package cmd

import (
	"context"
	"fmt"
	"sort"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultTotpGenerateSecretCmd = &cobra.Command{
	Use: "generate-secret",
	Aliases: []string{
		"gen-secret",
		"secret-gen",
		"secret-generate",
	},
	Short: "Manage Vault totp",
	Long: `The 'token' command group contains subcommands for interacting with Vault tokens.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		entityName := pkg.GetString(cmd, vaultIdentityEntityId)
		methodId := pkg.GetString(cmd, vaultTotpMethodId)
		force, _ := cmd.Flags().GetBool("force")

		var err error
		if methodId == "" {
			var availableMethods []string

			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					methods, err := client.TotpListMethods(ctx)
					if err == nil {
						availableMethods = methods
					}
					return nil
				}).
				Title("Loading available methods...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("could not fetch TOTP methods")
			}

			if len(availableMethods) > 0 {
				sort.Strings(availableMethods)
				methodId = tui.SelectInput("Enter method id", availableMethods)
			} else {
				methodId = tui.ReadInput("Enter method id", nil)
			}
		}

		if entityName == "" {
			var availableEntities []string

			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					entities, err := client.IdentityListEntities(ctx)
					if err == nil {
						availableEntities = entities
					}
					return nil
				}).
				Title("Loading available entities...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("could not fetch entities")
			}

			if len(availableEntities) > 0 {
				sort.Strings(availableEntities)
				entityName = tui.SelectInput("Enter entity id", availableEntities)
			} else {
				entityName = tui.ReadInput("Enter entity id", nil)
			}
		}

		var entityId string
		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()
		if err := spinner.New().
			Type(spinner.Line).
			ActionWithErr(func(ctx context.Context) error {
				entityId, err = client.IdentityGetEntityIdByName(ctx, entityName)
				return err
			}).
			Title(fmt.Sprintf("Fetching entity_id for entity_name %q", entityName)).
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not fetch entity_id")
		}

		log.Info().Msgf("Derived entity_id %q from entity_name %q", entityId, entityName)

		ctx, cancel = context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		otpUri, err := client.TotpGenerateSecretAdmin(ctx, methodId, entityId, force)
		if err != nil {
			log.Fatal().Err(err).Msg("could not generate totp secret")
		}

		fmt.Println(otpUri)
		fmt.Println("")
		qrEncoder := internal.TerminalQrEncoder{}
		if err := qrEncoder.Encode(otpUri); err != nil {
			log.Fatal().Err(err).Msg("could not display qr code")
		}
	},
}

func init() {
	vaultTotpCmd.AddCommand(vaultTotpGenerateSecretCmd)

	vaultTotpGenerateSecretCmd.Flags().StringP(vaultIdentityEntityId, "e", "", "Identity Entity ID")
	vaultTotpGenerateSecretCmd.Flags().StringP(vaultTotpMethodId, "m", "", "TOTP method ID")
	vaultTotpGenerateSecretCmd.Flags().BoolP(vaultForce, "f", false, "Force overwriting of existing TOTP secrets")
}
