package transmission

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type torrentItem Torrent

func (t torrentItem) Title() string       { return "" }
func (t torrentItem) Description() string { return "" }
func (t torrentItem) FilterValue() string { return *t.Name }

type torrentDelegate struct{}

func (d torrentDelegate) Height() int                               { return 1 }
func (d torrentDelegate) Spacing() int                              { return 0 }
func (d torrentDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d torrentDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	t := listItem.(torrentItem)

	pink := lipgloss.Color("205")
	style := lipgloss.NewStyle().Foreground(pink)

	id := style.Width(5).Render(fmt.Sprintf("%d", *t.ID))
	prog := style.Width(8).Render(fmt.Sprintf("%3.0f%%", *t.PercentDone*100))
	name := style.Bold(true).Render(*t.Name)

	fmt.Fprintf(w, "%s %s %s\n", id, prog, name)
}

func DisplayTorrents(torrents []Torrent) {
	var items []list.Item
	for _, t := range torrents {
		items = append(items, torrentItem(t))
	}

	l := list.New(items, torrentDelegate{}, 60, len(items)+4)
	l.Title = "Torrents"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	fmt.Println(l.View())
}
