package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	ssh "github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const agentCertsSshGetCertificateCmdFlagId = "id"

var agentCertsSshGetCertificateCmd = &cobra.Command{
	Use:   "get-cert",
	Short: "Get SSH certificate identified by its id",
	Run: func(cmd *cobra.Command, args []string) {
		client := ssh.MustBuildApp(cmd)

		id, err := cmd.Flags().GetString(agentCertsSshGetCertificateCmdFlagId)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		resp, err := client.CertsSshGetCertificate(context.Background(), id)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := ssh.MustResponse[api.SshManagedCertificate](resp)
		fmt.PrintManagedCertificateConfig(*data)
	},
}

func init() {
	agentCertsSshCmd.AddCommand(agentCertsSshGetCertificateCmd)
	agentCertsSshGetCertificateCmd.Flags().String(agentCertsSshGetCertificateCmdFlagId, "", "The id of the certificate to return")
	if err := agentCertsSshGetCertificateCmd.MarkFlagRequired(agentCertsSshGetCertificateCmdFlagId); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
