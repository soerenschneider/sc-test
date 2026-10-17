package cmd

import (
	"context"
	"time"

	"charm.land/huh/v2/spinner"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// vaultTokenRenewCmd represents the token renew command
var vaultTokenRenewCmd = &cobra.Command{
	Use:     "renew",
	Aliases: []string{"renew-self", "extend"},
	Short:   "Try to extend the TTL of the current Vault token",
	Long: `Renew the currently active Vault token to extend its TTL.

The requested TTL is passed to Vault as increment and counts from now, not from
the current expiry. If no TTL is given, Vault applies the token's default TTL.
Vault caps the resulting TTL at the token's max TTL, so the granted TTL may be
lower than requested. Tokens that are not renewable can not be extended.

Information about the renewed token is displayed afterwards.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		var increment time.Duration
		if ttl := pkg.GetString(cmd, vaultTtl); ttl != "" {
			var err error
			increment, err = time.ParseDuration(ttl)
			if err != nil || increment < 0 {
				log.Fatal().Err(err).Msgf("invalid ttl %q", ttl)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		var secret *api.Secret
		if err := spinner.New().
			ActionWithErr(func(ctx context.Context) error {
				var err error
				secret, err = client.RenewSelfToken(ctx, increment)
				return err
			}).
			Title("Renewing token...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not renew token")
		}

		if secret != nil && secret.Auth != nil {
			granted := time.Duration(secret.Auth.LeaseDuration) * time.Second
			if increment > 0 && granted < increment {
				log.Warn().Msgf("Requested TTL %s, but Vault only granted %s (capped by max TTL)", increment, granted)
			} else {
				log.Info().Msgf("Renewed token, new TTL is %s", granted)
			}
		}

		lookup, err := client.LookupToken(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("could not lookup token")
		}
		printTokenInfo(lookup.Data)
	},
}

func init() {
	vaultTokenCmd.AddCommand(vaultTokenRenewCmd)

	vaultTokenRenewCmd.Flags().StringP(vaultTtl, "t", "", "TTL to request, counted from now (e.g. 8h). Defaults to the token's default TTL")
}
