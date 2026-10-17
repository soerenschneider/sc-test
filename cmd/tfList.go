package cmd

import (
	"github.com/soerenschneider/sc/internal/tf"
	"github.com/spf13/cobra"
)

var tfListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the environments and their stacks",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		tf.List(tfRepo(cmd))
	},
}

func init() {
	tfCmd.AddCommand(tfListCmd)
}
