package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// public API
// ---------------------------------------------------------------------------

// Align is per-column horizontal alignment.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Table is a styled rectangular table. Construct a literal and call
// Render() or Print(). The zero value is a usable empty table.
type Table struct {
	// Title is an accent-coloured underlined heading rendered above the
	// table. Omit for a flush top edge.
	Title string

	// Caption is a dim italic line beneath the table — typically used
	// for metadata: row counts, "as of" timestamps, source notes.
	Caption string

	// Headers is the column-label row inside the border.
	Headers []string

	// Rows is the table body, one []string per row. Cells may already
	// contain ANSI escape codes; lipgloss preserves them, so callers can
	// embed per-cell colour (status badges, diff markers) by rendering
	// the cell with their own lipgloss style before adding it here.
	Rows [][]string

	// Aligns sets per-column alignment, indexed left to right. Missing
	// or short entries default to AlignLeft. Right-aligning numeric
	// columns is the single biggest readability win for most tables.
	Aligns []Align

	// FullWidth stretches the table to the terminal width when stdout
	// is a TTY. Ignored when piped, so downstream tools see a stable
	// column layout regardless of the receiver's terminal.
	FullWidth bool

	// Wrap allows long cell content to wrap inside its column rather
	// than be truncated. Off by default — most tables read better with
	// single-line rows; turn on for log-style messages.
	Wrap bool

	// Zebra paints alternating row backgrounds. Useful past ~10 rows;
	// looks needlessly busy on short tables.
	Zebra bool

	// EmptyMessage replaces the body when Rows is empty. Defaults to
	// "(no data)". To suppress entirely, set to a single space " ".
	EmptyMessage string

	// MaxWidths caps each column's width in display columns. Cell content
	// exceeding the limit is truncated with "…". Zero or missing entries
	// impose no limit for that column — it grows to fit its widest cell,
	// which is what blows up tables containing URLs or long messages.
	//
	// Truncation is rune-based and applies to headers too, so columns
	// never grow beyond their cap. Typical use:
	//
	//   MaxWidths: []int{40, 60, 20, 0, 0}  // cap title/URL/tags only
	//
	// Note: this is independent of Wrap. Wrap=false is "don't wrap and
	// let the column grow as needed" (lipgloss's default), not "truncate"
	// — if a column has long content and you want a stable table width,
	// you need MaxWidths regardless of Wrap.
	MaxWidths []int

	// LinkFunc, when non-nil, returns a URL for each body cell to make
	// that cell a clickable hyperlink in supporting terminals (iTerm2,
	// kitty, wezterm, gnome-terminal, recent alacritty, VS Code's
	// terminal). An empty return means "no link for this cell".
	//
	// LinkFunc is called with the FULL cell content even when MaxWidths
	// truncates the display — so a long URL in a narrow column shows as
	// "https://exampl…" but the link opens the full URL.
	//
	// Common patterns:
	//
	//   // URL column links to itself:
	//   LinkFunc: tui.LinkSelf(1),
	//
	//   // Title column links to URL column (need the rows in scope):
	//   LinkFunc: func(row, col int, _ string) string {
	//       if col == 0 { return rows[row][1] }
	//       return ""
	//   },
	//
	// Terminals without OSC 8 support render the escapes as no-ops, so
	// enabling LinkFunc is safe for interactive use. When piping the
	// table to grep/awk/files, the escapes appear as garbage — gate on
	// a TTY check in that case.
	LinkFunc func(row, col int, content string) string

	// StyleFunc overrides cell styling completely. When non-nil, Aligns
	// and Zebra are ignored — your function picks the style for every
	// cell, receiving table.HeaderRow for the header row.
	StyleFunc func(row, col int) lipgloss.Style
}

// Print writes the rendered table to stdout. Uses lipgloss.Println so
// colours are correctly downsampled for the active terminal.
func (t Table) Print() {
	_, _ = lipgloss.Println(t.Render())
}

// Fprint writes the rendered table to w followed by a newline.
func (t Table) Fprint(w io.Writer) {
	_, _ = fmt.Fprintln(w, t.Render())
}

