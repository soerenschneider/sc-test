package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentWolWakeupCmdFlagAlias = "alias"

var agentWolWakeupCmd = &cobra.Command{
	Use:   "wakeup",
	Short: "Wakes up another computer via wake-on-lan",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		alias, err := cmd.Flags().GetString(agentWolWakeupCmdFlagAlias)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		resp, err := client.WolPostMessage(context.Background(), alias)
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
	agentWolCmd.AddCommand(agentWolWakeupCmd)
	agentWolWakeupCmd.Flags().String(agentWolWakeupCmdFlagAlias, "", "The configured nice-name of the MAC address you want to wake up")
	if err := agentWolWakeupCmd.MarkFlagRequired(agentWolWakeupCmdFlagAlias); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
