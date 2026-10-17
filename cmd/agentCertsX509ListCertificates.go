package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/soerenschneider/sc/internal/agent/fmt"
	"github.com/spf13/cobra"
)

var agentCertsX509ListCertificatesCmd = &cobra.Command{
	Use:   "list-certificates",
	Short: "List all managed X509 certificates",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		resp, err := client.CertsX509GetCertificatesList(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		data := agent.MustResponse[api.X509ManagedCertificateList](resp)
		fmt.PrintPkiManagedCertificateConfigs(data.Data)
	},
}

func init() {
	agentCertsX509Cmd.AddCommand(agentCertsX509ListCertificatesCmd)
}
