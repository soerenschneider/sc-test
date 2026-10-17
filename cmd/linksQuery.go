package cmd

import (
	"context"
	"fmt"
	"time"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/linkding"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var linksQueryCmd = &cobra.Command{
	Use:     "query",
	Aliases: []string{"q"},
	Short:   "Interact with VictoriaLogs for querying and managing links",
	Long: `The 'links' command is the entry point for interacting with VictoriaLogs.

Use subcommands to query, tail, or manage links stored in VictoriaLogs, a high-performance log database.

Examples:
  sc links query -q='level="error"'   # Query links with specific filters`,
	Run: func(cmd *cobra.Command, args []string) {
		address := pkg.GetString(cmd, linksAddr)
		query := pkg.GetString(cmd, linksQuery)
		token, err := getLinkdingToken(cmd)
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting linkding token")
		}

		limit := 500

		client, err := linkding.NewLinkdingClient(address, token)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build client")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if query == "" && !cmd.Flags().Changed(linksQuery) {
			var tags []linkding.Tag
			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					var err error
					tags, err = client.GetAllTags(ctx)
					return err
				}).
				Title("Receiving tags...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("could not get tags")
			}

			normalizedTags := make([]string, len(tags))
			for idx, tag := range tags {
				normalizedTags[idx] = fmt.Sprintf("#%s", tag.Name)
			}

			noValidation := func(input string) error {
				return nil
			}
			query = tui.ReadInputWithValidation("Query", normalizedTags, noValidation)
		}

		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var bookmarks []linkding.Bookmark

		if err := spinner.New().
			Type(spinner.Line).
			ActionWithErr(func(ctx context.Context) error {
				var err error
				bookmarks, err = client.ListBookmarks(ctx, query, limit)
				return err
			}).
			Title("Receiving links data...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not get links")
		}

		headers, tableData := linkding.BookmarksAsTable(bookmarks)

		tui.Table{
			Title:     "Bookmarks",
			Headers:   headers,
			Rows:      tableData,
			Aligns:    []tui.Align{0, 0, tui.AlignRight}, // amount right-aligned
			Caption:   fmt.Sprintf("%d rows", len(tableData)),
			Zebra:     len(tableData) > 10,
			MaxWidths: []int{50, 60, 20, 14, 14},
			LinkFunc:  tui.LinkSelf(1),
		}.Print()
	},
}

func init() {
	linksCmd.AddCommand(linksQueryCmd)

	linksQueryCmd.PersistentFlags().StringP(linksQuery, "q", "", "Query. Can be empty to return all bookmarks. To search for tags, tags must be preceded by a '#'.")
}