// Render returns the styled table as a single string with no trailing
// newline. Safe to call without a TTY — colour output is downsampled by
// lipgloss when the destination doesn't support it.
func (t Table) Render() string {
	var out strings.Builder

	if t.Title != "" {
		out.WriteString(TableTitleStyle.Render(t.Title))
		out.WriteString("\n\n")
	}

	if len(t.Rows) == 0 {
		msg := t.EmptyMessage
		if msg == "" {
			msg = "(no data)"
		}
		out.WriteString(TableCaptionStyle.Render("  " + msg))
		return out.String()
	}

	headers, rows := t.preparedCells()

	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(TableBorderStyle).
		Headers(headers...).
		Rows(rows...).
		Wrap(t.Wrap).
		StyleFunc(t.makeStyleFunc())

	if t.FullWidth {
		if w, ok := terminalWidth(); ok {
			tbl = tbl.Width(w)
		}
	}

	out.WriteString(tbl.String())

	if t.Caption != "" {
		out.WriteString("\n\n")
		out.WriteString(TableCaptionStyle.Render(t.Caption))
	}
	return out.String()
}

// makeStyleFunc returns the cell-styling function the lipgloss table
// uses internally. When the caller provides a StyleFunc, that wins; the
// default path applies Aligns and (optionally) Zebra.
func (t Table) makeStyleFunc() func(row, col int) lipgloss.Style {
	if t.StyleFunc != nil {
		return t.StyleFunc
	}
	return func(row, col int) lipgloss.Style {
		pos := lipgloss.Left
		if col >= 0 && col < len(t.Aligns) {
			pos = positionFor(t.Aligns[col])
		}
		if row == table.HeaderRow {
			return TableHeaderStyle.Align(pos)
		}
		s := TableBodyStyle.Align(pos)
		if t.Zebra && row >= 0 && row%2 == 1 {
			s = s.Background(zebraBg)
		}
		return s
	}
}

// ---------------------------------------------------------------------------
// styles — exported so callers can theme globally; the defaults adapt to
// the terminal background detected once at package init.
// ---------------------------------------------------------------------------

var (
	// Foundation colours — picked to read well on both light and dark
	// backgrounds without screaming. Override these (or the *Style vars
	// below) for project-specific theming.
	accentFg = LightDark(lipgloss.Color("#5f4dc2"), lipgloss.Color("#c0a3ff"))
	mutedFg  = LightDark(lipgloss.Color("#737373"), lipgloss.Color("#8a8a8a"))
	borderFg = LightDark(lipgloss.Color("#bfbfbf"), lipgloss.Color("#4a4a4a"))
	zebraBg  = LightDark(lipgloss.Color("#f5f5f5"), lipgloss.Color("#1c1c1c"))

	// TableTitleStyle is the heading above the table.
	TableTitleStyle = lipgloss.NewStyle().
			Foreground(accentFg).
			Bold(true).
			Underline(true)

	// TableHeaderStyle is each cell in the header row.
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(accentFg).
				Bold(true).
				Padding(0, 2)

	// TableBodyStyle is each cell in the body rows.
	TableBodyStyle = lipgloss.NewStyle().
			Padding(0, 2)

	// TableCaptionStyle is the dim italic line beneath the table and
	// the "(no data)" empty-state message.
	TableCaptionStyle = lipgloss.NewStyle().
				Foreground(mutedFg).
				Italic(true)

	// TableBorderStyle paints the table's rounded border.
	TableBorderStyle = lipgloss.NewStyle().
				Foreground(borderFg)
)

// ---------------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------------

func positionFor(a Align) lipgloss.Position {
	switch a {
	case AlignCenter:
		return lipgloss.Center
	case AlignRight:
		return lipgloss.Right
	default:
		return lipgloss.Left
	}
}

