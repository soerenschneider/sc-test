// Package tf wraps terragrunt for the units below terraform/stacks/<stack>/envs/<env>
// of the soeren.cloud repository.
//
// Every unit that encrypts its state has its own passphrase in
// stacks/<stack>/envs/<env>/tf-encryption.sops.yaml, which is decrypted with
// sops and handed to terragrunt via TF_ENCRYPTION. The first run of a unit
// without state generates it. See terraform/README.md in the repository for
// state encryption and passphrase rotation.
//
// The conventions implemented here belong to the repository, which declares
// their version in terraform/.sc-contract. A repository declaring a version
// outside [MinContract, MaxContract] is refused.
package tf

import (
	"os"
	"regexp"
	"syscall"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// The range of terraform/.sc-contract versions this package implements. Raise
// MaxContract (and MinContract, once the old conventions are gone) together
// with the code that follows a new version.
const (
	MinContract = 1
	MaxContract = 1
)

const defaultKeyID = "mykey"

// The passphrase file in every unit directory that encrypts its state.
const secretsName = "tf-encryption.sops.yaml"

const contractName = ".sc-contract"

// Lists the units that are allowed to keep their state in plaintext, as
// {"<stack>/<env>": "<reason>"}. Every other unit has to encrypt it.
const whitelistName = "state-encryption-whitelist.json"

// How to invoke this program again, for the hints it prints.
const self = "sc tf"

var (
	nameRE       = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	keyIDRE      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	encryptionRE = regexp.MustCompile(`(?m)^\s*encryption\s*\{`)
	unitKeyRE    = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
	planSuffixRE = regexp.MustCompile(`^\d{8}T\d{6}Z\.tfplan$`)
	shellSafeRE  = regexp.MustCompile(`^[A-Za-z0-9_./=:@%+,-]+$`)
)

type Operation struct{ Name, Help string }

var Operations = []Operation{
	{"plan", "show what would change and write a plan file"},
	{"apply", "apply a saved plan, or plan and apply"},
	{"plan-destroy", "plan the destruction of everything"},
	{"destroy", "destroy everything"},
	{"show", "show a saved plan"},
	{"output", "print the outputs"},
	{"validate", "validate the configuration"},
	{"init", "initialise backend, providers and modules"},
	{"init-upgrade", "init with -upgrade"},
	{"refresh", "refresh the state"},
	{"fmt", "tofu fmt -recursive -check -diff"},
	{"test", "tofu test, no backend needed"},
	{"raw", "any tofu command, with encryption and lock"},
	{"rotate-passphrase-start", "start a passphrase rotation of the stack"},
	{"rotate-passphrase-finish", "finish a passphrase rotation of the stack"},
}

// Options is what the command line asked for.
type Options struct {
	// Root of the soeren.cloud checkout.
	Repo string

	Env, Stack, Operation, PlanFile, OutputName string
	OutputRaw, Yes, DetailedExitcode            bool
	// Ask for everything not given, even when the stack is given.
	Interactive bool
	// Run operations that write state although the checkout is behind its
	// upstream or the upstream could not be fetched.
	AllowStale  bool
	Passthrough []string
	// Options given on the command line, by their long flag name, which the
	// wizard does not ask for again.
	Set map[string]bool
}

type runner struct {
	tfRoot, repoRoot, stacksDir, plansDir string

	operation, env, stack, planFile, outputName string
	outputRaw, yes, detailedExitcode            bool
	wizardForced, interactive, allowStale       bool
	passthrough                                 []string
	set                                         map[string]bool

	workingDir, secretsFile string
	encrypted               bool

	// The decrypted secrets file of the unit. It is only ever held in this
	// process and handed to sops through a pipe, never written to disk or put
	// on a command line. Any of the recipients in stacks/.sops.yaml can unlock
	// it. Fields:
	//
	//   passphrase            the current passphrase
	//   key_id                name of its key provider, which OpenTofu
	//                         records in the state; "mykey" when absent
	//   previous_passphrase,  only while a rotation is in progress: the
	//   previous_key_id       passphrase the state may still be encrypted with
	secrets              map[string]any
	keyID, previousKeyID string
}

func newRunner(opts Options) *runner {
	r := &runner{
		operation:        opts.Operation,
		env:              opts.Env,
		stack:            opts.Stack,
		planFile:         opts.PlanFile,
		outputName:       opts.OutputName,
		outputRaw:        opts.OutputRaw,
		yes:              opts.Yes,
		detailedExitcode: opts.DetailedExitcode,
		wizardForced:     opts.Interactive,
		allowStale:       opts.AllowStale,
		passthrough:      opts.Passthrough,
		set:              opts.Set,
	}
	if r.env == "" {
		r.env = "prod"
	}
	if r.set == nil {
		r.set = map[string]bool{}
	}
	return r
}

// Run runs the operation, asking for whatever is missing when there is a
// terminal, and returns the exit code of tofu. It does not return on errors,
// which are printed before the process exits through Exit.
func Run(opts Options) int {
	// Plan files describe the whole infrastructure and are only readable by
	// the owner. For an encrypted stack OpenTofu encrypts them on top of that
	// (see the 'plan' block in the stack's tf_provider.tf).
	syscall.Umask(0o077)
	handleSignals()

	r := newRunner(opts)
	r.findPaths(opts.Repo)
	r.checkContract()
	//#nosec:G115
	r.interactive = term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

	if r.wizardForced || (r.interactive && (r.stack == "" || r.operation == "")) {
		if !r.interactive {
			die("the wizard needs a terminal, pass everything as flags instead")
		}
		r.wizard()
		logf("Equivalent command: %s", r.commandLine())
	}
	if r.operation == "" {
		die("no operation given, use one of the subcommands of '%s'", self)
	}

	return r.run()
}

// List prints the environments and their stacks.
func List(repo string) {
	r := newRunner(Options{})
	r.findPaths(repo)
	r.checkContract()

	whitelist := r.loadWhitelist()
	for _, env := range r.listEnvs() {
		lipgloss.Fprintln(os.Stdout, env)
		for _, s := range r.listStacks(env) {
			note := ""
			if !encryptsState(r.unitDir(s, env)) {
				if reason, ok := whitelist[s+"/"+env]; ok {
					note = dimStyle.Render("  plaintext state: " + reason)
				} else {
					note = errStyle.Render("  plaintext state, not whitelisted")
				}
			}
			lipgloss.Fprintln(os.Stdout, "  "+s+note)
		}
	}
}
