package cmd

import (
	"github.com/spf13/cobra"
)

var agentLibvirtCmd = &cobra.Command{
	Use:   "libvirt",
	Short: "Interact with libvirt",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	agentCmd.AddCommand(agentLibvirtCmd)
}