// preparedCells applies MaxWidths and LinkFunc to produce the rows and
// headers the lipgloss table actually renders. The order matters: the
// link URL must come from the FULL cell content, but the visible text
// is the truncated string — so we truncate first, then wrap. That way
// a truncated cell like "https://exampl…" still links to the full URL.
//
// Headers are never wrapped as links — they describe data, they aren't
// data — but they are truncated, so the column can't grow wider than
// its cap.
//
// When neither MaxWidths nor LinkFunc is set, the inputs pass through
// unchanged — no allocations, no copying.
func (t Table) preparedCells() ([]string, [][]string) {
	if len(t.MaxWidths) == 0 && t.LinkFunc == nil {
		return t.Headers, t.Rows
	}

	headers := t.Headers
	if len(t.MaxWidths) > 0 {
		headers = make([]string, len(t.Headers))
		for i, h := range t.Headers {
			headers[i] = clipCol(h, t.MaxWidths, i)
		}
	}

	rows := make([][]string, len(t.Rows))
	for i, r := range t.Rows {
		nr := make([]string, len(r))
		for j, full := range r {
			display := clipCol(full, t.MaxWidths, j)
			if t.LinkFunc != nil {
				if url := t.LinkFunc(i, j, full); url != "" {
					display = Hyperlink(display, url)
				}
			}
			nr[j] = display
		}
		rows[i] = nr
	}
	return headers, rows
}

func clipCol(s string, widths []int, col int) string {
	if col >= len(widths) || widths[col] <= 0 {
		return s
	}
	return clip(s, widths[col])
}

// clip is rune-based: one rune = one display column. That's wrong for
// CJK and emoji (which take 2 columns each) but right for ASCII — and
// ASCII covers URLs, IDs, code, log lines, and most of what people
// actually shove into wide columns. Callers who need exact display-width
// truncation can pre-truncate with lipgloss/ansi.Truncate themselves.
func clip(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}

// ---------------------------------------------------------------------------
// hyperlinks (OSC 8)
// ---------------------------------------------------------------------------

// Hyperlink wraps text in an OSC 8 escape sequence so terminals that
// support it render `text` as a clickable link to `url`. Supporting
// terminals include iTerm2, kitty, wezterm, gnome-terminal, recent
// alacritty, and most modern emulators (including the integrated
// terminals in VS Code and JetBrains IDEs). Terminals without support
// display just the text — the escapes are silently ignored — so this
// is safe to use unconditionally for interactive output.
//
// If url is empty, returns text unchanged (no escape sequence emitted).
//
// The form `ESC ] 8 ;; URL ESC \ TEXT ESC ] 8 ;; ESC \` follows the de
// facto OSC 8 spec. ESC-backslash (ST) is used as the terminator rather
// than BEL because all modern terminals accept it and ST is closer to
// the formal spec.
//
// Caveat: the escape codes look like garbage to tools that don't strip
// ANSI (grep, awk, plain `cat > file.txt`). When piping a table to
// those, gate LinkFunc behind a TTY check or strip codes downstream.
func Hyperlink(text, url string) string {
	if url == "" {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// LinkSelf returns a LinkFunc that turns the given columns' content
// into clickable links pointing at the content itself. Use this when a
// column contains URLs that should both display as the URL and open
// the URL on click:
//
//	Table{
//	    Headers: []string{"Title", "URL", "Tags"},
//	    Rows:    rows,
//	    LinkFunc: tui.LinkSelf(1),  // column 1 (URL) is clickable
//	}.Print()
//
// For "click cell A to open URL in cell B", write a small closure
// directly — it's a one-liner and avoids over-abstracting.
func LinkSelf(cols ...int) func(row, col int, content string) string {
	set := make(map[int]bool, len(cols))
	for _, c := range cols {
		set[c] = true
	}
	return func(_, col int, content string) string {
		if set[col] {
			return content
		}
		return ""
	}
}

// terminalWidth returns the stdout terminal width when stdout is a TTY.
// Returns (0, false) for pipes, redirects, or any other non-TTY — in those
// cases FullWidth should be ignored so output stays stable.
func terminalWidth() (int, bool) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0, false
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}
