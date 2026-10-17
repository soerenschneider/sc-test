package pkg

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseRelativeDuration extends time.ParseDuration with "d" (24h) and "w"
// (7d) suffixes that Go's stdlib refuses. Anything else falls through to
// the stdlib parser unchanged.
func ParseRelativeDuration(s string) (time.Duration, error) {
	if len(s) > 1 {
		switch s[len(s)-1] {
		case 'd':
			n, err := strconv.ParseFloat(s[:len(s)-1], 64)
			if err == nil {
				return time.Duration(n * float64(24*time.Hour)), nil
			}
		case 'w':
			n, err := strconv.ParseFloat(s[:len(s)-1], 64)
			if err == nil {
				return time.Duration(n * float64(7*24*time.Hour)), nil
			}
		}
	}
	return time.ParseDuration(s)
}

// ParseTime accepts:
//   - Go durations relative to now: "1h", "5m", "2h30m"; a leading "-" is
//     allowed for symmetry with `--since -5m`. Also accepts the convenience
//     suffixes "d" (24h) and "w" (7d) that Go's stdlib does not — common
//     enough in log-query CLIs that omitting them is more surprising than
//     adding them.
//   - RFC3339 with or without fractional seconds
//   - "YYYY-MM-DD HH:MM:SS" or "YYYY-MM-DDTHH:MM:SS"
//   - "YYYY-MM-DD" (midnight, local time)
//   - Unix epoch in seconds (10 digits), milliseconds (13), or nanoseconds (16/19)
func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if d, err := ParseRelativeDuration(strings.TrimPrefix(s, "-")); err == nil {
		return time.Now().Add(-d), nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		switch len(s) {
		case 10:
			return time.Unix(n, 0), nil
		case 13:
			return time.UnixMilli(n), nil
		case 16, 19:
			return time.Unix(0, n), nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"cannot parse %q: use duration (1h, 1d, 1w), RFC3339, YYYY-MM-DD[ HH:MM:SS], or Unix epoch",
		s)
}
