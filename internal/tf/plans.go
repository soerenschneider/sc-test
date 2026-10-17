package tf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func gitHead(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func gitDirty(repoRoot string) bool {
	out, err := exec.Command("git", "-C", repoRoot, "status", "--porcelain").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func fileMtime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func metaValue(file, key string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, key+"="); ok {
			return v
		}
	}
	return ""
}

func firstLine(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	line, _, _ := strings.Cut(string(out), "\n")
	return line
}

// plansOf lists the plans of the stack, newest first. The timestamp is matched
// exactly, so that 'aws-s3' does not pick up the plans of 'aws-s3-foo'.
func (r *runner) plansOf() []string {
	prefix := r.env + "-" + r.stack + "-"
	matches, _ := filepath.Glob(filepath.Join(r.plansDir, prefix+"*.tfplan"))
	var plans []string
	for _, m := range matches {
		if fileExists(m) && planSuffixRE.MatchString(strings.TrimPrefix(filepath.Base(m), prefix)) {
			plans = append(plans, m)
		}
	}
	sort.SliceStable(plans, func(i, j int) bool { return fileMtime(plans[i]).After(fileMtime(plans[j])) })
	return plans
}

// resolveExistingPlan resolves --plan-file for the operations that read one,
// and warns about everything that makes an old plan a bad thing to apply.
func (r *runner) resolveExistingPlan() {
	if r.planFile == "latest" {
		plans := r.plansOf()
		if len(plans) == 0 {
			die("no plan file for '%s/%s' below %s", r.env, r.stack, r.rel(r.plansDir))
		}
		r.planFile = plans[0]
	}

	if !fileExists(r.planFile) {
		die("no such plan file: %s", r.planFile)
	}
	if f, err := os.Open(r.planFile); err != nil {
		die("plan file is not readable: %s", r.planFile)
	} else {
		_ = f.Close()
	}
	r.planFile, _ = filepath.Abs(r.planFile)

	meta := r.planFile + ".meta"
	if e := metaValue(meta, "env"); e != "" && e != r.env {
		die("plan was created for environment '%s', not '%s'", e, r.env)
	}
	if s := metaValue(meta, "stack"); s != "" && s != r.stack {
		die("plan was created for stack '%s', not '%s'", s, r.stack)
	}

	// An encrypted plan file is unreadable without the passphrase it was
	// written with, so say so here instead of letting tofu fail on a garbled
	// file.
	if metaValue(meta, "encrypted") == "true" && !r.encrypted {
		die("plan was created with state encryption, but %s is gone", r.rel(r.secretsFile))
	}

	// Same for a plan encrypted with a passphrase that has been rotated out.
	if k := metaValue(meta, "key_id"); k != "" && k != r.keyID && k != r.previousKeyID {
		die("plan was encrypted with key '%s', which is no longer in %s", k, r.rel(r.secretsFile))
	}

	if age := time.Since(fileMtime(r.planFile)); age > time.Hour {
		warnf("plan is %dh old, the real infrastructure may have moved on", int(age.Hours()))
	}

	if c := metaValue(meta, "commit"); c != "" && c != "unknown" && c != gitHead(r.repoRoot) {
		warnf("plan was created on commit %s, HEAD is %s", c, gitHead(r.repoRoot))
	}

	if metaValue(meta, "dirty") == "true" {
		warnf("the working tree was dirty when the plan was created")
	}
}

// prepareNewPlan picks the destination for a freshly created plan and
// remembers what it was created from, so that applying it later can be sanity
// checked.
func (r *runner) prepareNewPlan() {
	if r.planFile == "none" {
		r.planFile = ""
		return
	}

	if r.planFile == "" {
		if err := os.MkdirAll(r.plansDir, 0o700); err != nil {
			die("could not create %s: %v", r.rel(r.plansDir), err)
		}
		_ = os.Chmod(r.plansDir, 0o700)
		r.planFile = filepath.Join(r.plansDir, fmt.Sprintf("%s-%s-%s.tfplan", r.env, r.stack, time.Now().UTC().Format("20060102T150405Z")))
	} else {
		abs, _ := filepath.Abs(r.planFile)
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			die("could not create %s: %v", filepath.Dir(abs), err)
		}
		r.planFile = abs
	}

	// Removed again on exit unless the run gets far enough to clear it.
	cleanup.Lock()
	cleanup.plan = r.planFile
	cleanup.Unlock()

	meta := fmt.Sprintf(`env=%s
stack=%s
operation=%s
encrypted=%t
key_id=%s
created=%s
commit=%s
dirty=%t
tofu=%s
terragrunt=%s
`, r.env, r.stack, r.operation, r.encrypted, r.keyID, time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		gitHead(r.repoRoot), gitDirty(r.repoRoot), firstLine("tofu", "version"), firstLine("terragrunt", "--version"))
	if err := os.WriteFile(r.planFile+".meta", []byte(meta), 0o600); err != nil {
		die("could not write %s: %v", r.rel(r.planFile+".meta"), err)
	}
}
