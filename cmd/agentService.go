package cmd

import (
	"github.com/spf13/cobra"
)

var agentServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Interact with services component",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentServiceCmd)
}
