package cmd

import (
	"context"
	"fmt"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/transmission"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var torrentDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del"},
	Short:   "Delete torrents from the Transmission client",
	Long: `This command deletes one or more torrents from the Transmission client using the Transmission API.

You can specify one or more torrent IDs to delete. Optionally, you can also delete the associated downloaded data.

Examples:
  sc torrent delete --ids 1,2,3         # Delete torrents with IDs 1, 2, and 3
  sc torrent delete --ids 4 -d          # Delete torrent with ID 4 and its data`,
	Run: func(cmd *cobra.Command, args []string) {
		addr := pkg.GetString(cmd, torrentAddr)
		client, err := transmission.NewClient(addr)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build transmission client")
		}

		deleteData, isDeleteDataChanged := pkg.GetBool(cmd, torrentDeleteData)

		ids := pkg.GetInt64Array(cmd, torrentIds)
		if len(ids) == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			var torrents []transmission.Torrent
			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					var err error
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
				log.Info().Msg("No torrents available to delete")
				return
			}

			formattedTorrentsIdMap, formattedTorrents := ToListEntries(torrents)

			var selectedTorrents []string
			fields := []huh.Field{
				huh.NewMultiSelect[string]().
					Title("Torrents to delete").
					Options(huh.NewOptions(formattedTorrents...)...).
					Value(&selectedTorrents),
			}

			if !isDeleteDataChanged {
				fields = append(fields,
					huh.NewConfirm().
						Title("Trash downloaded data for torrents to delete?").
						Value(&deleteData),
				)
			}

			err := huh.NewForm(
				huh.NewGroup(
					fields...,
				),
			).Run()
			if err != nil {
				log.Fatal().Err(err).Msg("could not select torrents")
			}

			if len(selectedTorrents) == 0 {
				log.Info().Msg("No torrents selected for deletion")
				return
			}

			var errs error
			for _, value := range selectedTorrents {
				id, found := formattedTorrentsIdMap[value]
				if found {
					ids = append(ids, id)
				}
			}

			if errs != nil {
				log.Warn().Err(err).Msg("could not get id for all selections")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		if err := client.RemoveTorrent(ctx, ids, deleteData); err != nil {
			log.Fatal().Err(err).Msg("could not delete torrents")
		}
	},
}

func init() {
	torrentCmd.AddCommand(torrentDeleteCmd)

	torrentDeleteCmd.Flags().Int64Slice(torrentIds, nil, "ID(s) to delete")
	torrentDeleteCmd.Flags().BoolP(torrentDeleteData, "d", false, "Delete all data of a torrent")
}

func ToListEntries(torrents []transmission.Torrent) (map[string]int64, []string) {
	ret := make(map[string]int64, len(torrents))
	entries := make([]string, len(torrents))
	for idx, torrent := range torrents {
		entry := torrent.String()

		var id int64
		if torrent.ID != nil {
			id = *torrent.ID
		}

		ret[entry] = id
		entries[idx] = entry
	}
	return ret, entries
}

func ToListEntry(torrent transmission.Torrent) (string, int64) {
	var id int64
	if torrent.ID != nil {
		id = *torrent.ID
	}
	return fmt.Sprintf("%s (%d)", *torrent.Name, id), id
}
