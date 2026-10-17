package fmt

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/soerenschneider/sc-agent/pkg/api"
)

func RenderServiceLogsCmd(data *api.ServiceLogsData) {
	t := table.NewWriter()
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 240},
	})
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Log"})
	for _, person := range data.Data.Logs {
		t.AppendRow(table.Row{person})
	}
	t.Render()
}
