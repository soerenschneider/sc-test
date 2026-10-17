package fmt

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/soerenschneider/sc-agent/pkg/api"
)

func RenderPackagesInstalledCmd(data *api.PackagesInstalled) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Version", "Repo"})
	for _, person := range data.Packages {
		t.AppendRow(table.Row{person.Name, person.Version, person.Repo})
	}
	t.Render()
}

func RenderPackagesUpdateableCmd(data *api.PackageUpdates) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Version", "Repo"})
	for _, pckg := range data.UpdatablePackages {
		t.AppendRow(table.Row{pckg.Name, pckg.Version, pckg.Repo})
	}
	t.Render()
}
