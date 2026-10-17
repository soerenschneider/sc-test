package cmd

import (
	"github.com/spf13/cobra"
)

var vaultPkiCmd = &cobra.Command{
	Use:   "pki",
	Short: "Manages the Vault PKI secret engine",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultPkiCmd)
	vaultPkiCmd.PersistentFlags().StringP(vaultMountPath, "m", "", "Path where the PKI secret engine is mounted. If not specified, available mounts are discovered.")
}
