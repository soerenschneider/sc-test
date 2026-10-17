package cmd

import (
	"context"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

const (
	vaultTransitKey     = "key"
	vaultTransitContext = "context"
	vaultTransitInput   = "input"
	vaultTransitOutput  = "output"
)

// transitKeyLister is the subset of the vault client needed to discover transit keys.
type transitKeyLister interface {
	prompt.CapabilityChecker
	TransitListKeys(ctx context.Context, mount string) ([]string, error)
}

var vaultTransitEncryptCmd = &cobra.Command{
	Use:     "encrypt",
	Aliases: []string{"enc"},
	Short:   "Encrypt data using a transit key",
	Long: `Encrypts data using a key of the Vault transit secret engine and prints the resulting ciphertext.

The plaintext is read from the file given by --input ('-' for stdin). If no input file is
given and data is piped to stdin, stdin is read. Otherwise, the plaintext is prompted for
interactively.

Mount and key may be passed as flags. Anything that is not passed is discovered and
offered for selection, falling back to free-form input if discovery is not possible.

Examples:
  sc vault transit encrypt
  sc vault transit encrypt -m transit -k my-key -i secret.txt
  echo -n "hunter2" | sc vault transit encrypt -m transit -k my-key`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		keyName := pkg.GetString(cmd, vaultTransitKey)
		keyContext := pkg.GetString(cmd, vaultTransitContext)
		inputFile := pkg.GetString(cmd, vaultTransitInput)
		outputFile := pkg.GetString(cmd, vaultTransitOutput)

		plaintext, err := readTransitInput(inputFile, func() string {
			return tui.ReadSensitiveInput("Enter plaintext")
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not read plaintext")
		}

		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "transit")
		}

		if keyName == "" {
			keyName = promptTransitKey(cmd.Context(), client, mount, "encrypt")
		}
		prompt.RequireCapability(cmd.Context(), client, vault.Path(mount, "encrypt", keyName), "update")

		ctx, cancel := context.WithTimeout(cmd.Context(), vaultDefaultTimeout)
		defer cancel()

		var ciphertext string
		title := fmt.Sprintf("Encrypting data using key %q on mount %q...", keyName, mount)
		if err := tui.RunWithSpinner(ctx, title, func(ctx context.Context) error {
			var err error
			ciphertext, err = client.TransitEncrypt(ctx, mount, keyName, plaintext, []byte(keyContext))
			return err
		}); err != nil {
			log.Fatal().Err(err).Msg("could not encrypt data")
		}

		if outputFile != "" {
			if err := os.WriteFile(pkg.GetExpandedFile(outputFile), []byte(ciphertext+"\n"), 0600); err != nil {
				log.Fatal().Err(err).Msg("could not write ciphertext")
			}
			log.Info().Msgf("Ciphertext written to %q", outputFile)
			return
		}

		fmt.Println(ciphertext)
	},
}

// readTransitInput reads the input from the given file, from stdin if the file is '-' or
// data is piped to stdin, and prompts for it interactively otherwise.
func readTransitInput(inputFile string, prompt func() string) ([]byte, error) {
	switch {
	case inputFile == "-":
		return io.ReadAll(os.Stdin)
	case inputFile != "":
		return os.ReadFile(pkg.GetExpandedFile(inputFile))
	case !isTerminal(os.Stdin):
		return io.ReadAll(os.Stdin)
	default:
		return []byte(prompt()), nil
	}
}

// promptTransitKey offers a selection of the keys of the given transit mount that the token may use for
// operation, e.g. "encrypt", and falls back to free-form input if the keys can not be listed.
func promptTransitKey(ctx context.Context, client transitKeyLister, mount, operation string) string {
	title := fmt.Sprintf("Loading keys of transit mount %q...", mount)
	keys, err := prompt.Load(ctx, title, func(ctx context.Context) ([]string, error) {
		return client.TransitListKeys(ctx, mount)
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not list transit keys, falling back to manual input")
	}

	keys = prompt.FilterByCapability(ctx, client, "transit key", keys, func(key string) string {
		return vault.Path(mount, operation, key)
	}, "update")

	return tui.SelectOrReadInput("transit key", keys, nil)
}

func init() {
	vaultTransitCmd.AddCommand(vaultTransitEncryptCmd)

	vaultTransitEncryptCmd.Flags().StringP(vaultTransitKey, "k", "", "Name of the transit key. If not specified, available keys are discovered.")
	vaultTransitEncryptCmd.Flags().StringP(vaultTransitContext, "c", "", "Context for key derivation, required if the key has derivation enabled")
	vaultTransitEncryptCmd.Flags().StringP(vaultTransitInput, "i", "", "File to read the plaintext from, '-' for stdin. If not specified, reads piped stdin or prompts.")
	vaultTransitEncryptCmd.Flags().StringP(vaultTransitOutput, "o", "", "File to write the ciphertext to. If not specified, prints to stdout.")
}
