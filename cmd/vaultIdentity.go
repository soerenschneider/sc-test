package cmd

import (
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultIdentityCmd = &cobra.Command{
	Use: "identity",
	Aliases: []string{
		"id",
	},
	Short: "Manage Vault identities",
	Long: `The 'identity' command group contains subcommands for interacting with Vault identities.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage identities.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultIdentityCmd)
}
