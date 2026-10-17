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

var vaultPkiReadCaCmd = &cobra.Command{
	Use:     "read",
	Aliases: []string{"read-ca"},
	Short:   "Reads ca data for the PKI secret engine",
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "pki")
		}

		fullChain, _ := cmd.Flags().GetBool(vaultFullChain)

		caData, err := prompt.Load(cmd.Context(), "Fetching ca data from Vault", func(ctx context.Context) ([]byte, error) {
			if fullChain {
				return client.PkiGetCaChain(ctx, mount)
			}
			return client.PkiGetCa(ctx, mount, false)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not fetch ca data")
		}

		fmt.Println(string(caData))
	},
}

func init() {
	vaultPkiCmd.AddCommand(vaultPkiReadCaCmd)

	vaultPkiReadCaCmd.Flags().BoolP(vaultFullChain, "f", true, "Read the full chain")
}
