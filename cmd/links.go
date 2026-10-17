package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

const (
	linksAddr      = "address"
	linksQuery     = "query"
	linksTags      = "tags"
	linksUrl       = "url"
	linksToken     = "token"
	linksTokenFile = "token-file"
)

var linksCmd = &cobra.Command{
	Use:   "links",
	Short: "Interact with Linkding for querying and managing links",
	Long: `The 'links' command is the entry point for interacting with Linkding.

Use subcommands to query, tail, or manage links stored in Linkding.

Examples:
  sc links query -q='#example-tag'   # Query links with specific filters`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(linksCmd)

	linksCmd.PersistentFlags().StringP(linksAddr, "a", "", "Linkding address.")
	_ = linksCmd.MarkPersistentFlagRequired(linksAddr)

	linksCmd.PersistentFlags().StringP(linksToken, "t", "", "Linkding token.")
	linksCmd.PersistentFlags().StringP(linksTokenFile, "T", "", "Path to a file containing the Linkding token.")

	// Exactly one source is allowed, and at least one is required.
	linksCmd.MarkFlagsMutuallyExclusive(linksToken, linksTokenFile)
	linksCmd.MarkFlagsOneRequired(linksToken, linksTokenFile)
}

func getLinkdingToken(cmd *cobra.Command) (string, error) {
	token, _ := cmd.Flags().GetString(linksToken)
	if tokenFile, _ := cmd.Flags().GetString(linksTokenFile); tokenFile != "" {
		tokenFile = pkg.GetExpandedFile(tokenFile)
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", fmt.Errorf("reading token file: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}

	return token, nil
}
