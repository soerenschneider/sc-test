// Generic prefix-tree TUI. Knows nothing about Vault or filesystems; data
// comes through a TreeProvider, and the UI behavior is configured per call.
//
// Two use modes:
//
//   - Standalone — TreeBrowser.Browse(ctx, startPrefix) runs the TUI in its
//     own tea.Program and returns when the user acts. Used by the Vault `ls`
//     command.
//
//   - Embedded — TreeBrowser.NewModel(startPrefix) returns a *TreeModel that
//     a parent bubbletea model can host as a sub-screen. The model never
//     calls tea.Quit itself; the parent polls TreeModel.Done() each tick and
//     swaps it out when the user is finished. Used by the editor's `E` key,
//     which keeps the editor running underneath while the user picks an fs
//     destination.

package picker

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
)

// ---------------------------------------------------------------------------
// public surface
// ---------------------------------------------------------------------------

// TreeNode is one item in a tree.
type TreeNode struct {
	Name  string
	IsDir bool
}

// TreeProvider is the data source the browser walks. Prefix is empty at the
// root; otherwise it ends with "/". Implementations should canonicalize empty
// prefix to whatever their "root" means (e.g. "/" for a filesystem).
type TreeProvider interface {
	Children(ctx context.Context, prefix string) ([]TreeNode, error)
}

// TreeMaker is an optional capability on a TreeProvider. When a provider
// implements it, the 'n' key creates the path in-place (mkdir-style) and
// the browser stays open, navigating into the new location. Providers that
// don't implement it still allow 'n' if AllowCreate is set, but the browser
// exits with ActionCreate and the consumer decides what "create" means in
// its context — e.g. for Vault, opening the editor at the new path (the
// secret only materializes once the user saves).
//
// The full path passed to MakePath is the same string the browser would
// report in TreeResult: m.prefix + the user-typed name, slashes included.
type TreeMaker interface {
	MakePath(ctx context.Context, fullPath string) error
}

// TreeAction is what the user did when they exited the browser.
type TreeAction int

const (
	// ActionQuit: user pressed q/esc/ctrl+c. Prefix/Name are not meaningful.
	ActionQuit TreeAction = iota
	// ActionOpen: user selected an existing leaf with enter.
	ActionOpen
	// ActionSaveAs: user picked a new name via 's' in a directory.
	ActionSaveAs
	// ActionCreate: user picked a new name via 'n' in a directory.
	ActionCreate
)

// TreeResult is the user's choice on exit.
type TreeResult struct {
	Action TreeAction
	Prefix string // directory the user was at
	Name   string // leaf they selected or typed (empty for ActionQuit)
}

// TreeBrowser configures the TUI.
type TreeBrowser struct {
	Provider    TreeProvider
	Title       string // shown in title bar, e.g. "vault ls", "export"
	LeafLabel   string // singular noun, e.g. "secret", "file"
	DirLabel    string // singular noun, e.g. "folder", "directory"
	AllowCreate bool   // enables 'n' key — create new leaf in current prefix
	AllowSaveAs bool   // enables 's' key — pick a destination name
	Timeout     time.Duration
}

// Browse runs the TUI as a standalone bubbletea program. Returns when the
// user acts. The caller can call Browse again with the returned Prefix to
// resume at the same location.
func (b *TreeBrowser) Browse(ctx context.Context, startPrefix string) (*TreeResult, error) {
	m, err := b.NewModel(startPrefix)
	if err != nil {
		return nil, err
	}
	shim := &standaloneShim{m: m}
	out, err := tea.NewProgram(shim).Run()
	if err != nil {
		return nil, err
	}
	final := out.(*standaloneShim).m
	return final.Result(), nil
}

