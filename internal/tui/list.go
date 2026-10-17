package tui

import (
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	cursor := "  "
	if index == m.Index() {
		cursor = "➜ "
	}
	fmt.Fprintf(w, "%s%s\n", cursor, i.Title())
}

type model struct {
	list list.Model
}

func newModel(strItems []string, readonly bool) model {
	extraHeight := 2 // title
	maxTotalHeight := 10 + extraHeight

	items := make([]list.Item, len(strItems))
	for i, s := range strItems {
		items[i] = item(s)
	}

	// Determine how much extra space is needed (title + pagination)
	paginate := false
	if len(items) > maxTotalHeight-1 {
		paginate = true
		extraHeight++
	}

	visibleItems := len(items)
	if visibleItems > maxTotalHeight-extraHeight {
		visibleItems = maxTotalHeight - extraHeight
	}

	// Total height = visible items + extras
	totalHeight := visibleItems + extraHeight

	l := list.New(items, itemDelegate{}, 40, totalHeight)
	l.Title = "Smart List"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(paginate)
	l.SetShowHelp(false)

	return model{list: l}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(m.list.View())
}

func DisplayList(items []string, readonly bool) {
	m := newModel(items, readonly)

	if readonly {
		fmt.Println(m.list.View())
		os.Exit(0)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
