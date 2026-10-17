package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentLibvirtShutdownDomainCmdFlagDomain = "domain"

var agentLibvirtShutdownDomainCmd = &cobra.Command{
	Use:   "shutdown-domain",
	Short: "Shuts a domain down",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		domain, err := cmd.Flags().GetString(agentLibvirtShutdownDomainCmdFlagDomain)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		params := &api.LibvirtPostDomainActionParams{
			Action: api.LibvirtPostDomainActionParamsActionShutdown,
		}
		resp, err := client.LibvirtPostDomainAction(context.Background(), domain, params)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send requests")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		_ = agent.MustResponse[any](resp)
	},
}

func init() {
	agentLibvirtCmd.AddCommand(agentLibvirtShutdownDomainCmd)
	agentLibvirtShutdownDomainCmd.Flags().String(agentLibvirtShutdownDomainCmdFlagDomain, "", "The domain to shutdown")
	if err := agentLibvirtShutdownDomainCmd.MarkFlagRequired(agentLibvirtShutdownDomainCmdFlagDomain); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
