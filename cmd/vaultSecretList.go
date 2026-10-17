package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var vaultKv2ListCmd = &cobra.Command{
	Use:     "ls [prefix]",
	Aliases: []string{"list"},
	Short:   "Browse Vault KV v2 secrets in a TUI",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := ""
		if len(args) == 1 {
			prefix = args[0]
		}

		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))
		mount := pkg.GetString(cmd, vaultMountPath)

		return BrowseWithStore(client, BrowseOptions{
			Mount:   mount,
			Prefix:  prefix,
			Timeout: vaultDefaultTimeout,
		})
	},
}

func init() {
	vaultSecretCmd.AddCommand(vaultKv2ListCmd)
}

// BrowseOptions configures BrowseWithStore.
type BrowseOptions struct {
	Mount   string
	Prefix  string
	Timeout time.Duration
}

// BrowseWithStore loops the browser and editor against a single store
// instance, so auth happens once and state survives between picks.
func BrowseWithStore(store vaultKv2Provider, opts BrowseOptions) error {
	if store == nil {
		return errors.New("nil vaultKv2Provider")
	}
	if opts.Mount == "" {
		return errors.New("mount is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}

	prefix := normalizePrefix(opts.Prefix)
	for {
		m := newBrowseModel(store, opts.Mount, prefix, opts.Timeout)
		if err := m.loadInitial(); err != nil {
			return err
		}
		out, err := tea.NewProgram(m).Run()
		if err != nil {
			return err
		}
		bm := out.(*browseModel)
		if bm.selectedPath == "" {
			return nil // user quit
		}

		editErr := RunWithStore(store, vaultKv2EditOptions{
			Mount:   opts.Mount,
			Path:    bm.selectedPath,
			Timeout: opts.Timeout,
		})
		if editErr != nil {
			fmt.Fprintln(os.Stderr, "edit:", editErr)
		}
		prefix = bm.prefix
	}
}

// ---------------------------------------------------------------------------
// path helpers
// ---------------------------------------------------------------------------

func normalizePrefix(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func parentPrefix(p string) string {
	p = strings.TrimSuffix(p, "/")
	if !strings.Contains(p, "/") {
		return ""
	}
	return p[:strings.LastIndex(p, "/")+1]
}

// ---------------------------------------------------------------------------
// model
// ---------------------------------------------------------------------------

type browseModel struct {
	store   vaultKv2Provider
	mount   string
	prefix  string
	timeout time.Duration

	items  []vault.Entry
	cursor int
	offset int

	filter   textinput.Model
	filterOn bool

	loading bool
	err     string
	status  string

	// huh form for the "new secret" prompt
	form    *huh.Form
	newName string

	// Output to BrowseWithStore.
	selectedPath string

	width, height int
}

func newBrowseModel(store vaultKv2Provider, mount, prefix string, timeout time.Duration) *browseModel {
	ti := textinput.New()
	ti.Placeholder = "filter"
	ti.Prompt = "/ "
	ti.CharLimit = 256
	return &browseModel{
		store:   store,
		mount:   mount,
		prefix:  prefix,
		timeout: timeout,
		filter:  ti,
	}
}

// ---------------------------------------------------------------------------
// store I/O
// ---------------------------------------------------------------------------

func (m *browseModel) loadInitial() error {
	items, err := m.list(m.prefix)
	if err != nil {
		return err
	}
	m.items = sortEntries(items)
	return nil
}

func (m *browseModel) list(prefix string) ([]vault.Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.store.List(ctx, m.mount, prefix)
}

func sortEntries(items []vault.Entry) []vault.Entry {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items
}

type listedMsg struct {
	prefix string
	items  []vault.Entry
	err    error
}

func (m *browseModel) loadCmd(prefix string) tea.Cmd {
	store, mount, timeout := m.store, m.mount, m.timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		items, err := store.List(ctx, mount, prefix)
		return listedMsg{prefix: prefix, items: sortEntries(items), err: err}
	}
}

// ---------------------------------------------------------------------------
// bubbletea
// ---------------------------------------------------------------------------

func (m *browseModel) Init() tea.Cmd { return nil }

func (m *browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case listedMsg:
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
	}

	// "New secret" prompt owns input while open.
	if m.form != nil {
		f, cmd := m.form.Update(msg)
		if ff, ok := f.(*huh.Form); ok {
			m.form = ff
		}
		switch m.form.State {
		case huh.StateCompleted:
			name := strings.TrimSpace(m.newName)
			m.form = nil
			if name != "" {
				m.selectedPath = m.prefix + name
				return m, tea.Quit
			}
			return m, cmd
		case huh.StateAborted:
			m.form = nil
			m.newName = ""
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

func (m *browseModel) filteredItems() []vault.Entry {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		return m.items
	}
	out := make([]vault.Entry, 0, len(m.items))
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Name), q) {
			out = append(out, it)
		}
	}
	return out
}

func (m *browseModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.filteredItems()

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.cursor, m.offset = 0, 0
		}
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
		m.selectedPath = m.prefix + it.Name
		return m, tea.Quit
	case "n":
		return m, m.openNewForm()
	}
	return m, nil
}

func (m *browseModel) openNewForm() tea.Cmd {
	m.newName = ""
	existing := map[string]bool{}
	for _, it := range m.items {
		if !it.IsDir {
			existing[it.Name] = true
		}
	}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("New secret").
				Description(fmt.Sprintf("%s/%s<name>", m.mount, m.prefix)),
			huh.NewInput().
				Title("Name").
				Value(&m.newName).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return errors.New("name cannot be empty")
					}
					if strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") {
						return errors.New("name cannot start or end with '/'")
					}
					if existing[s] {
						return fmt.Errorf("a secret named %q already exists here", s)
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

func (m *browseModel) View() tea.View {
	// Top-level program — this method owns AltScreen state. Either path
	// (form or main list) wraps its content into a tea.View and sets
	// AltScreen on the way out.
	if m.form != nil {
		v := tea.NewView(m.form.View())
		v.AltScreen = true
		return v
	}

	var b strings.Builder

	b.WriteString(tui.TitleStyle.Render(" vault ls "))
	b.WriteString(" ")
	b.WriteString(tui.PathStyle.Render(m.mount + "/" + m.prefix))
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

	// Viewport: reserve a few lines for header, status, footer.
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
			name = tui.SecretStyle.Render(it.Name)
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
	dirs, secrets := countEntries(items)
	b.WriteString(tui.HelpStyle.Render(fmt.Sprintf("  %d folder(s) · %d secret(s)", dirs, secrets)))
	if m.filter.Value() != "" {
		td, ts := countEntries(m.items)
		b.WriteString(tui.HelpStyle.Render(fmt.Sprintf(" of %d/%d", td, ts)))
	}
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString(tui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n")
	} else if m.status != "" {
		b.WriteString(tui.StatusStyle.Render("· " + m.status))
		b.WriteString("\n")
	}

	b.WriteString(tui.HelpStyle.Render(
		"\n  ↑/↓ move  ·  enter open  ·  h/← up  ·  / filter  ·  n new  ·  r reload  ·  q quit",
	))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m *browseModel) adjustOffset(view, n int) {
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

func countEntries(items []vault.Entry) (dirs, secrets int) {
	for _, it := range items {
		if it.IsDir {
			dirs++
		} else {
			secrets++
		}
	}
	return
}
