package cmd

import (
	"github.com/spf13/cobra"
)

const (
	passOtpPrefix = "otp-prefix"
	passOtp       = "otp"
	passShow      = "show"
)

var passCmd = &cobra.Command{
	Use:   "pass",
	Short: "Interacts with the pass unix password store",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(passCmd)
}
