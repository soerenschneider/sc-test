package tui

import (
	"os"

	"charm.land/lipgloss/v2"
)

func Max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// ---------------------------------------------------------------------------
// adaptive colour setup
// ---------------------------------------------------------------------------
//
// Evaluated once at package init based on the terminal's actual background.
// HasDarkBackground queries the controlling terminal via OSC 11 with a
// short timeout (~50-100ms); on a non-TTY it returns immediately with the
// default (dark). For CLIs that get piped a lot, the cost is paid only
// when stdout is actually a terminal.
//
// Override the exported *Style vars below to theme globally.

var (
	HasDarkBg = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	LightDark = lipgloss.LightDark(HasDarkBg)

	// Foundation colours. The light-mode shades skew darker so they
	// remain legible on a white background (pure ANSI yellow is
	// notoriously unreadable on white); the dark-mode shades are
	// brighter so they pop against black.
	errorFg = LightDark(lipgloss.Color("#c0392b"), lipgloss.Color("#fe5f86"))
	warnFg  = LightDark(lipgloss.Color("#b7791f"), lipgloss.Color("#f1c40f"))
	infoFg  = LightDark(lipgloss.Color("#0277bd"), lipgloss.Color("#5bc0eb"))
	debugFg = LightDark(lipgloss.Color("#737373"), lipgloss.Color("#8a8a8a"))
	dimFg   = LightDark(lipgloss.Color("#737373"), lipgloss.Color("#8a8a8a"))
	// identityFg deliberately uses a magenta-purple that's distinct from
	// every level colour above (red / amber / blue / grey). On light bg
	// it's a medium purple readable on white; on dark bg it's lifted to
	// a softer mauve that pops without competing with red errors.
	identityFg = LightDark(lipgloss.Color("#9c27b0"), lipgloss.Color("#d4b3ff"))
)

// ---------------------------------------------------------------------------
// shared styles
// ---------------------------------------------------------------------------

var (
	DirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)
	SecretStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	LeafStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("237")).
			Padding(0, 1)

	PathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	MetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	HistoricalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	HistoricalTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("214")).
				Bold(true).
				Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("110")).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	MaskedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("76"))

	DirtyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	// ErrorStyle is the level tag for error/fatal/crit/critical.
	// Bold + red to draw the eye even when scanning quickly.
	ErrorStyle = lipgloss.NewStyle().Foreground(errorFg).Bold(true)
	// WarnStyle is the level tag for warn/warning.
	WarnStyle = lipgloss.NewStyle().Foreground(warnFg)
	// InfoStyle is the level tag for info.
	InfoStyle = lipgloss.NewStyle().Foreground(infoFg)
	// DebugStyle is the level tag for debug/trace.
	DebugStyle = lipgloss.NewStyle().Foreground(debugFg)
	// DimStyle is for secondary text — timestamps, field-key prefixes,
	// anything structural that shouldn't compete with the message.
	DimStyle = lipgloss.NewStyle().Foreground(dimFg)
	// IdentityStyle renders the per-line origin (host/service/app/…) in
	// a colour distinct from levels and dim — magenta — so the reader
	// can scan "where did this come from" independently from "how bad is
	// it" and "what's structural".
	IdentityStyle = lipgloss.NewStyle().Foreground(identityFg)
)
