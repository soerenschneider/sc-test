package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const agentCertsX509GetCertificateCmdFlagId = "id"

var agentCertsX509GetCertificateCmd = &cobra.Command{
	Use:   "get-certificate",
	Short: "Get x509 certificate identified by its id",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		id, err := cmd.Flags().GetString(agentCertsX509GetCertificateCmdFlagId)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		resp, err := client.CertsX509GetCertificate(context.Background(), id)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.X509ManagedCertificate](resp)
		fmt.PrintPkiManagedCertificateConfig(*data)
	},
}

func init() {
	agentCertsX509Cmd.AddCommand(agentCertsX509GetCertificateCmd)
	agentCertsX509GetCertificateCmd.Flags().String(agentCertsX509GetCertificateCmdFlagId, "", "The id of the certificate to return")
	if err := agentCertsX509GetCertificateCmd.MarkFlagRequired(agentCertsX509GetCertificateCmdFlagId); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
