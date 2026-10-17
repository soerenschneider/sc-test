package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentServiceStartFlagUnit = "unit"

var agentServiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a service unit",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		unit, err := cmd.Flags().GetString(agentServiceStartFlagUnit)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		params := &api.ServicesUnitStatusPutParams{
			Action: api.Start,
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
	agentServiceCmd.AddCommand(agentServiceStartCmd)
	agentServiceStartCmd.Flags().String(agentServiceStartFlagUnit, "", "The unit to start")
	if err := agentServiceStartCmd.MarkFlagRequired(agentServiceStartFlagUnit); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
