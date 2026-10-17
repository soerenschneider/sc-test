package cmd

import (
	"github.com/spf13/cobra"
)

var vaultSshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Manages the Vault SSH secret engine",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultSshCmd)
	vaultSshCmd.PersistentFlags().StringP(vaultMountPath, "m", "", "Path where the SSH secret engine is mounted. If not specified, available mounts are discovered.")
}
