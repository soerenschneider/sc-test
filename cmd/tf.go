package cmd

import (
	"errors"
	"os"

	"github.com/soerenschneider/sc/internal/tf"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

const (
	tfFlagRepo        = "repo"
	tfFlagEnv         = "env"
	tfFlagStack       = "stack"
	tfFlagYes         = "yes"
	tfFlagInteractive = "interactive"
	tfFlagAllowStale  = "allow-stale"

	tfFlagPlanFile         = "plan-file"
	tfFlagDetailedExitcode = "detailed-exitcode"
	tfFlagName             = "name"
	tfFlagRaw              = "raw"

	tfRepoEnvKey = "SC_TF_REPO"
)

var tfCmd = &cobra.Command{
	Use:   "tf",
	Short: "Run terragrunt on the stacks of the soeren.cloud repository",
	Long: `Run terragrunt on the units below terraform/stacks/<stack>/envs/<env> of a
checkout of the soeren.cloud repository, with the unit's state encryption
passphrase decrypted from its tf-encryption.sops.yaml.

Without a subcommand, and with a terminal, a wizard asks for the stack, the
environment, the operation and whatever else the operation needs. Values given
as flags are not asked for again.

The checkout is given by --repo, a profile ("tf: {repo: ...}") or SC_TF_REPO.
Its terraform/.sc-contract must declare a version this sc supports. Operations
that change the infrastructure (apply, destroy, refresh, raw and the passphrase
rotation) refuse to run while the checkout is behind its upstream or the
upstream cannot be fetched, unless --allow-stale is given.

See terraform/README.md in the repository for state encryption and
passphrase rotation.`,
	Example: `  sc tf
  sc tf list
  sc tf plan -s aws-s3
  sc tf apply -s aws-s3 -p latest
  sc tf output -s aws-s3 -n user_credentials | jq -r '.someuser.secret_key'
  sc tf plan -s aws-s3 -- -target=module.buckets
  sc tf raw -s aws-s3 -- state list
  sc tf rotate-passphrase-start -s aws-s3`,
	Args: tfArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runTf(cmd, args, "")
	},
}

// tfArgs only accepts arguments after '--', which are passed to tofu.
func tfArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && cmd.ArgsLenAtDash() != 0 {
		return errors.New("arguments for tofu go after '--'")
	}
	return nil
}

func tfRepo(cmd *cobra.Command) string {
	if repo := pkg.GetString(cmd, tfFlagRepo); repo != "" {
		return repo
	}
	return os.Getenv(tfRepoEnvKey)
}

func runTf(cmd *cobra.Command, args []string, operation string) {
	opts := tf.Options{
		Repo:             tfRepo(cmd),
		Env:              pkg.GetString(cmd, tfFlagEnv),
		Stack:            pkg.GetString(cmd, tfFlagStack),
		Operation:        operation,
		PlanFile:         pkg.GetString(cmd, tfFlagPlanFile),
		OutputName:       pkg.GetString(cmd, tfFlagName),
		OutputRaw:        getBoolIfDefined(cmd, tfFlagRaw),
		Yes:              pkg.MustGetBool(cmd, tfFlagYes),
		DetailedExitcode: getBoolIfDefined(cmd, tfFlagDetailedExitcode),
		Interactive:      pkg.MustGetBool(cmd, tfFlagInteractive),
		AllowStale:       pkg.MustGetBool(cmd, tfFlagAllowStale),
		Passthrough:      args,
		Set:              map[string]bool{},
	}
	for _, name := range []string{tfFlagEnv, tfFlagStack, tfFlagPlanFile, tfFlagName, tfFlagRaw} {
		opts.Set[name] = cmd.Flags().Changed(name)
	}

	code := tf.Run(opts)
	checkUpdateWaitGroup.Wait()
	tf.Exit(code)
}

// getBoolIfDefined reads a flag that only some of the subcommands have.
func getBoolIfDefined(cmd *cobra.Command, name string) bool {
	if cmd.Flags().Lookup(name) == nil {
		return false
	}
	return pkg.MustGetBool(cmd, name)
}

func init() {
	rootCmd.AddCommand(tfCmd)

	tfCmd.PersistentFlags().String(tfFlagRepo, "", "Checkout of the soeren.cloud repository (default: $"+tfRepoEnvKey+")")
	tfCmd.PersistentFlags().StringP(tfFlagEnv, "e", "prod", "Environment, terraform/stacks/<stack>/envs/<env>")
	tfCmd.PersistentFlags().StringP(tfFlagStack, "s", "", "Stack below terraform/stacks/ that has a unit for the environment")
	tfCmd.PersistentFlags().BoolP(tfFlagYes, "y", false, "Do not ask for confirmation, run non-interactively")
	tfCmd.PersistentFlags().BoolP(tfFlagInteractive, "i", false, "Ask for everything not given as a flag, even when --stack is given")
	tfCmd.PersistentFlags().Bool(tfFlagAllowStale, false, "Change the infrastructure although the checkout is behind its upstream or it cannot be fetched")
}
