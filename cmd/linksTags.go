package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/linkding"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var linksTagsCmd = &cobra.Command{
	Use:     "tags",
	Aliases: []string{"t"},
	Short:   "Interact with VictoriaLogs for querying and managing links",
	Long: `The 'links' command is the entry point for interacting with VictoriaLogs.

Use subcommands to query, tail, or manage links stored in VictoriaLogs, a high-performance log database.

Examples:
  sc links query -q='level="error"'   # Query links with specific filters`,
	Run: func(cmd *cobra.Command, args []string) {
		address := pkg.GetString(cmd, linksAddr)
		token, err := getLinkdingToken(cmd)
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting linkding token")
		}

		client, err := linkding.NewLinkdingClient(address, token)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build client")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, t := range tags {
			_, _ = fmt.Fprintf(w, "%d\t%s\n", t.ID, t.Name)
		}
		_ = w.Flush()
	},
}

func init() {
	linksCmd.AddCommand(linksTagsCmd)
}