// NewModel constructs a TreeModel for embedding inside another bubbletea
// model. It loads the initial prefix synchronously so the first frame is
// consistent.
func (b *TreeBrowser) NewModel(startPrefix string) (*TreeModel, error) {
	m := newTreeModel(b, startPrefix)
	if err := m.loadInitial(); err != nil {
		return nil, err
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// model
// ---------------------------------------------------------------------------

// TreeModel is the bubbletea model. Exported so it can be embedded.
type TreeModel struct {
	cfg    *TreeBrowser
	prefix string

	items  []TreeNode
	cursor int
	offset int

	filter   textinput.Model
	filterOn bool

	loading bool
	err     string
	status  string

	form       *huh.Form
	newName    string
	formAction TreeAction

	// Output state — read by parent (embedded) or by the standalone shim.
	done   bool
	action TreeAction
	name   string

	width, height int
}

func newTreeModel(cfg *TreeBrowser, prefix string) *TreeModel {
	ti := textinput.New()
	ti.Placeholder = "filter"
	ti.Prompt = "/ "
	ti.CharLimit = 256
	return &TreeModel{
		cfg:    cfg,
		prefix: prefix,
		filter: ti,
	}
}

// Done reports whether the user has finished. Once true, Result() is valid
// and the model will not transition further on input.
func (m *TreeModel) Done() bool { return m.done }

// Result returns the user's choice. Only call when Done() is true.
func (m *TreeModel) Result() *TreeResult {
	return &TreeResult{
		Action: m.action,
		Prefix: m.prefix,
		Name:   m.name,
	}
}

func (m *TreeModel) loadInitial() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.Timeout)
	defer cancel()
	items, err := m.cfg.Provider.Children(ctx, m.prefix)
	if err != nil {
		return err
	}
	m.items = sortTreeNodes(items)
	return nil
}

type treeListedMsg struct {
	prefix string
	items  []TreeNode
	err    error
}

func (m *TreeModel) loadCmd(prefix string) tea.Cmd {
	provider, timeout := m.cfg.Provider, m.cfg.Timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		items, err := provider.Children(ctx, prefix)
		return treeListedMsg{prefix: prefix, items: sortTreeNodes(items), err: err}
	}
}

// treeCreatedMsg delivers the outcome of an in-place create. On success we
// navigate into the new path (so the user lands inside the directory they
// just made); on error we stay at the current prefix and surface the error.
type treeCreatedMsg struct {
	targetPrefix string     // prefix to land at after the create
	items        []TreeNode // children of targetPrefix
	fullPath     string     // for the success status line
	err          error
}

func (m *TreeModel) makePathCmd(maker TreeMaker, name string) tea.Cmd {
	prefix, provider, timeout := m.prefix, m.cfg.Provider, m.cfg.Timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		full := prefix + name
		if err := maker.MakePath(ctx, full); err != nil {
			return treeCreatedMsg{err: err, fullPath: full}
		}
		// Navigate into the newly created path; fall back to the current
		// prefix if it isn't navigable for some reason (e.g. the path was
		// already a non-directory file).
		target := full
		if !strings.HasSuffix(target, "/") {
			target += "/"
		}
		items, err := provider.Children(ctx, target)
		if err != nil {
			items, _ = provider.Children(ctx, prefix)
			return treeCreatedMsg{
				targetPrefix: prefix,
				items:        sortTreeNodes(items),
				fullPath:     full,
			}
		}
		return treeCreatedMsg{
			targetPrefix: target,
			items:        sortTreeNodes(items),
			fullPath:     full,
		}
	}
}

func (m *TreeModel) Init() tea.Cmd { return nil }

func (m *TreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		// Already finished — ignore further input so the parent can
		// safely defer the swap-out to its own tick.
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case treeListedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = vault.HumanizeError(msg.err, "list")
			return m, nil
		}
		m.err = ""
		m.prefix = msg.prefix
		m.items = msg.items
		m.cursor, m.offset = 0, 0
		m.filter.SetValue("")
		return m, nil
	case treeCreatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = vault.HumanizeError(msg.err, "create")
			m.status = ""
			return m, nil
		}
		m.err = ""
		m.prefix = msg.targetPrefix
		m.items = msg.items
		m.cursor, m.offset = 0, 0
		m.filter.SetValue("")
		m.status = "created " + msg.fullPath
		return m, nil
	}

	// Name-prompt form (create / save-as) owns input while open.
	if m.form != nil {
		if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "esc" {
			m.form = nil
			return m, nil
		}
		f, cmd := m.form.Update(msg)
		if ff, ok := f.(*huh.Form); ok {
			m.form = ff
		}
		switch m.form.State {
		case huh.StateCompleted:
			name := strings.TrimSpace(m.newName)
			m.form = nil
			if name == "" {
				return m, cmd
			}
			// In-place create when the provider supports it (fs mkdir);
			// otherwise emit ActionCreate and let the consumer handle it
			// (the vault `ls` flow opens the editor at the new path).
			if m.formAction == ActionCreate {
				if maker, ok := m.cfg.Provider.(TreeMaker); ok {
					m.loading = true
					m.status = "creating " + m.prefix + name + "…"
					return m, m.makePathCmd(maker, name)
				}
			}
			m.action = m.formAction
			m.name = name
			m.done = true
			return m, cmd
		case huh.StateAborted:
			m.form = nil
			return m, cmd
		}
		return m, cmd
	}

	// Filter input takes most keys when active.
	if m.filterOn {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "esc":
				m.filterOn = false
				m.filter.SetValue("")
				m.filter.Blur()
				m.cursor, m.offset = 0, 0
				return m, nil
			case "enter":
				m.filterOn = false
				m.filter.Blur()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.cursor, m.offset = 0, 0
		return m, cmd
	}

	if km, ok := msg.(tea.KeyPressMsg); ok {
		return m.handleKey(km)
	}
	return m, nil
}

