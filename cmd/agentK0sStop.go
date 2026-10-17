package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

var agentK0sStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop k0s",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		params := &api.K0sPostActionParams{
			Action: api.K0sPostActionParamsActionStop,
		}
		resp, err := client.K0sPostAction(context.Background(), params)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		_ = agent.MustResponse[any](resp)
	},
}

func init() {
	agentK0sCmd.AddCommand(agentK0sStopCmd)
}
