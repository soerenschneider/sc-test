package cmd

import (
	"github.com/spf13/cobra"
)

var agentReplicationHttpCmd = &cobra.Command{
	Use:   "http",
	Short: "Replicates files from a HTTPS source",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentReplicationCmd.AddCommand(agentReplicationHttpCmd)
}
