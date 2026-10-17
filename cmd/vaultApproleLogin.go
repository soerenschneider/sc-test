package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

// vaultLoginCmd represents the vaultLogin command
var vaultApproleLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Vault using approle",
	Run: func(cmd *cobra.Command, args []string) {
		approleConfig, err := parseVaultApproleConfig(cmd)
		if err != nil {
			log.Fatal().Err(err).Msg("Error parsing config")
		}

		client := vault.MustBuildClient(cmd)

		mount := pkg.GetString(cmd, vaultMountPath)
		accessors, err := client.GetAppRoleAccessors(cmd.Context(), mount, approleConfig.RoleName)
		if err != nil {
			log.Fatal().Err(err).Msg("Error authenticating")
		}
		fmt.Println(accessors)

		secret, err := client.ApproleAuth(cmd.Context(), mount, approleConfig)
		if err != nil {
			log.Fatal().Err(err).Msg("Error authenticating")
		}

		fmt.Println(secret.Data)
	},
}

func init() {
	vaultApproleCmd.AddCommand(vaultApproleLoginCmd)
}
