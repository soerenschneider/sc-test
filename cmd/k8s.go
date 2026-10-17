package cmd

import (
	"github.com/spf13/cobra"
)

const (
	k8sNamespace = "namespace"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Interact with Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(k8sCmd)
}
