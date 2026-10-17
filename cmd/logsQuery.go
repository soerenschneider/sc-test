package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/victorialogs"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const defaultLogsQuery = "error AND _time:45m"

var logsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logs from VictoriaLogs using a logQL-like expression",
	Long: `Query logs stored in VictoriaLogs by providing filter expressions and optional time ranges.

This command allows you to search logs with filters, labels, and time constraints. It supports various output formats
for integration with scripts or for human-readable views.`,
	Run: func(cmd *cobra.Command, args []string) {
		logsOpts := &logsOpts{
			url:             pkg.GetString(cmd, logsAddr),
			query:           pkg.GetString(cmd, logsQuery),
			format:          pkg.GetString(cmd, logsFormat),
			idFields:        pkg.GetStringArray(cmd, logsIdFields),
			since:           pkg.GetString(cmd, logsSince),
			until:           pkg.GetString(cmd, logsUntil),
			noColor:         pkg.MustGetBool(cmd, logsNoColor),
			limit:           pkg.GetInt(cmd, logsLimit),
			refreshInterval: pkg.GetDuration(cmd, logsRefreshInterval),
			follow:          pkg.MustGetBool(cmd, logsFollow),
			timeout:         pkg.GetDuration(cmd, logsTimeout),
		}

		if err := runLogs(logsOpts); err != nil {
			log.Fatal().Err(err).Msg("could not run query")
		}
	},
}

func init() {
	logsCmd.AddCommand(logsQueryCmd)

	logsQueryCmd.Flags().StringP(logsQuery, "q", defaultLogsQuery, "Query")
	logsQueryCmd.Flags().StringP(logsFormat, "", "text", "Output format: text, json, raw")
	logsQueryCmd.Flags().StringArrayP(logsIdFields, "i",
		[]string{"host", "hostname", "service", "svc", "app"},
		"Identity fields highlighted before the message (joined with '/'; use '_stream' to include the VictoriaLogs stream)")
	logsQueryCmd.Flags().StringP(logsSince, "s", "",
		"Start time: duration (1h), RFC3339, YYYY-MM-DD, YYYY-MM-DD HH:MM:SS, or Unix epoch")
	logsQueryCmd.Flags().StringP(logsUntil, "u", "", "End time (same formats as --since)")
	logsQueryCmd.Flags().DurationP(logsRefreshInterval, "r", 2*time.Second,
		"Tail refresh interval (must be > 0)")
	logsQueryCmd.Flags().BoolP(logsFollow, "f", false, "Whether to follow logs")
	logsQueryCmd.Flags().BoolP(logsNoColor, "", false, "Disable ANSI color in text output")
	logsQueryCmd.Flags().IntP(logsLimit, "l", 100, "Maximum entries (bounded mode and replay phase)")
	logsQueryCmd.Flags().DurationP(logsTimeout, "", 10*time.Second, "Timeout")
}

// logsOpts is the CLI-side option bag. Strings here are raw flag values;
// they get converted to typed inputs (time.Time, etc.) before being handed
// to the victorialogs package.
type logsOpts struct {
	url    string
	query  string
	follow bool
	limit  int

	since string
	until string

	fields []string
	format string

	idFields []string

	tenant   string
	username string
	password string
	token    string

	refreshInterval time.Duration
	noColor         bool
	insecure        bool
	timeout         time.Duration
}

// ---------------------------------------------------------------------------
// main flow
// ---------------------------------------------------------------------------

func runLogs(opts *logsOpts) error {
	cfg := victorialogs.Config{
		URL:      opts.url,
		Username: opts.username,
		Password: opts.password,
		Token:    opts.token,
		Tenant:   opts.tenant,
		Insecure: opts.insecure,
	}

	client, err := victorialogs.NewClient(cfg)
	if err != nil {
		return err
	}

	useColor := !opts.noColor && os.Getenv("NO_COLOR") == "" && isTerminal(os.Stdout)
	fmtr := &logsFormatter{
		out:      os.Stdout,
		format:   opts.format,
		fields:   opts.fields,
		idFields: opts.idFields,
		color:    useColor,
	}

	// SIGINT / SIGTERM cancel the context — tail unblocks, bounded query
	// stops mid-stream, command exits cleanly.
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if opts.follow {
		return doTail(ctx, client, opts, fmtr)
	}
	return doQuery(ctx, client, opts, fmtr)
}

// doQuery runs a bounded query and prints the result in chronological order.
//
// Behavior worth knowing:
//
//   - We use the server's URL "limit" parameter rather than `| sort | limit`
//     pipeline tricks: the sort pipe is blocking and forces the server to
//     buffer everything before emitting a single byte, which on an unbounded
//     query hits server-side query-duration caps.
//
//   - If the caller specifies neither --since nor --until, we apply a 1-day
//     lookback default. VictoriaLogs technically supports unbounded queries
//     ("from the smallest available timestamp"), but in practice that
//     produces silent empty responses on busy servers — the docs themselves
//     note that the Web UI and Grafana plugin always pass an explicit time
//     range. Users who genuinely want all of history pass --since 0 or a
//     long duration like 30d explicitly.
//
//   - Results may arrive in arbitrary order even with the URL limit applied,
//     so we buffer (bounded by limit) and sort by _time client-side.
func doQuery(ctx context.Context, c *victorialogs.Client, opts *logsOpts, out *logsFormatter) error {
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	qopts := victorialogs.QueryOptions{
		Query: opts.query,
		Limit: opts.limit,
	}

	since := opts.since
	if since == "" && opts.until == "" {
		since = "1d"
	}
	if since != "" {
		t, err := pkg.ParseTime(since)
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}
		qopts.Start = t
	}
	if opts.until != "" {
		t, err := pkg.ParseTime(opts.until)
		if err != nil {
			return fmt.Errorf("--until: %w", err)
		}
		qopts.End = t
	}

	bufSize := opts.limit
	if bufSize <= 0 {
		bufSize = 100
	}
	entries := make([]victorialogs.LogEntry, 0, bufSize)
	if err := c.Query(ctx, qopts, func(e victorialogs.LogEntry) {
		entries = append(entries, e)
	}); err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.Before(entries[j].Time)
	})
	for _, e := range entries {
		out.write(e)
	}
	return nil
}

