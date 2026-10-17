package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const (
	scAgentProviderUrl = "provider"
	scAgentClientID    = "client-id"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with a remote sc-agent instance",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.PersistentFlags().String(agent.AgentCmdFlagsServer, "", "The endpoint of the server running sc-agent")
	if err := agentCmd.MarkPersistentFlagRequired(agent.AgentCmdFlagsServer); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}

	agentCmd.PersistentFlags().StringP(agent.AgentCmdFlagsCertFile, "c", "", "The file that contains the x509 client certificate to authenticate with")
	agentCmd.PersistentFlags().StringP(agent.AgentCmdFlagsKeyFile, "k", "", "The file that contains the x509 client key to authenticate with")
	agentCmd.PersistentFlags().String(agent.AgentCmdFlagsCaFile, "", "The file that contains the x509 ca certificate")
	agentCmd.PersistentFlags().BoolP(agent.AgentCmdFlagsVerbose, "v", false, "Increase verbosity of output")
}
