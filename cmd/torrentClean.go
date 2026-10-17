package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/transmission"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var torrentCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove finished torrents from the Transmission client",
	Long: `This command removes finished torrents from the Transmission client using the Transmission API.

Examples:
  sc torrent clean`,
	Run: func(cmd *cobra.Command, args []string) {
		addr := pkg.GetString(cmd, torrentAddr)
		client, err := transmission.NewClient(addr)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build transmission client")
		}

		var torrents transmission.Torrents
		{
			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			var err error
			torrents, err = client.GetTorrents(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("could not get torrents")
			}
		}

		if len(torrents) == 0 {
			log.Info().Msg("No torrents found")
			return
		}

		finishedTorrents := filterFinishedTorrents(torrents)
		if len(finishedTorrents) == 0 {
			log.Info().Msgf("No finished torrents found, %d active torrents", len(torrents))
		}

		finishedTorrentIds := make([]int64, 0, len(finishedTorrents))
		for _, torrent := range finishedTorrents {
			finishedTorrentIds = append(finishedTorrentIds, *torrent.ID)
		}

		{
			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			if err := client.RemoveTorrent(ctx, finishedTorrentIds, false); err != nil {
				log.Fatal().Err(err).Msg("could not delete torrents")
			}
		}

		log.Info().Msgf("Removed %d finished torrents", len(finishedTorrentIds))
	},
}

func init() {
	torrentCmd.AddCommand(torrentCleanCmd)
}

func filterFinishedTorrents(torrents []transmission.Torrent) []transmission.Torrent {
	ret := make([]transmission.Torrent, 0, len(torrents))
	for _, torrent := range torrents {
		if torrent.PercentDone != nil && *torrent.PercentDone == 1 {
			ret = append(ret, torrent)
		}
	}
	return ret
}
