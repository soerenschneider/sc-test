package vault

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPath(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "plain parts",
			parts: []string{"transit", "encrypt", "app"},
			want:  "transit/encrypt/app",
		},
		{
			name:  "mount with trailing slash",
			parts: []string{"transit/", "encrypt", "app"},
			want:  "transit/encrypt/app",
		},
		{
			name:  "nested mount with surrounding slashes",
			parts: []string{"/ssh/clients/", "sign", "users"},
			want:  "ssh/clients/sign/users",
		},
		{
			name:  "empty parts are skipped",
			parts: []string{"pki", "", "issue", "web"},
			want:  "pki/issue/web",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Path(tt.parts...); got != tt.want {
				t.Errorf("Path() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasCapability(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		wanted       string
		want         bool
	}{
		{
			name:         "capability granted",
			capabilities: []string{"read", "update"},
			wanted:       "update",
			want:         true,
		},
		{
			name:         "root grants everything",
			capabilities: []string{"root"},
			wanted:       "update",
			want:         true,
		},
		{
			name:         "denied",
			capabilities: []string{"deny"},
			wanted:       "update",
			want:         false,
		},
		{
			name:         "other capability only",
			capabilities: []string{"read"},
			wanted:       "update",
			want:         false,
		},
		{
			name:         "no capabilities",
			capabilities: nil,
			wanted:       "update",
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCapability(tt.capabilities, tt.wanted); got != tt.want {
				t.Errorf("HasCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapTtl(t *testing.T) {
	tests := []struct {
		name   string
		ttl    string
		maxTtl time.Duration
		want   string
	}{
		{
			name:   "ttl exceeds max_ttl",
			ttl:    "24h",
			maxTtl: 2 * time.Hour,
			want:   "2h0m0s",
		},
		{
			name:   "ttl within max_ttl",
			ttl:    "1h",
			maxTtl: 2 * time.Hour,
			want:   "1h",
		},
		{
			name:   "ttl equals max_ttl",
			ttl:    "2h",
			maxTtl: 2 * time.Hour,
			want:   "2h",
		},
		{
			name:   "no max_ttl",
			ttl:    "24h",
			maxTtl: 0,
			want:   "24h",
		},
		{
			name:   "unparseable ttl is left to vault",
			ttl:    "1d",
			maxTtl: 2 * time.Hour,
			want:   "1d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CapTtl(tt.ttl, tt.maxTtl, "role"); got != tt.want {
				t.Errorf("CapTtl() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseVaultDuration(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want time.Duration
	}{
		{
			name: "json number of seconds",
			val:  json.Number("3600"),
			want: time.Hour,
		},
		{
			name: "float of seconds",
			val:  float64(60),
			want: time.Minute,
		},
		{
			name: "duration string",
			val:  "30m",
			want: 30 * time.Minute,
		},
		{
			name: "string of seconds",
			val:  "120",
			want: 2 * time.Minute,
		},
		{
			name: "unparseable",
			val:  "forever",
			want: 0,
		},
		{
			name: "nil",
			val:  nil,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVaultDuration(tt.val); got != tt.want {
				t.Errorf("parseVaultDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
