package cmd

import (
	"context"
	"fmt"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/transmission"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var torrentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List torrents in the Transmission client",
	Long: `This command lists all torrents currently managed by the Transmission client using the Transmission API.

It displays relevant information such as torrent ID, name, status, progress, and more.

Examples:
  sc torrent list     # List all torrents`,
	Run: func(cmd *cobra.Command, args []string) {
		addr := pkg.GetString(cmd, torrentAddr)
		client, err := transmission.NewClient(addr)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build transmission client")
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		var torrents transmission.Torrents
		if err := spinner.New().
			Type(spinner.Line).
			ActionWithErr(func(ctx context.Context) error {
				torrents, err = client.GetTorrents(ctx)
				return err
			}).
			Title("Fetching list of torrents...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not fetch list of torrents")
		}

		if len(torrents) == 0 {
			log.Info().Msg("No torrents available")
			return
		}

		headers, data := torrents.AsTable()

		tui.Table{
			Title:   "Torrents",
			Headers: headers,
			Rows:    data,
			Aligns:  []tui.Align{0, 0, tui.AlignRight},
			Caption: fmt.Sprintf("%d rows", len(data)),
			Zebra:   len(data) > 10,
		}.Print()
	},
}

func init() {
	torrentCmd.AddCommand(torrentListCmd)
}