func (m *TreeModel) filteredItems() []TreeNode {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		return m.items
	}
	out := make([]TreeNode, 0, len(m.items))
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Name), q) {
			out = append(out, it)
		}
	}
	return out
}

func (m *TreeModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.filteredItems()
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.action = ActionQuit
		m.done = true
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor, m.offset = 0, 0
	case "G", "end":
		m.cursor = tui.Max0(len(items) - 1)
	case "/":
		m.filterOn = true
		return m, m.filter.Focus()
	case "r":
		m.loading = true
		m.status = "reloading…"
		return m, m.loadCmd(m.prefix)
	case "backspace", "h", "left", "-":
		if m.prefix == "" {
			return m, nil
		}
		m.loading = true
		return m, m.loadCmd(parentPrefix(m.prefix))
	case "enter", "right", "l":
		if len(items) == 0 {
			return m, nil
		}
		it := items[m.cursor]
		if it.IsDir {
			m.loading = true
			return m, m.loadCmd(m.prefix + it.Name + "/")
		}
		m.action = ActionOpen
		m.name = it.Name
		m.done = true
		return m, nil
	case "n":
		if m.cfg.AllowCreate {
			return m, m.openNamePrompt(ActionCreate, "New "+m.cfg.LeafLabel)
		}
	case "s":
		if m.cfg.AllowSaveAs {
			return m, m.openNamePrompt(ActionSaveAs, "Save here as")
		}
	}
	return m, nil
}

func (m *TreeModel) openNamePrompt(action TreeAction, title string) tea.Cmd {
	m.newName = ""
	m.formAction = action

	existing := map[string]bool{}
	for _, it := range m.items {
		if !it.IsDir {
			existing[it.Name] = true
		}
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(title).
				Description(fmt.Sprintf("in %s — esc to cancel", displayPrefix(m.prefix))),
			huh.NewInput().
				Title("Name").
				Value(&m.newName).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return errors.New("name cannot be empty")
					}
					if strings.HasPrefix(s, "/") {
						return errors.New("path is relative to the current location; don't start with '/'")
					}
					// SaveAs picks a single filename in the current
					// directory; nested paths don't fit. Create allows
					// arbitrary subpaths — vault folders are implicit
					// and a TreeMaker provider does mkdir -p.
					if action == ActionSaveAs && strings.Contains(s, "/") {
						return errors.New("name cannot contain '/' (use 'n' to create a nested path)")
					}
					for _, seg := range strings.Split(s, "/") {
						if seg == ".." {
							return errors.New("'..' segments not allowed")
						}
					}
					// Duplicate check only meaningful for a simple
					// leaf in the current directory.
					if action == ActionCreate && !strings.Contains(s, "/") && existing[s] {
						return fmt.Errorf("a %s named %q already exists here", m.cfg.LeafLabel, s)
					}
					return nil
				}),
		),
	).WithShowHelp(true).WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	return m.form.Init()
}

// ---------------------------------------------------------------------------
// view
// ---------------------------------------------------------------------------

