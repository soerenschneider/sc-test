package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

const (
	vaultApproleRoleId       = "role-id"
	vaultApproleRoleIdFile   = "role-id–file"
	vaultApproleRoleName     = "role"
	vaultApproleRoleNameFile = "role-file"
	vaultApproleSecretId     = "secret-id"
	vaultApproleSecretIdFile = "secret-id-file"
	vaultApproleSecretIdEnv  = "secret-id-env" //nolint G101
)

// vaultLoginCmd represents the vaultLogin command
var vaultApproleCmd = &cobra.Command{
	Use:     "approle",
	Aliases: []string{"approles"},
	Short:   "Manage Vault approle auth",
	Long: `The 'aws' command group contains subcommands for interacting with Vault AWS secret engine.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	vaultCmd.AddCommand(vaultApproleCmd)

	vaultApproleCmd.PersistentFlags().StringP(vaultMountPath, "m", "approle", "Vault mount for userpass auth engine")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleRoleId, "i", "", "AppRole role id")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleRoleIdFile, "", "", "AppRole role id file")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleRoleName, "r", "", "AppRole role name")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleRoleNameFile, "", "", "AppRole role name file")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleSecretId, "s", "", "AppRole secret id")
	vaultApproleCmd.PersistentFlags().StringP(vaultApproleSecretIdFile, "", "", "AppRole secret id file")

	// role-id vs role-name
	vaultApproleCmd.MarkFlagsMutuallyExclusive(vaultApproleRoleId, vaultApproleRoleName)

	// role-id vs role-id-file
	vaultApproleCmd.MarkFlagsMutuallyExclusive(vaultApproleRoleId, vaultApproleRoleIdFile)

	// role-name vs role-name-file
	vaultApproleCmd.MarkFlagsMutuallyExclusive(vaultApproleRoleName, vaultApproleRoleNameFile)

	// secret-id vs secret-id-file
	vaultApproleCmd.MarkFlagsMutuallyExclusive(vaultApproleSecretId, vaultApproleSecretIdFile)
}

func readValueOrFile(value, file, field string) (string, error) {
	if value != "" && file != "" {
		return "", fmt.Errorf("%s and %s-file are mutually exclusive", field, field)
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed reading %s-file: %w", field, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	return value, nil
}

func readValueOrFileOrEnv(value, file, env, field string) (string, error) {
	if value != "" && file != "" && env != "" {
		return "", fmt.Errorf("%s, %s-file and %s-env are mutually exclusive", field, field, field)
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed reading %s-file: %w", field, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	if env != "" {
		data := os.Getenv(env)
		if data != "" {
			return data, nil
		}
	}

	return value, nil
}

func parseVaultApproleConfig(cmd *cobra.Command) (*vault.VaultApproleConfig, error) {
	roleID, _ := cmd.Flags().GetString(vaultApproleRoleId)
	roleIDFile, _ := cmd.Flags().GetString(vaultApproleRoleIdFile)

	roleName, _ := cmd.Flags().GetString(vaultApproleRoleName)
	roleNameFile, _ := cmd.Flags().GetString(vaultApproleRoleNameFile)

	secretIdConf := &approle.SecretID{
		FromFile:   pkg.GetString(cmd, vaultApproleSecretIdFile),
		FromEnv:    pkg.GetString(cmd, vaultApproleSecretIdEnv),
		FromString: pkg.GetString(cmd, vaultApproleSecretId),
	}

	resolvedRoleID, err := readValueOrFile(roleID, roleIDFile, "role-id")
	if err != nil {
		return nil, err
	}

	resolvedRoleName, err := readValueOrFile(roleName, roleNameFile, "role-name")
	if err != nil {
		return nil, err
	}

	// role-id vs role-name (final resolved values)
	if resolvedRoleID != "" && resolvedRoleName != "" {
		return nil, fmt.Errorf("role-id and role-name are mutually exclusive")
	}

	if resolvedRoleID == "" && resolvedRoleName == "" {
		return nil, fmt.Errorf("either role-id or role-name must be provided")
	}

	return &vault.VaultApproleConfig{
		RoleID:   resolvedRoleID,
		RoleName: resolvedRoleName,
		SecretID: secretIdConf,
	}, nil
}
