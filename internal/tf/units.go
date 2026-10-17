package tf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// units lists every stacks/<stack>/envs/<env>/terragrunt.hcl as (stack, env).
func (r *runner) units() [][2]string {
	matches, _ := filepath.Glob(filepath.Join(r.stacksDir, "*", "envs", "*", "terragrunt.hcl"))
	var units [][2]string
	for _, m := range matches {
		envDir := filepath.Dir(m)
		stackDir := filepath.Dir(filepath.Dir(envDir))
		stack, env := filepath.Base(stackDir), filepath.Base(envDir)
		if strings.HasPrefix(stack, ".") || strings.HasPrefix(env, ".") {
			continue
		}
		units = append(units, [2]string{stack, env})
	}
	return units
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	return slices.Compact(values)
}

func (r *runner) listEnvs() []string {
	var envs []string
	for _, u := range r.units() {
		envs = append(envs, u[1])
	}
	return uniqueSorted(envs)
}

// listStacks lists the stacks with a unit for the environment, or every stack
// with a unit at all for an empty one.
func (r *runner) listStacks(env string) []string {
	var stacks []string
	for _, u := range r.units() {
		if env == "" || u[1] == env {
			stacks = append(stacks, u[0])
		}
	}
	return uniqueSorted(stacks)
}

func (r *runner) envsOf(stack string) []string {
	var envs []string
	for _, u := range r.units() {
		if u[0] == stack {
			envs = append(envs, u[1])
		}
	}
	return uniqueSorted(envs)
}

func (r *runner) unitDir(stack, env string) string {
	return filepath.Join(r.stacksDir, stack, "envs", env)
}

// tfFiles lists dir/*.tf and dir/*/*.tf, skipping hidden directories such as
// .terraform.
func tfFiles(dir string, nested bool) []string {
	files, _ := filepath.Glob(filepath.Join(dir, "*.tf"))
	if !nested {
		return files
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			more, _ := filepath.Glob(filepath.Join(dir, e.Name(), "*.tf"))
			files = append(files, more...)
		}
	}
	return files
}

func anyFileMatches(files []string, match func([]byte) bool) bool {
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil && match(data) {
			return true
		}
	}
	return false
}

// unitMatches reports whether any .tf file of the unit matches.
func (r *runner) unitMatches(match func([]byte) bool) bool {
	return anyFileMatches(tfFiles(r.workingDir, true), match)
}

// stripHCLComments drops '#', '//' and '/* */' comments outside of string
// literals, so that a commented-out block is not mistaken for a real one.
func stripHCLComments(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inString := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case inString:
			out = append(out, c)
			switch {
			case c == '\\' && i+1 < len(src):
				i++
				out = append(out, src[i])
			case c == '"' || c == '\n':
				inString = false
			}
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '#' || (c == '/' && i+1 < len(src) && src[i+1] == '/'):
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				out = append(out, '\n')
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return out
			}
			i += end + 3
			out = append(out, ' ')
		default:
			out = append(out, c)
		}
	}
	return out
}

// encryptsState reports whether the unit's root module has an active
// 'encryption' block. Which methods it enforces is up to OpenTofu, which
// refuses to run without the key material on its own.
func encryptsState(dir string) bool {
	return anyFileMatches(tfFiles(dir, false), func(b []byte) bool {
		return encryptionRE.Match(stripHCLComments(b))
	})
}

// loadWhitelist reads the units allowed to keep plaintext state. A missing
// file allows none of them.
func (r *runner) loadWhitelist() map[string]string {
	path := filepath.Join(r.stacksDir, whitelistName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}
	}
	if err != nil {
		die("could not read %s: %v", r.rel(path), err)
	}

	var whitelist map[string]string
	if err := json.Unmarshal(data, &whitelist); err != nil {
		die("%s: %v, expected {\"<stack>/<env>\": \"<reason>\"}", r.rel(path), err)
	}
	for key, reason := range whitelist {
		if !unitKeyRE.MatchString(key) {
			die("%s: '%s' is not of the form <stack>/<env>", r.rel(path), key)
		}
		if strings.TrimSpace(reason) == "" {
			die("%s: '%s' needs a reason why its state may stay unencrypted", r.rel(path), key)
		}
	}
	return whitelist
}

// checkEncryptionPolicy fails the run if the unit neither encrypts its state
// nor is whitelisted, and warns about whitelist entries that no longer mean
// anything. Forgetting the 'encryption' block in a new stack therefore stops
// the run instead of quietly writing plaintext state.
func (r *runner) checkEncryptionPolicy() {
	whitelist := r.loadWhitelist()
	wlPath := r.rel(filepath.Join(r.stacksDir, whitelistName))

	existing := map[string]bool{}
	for _, u := range r.units() {
		existing[u[0]+"/"+u[1]] = true
	}
	for _, key := range slices.Sorted(maps.Keys(whitelist)) {
		if !existing[key] {
			warnf("%s lists '%s', which is not a unit, remove the entry", wlPath, key)
		}
	}

	key := r.stack + "/" + r.env
	_, listed := whitelist[key]
	switch encrypted := encryptsState(r.workingDir); {
	case encrypted && listed:
		warnf("'%s' encrypts its state but is listed in %s, remove the entry", key, wlPath)
	case !encrypted && !listed:
		die("the state of %s is not encrypted: add an 'encryption' block to the stack's tf_provider.tf, "+
			"or list it in %s with the reason why plaintext state is fine", key, wlPath)
	}
}
