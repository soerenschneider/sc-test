package cmd

import (
	"context"
	"fmt"
	"sort"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// totpMethodLister is the subset of the vault client needed to discover TOTP methods.
type totpMethodLister interface {
	TotpListMethods(ctx context.Context) ([]string, error)
}

// identityEntityResolver is the subset of the vault client needed to discover and
// resolve identity entities.
type identityEntityResolver interface {
	IdentityListEntities(ctx context.Context) ([]string, error)
	IdentityGetEntityIdByName(ctx context.Context, entityName string) (string, error)
}

// vaultTotpDeleteSecretCmd represents the totp delete-secret command
var vaultTotpDeleteSecretCmd = &cobra.Command{
	Use: "delete-secret",
	Aliases: []string{
		"del-secret",
		"secret-del",
		"secret-delete",
		"destroy-secret",
	},
	Short: "Delete an entity's TOTP secret",
	Long: `Destroys the TOTP secret of an identity entity for a given TOTP MFA method.

Once the secret has been destroyed, the entity can no longer satisfy this MFA method
until a new secret has been generated using the 'generate-secret' subcommand.

Method id and entity name may be passed as flags. Anything that is not passed is
prompted for interactively.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		methodId := pkg.GetString(cmd, vaultTotpMethodId)
		entityName := pkg.GetString(cmd, vaultIdentityEntityId)
		force, _ := cmd.Flags().GetBool(vaultForce)

		if methodId == "" {
			methodId = promptTotpMethodId(client)
		}

		if entityName == "" {
			entityName = promptIdentityEntityName(client)
		}

		entityId := mustResolveEntityId(client, entityName)
		log.Info().Msgf("Derived entity_id %q from entity_name %q", entityId, entityName)

		if !force {
			prompt := fmt.Sprintf("Really destroy TOTP secret of entity %q for method %q?", entityName, methodId)
			if tui.SelectInput(prompt, []string{"no", "yes"}) != "yes" {
				log.Info().Msg("Aborted, nothing has been deleted")
				return
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		if err := spinner.New().
			ActionWithErr(func(ctx context.Context) error {
				return client.TotpDestroySecretAdmin(ctx, methodId, entityId)
			}).
			Title("Destroying TOTP secret...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not delete totp secret")
		}

		log.Info().Msgf("Destroyed TOTP secret of entity_name %q (entity_id %q) for method_id %q", entityName, entityId, methodId)
	},
}

// promptTotpMethodId tries to offer a selection of the configured TOTP methods and
// falls back to free-form input if the methods can not be listed.
func promptTotpMethodId(client totpMethodLister) string {
	var availableMethods []string

	ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
	defer cancel()

	if err := spinner.New().
		ActionWithErr(func(ctx context.Context) error {
			methods, err := client.TotpListMethods(ctx)
			if err != nil {
				return err
			}
			availableMethods = methods
			return nil
		}).
		Title("Loading available methods...").
		Context(ctx).
		Type(spinner.Dots).
		Run(); err != nil {
		log.Warn().Err(err).Msg("could not fetch TOTP methods, falling back to manual input")
	}

	if len(availableMethods) > 0 {
		sort.Strings(availableMethods)
		return tui.SelectInput("Enter method id", availableMethods)
	}

	return tui.ReadInput("Enter method id", nil)
}

// promptIdentityEntityName tries to offer a selection of the existing identity
// entities and falls back to free-form input if the entities can not be listed.
func promptIdentityEntityName(client identityEntityResolver) string {
	var availableEntities []string

	ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
	defer cancel()

	if err := spinner.New().
		ActionWithErr(func(ctx context.Context) error {
			entities, err := client.IdentityListEntities(ctx)
			if err != nil {
				return err
			}
			availableEntities = entities
			return nil
		}).
		Title("Loading available entities...").
		Context(ctx).
		Type(spinner.Dots).
		Run(); err != nil {
		log.Warn().Err(err).Msg("could not fetch entities, falling back to manual input")
	}

	if len(availableEntities) > 0 {
		sort.Strings(availableEntities)
		return tui.SelectInput("Enter entity id", availableEntities)
	}

	return tui.ReadInput("Enter entity id", nil)
}

// mustResolveEntityId translates an entity name into its entity id and terminates
// the process if that is not possible.
func mustResolveEntityId(client identityEntityResolver, entityName string) string {
	var entityId string

	ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
	defer cancel()

	if err := spinner.New().
		ActionWithErr(func(ctx context.Context) error {
			var err error
			entityId, err = client.IdentityGetEntityIdByName(ctx, entityName)
			return err
		}).
		Title(fmt.Sprintf("Fetching entity_id for entity_name %q", entityName)).
		Context(ctx).
		Type(spinner.Dots).
		Run(); err != nil {
		log.Fatal().Err(err).Msg("could not fetch entity_id")
	}

	return entityId
}

func init() {
	vaultTotpCmd.AddCommand(vaultTotpDeleteSecretCmd)

	vaultTotpDeleteSecretCmd.Flags().StringP(vaultIdentityEntityId, "e", "", "Identity Entity ID")
	vaultTotpDeleteSecretCmd.Flags().StringP(vaultTotpMethodId, "m", "", "TOTP method ID")
	vaultTotpDeleteSecretCmd.Flags().BoolP(vaultForce, "f", false, "Skip the confirmation prompt")
}
