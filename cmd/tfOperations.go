package cmd

import (
	"github.com/soerenschneider/sc/internal/tf"
	"github.com/spf13/cobra"
)

//nolint:gosec // G101: usage examples, the output names are not credentials
var tfOperationExamples = map[string]string{
	"plan":                     "  sc tf plan -s aws-s3\n  sc tf plan -s aws-s3 -- -target=module.buckets",
	"apply":                    "  sc tf apply -s aws-s3 -p latest",
	"show":                     "  sc tf show -s aws-s3 -p latest",
	"output":                   "  sc tf output -s aws-s3 -n user_credentials | jq -r '.someuser.secret_key'",
	"raw":                      "  sc tf raw -s aws-s3 -- import 'module.buckets[\"x\"].aws_s3_bucket.this' x\n  sc tf raw -s aws-s3 -- state list",
	"rotate-passphrase-start":  "  sc tf rotate-passphrase-start -s aws-s3",
	"rotate-passphrase-finish": "  sc tf rotate-passphrase-finish -s aws-s3",
}

var tfOperationLong = map[string]string{
	"fmt":  "Runs tofu fmt -recursive -check -diff in the stack directory, bypassing terragrunt: it needs neither state nor a backend.",
	"test": "Runs tofu test in the stack directory, bypassing terragrunt: it needs neither state nor a backend. Installs the modules first.",
	"raw": "Passes everything after '--' to tofu unchanged, for the commands sc tf does not wrap. It still sets up state " +
		"encryption and the lock, which is why it exists at all.",
	"rotate-passphrase-start": "Generates a new passphrase and keeps the current one as the previous one, which the stack " +
		"falls back to until it was applied once. Commit and push the secrets file, apply the stack, then run " +
		"rotate-passphrase-finish.",
	"rotate-passphrase-finish": "Checks that the stack reads its state with the new passphrase alone, then drops the " +
		"previous one. Commit and push the secrets file afterwards.",
}

func init() {
	for _, op := range tf.Operations {
		name := op.Name
		cmd := &cobra.Command{
			Use:     name + " [-- <extra arguments for tofu>]",
			Short:   op.Help,
			Long:    tfOperationLong[name],
			Example: tfOperationExamples[name],
			Args:    tfArgs,
			Run: func(cmd *cobra.Command, args []string) {
				runTf(cmd, args, name)
			},
		}

		switch name {
		case "plan", "plan-destroy":
			cmd.Flags().StringP(tfFlagPlanFile, "p", "", "Where to write the plan file, 'none' disables writing one (default: plans/<env>-<stack>-<timestamp>.tfplan)")
			cmd.Flags().BoolP(tfFlagDetailedExitcode, "d", false, "Exit 2 if there are changes, 0 if not")
		case "apply":
			cmd.Flags().StringP(tfFlagPlanFile, "p", "", "Plan file to apply, 'latest' picks the newest plan of the stack")
		case "show":
			cmd.Flags().StringP(tfFlagPlanFile, "p", "", "Plan file to show, 'latest' picks the newest plan of the stack")
		case "output":
			cmd.Flags().StringP(tfFlagName, "n", "", "Name of the output to print. Sensitive outputs are printed as well, asking for one by name is how tofu unredacts it. JSON unless --raw is given")
			cmd.Flags().Bool(tfFlagRaw, false, "Print the output verbatim instead of as JSON, only works for outputs that are a plain string")
		}

		tfCmd.AddCommand(cmd)
	}
}
