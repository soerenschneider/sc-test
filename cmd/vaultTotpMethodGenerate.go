package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultTotpGenerateMethodCmd = &cobra.Command{
	Use: "generate-method",
	Aliases: []string{
		"gen-method",
		"method-gen",
		"method-generate",
	},
	Short: "Manage Vault totp",
	Long: `The 'token' command group contains subcommands for interacting with Vault tokens.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		req := vault.MfaCreateMethodRequest{}
		req.Issuer = pkg.GetString(cmd, vaultIssuer)
		req.MethodName = pkg.GetString(cmd, vaultTotpMethodName)
		req.Algorithm = pkg.GetString(cmd, vaultAlgorithm)

		validateFunc := func(val string) error {
			if strings.TrimSpace(val) == "" {
				return errors.New("input can not be empty")
			}
			return nil
		}

		var fields []huh.Field
		var err error
		if req.Issuer == "" {
			fields = append(fields, huh.NewInput().Title("Issuer").Value(&req.Issuer).Validate(validateFunc))
		}

		if req.MethodName == "" {
			fields = append(fields, huh.NewInput().Title("Method Name").Value(&req.MethodName).Validate(validateFunc))
		}

		if req.Algorithm == "" {
			fields = append(fields, huh.NewSelect[string]().Options(huh.NewOptions(vaultTotpAlgorithms...)...).Title("TOTP Algorithm").Value(&req.Algorithm))
		}

		if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
			log.Fatal().Err(err).Msg("could not get info from user")
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()
		var methodId any
		if err := spinner.New().
			Type(spinner.Line).
			ActionWithErr(func(ctx context.Context) error {
				methodId, err = client.TotpGenerateMethod(ctx, req)
				return err
			}).
			Title("Generating TOTP method").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("could not generate TOTP method")
		}

		fmt.Println(methodId)
	},
}

func init() {
	vaultTotpCmd.AddCommand(vaultTotpGenerateMethodCmd)

	vaultTotpGenerateMethodCmd.Flags().StringP(vaultIssuer, "i", "", "The name of the key's issuing organization")
	vaultTotpGenerateMethodCmd.Flags().StringP(vaultTotpMethodName, "m", "", "The unique name identifier for this MFA method")
	vaultTotpGenerateMethodCmd.Flags().StringP(vaultAlgorithm, "", "SHA256", "Specifies the hashing algorithm used to generate the TOTP code. Options include \"SHA1\", \"SHA256\" and \"SHA512\"")
}
