package cmd

import (
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Taskwarrior helpers",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
}
