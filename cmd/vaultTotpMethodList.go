package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultTotpListMethodsCmd = &cobra.Command{
	Use: "list-methods",
	Aliases: []string{
		"list",
	},
	Short: "Manage Vault totp",
	Long: `The 'token' command group contains subcommands for interacting with Vault tokens.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))
		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		secret, err := client.TotpListMethods(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("could not list totp methods")
		}
		fmt.Println(secret)
	},
}

func init() {
	vaultTotpCmd.AddCommand(vaultTotpListMethodsCmd)
}
