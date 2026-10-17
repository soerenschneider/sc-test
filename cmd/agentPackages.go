package cmd

import (
	"github.com/spf13/cobra"
)

var agentPackagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Interact with the package component",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentPackagesCmd)
}
