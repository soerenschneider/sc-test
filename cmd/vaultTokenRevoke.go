package cmd

import (
	"context"
	"errors"
	"os"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/spf13/cobra"
)

// vaultTokenRevokeCmd represents the token revoke command
var vaultTokenRevokeCmd = &cobra.Command{
	Use:     "revoke",
	Aliases: []string{"revoke-self", "logout"},
	Short:   "Revoke the current Vault token",
	Long: `Revoke the currently active Vault token, including all of its child tokens.

Before revoking, information about the token is displayed and confirmation is
requested. Use --force to skip the confirmation prompt.

If the revoked token was read from the local token file (~/.vault-token), the
file is removed afterwards as the token is no longer usable.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))
		force, _ := cmd.Flags().GetBool(vaultForce)

		if !force {
			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			secret, err := client.LookupToken(ctx)
			cancel()
			if err != nil {
				log.Fatal().Err(err).Msg("could not lookup token")
			}

			printTokenInfo(secret.Data)

			if tui.SelectInput("Really revoke this token?", []string{"no", "yes"}) != "yes" {
				log.Info().Msg("Aborted, token has not been revoked")
				return
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		if err := spinner.New().
			ActionWithErr(func(ctx context.Context) error {
				return client.RevokeSelfToken(ctx)
			}).
			Title("Revoking token...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not revoke token")
		}

		log.Info().Msg("Revoked token")
		removeRevokedTokenFile()
	},
}

// removeRevokedTokenFile deletes the local token file if it holds the token that has
// just been revoked. Tokens supplied via the environment are left untouched.
func removeRevokedTokenFile() {
	if _, ok := os.LookupEnv("VAULT_TOKEN"); ok {
		log.Info().Msg("Token was supplied via VAULT_TOKEN, consider unsetting it")
		return
	}

	tokenFilePath, err := vault.GetVaultTokenFilePath()
	if err != nil {
		log.Warn().Err(err).Msg("could not determine token file path")
		return
	}

	if err := os.Remove(tokenFilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warn().Err(err).Msgf("could not remove token file %q", tokenFilePath)
		}
		return
	}

	log.Info().Msgf("Removed token file %q", tokenFilePath)
}

func init() {
	vaultTokenCmd.AddCommand(vaultTokenRevokeCmd)

	vaultTokenRevokeCmd.Flags().BoolP(vaultForce, "f", false, "Skip displaying token info and the confirmation prompt")
}
