package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/userdata"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/soerenschneider/sc/pkg/clipboard"
	"github.com/spf13/cobra"
)

type vaultLoginUserdata struct {
	LastUser string `json:"last_username"`
}

// vaultLoginCmd represents the vaultLogin command
var vaultLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Vault server using username and password",
	Long: `Authenticate with a HashiCorp Vault server using the "userpass" authentication method.

This command logs you into Vault using a username and password. If two-factor
authentication is enabled, you may also provide a one-time password (OTP).

After successful login, a Vault token is returned. The token is saved to a file.
If the file cannot be written—due to permission issues or missing directories—the token is printed
to stdout as a fallback.`,
	Run: func(cmd *cobra.Command, args []string) {
		username := pkg.GetString(cmd, vaultLoginUsername)
		otp := pkg.GetString(cmd, vaultLoginOtp)
		mfaId := pkg.GetString(cmd, vaultLoginMfaId)
		mount := pkg.GetString(cmd, vaultMountPath)

		client := vault.MustBuildClient(cmd)

		// load userdata
		commandName := getCommandName(cmd)
		userData, err := userdata.LoadCommandData[vaultLoginUserdata](cmp.Or(profile, defaultProfileName), commandName)
		if err != nil {
			log.Warn().Err(err).Msg("could not load userdata")
		}

		validateFunc := func(val string) error {
			if strings.TrimSpace(val) == "" {
				return errors.New("input can not be empty")
			}
			return nil
		}

		var fields []huh.Field
		if username == "" {
			var suggestions []string
			username = userData.LastUser
			currentUser, err := user.Current()
			if err == nil {
				suggestions = append(suggestions, currentUser.Username)
			}
			fields = append(fields, huh.NewInput().Title("Username").Suggestions(suggestions).Value(&username).Validate(validateFunc))
		}

		var password string
		fields = append(fields, huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&password).Validate(validateFunc))

		form := huh.NewForm(huh.NewGroup(fields...))

		// Jump directly to the password field if the username field is already filled out
		if username != "" {
			form.NextField()
		}

		if err := form.Run(); err != nil {
			log.Fatal().Err(err).Msg("could not get info from user")
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), vaultDefaultTimeout)
		defer cancel()

		var vaultSecret *api.Secret
		sendPasswordFunc := func(ctx context.Context) error {
			var err error
			if mfaId != "" && otp != "" {
				vaultSecret, err = client.UserpassAuthMfa(ctx, username, password, mfaId, otp)
				return err
			} else {
				vaultSecret, err = client.UserpassAuth(ctx, mount, username, password)
				return err
			}
		}

		if err := spinner.New().
			Type(spinner.Line).
			ActionWithErr(sendPasswordFunc).
			Title("Sending login request...").
			Context(ctx).
			Type(spinner.Dots).
			Run(); err != nil {
			log.Fatal().Err(err).Msg("sending login to request to Vault failed")
		}

		if vaultSecret == nil || vaultSecret.Auth == nil {
			log.Fatal().Msg("received empty response from Vault")
		}

		if vaultSecret.Auth.MFARequirement != nil {
			mfaReq, otpId, err := vault.GetVaultMfaRequirement(vaultSecret)
			if err != nil {
				log.Fatal().Err(err).Msg("error retrieving MFA information")
			}

			if otp == "" {
				// try to parse OTP from clipboard (for cases when pass otp or similar is used to generate otp)
				clipboardContent, err := clipboard.PasteClipboard(ctx)
				if err == nil {
					if (len(clipboardContent) == 6 || len(clipboardContent) == 8) && pkg.IsAsciiNumeric(clipboardContent) {
						otp = clipboardContent
					}
				}
				tui.ReadInputSuggestionWithValidation(&otp, fmt.Sprintf("Enter OTP for id %q", otpId), nil, pkg.OtpValidation)
			}

			ctx, cancel = context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()
			sendOtpFunc := func(ctx context.Context) error {
				vaultSecret, err = client.SendOtp(ctx, mfaReq, otp)
				return err
			}

			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(sendOtpFunc).
				Title("Sending OTP...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("sending OTP to Vault failed")
			}
		}

		tokenFile, err := vault.GetVaultTokenFilePath()
		if err != nil {
			printToken(vaultSecret.Auth.ClientToken)
			return
		}
		if err := os.WriteFile(tokenFile, []byte(vaultSecret.Auth.ClientToken), 0600); err != nil {
			printToken(vaultSecret.Auth.ClientToken)
			log.Fatal().Msgf("Failed to save token to %s: %v", tokenFile, err)
		}

		var outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("Token saved")
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), tokenFile))

		userData.LastUser = username
		if err := userdata.Upsert[vaultLoginUserdata](cmp.Or(profile, defaultProfileName), commandName, userData); err != nil {
			log.Warn().Err(err).Msg("could not save userdata")
		}
	},
}

func printToken(token string) {
	var outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("Vault token")
	fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), token))
}

func init() {
	vaultCmd.AddCommand(vaultLoginCmd)

	vaultLoginCmd.Flags().StringP(vaultLoginUsername, "u", "", "Username for login")
	vaultLoginCmd.Flags().StringP(vaultMountPath, "m", "userpass", "Vault mount for userpass auth engine")
	vaultLoginCmd.Flags().StringP(vaultLoginOtp, "o", "", "OTP value for non-interactive login")
	vaultLoginCmd.Flags().StringP(vaultLoginMfaId, "", "", "MFA ID")
}
