package cmd

import (
	"github.com/spf13/cobra"
)

var agentK0sCmd = &cobra.Command{
	Use:   "k0s",
	Short: "Interact with k0s",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentK0sCmd)
}