// doTail opens a live-tail stream. When --since is set, the historical
// window is replayed first via a bounded query, then the tail takes over.
// The replay phase uses --limit as a safety cap (defaulting to 1000 when
// unset) so very chatty streams don't blast 20 minutes of logs first.
func doTail(ctx context.Context, c *victorialogs.Client, opts *logsOpts, out *logsFormatter) error {
	if opts.since != "" {
		replay := *opts
		replay.follow = false
		replay.until = "" // [since, now]
		if replay.limit <= 0 {
			replay.limit = 1000
		}
		if err := doQuery(ctx, c, &replay, out); err != nil {
			return fmt.Errorf("replay: %w", err)
		}
	}
	return c.Tail(ctx, victorialogs.TailOptions{
		Query:           opts.query,
		RefreshInterval: opts.refreshInterval,
	}, out.write)
}

// ---------------------------------------------------------------------------
// output formatting
// ---------------------------------------------------------------------------

type logsFormatter struct {
	out      io.Writer
	format   string
	fields   []string
	idFields []string
	color    bool
}

func (f *logsFormatter) write(e victorialogs.LogEntry) {
	switch f.format {
	case "json":
		f.writeJSON(e)
	case "raw":
		_, _ = fmt.Fprintln(f.out, e.Msg)
	default:
		f.writeText(e)
	}
}

func (f *logsFormatter) writeJSON(e victorialogs.LogEntry) {
	obj := make(map[string]string, len(e.Fields)+3)
	for k, v := range e.Fields {
		obj[k] = v
	}
	if !e.Time.IsZero() {
		obj["_time"] = e.Time.UTC().Format(time.RFC3339Nano)
	}
	if e.Msg != "" {
		obj["_msg"] = e.Msg
	}
	if e.Stream != "" {
		obj["_stream"] = e.Stream
	}
	enc := json.NewEncoder(f.out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(obj)
}

func (f *logsFormatter) writeText(e victorialogs.LogEntry) {
	var b strings.Builder

	if !e.Time.IsZero() {
		b.WriteString(dim(e.Time.Local().Format("15:04:05.000"), f.color))
		b.WriteByte(' ')
	}
	if id := f.renderIdentity(e); id != "" {
		b.WriteString(id)
		b.WriteByte(' ')
	}
	if level := commonLevel(e); level != "" {
		b.WriteString(colorLevel(level, f.color))
		b.WriteByte(' ')
	}
	b.WriteString(e.Msg)

	for _, fld := range f.fields {
		if v, ok := e.Fields[fld]; ok && v != "" {
			b.WriteByte(' ')
			b.WriteString(dim(fld+"=", f.color))
			b.WriteString(v)
		}
	}
	b.WriteByte('\n')
	_, _ = f.out.Write([]byte(b.String()))
}

// renderIdentity walks the configured id fields in order and joins their
// non-empty values with '/'. Values that already appeared (e.g. when both
// host and hostname carry the same string) are deduplicated so the output
// stays compact. Returns the empty string when no id fields are populated,
// so the writer can skip the leading space.
//
// The special name "_stream" reads the LogEntry.Stream typed field rather
// than the Fields map, since VictoriaLogs extracts that field at decode
// time and it isn't visible in Fields.
func (f *logsFormatter) renderIdentity(e victorialogs.LogEntry) string {
	if len(f.idFields) == 0 {
		return ""
	}
	values := make([]string, 0, len(f.idFields))
	seen := map[string]bool{}
	for _, name := range f.idFields {
		var v string
		if name == "_stream" {
			v = e.Stream
		} else {
			v = e.Fields[name]
		}
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		values = append(values, v)
	}
	if len(values) == 0 {
		return ""
	}
	joined := strings.Join(values, "/")
	if !f.color {
		return joined
	}
	return tui.IdentityStyle.Render(joined)
}

// commonLevel checks the conventional level-field names in order:
// level (stdlib, zap, logrus), severity (GCP), log.level (Elastic),
// severity_text (OpenTelemetry).
func commonLevel(e victorialogs.LogEntry) string {
	for _, key := range []string{"level", "severity", "log.level", "severity_text"} {
		if v, ok := e.Fields[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

// colorLevel renders the LEVEL token of a log line, e.g. "[ERROR]".
//
// When color is false, returns the bracketed plain text — used when stdout
// isn't a TTY or the user passed --no-color. lipgloss also auto-downsamples
// when the destination isn't a TTY, but short-circuiting here saves the
// styling work entirely and keeps grep-friendly output truly plain.
func colorLevel(level string, color bool) string {
	tag := "[" + strings.ToUpper(level) + "]"
	if !color {
		return tag
	}
	switch strings.ToLower(level) {
	case "error", "err", "fatal", "crit", "critical":
		return tui.ErrorStyle.Render(tag)
	case "warn", "warning":
		return tui.WarnStyle.Render(tag)
	case "info":
		return tui.InfoStyle.Render(tag)
	case "debug", "trace":
		return tui.DebugStyle.Render(tag)
	default:
		return tag
	}
}

// dim returns text in a muted foreground colour, for secondary information
// like timestamps and field-key prefixes. No-op when color is false.
func dim(s string, color bool) string {
	if !color {
		return s
	}
	return tui.DimStyle.Render(s)
}

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
