// Package prompt contains interactive helpers that discover Vault resources and let the user choose
// between them, offering only what the current token is allowed to use.
package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
)

// MountLister is the subset of the vault client needed to discover mounted secret engines.
type MountLister interface {
	ListMountsByType(ctx context.Context, engineType string) ([]string, error)
}

// CapabilityChecker is the subset of the vault client needed to check the capabilities of the token.
type CapabilityChecker interface {
	Capabilities(ctx context.Context, paths []string) (map[string][]string, error)
}

// Load runs fetch with vault.DefaultTimeout while displaying a spinner with the given title.
func Load[T any](ctx context.Context, title string, fetch func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, vault.DefaultTimeout)
	defer cancel()

	return tui.LoadWithSpinner(ctx, title, fetch)
}

// Mount discovers the mounts of the given secret engine type. If exactly one mount is found, it is
// used without asking. If multiple mounts are found, a selection is offered. Otherwise, it falls
// back to free-form input.
func Mount(ctx context.Context, client MountLister, engineType string) string {
	title := fmt.Sprintf("Discovering %s mounts...", engineType)
	mounts, err := Load(ctx, title, func(ctx context.Context) ([]string, error) {
		return client.ListMountsByType(ctx, engineType)
	})
	if err != nil {
		log.Warn().Err(err).Msgf("could not discover %s mounts, falling back to manual input", engineType)
	}

	return tui.SelectOrReadInput(engineType+" mount", mounts, []string{engineType})
}

// FilterByCapability returns the names whose path, as built by pathFor, the token has the given capability
// for. As this only serves to offer a better selection, all names are returned if the capabilities can
// not be looked up or if the token lacks the capability for all of them.
func FilterByCapability(ctx context.Context, client CapabilityChecker, kind string, names []string, pathFor func(name string) string, capability string) []string {
	if len(names) == 0 {
		return names
	}

	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, pathFor(name))
	}

	capabilities, err := Load(ctx, "Checking permissions...", func(ctx context.Context) (map[string][]string, error) {
		return client.Capabilities(ctx, paths)
	})
	if err != nil {
		log.Debug().Err(err).Msgf("could not check permissions, offering all %ss", kind)
		return names
	}

	var allowed []string
	for _, name := range names {
		if vault.HasCapability(capabilities[pathFor(name)], capability) {
			allowed = append(allowed, name)
		} else {
			log.Debug().Msgf("Skipping %s %q as the token lacks the %q capability for %q", kind, name, capability, pathFor(name))
		}
	}

	if len(allowed) == 0 {
		log.Warn().Msgf("Token lacks the %q capability for all %ss, offering all of them", capability, kind)
		return names
	}

	return allowed
}

// RequireCapability terminates the process if the token lacks the given capability for path. If the
// capabilities can not be looked up, the check is skipped and Vault decides once the request is sent.
func RequireCapability(ctx context.Context, client CapabilityChecker, path, capability string) {
	capabilities, err := Load(ctx, "Checking permissions...", func(ctx context.Context) (map[string][]string, error) {
		return client.Capabilities(ctx, []string{path})
	})
	if err != nil {
		log.Debug().Err(err).Msgf("could not check permissions for %q", path)
		return
	}

	if !vault.HasCapability(capabilities[path], capability) {
		log.Fatal().Msgf("Token lacks the %q capability for %q (has %q), please check your policies", capability, path, strings.Join(capabilities[path], ", "))
	}
}
