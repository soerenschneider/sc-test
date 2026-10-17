package cmd

import (
	"github.com/rs/zerolog/log"
	oidc2 "github.com/soerenschneider/sc/internal/oidc"
	"github.com/soerenschneider/sc/internal/storage"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var agentLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Get OIDC token",
	Run: func(cmd *cobra.Command, args []string) {
		tokenStorage := storage.NewStorage()

		provider, err := oidc.NewProvider(cmd.Context(), pkg.GetString(cmd, scAgentProviderUrl))
		if err != nil {
			log.Fatal().Err(err).Msg("could not acquire provider")
		}

		config := &oauth2.Config{
			ClientID:     pkg.GetString(cmd, scAgentClientID),
			ClientSecret: "", // public client, no secret
			Endpoint:     provider.Endpoint(),
			RedirectURL:  "http://localhost:8651/callback",
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		_, err = oidc2.GetAuthToken(cmd.Context(), tokenStorage, config)
		if err != nil {
			log.Fatal().Err(err).Msg("could not acquire token")
		}
	},
}

func init() {
	agentCmd.AddCommand(agentLoginCmd)

	agentLoginCmd.Flags().StringP(scAgentProviderUrl, "p", "https://auth.dd.soeren.cloud/realms/soerencloud", "Username for login")
	agentLoginCmd.Flags().StringP(scAgentClientID, "i", "sc_agent", "OIDC client id")
}
