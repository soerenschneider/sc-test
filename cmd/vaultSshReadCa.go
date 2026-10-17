package cmd

import (
	"context"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"

	"github.com/spf13/cobra"
)

var vaultSshReadCaCmd = &cobra.Command{
	Use:     "read",
	Aliases: []string{"read-ca"},
	Short:   "Reads ca data for the SSH secret engine",
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "ssh")
		}

		caData, err := prompt.Load(cmd.Context(), "Fetching ca data from Vault", func(ctx context.Context) ([]byte, error) {
			return client.SshGetCa(ctx, mount)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not fetch ca data")
		}

		fmt.Println(string(caData))
	},
}

func init() {
	vaultSshCmd.AddCommand(vaultSshReadCaCmd)
}
