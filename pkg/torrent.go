package pkg

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	maxDNLength = 4096
)

// Sentinel errors for callers that want structured handling.
var (
	ErrInvalidMagnetLink = errors.New("invalid magnet link")
	ErrMissingDN         = errors.New("missing dn parameter")
	ErrInvalidDN         = errors.New("invalid dn parameter")
)

func ParseMagnetDisplayName(magnetURI string) (string, error) {
	if magnetURI == "" {
		return "", ErrInvalidMagnetLink
	}

	u, err := url.Parse(magnetURI)
	if err != nil {
		return "", ErrInvalidMagnetLink
	}

	// RFC 6068-style magnet URIs use the "magnet" scheme.
	if !strings.EqualFold(u.Scheme, "magnet") {
		return "", ErrInvalidMagnetLink
	}

	values := u.Query()

	// Query() already percent-decodes safely.
	dn := values.Get("dn")
	if dn == "" {
		return "", ErrMissingDN
	}

	// Trim surrounding whitespace.
	dn = strings.TrimSpace(dn)

	// Remove ASCII control chars and normalize whitespace.
	dn = sanitizeDisplayName(dn)

	if dn == "" {
		return "", ErrInvalidDN
	}

	if len(dn) > maxDNLength {
		return "", ErrInvalidDN
	}

	if !utf8.ValidString(dn) {
		return "", ErrInvalidDN
	}

	return dn, nil
}

// sanitizeDisplayName removes control characters and normalizes
// repeated whitespace into single spaces.
func sanitizeDisplayName(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	lastWasSpace := false

	for _, r := range s {
		// Drop control chars.
		if r < 0x20 || r == 0x7F {
			continue
		}

		// Normalize all unicode whitespace to a single ASCII space.
		if isWhitespace(r) {
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
			continue
		}

		b.WriteRune(r)
		lastWasSpace = false
	}

	return strings.TrimSpace(b.String())
}

func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}
