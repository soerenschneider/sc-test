package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

var agentReplicationSecretsListItemsCmd = &cobra.Command{
	Use:   "list-items",
	Short: "Lists all items that describe a secret replication",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		resp, err := client.ReplicationGetSecretsItemsList(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		items := agent.MustResponse[api.ReplicationSecretsItemsList](resp)
		fmt.PrintSecretReplicationItems(items.Data)
	},
}

func init() {
	agentSecretsReplicationCmd.AddCommand(agentReplicationSecretsListItemsCmd)
}
