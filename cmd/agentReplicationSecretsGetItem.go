package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const agentReplicationSecretsGetItemCmdFlagId = "id"

var agentReplicationSecretsGetItemCmd = &cobra.Command{
	Use:   "get-item",
	Short: "Retrieves an item that describes a secret replication",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		id, err := cmd.Flags().GetString(agentReplicationSecretsGetItemCmdFlagId)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		resp, err := client.ReplicationGetSecretsItem(context.Background(), id)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		item := agent.MustResponse[api.ReplicationSecretsItem](resp)
		fmt.PrintSecretReplicationItem(*item)
	},
}

func init() {
	agentSecretsReplicationCmd.AddCommand(agentReplicationSecretsGetItemCmd)
	agentReplicationSecretsGetItemCmd.Flags().String(agentReplicationSecretsGetItemCmdFlagId, "", "The id of the replication item to return")
	if err := agentReplicationSecretsGetItemCmd.MarkFlagRequired(agentReplicationSecretsGetItemCmdFlagId); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
