package cmd

import (
	"github.com/spf13/cobra"
)

var agentCertsX509Cmd = &cobra.Command{
	Use:   "x509",
	Short: "Manage x509 certificates",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCertsCmd.AddCommand(agentCertsX509Cmd)
}
