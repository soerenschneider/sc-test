package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// BrowseResult describes the item that was displayed when the user quit the browser.
type BrowseResult struct {
	// Name is the name of the displayed item, empty if the user quit from the list.
	Name string
	// Content is the content of the displayed item.
	Content string
}

// Browse displays a filterable list of items. Selecting an item loads its content using load and
// displays it in a scrollable view, from which the user can go back to the list. If the user quits
// while viewing an item, the item is returned so it can be printed after the alt screen is left.
func Browse(ctx context.Context, title string, items []string, load func(ctx context.Context, name string) (string, error)) (BrowseResult, error) {
	m := newBrowseModel(ctx, title, items, load)
	out, err := tea.NewProgram(m, tea.WithContext(ctx)).Run()
	if err != nil {
		return BrowseResult{}, err
	}

	final := out.(*browseModel)
	if final.viewing == "" {
		return BrowseResult{}, nil
	}
	return BrowseResult{Name: final.viewing, Content: final.cache[final.viewing]}, nil
}

type browseLoadedMsg struct {
	name    string
	content string
	err     error
}

type browseModel struct {
	ctx  context.Context
	load func(ctx context.Context, name string) (string, error)

	list     list.Model
	viewport viewport.Model
	cache    map[string]string

	// viewing is the name of the item currently displayed, empty while the list is displayed.
	viewing string
	loading string
	status  string
	width   int
	height  int
}

func newBrowseModel(ctx context.Context, title string, names []string, load func(ctx context.Context, name string) (string, error)) *browseModel {
	items := make([]list.Item, len(names))
	for i, name := range names {
		items[i] = item(name)
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.SetStatusBarItemName("item", "items")
	// the list binds quitting to "v" by default
	l.KeyMap.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view"))}
	}

	vp := viewport.New()
	vp.SoftWrap = true

	return &browseModel{
		ctx:      ctx,
		load:     load,
		list:     l,
		viewport: vp,
		cache:    map[string]string{},
	}
}

func (m *browseModel) Init() tea.Cmd {
	return nil
}

func (m *browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-1)
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - 2)
		return m, nil

	case browseLoadedMsg:
		m.loading = ""
		if msg.err != nil {
			m.status = ErrorStyle.Render(fmt.Sprintf("could not load %q: %v", msg.name, msg.err))
			return m, nil
		}
		m.cache[msg.name] = msg.content
		m.show(msg.name)
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.viewing != "" {
			return m.updateDetail(msg)
		}
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			return m.selectItem()
		}
	}

	if m.viewing != "" {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *browseModel) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "backspace", "left", "h":
		m.viewing = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *browseModel) selectItem() (tea.Model, tea.Cmd) {
	selected, ok := m.list.SelectedItem().(item)
	if !ok || m.loading != "" {
		return m, nil
	}

	name := string(selected)
	if _, cached := m.cache[name]; cached {
		m.show(name)
		return m, nil
	}

	m.loading = name
	m.status = ""
	return m, func() tea.Msg {
		content, err := m.load(m.ctx, name)
		return browseLoadedMsg{name: name, content: content, err: err}
	}
}

func (m *browseModel) show(name string) {
	m.viewing = name
	m.status = ""
	m.viewport.SetContent(strings.TrimRight(m.cache[name], "\n"))
	m.viewport.GotoTop()
}

func (m *browseModel) View() tea.View {
	var content string
	if m.viewing != "" {
		header := TitleStyle.Render(m.viewing)
		help := HelpStyle.Render(fmt.Sprintf("%3.f%% • ↑/↓ scroll • esc back • q quit", m.viewport.ScrollPercent()*100))
		content = lipgloss.JoinVertical(lipgloss.Left, header, m.viewport.View(), help)
	} else {
		status := m.status
		if m.loading != "" {
			status = StatusStyle.Render(fmt.Sprintf("Loading %q...", m.loading))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, m.list.View(), status)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
