package cmd

import (
	"github.com/spf13/cobra"
)

var agentSecretsReplicationCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Replicates secrets from Hashicorp Vault",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentSecretsReplicationCmd)
}
