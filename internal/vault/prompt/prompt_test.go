package prompt

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/soerenschneider/sc/internal/vault"
)

type fakeCapabilityChecker struct {
	capabilities map[string][]string
	err          error
}

func (f *fakeCapabilityChecker) Capabilities(_ context.Context, paths []string) (map[string][]string, error) {
	if f.err != nil {
		return nil, f.err
	}

	ret := map[string][]string{}
	for _, path := range paths {
		ret[path] = f.capabilities[path]
		if ret[path] == nil {
			ret[path] = []string{"deny"}
		}
	}
	return ret, nil
}

func TestFilterByCapability(t *testing.T) {
	keys := []string{"app", "db", "tenant"}
	pathFor := func(key string) string {
		return vault.Path("transit", "encrypt", key)
	}

	tests := []struct {
		name   string
		client *fakeCapabilityChecker
		names  []string
		want   []string
	}{
		{
			name: "only allowed names are returned",
			client: &fakeCapabilityChecker{capabilities: map[string][]string{
				"transit/encrypt/app":    {"update"},
				"transit/encrypt/tenant": {"read", "update"},
			}},
			names: keys,
			want:  []string{"app", "tenant"},
		},
		{
			name: "root token is allowed everything",
			client: &fakeCapabilityChecker{capabilities: map[string][]string{
				"transit/encrypt/app":    {"root"},
				"transit/encrypt/db":     {"root"},
				"transit/encrypt/tenant": {"root"},
			}},
			names: keys,
			want:  keys,
		},
		{
			name:   "all names returned if none is allowed",
			client: &fakeCapabilityChecker{},
			names:  keys,
			want:   keys,
		},
		{
			name:   "all names returned if capabilities can not be looked up",
			client: &fakeCapabilityChecker{err: errors.New("permission denied")},
			names:  keys,
			want:   keys,
		},
		{
			name:   "no names",
			client: &fakeCapabilityChecker{},
			names:  nil,
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByCapability(context.Background(), tt.client, "transit key", tt.names, pathFor, "update")
			if !slices.Equal(got, tt.want) {
				t.Errorf("FilterByCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
