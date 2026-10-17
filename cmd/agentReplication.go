package cmd

import (
	"github.com/spf13/cobra"
)

var agentReplicationCmd = &cobra.Command{
	Use:   "replication",
	Short: "Interacts with the replication component",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentReplicationCmd)
}
