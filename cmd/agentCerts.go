package cmd

import (
	"github.com/spf13/cobra"
)

var agentCertsCmd = &cobra.Command{
	Use:   "certs",
	Short: "Interact with certificates",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentCertsCmd)
}
