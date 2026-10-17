package tf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/soerenschneider/sc/pkg"
)

// findPaths locates terraform/ in the given checkout of the repository.
func (r *runner) findPaths(repo string) {
	if repo == "" {
		die("no repository given, pass --repo, set it in a profile or set SC_TF_REPO")
	}
	expanded := pkg.GetExpandedFile(repo)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		die("invalid repository path %s: %v", repo, err)
	}

	r.repoRoot = abs
	r.tfRoot = filepath.Join(abs, "terraform")
	r.stacksDir = filepath.Join(r.tfRoot, "stacks")
	r.plansDir = filepath.Join(r.tfRoot, "plans")

	if !isDir(r.stacksDir) {
		die("%s has no terraform/stacks, is it a checkout of the soeren.cloud repository?", abs)
	}
}

// rel shortens a path for messages.
func (r *runner) rel(path string) string {
	return strings.TrimPrefix(path, r.repoRoot+"/")
}

// checkContract fails the run unless the repository declares a version of its
// conventions this package implements, so that an outdated binary stops
// instead of misreading a secrets file.
func (r *runner) checkContract() {
	path := filepath.Join(r.tfRoot, contractName)
	version, err := readContract(path)
	if err != nil {
		die("%s: %v", r.rel(path), err)
	}
	switch {
	case version > MaxContract:
		die("%s declares contract %d, this sc supports %d to %d: upgrade sc", r.rel(path), version, MinContract, MaxContract)
	case version < MinContract:
		die("%s declares contract %d, this sc supports %d to %d: pull the repository, or use an older sc", r.rel(path), version, MinContract, MaxContract)
	}
}

func readContract(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	version, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer", strings.TrimSpace(string(data)))
	}
	return version, nil
}

// changesInfrastructure reports whether the operation changes what is deployed
// or the passphrase it is encrypted with, which a stale checkout must not do:
// it would roll back whatever the newer commits changed.
func changesInfrastructure(op string) bool {
	return writesState(op) || strings.HasPrefix(op, "rotate-passphrase-")
}

const fetchTimeout = 30 * time.Second

// gitFreshness fetches the upstream of the checked out branch and counts the
// commits the checkout is behind and ahead of it.
func gitFreshness(repo string) (upstream string, behind, ahead int, err error) {
	git := func(ctx context.Context, args ...string) (string, error) {
		//nolint:gosec // G204: git with arguments built here, the repository path is the user's own
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
		// A fetch must fail rather than sit at a credentials prompt.
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if os.Getenv("GIT_SSH_COMMAND") == "" {
			cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
		}
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			// git explains itself over several lines, the first one says it.
			if msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n"); msg != "" {
				return "", errors.New(msg)
			}
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}

	ctx := context.Background()
	upstream, err = git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return "", 0, 0, fmt.Errorf("the checked out branch has no upstream: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	if _, err := git(fetchCtx, "fetch", "--quiet"); err != nil {
		return upstream, 0, 0, fmt.Errorf("could not fetch %s: %w", upstream, err)
	}

	counts, err := git(ctx, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return upstream, 0, 0, err
	}
	fields := strings.Fields(counts)
	if len(fields) != 2 {
		return upstream, 0, 0, fmt.Errorf("unexpected output of git rev-list: %q", counts)
	}
	ahead, _ = strconv.Atoi(fields[0])
	behind, _ = strconv.Atoi(fields[1])
	return upstream, behind, ahead, nil
}

// checkFreshness refuses to change the infrastructure from a checkout that is
// behind its upstream, or whose upstream cannot be fetched, unless allowed to
// with --allow-stale. For everything else a stale checkout is only a warning.
func (r *runner) checkFreshness() {
	strict := changesInfrastructure(r.operation) && !r.allowStale
	fail := func(format string, a ...any) {
		msg := fmt.Sprintf(format, a...)
		if strict {
			die("%s. '%s' changes the infrastructure, pass --allow-stale to run it anyway", msg, r.operation)
		}
		warnf("%s", msg)
	}

	logf("Checking that %s is up to date ...", r.repoRoot)
	upstream, behind, ahead, err := gitFreshness(r.repoRoot)
	if err != nil {
		fail("%v", err)
		return
	}
	if behind > 0 {
		fail("the checkout is %d commit(s) behind %s, pull first", behind, upstream)
	}
	if ahead > 0 && changesInfrastructure(r.operation) {
		warnf("the checkout is %d commit(s) ahead of %s, push them: a passphrase that only exists here is not backed up", ahead, upstream)
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
