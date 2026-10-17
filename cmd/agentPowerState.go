package cmd

import (
	"github.com/spf13/cobra"
)

var agentPowerStateCmd = &cobra.Command{
	Use:   "power-state",
	Short: "Interacts with the power-state component to either shutdown or reboot a machine.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentPowerStateCmd)
}
