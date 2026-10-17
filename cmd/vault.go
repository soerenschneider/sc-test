package cmd

import (
	"time"

	"github.com/soerenschneider/sc/internal/vault"
	"github.com/spf13/cobra"
)

const (
	vaultAddr      = "address"
	vaultTokenFile = "token-file"

	vaultMountPath = "mount"
	vaultTtl       = "ttl"
	vaultRoleName  = "role"

	vaultAwsProfile                    = "aws-profile"
	vaultAwsDefaultTtl                 = 3600
	vaultAwsDefaultCredentialsFilename = "~/.aws/credentials" //nolint G101
	vaultAwsDefaultProfile             = "default"
	vaultAwsNoWait                     = "no-wait"
	vaultAwsWaitTimeout                = "wait-timeout"
	vaultAwsDefaultWaitTimeout         = 30 * time.Second

	vaultPrivateKeyFile  = "key-file"
	vaultCertificateFile = "cert-file"
	vaultCaFile          = "ca-file"
	vaultCertMinLifetime = "min-lifetime"
	vaultCertMinDuration = "min-duration"
	vaultFullChain       = "full-chain"

	vaultPublicKeyFile = "pub-key-file"
	vaultPrincipals    = "principals"
	vaultCommonName    = "common-name"
	vaultIpSans        = "ip-sans"
	vaultAltNames      = "alt-names"

	vaulEntityName      = "entity-name"
	vaultLoginUsername  = "username"
	vaultLoginOtp       = "otp"
	vaultLoginMfaId     = "mfa-id"
	vaultIssuer         = "issuer"
	vaultTotpMethodName = "method-name"
	vaultAlgorithm      = "algorithm"

	vaultIdentityEntityId = "entity-id"
	vaultTotpMethodId     = "method-id"

	vaultForce          = "force"
	vaultDefaultTimeout = vault.DefaultTimeout

	vaultSecretSyncName   = "name"
	vaultSecretSyncConfig = "sync-config" //nolint G101
	vaultSecretSyncAll    = "all"
)

var vaultTotpAlgorithms = []string{"SHA1", "SHA256", "SHA512"}

// vaultCmd represents the vault command
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Commands for interacting with HashiCorp Vault",
	Long: `This command serves as the entry point for Vault-related functionality.

Use one of the available subcommands to interact with HashiCorp Vault for tasks
such as reading and writing secrets, authentication, or policy management.

Examples:
  sc vault login             # Authenticate with Vault
  sc vault ssh sign-key      # Interact with SSH secret engine`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(vaultCmd)

	vaultCmd.Flags().StringP(vaultAddr, "a", "", "Vault address. If not specified, tries to read env variable VAULT_ADDR.")
	vaultCmd.Flags().StringP(vaultTokenFile, "t", "~/.vault-token", "Vault token file.")
}