// View returns the rendered tree. Note: this method does NOT set AltScreen
// or other terminal-state fields on the returned tea.View — those are the
// parent's concern. For standalone use the standaloneShim wraps and adds
// them; for embedded use the parent model does the same.
func (m *TreeModel) View() tea.View {
	if m.form != nil {
		return tea.NewView(m.form.View())
	}

	var b strings.Builder
	b.WriteString(tui.TitleStyle.Render(" " + m.cfg.Title + " "))
	b.WriteString(" ")
	b.WriteString(tui.PathStyle.Render(displayPrefix(m.prefix)))
	if m.loading {
		b.WriteString(" ")
		b.WriteString(tui.MetaStyle.Render("· loading"))
	}
	b.WriteString("\n")

	if m.filterOn || m.filter.Value() != "" {
		b.WriteString(m.filter.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")

	items := m.filteredItems()

	view := m.height - 6
	if view < 5 {
		view = 5
	}
	m.adjustOffset(view, len(items))

	if len(items) == 0 {
		if m.filter.Value() != "" {
			b.WriteString(tui.HelpStyle.Render("  (no matches)\n"))
		} else {
			b.WriteString(tui.HelpStyle.Render("  (empty)\n"))
		}
	}

	end := m.offset + view
	if end > len(items) {
		end = len(items)
	}
	if m.offset > 0 {
		b.WriteString(tui.HelpStyle.Render(fmt.Sprintf("  ↑ %d more\n", m.offset)))
	}
	for i := m.offset; i < end; i++ {
		it := items[i]
		marker := "  "
		var name string
		if it.IsDir {
			name = tui.DirStyle.Render(it.Name + "/")
		} else {
			name = tui.LeafStyle.Render(it.Name)
		}
		if i == m.cursor {
			marker = tui.SelectedStyle.Render("▸ ")
			if it.IsDir {
				name = tui.SelectedStyle.Render(it.Name + "/")
			} else {
				name = tui.SelectedStyle.Render(it.Name)
			}
		}
		b.WriteString(marker)
		b.WriteString(name)
		b.WriteString("\n")
	}
	if end < len(items) {
		b.WriteString(tui.HelpStyle.Render(fmt.Sprintf("  ↓ %d more\n", len(items)-end)))
	}

	b.WriteString("\n")
	dirs, leaves := countTreeNodes(items)
	b.WriteString(tui.HelpStyle.Render(fmt.Sprintf("  %d %s · %d %s",
		dirs, plural(dirs, m.cfg.DirLabel),
		leaves, plural(leaves, m.cfg.LeafLabel))))
	if m.filter.Value() != "" {
		td, tl := countTreeNodes(m.items)
		b.WriteString(tui.HelpStyle.Render(fmt.Sprintf(" of %d/%d", td, tl)))
	}
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString(tui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n")
	} else if m.status != "" {
		b.WriteString(tui.StatusStyle.Render("· " + m.status))
		b.WriteString("\n")
	}

	foot := "  ↑/↓ move  ·  enter open  ·  h/← up  ·  / filter  ·  r reload"
	if m.cfg.AllowCreate {
		foot += "  ·  n new"
	}
	if m.cfg.AllowSaveAs {
		foot += "  ·  s save as"
	}
	foot += "  ·  esc cancel"
	b.WriteString(tui.HelpStyle.Render("\n" + foot))
	return tea.NewView(b.String())
}

func (m *TreeModel) adjustOffset(view, n int) {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+view {
		m.offset = m.cursor - view + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	if m.offset > tui.Max0(n-view) {
		m.offset = tui.Max0(n - view)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// standaloneShim wraps a TreeModel as a top-level bubbletea program, issuing
// tea.Quit when the model signals done.
type standaloneShim struct{ m *TreeModel }

func (s *standaloneShim) Init() tea.Cmd { return s.m.Init() }
func (s *standaloneShim) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newM, cmd := s.m.Update(msg)
	if tm, ok := newM.(*TreeModel); ok {
		s.m = tm
	}
	if s.m.Done() {
		return s, tea.Quit
	}
	return s, cmd
}

// View returns the underlying TreeModel's content with AltScreen turned on.
// This is the standalone case — when the browser is run via Browse() — so
// the program needs the alt-screen state set somewhere; the shim is the
// natural home for it.
func (s *standaloneShim) View() tea.View {
	v := s.m.View()
	v.AltScreen = true
	return v
}

func sortTreeNodes(items []TreeNode) []TreeNode {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func countTreeNodes(items []TreeNode) (dirs, leaves int) {
	for _, it := range items {
		if it.IsDir {
			dirs++
		} else {
			leaves++
		}
	}
	return
}

func plural(n int, label string) string {
	if n == 1 {
		return label
	}
	return label + "s"
}

// parentPrefix returns the parent of p, or "" for root.
func parentPrefix(p string) string {
	p = strings.TrimSuffix(p, "/")
	if !strings.Contains(p, "/") {
		return ""
	}
	return p[:strings.LastIndex(p, "/")+1]
}

// displayPrefix returns a user-facing rendering of a prefix. Empty becomes
// "/" so the user always sees a slash.
func displayPrefix(p string) string {
	if p == "" {
		return "/"
	}
	return p
}
