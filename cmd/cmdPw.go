package cmd

import (
	"github.com/spf13/cobra"
)

var pwCmd = &cobra.Command{
	Use:   "pw",
	Short: "Generate secure passwords",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(pwCmd)
}
