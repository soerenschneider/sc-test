package cmd

import (
	"github.com/spf13/cobra"
)

var agentWolCmd = &cobra.Command{
	Use:   "wol",
	Short: "Interacts with the wake-on-lan component",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentWolCmd)
}
