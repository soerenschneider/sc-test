package cmd

import (
	"github.com/spf13/cobra"
)

var agentCertsSshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Manage ssh certificates",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCertsCmd.AddCommand(agentCertsSshCmd)
}
