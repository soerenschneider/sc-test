package cmd

import (
	"github.com/spf13/cobra"
)

var k8sBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Interact with Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	k8sCmd.AddCommand(k8sBackupCmd)

	k8sBackupCmd.PersistentFlags().StringP(k8sNamespace, "n", "", "Namespace")
}
