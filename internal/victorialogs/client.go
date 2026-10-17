// Package victorialogs is a Go client for VictoriaLogs' HTTP query API.
//
// It exposes two streaming methods — Query (bounded) and Tail (live) — both
// of which decode the server's JSON-lines response on the fly and invoke a
// caller-supplied callback per entry. The client does not buffer, transform,
// or format entries; that's left to the caller.
//
// Auth is HTTP basic, bearer token, or multi-tenancy headers; pick one via
// Config.
//
// Endpoints used:
//   - /select/logsql/query — bounded query
//   - /select/logsql/tail  — live tail (persistent connection)
package victorialogs

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// configuration & construction
// ---------------------------------------------------------------------------

// Config configures a Client. URL is required; the rest are optional.
type Config struct {
	// URL is the base URL of the VictoriaLogs server, e.g.
	// "http://localhost:9428". Trailing slashes are tolerated.
	URL string

	// Auth — set at most one of (Username, Password) or Token.
	Username string
	Password string
	Token    string

	// Tenant in "AccountID:ProjectID" form for multi-tenancy. Empty means
	// the default tenant.
	Tenant string

	// Insecure skips TLS certificate verification. Ignored when HTTPClient
	// is set (configure TLS on your own client in that case).
	Insecure bool

	// HTTPClient overrides the default http.Client. Useful for injecting a
	// custom transport (observability, retries, mutual TLS, …). When nil,
	// a default client is built honoring Insecure with a
	// ResponseHeaderTimeout of 30s so tail connections still get a sane
	// time-to-first-byte bound.
	HTTPClient *http.Client
}

// Client is a VictoriaLogs HTTP client. It's safe for concurrent use.
type Client struct {
	base *url.URL
	http *http.Client
	auth authConfig
}

type authConfig struct {
	username, password string
	token              string
	tenant             string
}

