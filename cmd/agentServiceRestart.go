package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentServiceRestartFlagUnit = "unit"

var agentServiceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart a service unit",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		unit, err := cmd.Flags().GetString(agentServiceRestartFlagUnit)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		params := &api.ServicesUnitStatusPutParams{
			Action: api.Restart,
		}
		resp, err := client.ServicesUnitStatusPut(context.Background(), unit, params)
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
	agentServiceCmd.AddCommand(agentServiceRestartCmd)
	agentServiceRestartCmd.Flags().String(agentServiceRestartFlagUnit, "", "The unit to restart")
	if err := agentServiceRestartCmd.MarkFlagRequired(agentServiceRestartFlagUnit); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
