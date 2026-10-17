package cmd

import (
	"github.com/spf13/cobra"
)

const (
	torrentAddr           = "address"
	torrentIds            = "id"
	torrentDeleteData     = "delete-data"
	torrentMagnet         = "magnet"
	torrentNoConfirmation = "no-confirm"
)

var torrentCmd = &cobra.Command{
	Use:   "torrent",
	Short: "Commands for interacting with the Transmission torrent client",
	Long: `This command serves as the entry point for managing torrents via the Transmission API.

Use one of the available subcommands to interact with the Transmission client for tasks
such as adding new torrents, deleting existing ones, or listing current torrent activity.

Examples:
  sc torrent add -m "magnet:?xt=urn:btih:..."
  sc torrent list               # List all current torrents
  sc torrent delete <id>        # Delete a torrent by ID`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(torrentCmd)

	torrentCmd.PersistentFlags().StringP(torrentAddr, "a", "", "Transmission address.")
	_ = torrentCmd.MarkPersistentFlagRequired(torrentAddr)
}
