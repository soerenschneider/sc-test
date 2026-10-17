package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

const vaultPolicyName = "name"

var vaultPolicyReadCmd = &cobra.Command{
	Use:     "read [name]",
	Aliases: []string{"get", "show"},
	Short:   "Display the rules of a Vault ACL policy",
	Long: `Displays the rules of a Vault ACL policy.

The policy may be passed as argument or via --name. If it is not specified, all policies
are listed in an interactive browser: select a policy to display its rules and go back to
the list with esc. The policy displayed when quitting with q is printed. If the policies
can not be listed, the name is prompted for instead.

Examples:
  sc vault policy read
  sc vault policy read default
  sc vault policy read -n my-policy`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		name := pkg.GetString(cmd, vaultPolicyName)
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			policies, err := prompt.Load(cmd.Context(), "Loading policies...", client.PolicyList)
			if err != nil {
				log.Warn().Err(err).Msg("could not list policies, falling back to manual input")
			}

			if len(policies) > 1 && isTerminal(os.Stdin) && isTerminal(os.Stdout) {
				browsePolicies(cmd.Context(), client, policies)
				return
			}

			name = tui.SelectOrReadInput("policy", policies, nil)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), vaultDefaultTimeout)
		defer cancel()

		rules, err := client.PolicyRead(ctx, name)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read policy")
		}

		printPolicy(name, rules)
	},
}

// browsePolicies lets the user browse the given policies interactively. The policy displayed when
// quitting is printed, so it remains visible after leaving the browser.
func browsePolicies(ctx context.Context, client *vault.VaultClient, policies []string) {
	result, err := tui.Browse(ctx, "Vault policies", policies, func(ctx context.Context, name string) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, vaultDefaultTimeout)
		defer cancel()
		return client.PolicyRead(ctx, name)
	})
	if err != nil {
		log.Fatal().Err(err).Msg("could not browse policies")
	}

	if result.Name != "" {
		printPolicy(result.Name, result.Content)
	}
}

// printPolicy prints the rules of a policy, preceded by a header with its name if stdout is a terminal.
func printPolicy(name, rules string) {
	if isTerminal(os.Stdout) {
		header := lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#6C50FF")).Bold(true).Padding(0, 1).SetString("Policy " + name)
		fmt.Println(header.String())
		fmt.Println()
	}

	fmt.Println(strings.TrimRight(rules, "\n"))
}

func init() {
	vaultPolicyCmd.AddCommand(vaultPolicyReadCmd)

	vaultPolicyReadCmd.Flags().StringP(vaultPolicyName, "n", "", "Name of the policy. If not specified, available policies are offered for selection.")
}
