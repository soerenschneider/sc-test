package tui

import (
	"context"
	"os"
	"time"

	"charm.land/huh/v2/spinner"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

const (
	// spinnerDelay is how long an action may take before a spinner is displayed.
	spinnerDelay = 200 * time.Millisecond
	// spinnerMinVisible is how long a spinner is displayed at least, once shown.
	spinnerMinVisible = 300 * time.Millisecond
)

// RunWithSpinner runs action while displaying a spinner. As the spinner requires a terminal,
// action is run without a spinner if stderr is not a terminal, e.g. when used in scripts.
//
// The spinner is only displayed if action does not finish within spinnerDelay and is then
// displayed for at least spinnerMinVisible. Besides avoiding flickering, this makes sure the
// spinner lives long enough to consume the terminal's replies to the queries bubbletea sends
// on startup, which would otherwise leak into the shell.
func RunWithSpinner(ctx context.Context, title string, action func(ctx context.Context) error) error {
	if !term.IsTerminal(int(os.Stderr.Fd())) { //#nosec:G115
		return action(ctx)
	}

	done := make(chan error, 1)
	go func() {
		done <- action(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(spinnerDelay):
	}

	return spinner.New().
		ActionWithErr(func(_ context.Context) error {
			minVisible := time.After(spinnerMinVisible)
			err := <-done
			<-minVisible
			return err
		}).
		Title(title).
		Context(ctx).
		Type(spinner.Dots).
		WithOutput(os.Stderr).
		Run()
}

// LoadWithSpinner runs fetch while displaying a spinner with the given title, see RunWithSpinner.
func LoadWithSpinner[T any](ctx context.Context, title string, fetch func(ctx context.Context) (T, error)) (T, error) {
	var ret T
	err := RunWithSpinner(ctx, title, func(ctx context.Context) error {
		var err error
		ret, err = fetch(ctx)
		return err
	})
	return ret, err
}

// SelectOrReadInput returns the only choice if there is exactly one, offers a selection if there
// are multiple choices and falls back to free-form input with the given suggestions otherwise.
func SelectOrReadInput(name string, choices []string, suggestions []string) string {
	switch len(choices) {
	case 0:
		return ReadInput("Enter "+name, suggestions)
	case 1:
		log.Info().Msgf("Using only available %s %q", name, choices[0])
		return choices[0]
	default:
		return SelectInput("Select "+name, choices)
	}
}
