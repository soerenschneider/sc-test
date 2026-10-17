package cmd

import (
	"context"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var vaultIdentityGetEntityCmd = &cobra.Command{
	Use: "get-entity",
	Aliases: []string{
		"get",
	},
	Short: "Retrieve a Vault identity entity",
	Long: `The 'get-entity' command is part of the 'identity' command group, which provides 
functionality for interacting with Vault identity entities.

This command retrieves detailed information about a specific entity managed by the Vault identity system. 
It can be useful for inspecting entity metadata, aliases, and associated policies.

Note: This command may require appropriate Vault permissions to access identity resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		name := pkg.GetString(cmd, "name")
		if name == "" {
			var suggestions []string

			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					entities, err := client.IdentityListEntities(ctx)
					if err == nil {
						suggestions = entities
					}
					return nil
				}).
				Title("Loading available entities...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("sending login to request to Vault failed")
			}

			name = tui.ReadInput("Enter entity name", suggestions)
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		data, err := client.IdentityGetEntityByName(ctx, name)
		if err != nil {
			log.Fatal().Err(err).Msg("could not list identity entities")
		}

		tui.PrintMapOutput(data)
	},
}

func init() {
	vaultIdentityCmd.AddCommand(vaultIdentityGetEntityCmd)

	vaultIdentityGetEntityCmd.Flags().StringP(vaulEntityName, "n", "", "Name of the entity")
}
