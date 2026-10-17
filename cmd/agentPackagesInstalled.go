package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

var agentPackagesInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List all installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		resp, err := client.PackagesInstalledGet(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.PackagesInstalled](resp)
		fmt.RenderPackagesInstalledCmd(data)
	},
}

func init() {
	agentPackagesCmd.AddCommand(agentPackagesInstalledCmd)
}
