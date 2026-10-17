package cmd

import (
	"github.com/spf13/cobra"
)

var vaultTransitCmd = &cobra.Command{
	Use:   "transit",
	Short: "Manage Vault transit secret engine",
	Long: `The 'transit' command group contains subcommands for interacting with the Vault transit secret engine.

This command itself does not perform any actions. Instead, use one of its subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultTransitCmd)
	vaultTransitCmd.PersistentFlags().StringP(vaultMountPath, "m", "", "Path where the transit secret engine is mounted. If not specified, available mounts are discovered.")
}
