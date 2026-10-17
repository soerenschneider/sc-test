package cmd

import (
	"fmt"

	"github.com/soerenschneider/sc/internal"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(internal.BuildVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
