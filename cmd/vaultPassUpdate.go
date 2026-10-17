package cmd

import (
	"context"
	"errors"
	"os/user"

	"charm.land/huh/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultPassUpdateCmd = &cobra.Command{
	Use: "update-password",
	Aliases: []string{
		"update-pass",
	},
	Short: "Update the password for Vault userpass authentication method",
	Long: `Update the password on a HashiCorp Vault server using the "userpass" authentication method.

This command allows a user to change their Vault password by providing their.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		username := pkg.GetString(cmd, vaultLoginUsername)
		mount := pkg.GetString(cmd, vaultMountPath)

		if username == "" {
			var suggestions []string
			currentUser, err := user.Current()
			if err == nil {
				suggestions = []string{currentUser.Username}
			}

			username = tui.ReadInput("Enter username", suggestions)
		}

		var password string
		var passwordConfirmation string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("New password").
					Value(&password).
					EchoMode(huh.EchoModePassword).
					Validate(func(str string) error {
						if len(str) < 12 {
							return errors.New("password must be >= 12")
						}
						return nil
					}),

				huh.NewInput().
					Title("Confirm password").
					Value(&passwordConfirmation).
					EchoMode(huh.EchoModePassword).
					Validate(func(str string) error {
						if str != password {
							return errors.New("passwords do not match")
						}
						return nil
					}),
			),
		)

		if err := form.Run(); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
		defer cancel()

		if err := client.UpdatePassword(ctx, mount, username, password); err != nil {
			log.Fatal().Err(err).Msg("could not update password")
		}
	},
}

func init() {
	vaultCmd.AddCommand(vaultPassUpdateCmd)

	vaultPassUpdateCmd.Flags().StringP(vaultLoginUsername, "u", "", "Username for login")
	vaultPassUpdateCmd.Flags().StringP(vaultMountPath, "m", "userpass", "Vault mount for userpass auth engine")
}
