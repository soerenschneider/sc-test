package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var vaultTransitDecryptCmd = &cobra.Command{
	Use:     "decrypt",
	Aliases: []string{"dec"},
	Short:   "Decrypt data using a transit key",
	Long: `Decrypts a ciphertext using a key of the Vault transit secret engine and prints the resulting plaintext.

The ciphertext is read from the file given by --input ('-' for stdin). If no input file is
given and data is piped to stdin, stdin is read. Otherwise, the ciphertext is prompted for
interactively.

Mount and key may be passed as flags. Anything that is not passed is discovered and
offered for selection, falling back to free-form input if discovery is not possible.

Examples:
  sc vault transit decrypt
  sc vault transit decrypt -m transit -k my-key -i secret.enc -o secret.txt
  sc vault transit encrypt -m transit -k my-key | sc vault transit decrypt -m transit -k my-key`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		mount := pkg.GetString(cmd, vaultMountPath)
		keyName := pkg.GetString(cmd, vaultTransitKey)
		keyContext := pkg.GetString(cmd, vaultTransitContext)
		inputFile := pkg.GetString(cmd, vaultTransitInput)
		outputFile := pkg.GetString(cmd, vaultTransitOutput)

		input, err := readTransitInput(inputFile, func() string {
			return tui.ReadInputWithValidation("Enter ciphertext", nil, validateTransitCiphertext)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not read ciphertext")
		}

		ciphertext := strings.TrimSpace(string(input))
		if err := validateTransitCiphertext(ciphertext); err != nil {
			log.Fatal().Err(err).Msg("invalid ciphertext")
		}

		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "transit")
		}

		if keyName == "" {
			keyName = promptTransitKey(cmd.Context(), client, mount, "decrypt")
		}
		prompt.RequireCapability(cmd.Context(), client, vault.Path(mount, "decrypt", keyName), "update")

		ctx, cancel := context.WithTimeout(cmd.Context(), vaultDefaultTimeout)
		defer cancel()

		var plaintext []byte
		title := fmt.Sprintf("Decrypting data using key %q on mount %q...", keyName, mount)
		if err := tui.RunWithSpinner(ctx, title, func(ctx context.Context) error {
			var err error
			plaintext, err = client.TransitDecrypt(ctx, mount, keyName, ciphertext, []byte(keyContext))
			return err
		}); err != nil {
			log.Fatal().Err(err).Msg("could not decrypt data")
		}

		if outputFile != "" {
			if err := os.WriteFile(pkg.GetExpandedFile(outputFile), plaintext, 0600); err != nil {
				log.Fatal().Err(err).Msg("could not write plaintext")
			}
			log.Info().Msgf("Plaintext written to %q", outputFile)
			return
		}

		// Write the plaintext as-is so it can be piped, only add a trailing newline for terminals.
		if isTerminal(os.Stdout) && !bytes.HasSuffix(plaintext, []byte("\n")) {
			plaintext = append(plaintext, '\n')
		}
		if _, err := os.Stdout.Write(plaintext); err != nil {
			log.Fatal().Err(err).Msg("could not write plaintext")
		}
	},
}

// validateTransitCiphertext performs a basic sanity check that s looks like a transit ciphertext,
// e.g. "vault:v1:...".
func validateTransitCiphertext(s string) error {
	if s == "" {
		return errors.New("ciphertext must not be empty")
	}
	if !strings.HasPrefix(s, "vault:v") {
		return errors.New("ciphertext must start with 'vault:v'")
	}
	return nil
}

func init() {
	vaultTransitCmd.AddCommand(vaultTransitDecryptCmd)

	vaultTransitDecryptCmd.Flags().StringP(vaultTransitKey, "k", "", "Name of the transit key. If not specified, available keys are discovered.")
	vaultTransitDecryptCmd.Flags().StringP(vaultTransitContext, "c", "", "Context for key derivation, required if the key has derivation enabled")
	vaultTransitDecryptCmd.Flags().StringP(vaultTransitInput, "i", "", "File to read the ciphertext from, '-' for stdin. If not specified, reads piped stdin or prompts.")
	vaultTransitDecryptCmd.Flags().StringP(vaultTransitOutput, "o", "", "File to write the plaintext to. If not specified, prints to stdout.")
}
