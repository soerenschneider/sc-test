package cmd

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/api"
	"github.com/soerenschneider/sc/internal/agent"
	"github.com/spf13/cobra"
)

const agentLibvirtRebootDomainCmdFlagDomain = "domain"

var agentLibvirtRebootDomainCmd = &cobra.Command{
	Use:   "reboot-domain",
	Short: "Reboots a domain",
	Run: func(cmd *cobra.Command, args []string) {
		client := agent.MustBuildApp(cmd)

		domain, err := cmd.Flags().GetString(agentLibvirtRebootDomainCmdFlagDomain)
		if err != nil {
			log.Fatal().Err(err).Str("flag", agentLibvirtRebootDomainCmdFlagDomain).Msg("could not get flag")
		}

		params := &api.LibvirtPostDomainActionParams{
			Action: api.LibvirtPostDomainActionParamsActionReboot,
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
	agentLibvirtCmd.AddCommand(agentLibvirtRebootDomainCmd)
	agentLibvirtRebootDomainCmd.Flags().String(agentLibvirtRebootDomainCmdFlagDomain, "", "The domain to reboot")
	if err := agentLibvirtRebootDomainCmd.MarkFlagRequired(agentLibvirtRebootDomainCmdFlagDomain); err != nil {
		log.Fatal().Err(err).Msg("could not mark flag required")
	}
}