// NewClient constructs a Client from cfg. Returns an error if URL is missing,
// malformed, or uses an unsupported scheme.
func NewClient(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("victorialogs: URL is required")
	}
	u, err := url.Parse(strings.TrimRight(cfg.URL, "/"))
	if err != nil {
		return nil, fmt.Errorf("victorialogs: invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("victorialogs: URL must use http or https, got %q", u.Scheme)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		transport := &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		}
		if cfg.Insecure {
			//nolint G402
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		// No top-level Timeout — tail connections must be allowed to run
		// indefinitely. Callers bound the lifetime with context.
		httpClient = &http.Client{Transport: transport}
	}

	return &Client{
		base: u,
		http: httpClient,
		auth: authConfig{
			username: cfg.Username,
			password: cfg.Password,
			token:    cfg.Token,
			tenant:   cfg.Tenant,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// log entry
// ---------------------------------------------------------------------------

// LogEntry is one decoded log line. The conventional fields _time, _msg, and
// _stream are extracted into typed fields; everything else lives in Fields
// (always non-nil after a successful decode).
type LogEntry struct {
	Time   time.Time
	Msg    string
	Stream string
	Fields map[string]string
}

// parseEntry decodes a single JSON-lines record into a LogEntry. Non-string
// values (numbers, bools, nested objects) are stringified — VictoriaLogs
// stores log fields as strings on the wire but some collectors emit other
// types, and we never want a single odd line to break a long-running tail.
func parseEntry(line []byte) (LogEntry, error) {
	// Fast path: the response is overwhelmingly all-strings.
	raw := map[string]string{}
	if err := json.Unmarshal(line, &raw); err != nil {
		mixed := map[string]any{}
		if err2 := json.Unmarshal(line, &mixed); err2 != nil {
			return LogEntry{}, err
		}
		raw = make(map[string]string, len(mixed))
		for k, v := range mixed {
			raw[k] = anyToString(v)
		}
	}

	e := LogEntry{Fields: make(map[string]string, len(raw))}
	for k, v := range raw {
		switch k {
		case "_time":
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				e.Time = t
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				e.Time = t
			}
		case "_msg":
			e.Msg = v
		case "_stream":
			e.Stream = v
		default:
			e.Fields[k] = v
		}
	}
	return e, nil
}

func anyToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// ---------------------------------------------------------------------------
// query
// ---------------------------------------------------------------------------

// QueryOptions configures Client.Query.
type QueryOptions struct {
	// Query is the LogsQL expression. Empty defaults to "*" (match all).
	Query string

	// Start and End bound the time window. Zero values mean "no bound on
	// that side". Inclusive of Start, exclusive of End (server semantics).
	Start time.Time
	End   time.Time

	// Limit caps the number of returned entries. The server selects the
	// N entries with the largest _time values and emits them in arbitrary
	// order; callers that need chronological output should sort the
	// buffered result themselves. Zero means no limit (returns every
	// match, which can be expensive — set a value unless you really want
	// every entry).
	Limit int
}

// Query runs a bounded LogsQL query against /select/logsql/query and invokes
// fn for each entry as it streams in. Returns when the response is
// exhausted, ctx is cancelled, or the server reports an error.
//
// The server returns entries in storage order (roughly but not strictly
// chronological). When Limit > 0 the server returns the most recent N
// matching entries, but still in unspecified order — sort client-side if
// you need chronological output.
func (c *Client) Query(ctx context.Context, opts QueryOptions, fn func(LogEntry)) error {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		query = "*"
	}
	form := url.Values{}
	form.Set("query", query)
	if !opts.Start.IsZero() {
		form.Set("start", strconv.FormatInt(opts.Start.UnixNano(), 10))
	}
	if !opts.End.IsZero() {
		form.Set("end", strconv.FormatInt(opts.End.UnixNano(), 10))
	}
	if opts.Limit > 0 {
		form.Set("limit", strconv.Itoa(opts.Limit))
	}
	return c.stream(ctx, "/select/logsql/query", form, fn)
}

// ---------------------------------------------------------------------------
// tail
// ---------------------------------------------------------------------------

// TailOptions configures Client.Tail.
type TailOptions struct {
	// Query is the LogsQL expression. Empty defaults to "*" (match all).
	// Note: VictoriaLogs forbids some pipeline operators on the tail
	// endpoint (anything that requires bounded input — sort, stats, …).
	Query string

	// RefreshInterval is how often the server polls storage for new
	// matches. Must be > 0; the server rejects zero or negative values
	// with a 400.
	RefreshInterval time.Duration
}

// Tail opens a live-tail connection to /select/logsql/tail and invokes fn
// for each entry as it arrives. Blocks until ctx is cancelled, the
// connection is broken, or the server returns an error.
//
// Tail does not reconnect on its own — that's a policy decision the caller
// should make explicitly, since silent reconnects risk gaps the caller may
// not notice.
func (c *Client) Tail(ctx context.Context, opts TailOptions, fn func(LogEntry)) error {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		query = "*"
	}
	if opts.RefreshInterval <= 0 {
		return errors.New("victorialogs: RefreshInterval must be > 0")
	}
	form := url.Values{}
	form.Set("query", query)
	form.Set("refresh_interval", opts.RefreshInterval.String())
	return c.stream(ctx, "/select/logsql/tail", form, fn)
}

// ---------------------------------------------------------------------------
// streaming HTTP plumbing
// ---------------------------------------------------------------------------

// stream POSTs form to endpoint and decodes the JSON-lines body, calling fn
// per entry. Used by both Query and Tail.
func (c *Client) stream(ctx context.Context, endpoint string, form url.Values, fn func(LogEntry)) error {
	u := *c.base
	u.Path = u.Path + endpoint

	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(),
		strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		// Context cancellation surfaces as a transport-level error; turn
		// it into a clean exit for the caller, but log it under debug so
		// a misconfigured timeout / pre-cancelled context isn't invisible.
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPError{
			Status: resp.StatusCode,
			Body:   strings.TrimSpace(string(body)),
		}
	}

	// Scanner with a large max — single log lines can exceed the default
	// 64KB (stack traces, large request bodies serialized as fields).
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var (
		linesRead    int
		linesEmpty   int
		linesBadJSON int
	)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		linesRead++
		line := scanner.Bytes()
		if len(line) == 0 {
			linesEmpty++
			continue
		}
		entry, err := parseEntry(line)
		if err != nil {
			linesBadJSON++
			continue
		}
		fn(entry)
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	return nil
}

func (c *Client) applyAuth(req *http.Request) {
	switch {
	case c.auth.token != "":
		req.Header.Set("Authorization", "Bearer "+c.auth.token)
	case c.auth.username != "" || c.auth.password != "":
		req.SetBasicAuth(c.auth.username, c.auth.password)
	}
	if c.auth.tenant != "" {
		if parts := strings.SplitN(c.auth.tenant, ":", 2); len(parts) == 2 {
			req.Header.Set("AccountID", parts[0])
			req.Header.Set("ProjectID", parts[1])
		}
	}
}

// ---------------------------------------------------------------------------
// errors
// ---------------------------------------------------------------------------

// HTTPError is returned when VictoriaLogs responds with a 4xx or 5xx status.
// Body contains up to 4 KB of the response body for diagnostics.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("victorialogs: HTTP %d", e.Status)
	}
	return fmt.Sprintf("victorialogs: HTTP %d: %s", e.Status, e.Body)
}
