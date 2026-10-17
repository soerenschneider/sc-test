package tf

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runTerragrunt runs tofu through terragrunt. terragrunt runs tofu inside its
// own cache directory, so every path handed to tofu has to be absolute.
func (r *runner) runTerragrunt(stdout io.Writer, args ...string) int {
	a := []string{"run", "--working-dir", r.workingDir}
	if r.yes {
		a = append(a, "--non-interactive")
	}
	if !r.interactive {
		a = append(a, "--no-color")
	}
	a = append(append(a, "--"), args...)
	return runCmd("", stdout, "terragrunt", a...)
}

func (r *runner) run() int {
	if !validOperation(r.operation) {
		die("unknown operation: %s", r.operation)
	}

	// Both end up in file paths, so keep them to something harmless.
	if !nameRE.MatchString(r.env) {
		die("invalid environment name: %s", r.env)
	}
	if r.stack != "" && !nameRE.MatchString(r.stack) {
		die("invalid stack name: %s", r.stack)
	}

	if len(r.listStacks(r.env)) == 0 {
		die("no such environment: %s (have: %s)", r.env, strings.Join(r.listEnvs(), " "))
	}

	rotate := strings.HasPrefix(r.operation, "rotate-passphrase-")
	if r.stack == "" {
		die("--stack is required (have: %s)", strings.Join(r.listStacks(r.env), " "))
	}
	r.workingDir = r.unitDir(r.stack, r.env)
	if !fileExists(filepath.Join(r.workingDir, "terragrunt.hcl")) {
		die("no such stack: %s (have: %s)", r.rel(r.workingDir), strings.Join(r.listStacks(r.env), " "))
	}

	// One passphrase per unit, so that whoever can decrypt it can read this
	// unit's state and no other. The recipients are in stacks/.sops.yaml.
	r.secretsFile = filepath.Join(r.workingDir, secretsName)

	for _, cmd := range []string{"terragrunt", "tofu"} {
		if _, err := exec.LookPath(cmd); err != nil {
			die("%s is not installed", cmd)
		}
	}

	// fmt and test never touch state or a backend, so they run tofu directly
	// in the stack directory. Going through terragrunt would copy the stack
	// into its cache and report paths that do not exist in the working tree.
	if r.operation == "fmt" || r.operation == "test" {
		if r.planFile != "" {
			die("--plan-file does not apply to '%s'", r.operation)
		}
		if r.operation == "fmt" {
			args := append(append([]string{"fmt", "-recursive", "-check", "-diff"}, r.passthrough...), ".")
			return runCmd(r.workingDir, os.Stdout, "tofu", args...)
		}
		// 'tofu test' needs the modules on disk but no backend. Idempotent
		// and cached, so it just runs every time rather than guessing whether
		// .terraform is current.
		logf("Installing modules for '%s/%s' (no backend) ...", r.env, r.stack)
		if runCmd(r.workingDir, io.Discard, "tofu", "init", "-backend=false", "-input=false") != 0 {
			die("could not install modules for %s", r.rel(r.workingDir))
		}
		return runCmd(r.workingDir, os.Stdout, "tofu", append([]string{"test"}, r.passthrough...)...)
	}

	r.checkEncryptionPolicy()

	// The 'encryption' block in the stack's configuration decides whether its
	// state is encrypted, and the passphrase file follows it. A unit that
	// encrypts but has no passphrase yet gets one generated further down, once
	// the lock is held and it is certain that the unit has no state.
	generate := false
	switch {
	case !encryptsState(r.workingDir):
		if fileExists(r.secretsFile) {
			warnf("%s exists, but the stack does not encrypt its state, the file is ignored", r.rel(r.secretsFile))
		}
		if rotate {
			die("'%s/%s' does not encrypt its state, there is no passphrase to rotate", r.stack, r.env)
		}
	case fileExists(r.secretsFile):
		r.encrypted = true
	case rotate:
		die("%s does not exist, there is no passphrase to rotate", r.rel(r.secretsFile))
	default:
		r.encrypted = true
		generate = true
	}
	if r.encrypted {
		if _, err := exec.LookPath("sops"); err != nil {
			die("sops is not installed, but %s needs it", r.rel(r.secretsFile))
		}
	}

	if !r.interactive {
		// Keeps tofu from emitting hints that assume a human at the terminal.
		_ = os.Setenv("TF_IN_AUTOMATION", "1")
	}

	// Argument validation. Deliberately ahead of the lock and the sops
	// decryption: a typo should not first wait on a pinentry prompt only to
	// be rejected after.
	if r.operation != "output" {
		if r.outputName != "" {
			die("--name only applies to 'output'")
		}
		if r.outputRaw {
			die("--raw only applies to 'output'")
		}
	}
	if r.outputRaw && r.outputName == "" {
		die("--raw needs --name")
	}

	switch r.operation {
	case "init", "init-upgrade", "validate", "output", "refresh", "destroy", "raw":
		if r.planFile != "" {
			die("--plan-file does not apply to '%s'", r.operation)
		}
	case "show":
		// apply without a plan file is still allowed, it just prompts the
		// way tofu normally does.
		if r.planFile == "" {
			die("--plan-file is required for 'show'")
		}
	}

	if rotate {
		if r.planFile != "" {
			die("--plan-file does not apply to '%s'", r.operation)
		}
		if len(r.passthrough) > 0 {
			die("'%s' takes no arguments after '--'", r.operation)
		}
	}

	if r.operation == "raw" && len(r.passthrough) == 0 {
		die("'raw' needs a tofu command after '--', e.g. %s raw -s <stack> -- state list", self)
	}

	// Also ahead of the decryption, and of anything that touches the world.
	r.checkFreshness()

	// Everything below this point touches the world.

	r.acquireLock()

	switch {
	case generate:
		// Leaves r.secrets set, so there is nothing to decrypt.
		r.generateSecrets()
		_ = os.Setenv("TF_ENCRYPTION", r.buildTFEncryption(true))
	case r.encrypted:
		if writesState(r.operation) {
			r.requireCommitted()
		}
		logf("Reading opentofu state encryption passphrase from %s ...", r.rel(r.secretsFile))
		r.readSecrets()
		_ = os.Setenv("TF_ENCRYPTION", r.buildTFEncryption(true))

		if r.previousKeyID != "" && !rotate {
			warnf("passphrase rotation in progress, state encrypted with '%s' is re-encrypted with '%s' on its next write. "+
				"Apply the stack, then run: %s", r.previousKeyID, r.keyID, r.hint("rotate-passphrase-finish"))
		}
	default:
		logf("'%s/%s' does not encrypt its state, running without a passphrase ...", r.stack, r.env)
		// A passphrase inherited from the caller's shell would silently apply
		// to a unit that is not supposed to use one.
		_ = os.Unsetenv("TF_ENCRYPTION")
	}

	// A passphrase that was just generated is not committed yet. Checked
	// after generating it, so that the first run of a new unit still gets as
	// far as creating the file.
	if generate && writesState(r.operation) {
		r.requireCommitted()
	}

	// The vault provider reads VAULT_TOKEN, falling back to ~/.vault-token.
	// The aws provider uses the usual AWS_* variables or the shared config
	// file.
	if r.unitMatches(func(b []byte) bool { return bytes.Contains(b, []byte("hashicorp/vault")) }) {
		home, _ := os.UserHomeDir()
		if os.Getenv("VAULT_TOKEN") == "" && !fileExists(filepath.Join(home, ".vault-token")) {
			warnf("neither VAULT_TOKEN nor ~/.vault-token is present, the vault provider will fail to authenticate if a stack needs it")
		}
	}

	pt := r.passthrough
	switch r.operation {
	case "init":
		return r.runTerragrunt(os.Stdout, append([]string{"init"}, pt...)...)
	case "init-upgrade":
		return r.runTerragrunt(os.Stdout, append([]string{"init", "-upgrade"}, pt...)...)
	case "validate":
		return r.runTerragrunt(os.Stdout, append([]string{"validate"}, pt...)...)
	case "refresh":
		return r.runTerragrunt(os.Stdout, append([]string{"refresh"}, pt...)...)
	case "raw":
		return r.runTerragrunt(os.Stdout, pt...)

	case "output":
		args := []string{"output"}
		// A sensitive output is redacted in the overview but printed in full
		// when it is requested by name, which is the only way to get at the
		// credentials of a stack that does not write them to OpenBao.
		if r.outputName != "" {
			if r.outputRaw {
				args = append(args, "-raw")
			} else {
				args = append(args, "-json")
			}
			args = append(args, r.outputName)
		}
		return r.runTerragrunt(os.Stdout, append(args, pt...)...)

	case "plan", "plan-destroy":
		r.prepareNewPlan()
		args := []string{"plan"}
		if r.operation == "plan-destroy" {
			args = append(args, "-destroy")
		}
		if r.planFile != "" {
			args = append(args, "-out="+r.planFile)
		}
		if r.detailedExitcode {
			args = append(args, "-detailed-exitcode")
		}

		// -detailed-exitcode turns 2 into 'there are changes', which must not
		// look like a failure here.
		rc := r.runTerragrunt(os.Stdout, append(args, pt...)...)
		if rc != 0 && (!r.detailedExitcode || rc != 2) {
			return rc
		}

		if r.planFile != "" {
			cleanup.Lock()
			cleanup.plan = ""
			cleanup.Unlock()
			_ = os.Chmod(r.planFile, 0o600)
			_ = os.Chmod(r.planFile+".meta", 0o600)
			if !r.encrypted {
				warnf("the plan file of '%s/%s' is not encrypted and holds every value the state holds, including credentials", r.env, r.stack)
			}
			logf("Plan written to %s", r.rel(r.planFile))
			logf("Apply it with: %s -p latest", r.hint("apply"))
		}
		return rc

	case "apply":
		if r.planFile != "" {
			r.resolveExistingPlan()
			r.confirm(fmt.Sprintf("Apply %s to '%s/%s'?", r.rel(r.planFile), r.env, r.stack))
			// A saved plan is applied as it is, tofu does not prompt again.
			return r.runTerragrunt(os.Stdout, append([]string{"apply", r.planFile}, pt...)...)
		}
		args := []string{"apply"}
		if r.yes {
			args = append(args, "-auto-approve")
		}
		return r.runTerragrunt(os.Stdout, append(args, pt...)...)

	case "destroy":
		r.confirmTyped(fmt.Sprintf("This destroys everything in '%s/%s'.", r.env, r.stack), r.env)
		args := []string{"destroy"}
		if r.yes {
			args = append(args, "-auto-approve")
		}
		return r.runTerragrunt(os.Stdout, append(args, pt...)...)

	case "show":
		r.resolveExistingPlan()
		return r.runTerragrunt(os.Stdout, append([]string{"show", r.planFile}, pt...)...)

	case "rotate-passphrase-start":
		if r.previousKeyID != "" {
			die("a rotation is already in progress (previous key '%s'), finish it with %s first", r.previousKeyID, r.hint("rotate-passphrase-finish"))
		}

		newKeyID, newPassphrase := newKey()

		r.confirm(fmt.Sprintf("Rotate the passphrase of '%s/%s' from '%s' to '%s'?", r.stack, r.env, r.keyID, newKeyID))

		next := maps.Clone(r.secrets)
		next["previous_key_id"] = r.keyID
		next["previous_passphrase"] = r.secretString("passphrase")
		next["key_id"] = newKeyID
		next["passphrase"] = newPassphrase
		r.writeSecrets(next)

		logf("Wrote %s, the new key is '%s'.", r.rel(r.secretsFile), newKeyID)
		logf("Commit and push it, then apply the stack, which re-encrypts its state:")
		logf("  %s", r.hint("apply"))
		logf("Then: %s", r.hint("rotate-passphrase-finish"))
		return 0

	case "rotate-passphrase-finish":
		if r.previousKeyID == "" {
			die("no rotation in progress for '%s/%s'", r.stack, r.env)
		}

		// The state has to be readable without the fallback before the
		// previous passphrase may go, or it would be lost for good.
		logf("Checking that '%s/%s' reads its state with '%s' alone ...", r.stack, r.env, r.keyID)
		_ = os.Setenv("TF_ENCRYPTION", r.buildTFEncryption(false))
		if r.runTerragrunt(io.Discard, "state", "pull") != 0 {
			die("the state still needs '%s', apply the stack and run %s again", r.previousKeyID, r.hint("rotate-passphrase-finish"))
		}

		r.confirm(fmt.Sprintf("Drop the previous passphrase '%s' of '%s/%s'? Plans encrypted with it become unreadable.", r.previousKeyID, r.stack, r.env))

		next := maps.Clone(r.secrets)
		delete(next, "previous_key_id")
		delete(next, "previous_passphrase")
		r.writeSecrets(next)

		logf("Wrote %s without '%s', commit and push it.", r.rel(r.secretsFile), r.previousKeyID)
		return 0
	}
	return 0
}
