package selfupdate

import (
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		local, remote string
		want          bool
		wantErr       bool
	}{
		{"v1.10.0", "v1.11.0", true, false},
		{"v1.11.0", "v1.11.0", false, false},
		{"v1.12.0", "v1.11.0", false, false},
		{"v1.9.0", "v1.10.0", true, false},
		{"1.9.0", "v1.10.0", true, false},
		{"v1.11.0-rc1", "v1.11.0", true, false},
		{"dev", "v1.11.0", false, true},
		{"v1.0.0", "", false, true},
	}
	for _, tt := range tests {
		got, err := IsNewer(tt.local, tt.remote)
		if (err != nil) != tt.wantErr {
			t.Errorf("IsNewer(%q, %q) error = %v, wantErr %v", tt.local, tt.remote, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.local, tt.remote, got, tt.want)
		}
	}
}

func TestParseChecksum(t *testing.T) {
	sums := `78ce0f193213f5b69c64a9a437ccf7945f07c7e4b8da4296e305ec8741f2ff46  builds/sc-linux-aarch64
404b1ae1d5829fc62981eecf76e1fe6cd819df04dc2d940a54f78c19afed42fc  builds/sc-linux-amd64
5a58c9593040f5ad283274c2bb7fea06e57c024664ed94f92b8e987d9bc5bcf8 *sc-linux-armv6
`
	got, err := parseChecksum(strings.NewReader(sums), "sc-linux-amd64")
	if err != nil || got != "404b1ae1d5829fc62981eecf76e1fe6cd819df04dc2d940a54f78c19afed42fc" {
		t.Errorf("parseChecksum() = %q, %v", got, err)
	}
	got, err = parseChecksum(strings.NewReader(sums), "sc-linux-armv6")
	if err != nil || got != "5a58c9593040f5ad283274c2bb7fea06e57c024664ed94f92b8e987d9bc5bcf8" {
		t.Errorf("parseChecksum() = %q, %v", got, err)
	}
	if _, err := parseChecksum(strings.NewReader(sums), "sc-darwin-arm64"); err == nil {
		t.Error("expected error for missing asset")
	}
}

func TestAssetName(t *testing.T) {
	tests := map[string]string{
		"linux/amd64":  "sc-linux-amd64",
		"linux/arm64":  "sc-linux-aarch64",
		"linux/arm":    "sc-linux-armv6",
		"darwin/arm64": "sc-darwin-arm64",
	}
	for platform, want := range tests {
		goos, goarch, _ := strings.Cut(platform, "/")
		got, err := AssetName(goos, goarch)
		if err != nil || got != want {
			t.Errorf("AssetName(%s) = %q, %v, want %q", platform, got, err, want)
		}
	}
	if _, err := AssetName("darwin", "amd64"); err == nil {
		t.Error("expected error for darwin/amd64")
	}
}
