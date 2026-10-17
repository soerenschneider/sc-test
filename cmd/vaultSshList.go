package cmd

import (
	"context"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var sshListCmd = &cobra.Command{
	Use:     "list-roles",
	Aliases: []string{"list"},
	Short:   "Lists all roles for the SSH secrets engine",
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "ssh")
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), vaultDefaultTimeout)
		defer cancel()

		roles, err := client.SshListRoles(ctx, mount)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to list SSH roles for mount %q", mount)
		}

		var outputHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).MarginRight(1).SetString("SSH roles")
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, outputHeader.String(), strings.Join(roles, ", ")))
	},
}

func init() {
	vaultSshCmd.AddCommand(sshListCmd)
}
