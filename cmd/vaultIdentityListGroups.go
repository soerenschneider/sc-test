package cmd

import (
	"context"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultIdentityListGroupsCmd = &cobra.Command{
	Use: "list-groups",
	Aliases: []string{
		"groups",
	},
	Short: "List identity entities in Vault",
	Long: `The 'list-entities' command is part of the 'identity' command group, which provides 
tools for managing identity-related resources in Vault.

This command retrieves and lists all entities currently managed by the Vault identity system.
It can be used to view entity IDs, names, and associated metadata.

Note: Appropriate Vault permissions are required to access identity entity data.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))
		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		keys, err := client.IdentityListGroups(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("could not list identity groups")
		}

		sort.Strings(keys)
		tui.WriteListOutput(keys)
	},
}

func init() {
	vaultIdentityCmd.AddCommand(vaultIdentityListGroupsCmd)
}
