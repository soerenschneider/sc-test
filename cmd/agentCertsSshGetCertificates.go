package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

const (
	agentCertsSshGetCertificatesCmdCertTypeUser = "user"
	agentCertsSshGetCertificatesCmdCertTypeHost = "host"

	agentCertsSshGetCertificatesCmdFlagsCertType        = "cert-type"
	agentCertsSshGetCertificatesCmdFlagsCertTypeDefault = agentCertsSshGetCertificatesCmdCertTypeUser
)

var agentCertsSshGetCertificatesCmd = &cobra.Command{
	Use:   "list-certificates",
	Short: "List all managed SSH certificates",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		var certType api.CertsSshGetCertificatesParamsType
		certTypeParam, err := cmd.Flags().GetString(agentCertsSshGetCertificatesCmdFlagsCertType)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		switch certTypeParam {
		case agentCertsSshGetCertificatesCmdCertTypeUser:
			certType = api.User
		case agentCertsSshGetCertificatesCmdCertTypeHost:
			certType = api.Host
		default:
			log.Fatal().Msgf("invalid cert type %q provided, only %q and %q allowed", certTypeParam, agentCertsSshGetCertificatesCmdCertTypeUser, agentCertsSshGetCertificatesCmdCertTypeHost)
		}
		params := &api.CertsSshGetCertificatesParams{
			Type: &certType,
		}
		resp, err := client.CertsSshGetCertificates(context.Background(), params)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.SshManagedCertificatesList](resp)
		fmt.PrintManagedCertificateConfigs(data.Data)
	},
}

func init() {
	agentCertsSshCmd.AddCommand(agentCertsSshGetCertificatesCmd)

	agentCertsSshGetCertificatesCmd.Flags().String(agentCertsSshGetCertificatesCmdFlagsCertType, agentCertsSshGetCertificatesCmdFlagsCertTypeDefault, "The type of certs to return, must be either 'user' or 'host'")
}
