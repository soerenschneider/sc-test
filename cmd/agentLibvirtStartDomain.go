package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentLibvirtStartDomainCmdCmdFlagDomain = "domain"

var agentLibvirtStartDomainCmd = &cobra.Command{
	Use:   "start-domain",
	Short: "Starts a domain",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		domain, err := cmd.Flags().GetString(agentLibvirtStartDomainCmdCmdFlagDomain)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get flag")
		}

		params := &api.LibvirtPostDomainActionParams{
			Action: api.LibvirtPostDomainActionParamsActionStart,
		}
		resp, err := client.LibvirtPostDomainAction(context.Background(), domain, params)
		if err != nil {
			log.Fatal().Err(err).Msg("could not send request")
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		_ = agent.MustResponse[any](resp)
	},
}

func init() {
	agentLibvirtCmd.AddCommand(agentLibvirtStartDomainCmd)
	agentLibvirtStartDomainCmd.Flags().String(agentLibvirtStartDomainCmdCmdFlagDomain, "", "The domain to start")
	if err := agentLibvirtStartDomainCmd.MarkFlagRequired(agentLibvirtStartDomainCmdCmdFlagDomain); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
