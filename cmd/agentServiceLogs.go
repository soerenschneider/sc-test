package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const serviceLogsFlagUnit = "unit"

var agentServiceLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Retrieve logs of a service unit",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		unit, err := cmd.Flags().GetString(serviceLogsFlagUnit)
		if err != nil {
			log.Fatal().Err(err).Msg("could net get flag")
		}

		resp, err := client.ServicesUnitLogsGet(context.Background(), unit)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.ServiceLogsData](resp)
		fmt.RenderServiceLogsCmd(data)
	},
}

func init() {
	agentServiceCmd.AddCommand(agentServiceLogsCmd)
	agentServiceLogsCmd.Flags().String(serviceLogsFlagUnit, "", "The unit to request the logs for")
	if err := agentServiceLogsCmd.MarkFlagRequired(serviceLogsFlagUnit); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
