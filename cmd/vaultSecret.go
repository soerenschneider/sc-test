package cmd

import "github.com/spf13/cobra"

var vaultSecretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"secrets", "kv2"},
	Short:   "Manages the Vault KV2 secret engine",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultSecretCmd)
	vaultSecretCmd.PersistentFlags().StringP(vaultMountPath, "m", "secret", "Path where the KV2 secret engine is mounted")
}
