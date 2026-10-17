package cmd

import (
	"github.com/spf13/cobra"
)

var k8sRenovateCmd = &cobra.Command{
	Use:   "renovate",
	Short: "Interact with Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	k8sCmd.AddCommand(k8sRenovateCmd)

	k8sRenovateCmd.PersistentFlags().StringP(k8sNamespace, "n", "renovate", "Namespace")
}
