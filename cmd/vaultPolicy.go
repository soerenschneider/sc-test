package cmd

import (
	"github.com/spf13/cobra"
)

var vaultPolicyCmd = &cobra.Command{
	Use:     "policy",
	Aliases: []string{"policies", "pol"},
	Short:   "Inspect Vault ACL policies",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultPolicyCmd)
}
