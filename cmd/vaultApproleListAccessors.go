package cmd

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var vaultApproleListAccessorsCmd = &cobra.Command{
	Use:     "list-accessors",
	Aliases: []string{"list"},
	Short:   "List approle accessors",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("hello")

		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)

		roles, err := client.ListAppRoles(cmd.Context(), mount)
		if err != nil {
			log.Fatal().Err(err).Msg("Error listing roles")
		}
		var errs error
		for _, role := range roles {
			accessors, err := client.GetAppRoleAccessors(cmd.Context(), mount, role)
			if err != nil {
				log.Error().Err(err).Str("role", role).Msg("Error getting accessors")
				errs = multierror.Append(errs, err)
				continue
			}

			// Step 2: Iterate over accessors
			for _, accessor := range accessors {
				if err := client.GetAppRoleAccessorMetadata(cmd.Context(), mount, role, accessor); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		}

		return errs
	},
}

func init() {
	vaultApproleCmd.AddCommand(vaultApproleListAccessorsCmd)
}
