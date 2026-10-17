package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/pw"
	"github.com/soerenschneider/sc/pkg/clipboard"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const passwordClearTimeout = 30 * time.Second

var passwordOpts pw.PasswordOptions

var pwGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generates a random password",
	Run: func(cmd *cobra.Command, args []string) {
		generatedPw, err := pw.GenerateXkcd(passwordOpts)
		if err != nil {
			log.Fatal().Err(err).Msg("could not generate password")
		}

		printPassword, _ := cmd.Flags().GetBool("print")
		if printPassword {
			printPasswordTemporarily(cmd.Context(), generatedPw)
		} else {
			if err := clipboard.CopyClipboard(cmd.Context(), generatedPw); err != nil {
				log.Fatal().Err(err).Msg("could not copy password to clipboard")
			}
			log.Info().Msgf("Generated password copied to clipboard")

			title := fmt.Sprintf("Waiting %v to wipe password, press ctrl+c to wipe now", passwordClearTimeout)
			err := spinner.New().
				Context(cmd.Context()).
				ActionWithErr(func(ctx context.Context) error {
					waitForPasswordClearTimeout(ctx)
					return nil
				}).
				Title(title).
				Type(spinner.Dots).
				Run()

			// Ctrl+C does not raise SIGINT while the spinner holds the terminal in raw mode, it is
			// handled by the spinner itself and reported as tea.ErrInterrupted instead.
			if err != nil && !errors.Is(err, tea.ErrInterrupted) && cmd.Context().Err() == nil {
				log.Warn().Err(err).Msg("could not display spinner")
				waitForPasswordClearTimeout(cmd.Context())
			}

			clearPassword(cmd.Context(), generatedPw)
		}
	},
}

// waitForPasswordClearTimeout blocks until passwordClearTimeout has passed or ctx is canceled.
func waitForPasswordClearTimeout(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(passwordClearTimeout):
	}
}

// clearPassword wipes the clipboard if it still contains generatedPw.
func clearPassword(ctx context.Context, generatedPw string) {
	// ctx may already be canceled, the clipboard needs to be wiped nevertheless.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	current, err := clipboard.PasteClipboard(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("could not read clipboard, not wiping it")
		return
	}

	if current == generatedPw {
		if err := clipboard.CopyClipboard(ctx, ""); err != nil {
			log.Warn().Err(err).Msg("could not wipe clipboard")
			return
		}
		log.Info().Msg("Wiped password from clipboard")
	}
}

// printPasswordTemporarily prints pw and removes it again after passwordClearTimeout, as soon as
// ctx is canceled or any key is pressed.
func printPasswordTemporarily(ctx context.Context, pw string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fd := int(os.Stdin.Fd()) //#nosec:G115

	// put terminal in raw mode (no Enter key, no echo). As this also disables signals, Ctrl+C
	// does not raise SIGINT anymore, so any key press cancels instead.
	oldState, err := term.MakeRaw(fd)
	if err == nil {
		defer func() {
			_ = term.Restore(fd, oldState)
		}()

		go func() {
			buf := make([]byte, 1)
			if _, err := os.Stdin.Read(buf); err == nil {
				cancel()
			}
		}()
	}

	fmt.Print(pw)
	waitForPasswordClearTimeout(ctx)
	fmt.Print("\r\033[K")
}

func init() {
	pwCmd.AddCommand(pwGenCmd)
	pwGenCmd.Flags().BoolVarP(&passwordOpts.Lowercase, "lowercase", "", false, "Converts all words to lowercase")
	pwGenCmd.Flags().BoolVarP(&passwordOpts.Special, "special", "", false, "Include special character")
	pwGenCmd.Flags().BoolP("print", "p", false, "Print password to stdout, do not copy to clipboard")
	pwGenCmd.Flags().IntVarP(&passwordOpts.NumWords, "words", "w", 4, "Number of words in the password")
	pwGenCmd.Flags().VarP(&passwordOpts.SpecialPos, "special-pos", "", "Position of special characters")
	pwGenCmd.Flags().StringVarP(&passwordOpts.Language, "language", "l", "de", "Wordlist language, e.g. 'en', 'de'")
	pwGenCmd.Flags().StringVarP(&passwordOpts.Separator, "separator", "s", " ", "Separator between words")
}
