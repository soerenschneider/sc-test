package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	table "github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

var agentPackagesUpdateableCmd = &cobra.Command{
	Use:   "updateable",
	Short: "Show which packages are updatable",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		resp, err := client.PackagesUpdatesGet(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.PackageUpdates](resp)
		if !data.UpdatesAvailable {
			fmt.Println("No packages marked for update")
		} else {
			table.RenderPackagesUpdateableCmd(data)
		}
	},
}

func init() {
	agentPackagesCmd.AddCommand(agentPackagesUpdateableCmd)
}
