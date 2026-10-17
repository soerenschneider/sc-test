package cmd

import (
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultAwsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Manage AWS secret engine",
	Long: `The 'aws' command group contains subcommands for interacting with Vault AWS secret engine.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultAwsCmd)
	vaultAwsCmd.PersistentFlags().StringP(vaultMountPath, "m", "", "Path where the AWS secret engine is mounted. If not specified, available mounts are discovered.")
}
