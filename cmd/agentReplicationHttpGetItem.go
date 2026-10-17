package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const agentReplicationHttpGetItemCmdFlagId = "id"

var agentReplicationHttpGetItemCmd = &cobra.Command{
	Use:   "get-item",
	Short: "Retrieves an item that describes a http replication",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		id, err := cmd.Flags().GetString(agentReplicationHttpGetItemCmdFlagId)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		resp, err := client.ReplicationGetHttpItem(context.Background(), id)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		item := agent.MustResponse[api.ReplicationHttpItem](resp)
		fmt.PrintReplicationHttpItem(*item)
	},
}

func init() {
	agentReplicationHttpCmd.AddCommand(agentReplicationHttpGetItemCmd)
	agentReplicationHttpGetItemCmd.Flags().String(agentReplicationHttpGetItemCmdFlagId, "", "The id of the replication item to return")
	if err := agentReplicationHttpGetItemCmd.MarkFlagRequired(agentReplicationHttpGetItemCmdFlagId); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
