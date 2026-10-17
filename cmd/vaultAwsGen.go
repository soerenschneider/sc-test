package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var vaultAwsGenerateCredentialsCmd = &cobra.Command{
	Use:     "generate-credentials",
	Aliases: []string{"gen-credentials", "gen"},
	Short:   "Generate AWS credentials using Vault",
	Long: `The 'generate-credentials' command is part of the 'aws' command group, which provides tools
for working with the Vault AWS secrets engine.

This command is used to generate dynamic AWS credentials by interacting with Vault. 
It requires appropriate configuration and permissions set in Vault to access the AWS secrets engine.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		profile := pkg.GetString(cmd, vaultAwsProfile)
		role := pkg.GetString(cmd, vaultRoleName)
		ttl := pkg.GetInt(cmd, vaultTtl)
		noWait := pkg.MustGetBool(cmd, vaultAwsNoWait)
		waitTimeout := pkg.GetDuration(cmd, vaultAwsWaitTimeout)

		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "aws")
		}

		if role == "" {
			var availableRoles []string

			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					roles, err := client.AwsListRoles(ctx, mount)
					if err == nil {
						availableRoles = roles
					}
					return nil
				}).
				Title("Loading available roles...").
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("sending login to request to Vault failed")
			}

			if len(availableRoles) > 0 {
				sort.Strings(availableRoles)
				role = tui.SelectInput("Enter role", availableRoles)
			} else {
				role = tui.ReadInput("Enter role", nil)
			}
		}

		{
			ctx, cancel := context.WithTimeout(context.Background(), vaultDefaultTimeout)
			defer cancel()

			var creds *vault.AwsCredentials
			if err := spinner.New().
				Type(spinner.Line).
				ActionWithErr(func(ctx context.Context) error {
					var err error
					creds, err = client.AwsGenCreds(ctx, mount, role, strconv.Itoa(ttl))
					return err
				}).
				Title(fmt.Sprintf("Generating credentials for role %s", role)).
				Context(ctx).
				Type(spinner.Dots).
				Run(); err != nil {
				log.Fatal().Err(err).Msg("failed to generate credentials")
			}

			if err := updateAwsCredentialsFile(profile, *creds); err != nil {
				var outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("access_key")
				fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), creds.AccessKeyID))

				outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("secret_key")
				fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), creds.SecretAccessKey))

				log.Fatal().Err(err).Msg("could not write credentials to file")
			}

			if !noWait {
				waitForAwsCredentials(*creds, waitTimeout)
			}
		}

		var outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("Wrote credentials")
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), vaultAwsDefaultCredentialsFilename))
	},
}

// waitForAwsCredentials blocks until AWS accepts the freshly generated
// credentials. Not succeeding is not fatal, as the credentials have already
// been written and will most likely work a bit later.
func waitForAwsCredentials(creds vault.AwsCredentials, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var arn string
	if err := spinner.New().
		ActionWithErr(func(ctx context.Context) error {
			var err error
			arn, err = vault.WaitForAwsCredentials(ctx, creds, time.Second)
			return err
		}).
		Title("Waiting for credentials to become effective").
		Context(ctx).
		Type(spinner.Dots).
		Run(); err != nil {
		log.Warn().Err(err).Msgf("credentials not accepted by AWS after %s, they may need a bit longer to become effective", timeout)
		return
	}

	log.Info().Msgf("Credentials are effective, authenticated as %s", arn)
}

func updateAwsCredentialsFile(profile string, creds vault.AwsCredentials) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	credsPath := filepath.Join(awsDir, "credentials")

	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	cfg := ini.Empty()
	if _, err := os.Stat(credsPath); err == nil {
		if err := cfg.Append(credsPath); err != nil {
			return fmt.Errorf("failed to load existing credentials file: %w", err)
		}
	}

	section, err := cfg.GetSection(profile)
	if err != nil {
		section, err = cfg.NewSection(profile)
		if err != nil {
			return fmt.Errorf("failed to create profile section: %w", err)
		}
	}
	section.Key("aws_access_key_id").SetValue(creds.AccessKeyID)
	section.Key("aws_secret_access_key").SetValue(creds.SecretAccessKey)
	if creds.SessionToken != "" {
		section.Key("aws_session_token").SetValue(creds.SessionToken)
	} else {
		// Don't leave a stale token from earlier credentials behind.
		section.DeleteKey("aws_session_token")
	}

	if err := cfg.SaveTo(credsPath); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

func init() {
	vaultAwsCmd.AddCommand(vaultAwsGenerateCredentialsCmd)

	vaultAwsGenerateCredentialsCmd.Flags().IntP(vaultTtl, "t", vaultAwsDefaultTtl, "Specify how long the credentials should be valid for in seconds")
	vaultAwsGenerateCredentialsCmd.Flags().StringP(vaultRoleName, "r", "", "Specifies the name of the role to generate credentials for")
	vaultAwsGenerateCredentialsCmd.Flags().StringP(vaultAwsProfile, "p", vaultAwsDefaultProfile, "Specifies the name of the AWS credentials profile section")
	vaultAwsGenerateCredentialsCmd.Flags().Bool(vaultAwsNoWait, false, "Do not wait for the credentials to become effective")
	vaultAwsGenerateCredentialsCmd.Flags().Duration(vaultAwsWaitTimeout, vaultAwsDefaultWaitTimeout, "How long to wait for the credentials to become effective")
}
