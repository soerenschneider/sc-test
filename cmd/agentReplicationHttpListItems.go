package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

var agentReplicationHttpListItems = &cobra.Command{
	Use:   "list-items",
	Short: "Lists all items that describe a http replication",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		resp, err := client.ReplicationGetHttpItemsList(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		item := agent.MustResponse[api.ReplicationHttpItemsList](resp)
		fmt.PrintReplicationHttpItemsList(item.Data)
	},
}

func init() {
	agentReplicationHttpCmd.AddCommand(agentReplicationHttpListItems)
}
