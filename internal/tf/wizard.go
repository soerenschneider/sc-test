package tf

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/huh/v2"
)

func validOperation(op string) bool {
	for _, o := range Operations {
		if o.Name == op {
			return true
		}
	}
	return false
}

func shellQuote(s string) string {
	if shellSafeRE.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// splitArgs splits a line the way a POSIX shell would, minus expansions.
func splitArgs(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inArg := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 {
				return nil, errors.New("unterminated single quote")
			}
			cur.WriteString(s[i+1 : i+1+j])
			i += j + 1
			inArg = true
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`", s[i+1]) >= 0 {
					i++
				}
				cur.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, errors.New("unterminated double quote")
			}
			inArg = true
		case '\\':
			if i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
				inArg = true
			}
		case ' ', '\t', '\n':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteByte(c)
			inArg = true
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}

// hint is the invocation of another operation on the same unit.
func (r *runner) hint(op string) string {
	return fmt.Sprintf("%s %s -e %s -s %s", self, op, shellQuote(r.env), shellQuote(r.stack))
}

// commandLine is the invocation that does what the wizard was told.
func (r *runner) commandLine() string {
	parts := []string{self, r.operation, "-e", shellQuote(r.env)}
	if r.stack != "" {
		parts = append(parts, "-s", shellQuote(r.stack))
	}
	if r.planFile != "" {
		parts = append(parts, "-p", shellQuote(r.planFile))
	}
	if r.outputName != "" {
		parts = append(parts, "-n", shellQuote(r.outputName))
	}
	if r.outputRaw {
		parts = append(parts, "--raw")
	}
	if r.yes {
		parts = append(parts, "-y")
	}
	if r.detailedExitcode {
		parts = append(parts, "-d")
	}
	if r.allowStale {
		parts = append(parts, "--allow-stale")
	}
	if len(r.passthrough) > 0 {
		parts = append(parts, "--")
		for _, a := range r.passthrough {
			parts = append(parts, shellQuote(a))
		}
	}
	return strings.Join(parts, " ")
}

func ask(field huh.Field) {
	err := huh.NewForm(huh.NewGroup(field)).WithTheme(huh.ThemeFunc(huh.ThemeCharm)).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		die("aborted")
	}
	if err != nil {
		die("%v", err)
	}
}

func (r *runner) wizard() {
	// 1. Stack
	if !r.set["stack"] {
		var options []huh.Option[string]
		for _, s := range r.listStacks("") {
			label := fmt.Sprintf("%s %s", s, dimStyle.Render("("+strings.Join(r.envsOf(s), ", ")+")"))
			options = append(options, huh.NewOption(label, s))
		}
		if len(options) == 0 {
			die("no stacks below %s", r.rel(r.stacksDir))
		}
		ask(huh.NewSelect[string]().Title("Stack").Options(options...).Value(&r.stack))
	}

	// 2. Environment
	if !r.set["env"] {
		candidates := r.listEnvs()
		if r.stack != "" {
			candidates = r.envsOf(r.stack)
		}
		switch len(candidates) {
		case 0:
			die("no such stack: %s (have: %s)", r.stack, strings.Join(r.listStacks(""), " "))
		case 1:
			r.env = candidates[0]
			logf("Environment: %s, the only one", r.env)
		default:
			if !slices.Contains(candidates, r.env) {
				r.env = candidates[0]
			}
			var options []huh.Option[string]
			for _, e := range candidates {
				options = append(options, huh.NewOption(e, e))
			}
			ask(huh.NewSelect[string]().Title("Environment").Options(options...).Value(&r.env))
		}
	}

	// 3. Operation
	if r.operation == "" {
		var options []huh.Option[string]
		for _, o := range Operations {
			label := fmt.Sprintf("%-24s %s", o.Name, dimStyle.Render(o.Help))
			options = append(options, huh.NewOption(label, o.Name))
		}
		ask(huh.NewSelect[string]().Title("Operation").Options(options...).Value(&r.operation))
	}

	// 4. Whatever the operation needs
	if (r.operation == "apply" || r.operation == "show") && !r.set["plan-file"] {
		r.askPlanFile()
	}

	if r.operation == "output" && !r.set["name"] {
		ask(huh.NewInput().
			Title("Output name").
			Description("Empty prints every output. A sensitive output is only shown in full when asked for by name.").
			Value(&r.outputName))
		r.outputName = strings.TrimSpace(r.outputName)
		if r.outputName != "" && !r.set["raw"] {
			ask(huh.NewConfirm().
				Title("Print it verbatim instead of as JSON?").
				Description("Only works for an output that is a plain string.").
				Affirmative("Raw").Negative("JSON").
				Value(&r.outputRaw))
		}
	}

	if len(r.passthrough) == 0 && !strings.HasPrefix(r.operation, "rotate-passphrase-") {
		var line string
		input := huh.NewInput().Value(&line).Validate(func(s string) error {
			args, err := splitArgs(s)
			if err == nil && r.operation == "raw" && len(args) == 0 {
				err = errors.New("'raw' needs a tofu command")
			}
			return err
		})
		if r.operation == "raw" {
			input.Title("tofu command").Description("Quoted like in a shell, e.g. state list")
		} else {
			input.Title("Extra arguments for tofu").Description("Optional, quoted like in a shell, e.g. -target=module.buckets")
		}
		ask(input)
		r.passthrough, _ = splitArgs(line)
	}
}

func (r *runner) askPlanFile() {
	plans := r.plansOf()
	if len(plans) == 0 {
		if r.operation == "show" {
			die("no plan file for '%s/%s' below %s", r.env, r.stack, r.rel(r.plansDir))
		}
		logf("No saved plan for '%s/%s', tofu plans and asks before applying", r.env, r.stack)
		return
	}

	var options []huh.Option[string]
	for _, p := range plans {
		meta := p + ".meta"
		details := []string{humanAge(time.Since(fileMtime(p))) + " old"}
		if c := metaValue(meta, "commit"); c != "" {
			details = append(details, "commit "+c)
		}
		if metaValue(meta, "dirty") == "true" {
			details = append(details, "dirty tree")
		}
		if op := metaValue(meta, "operation"); op == "plan-destroy" {
			details = append(details, "DESTROY")
		}
		label := fmt.Sprintf("%s %s", filepath.Base(p), dimStyle.Render(strings.Join(details, " · ")))
		options = append(options, huh.NewOption(label, p))
	}
	if r.operation == "apply" {
		options = append(options, huh.NewOption("Without a saved plan, tofu plans and asks", ""))
	}

	r.planFile = plans[0]
	ask(huh.NewSelect[string]().Title("Plan file").Options(options...).Value(&r.planFile))
}

func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func (r *runner) confirm(prompt string) {
	if r.yes {
		return
	}
	if !r.interactive {
		die("%s: no terminal to ask on, pass --yes to skip the confirmation", prompt)
	}
	ok := false
	ask(huh.NewConfirm().Title(prompt).Affirmative("Yes").Negative("No").Value(&ok))
	if !ok {
		die("aborted")
	}
}

// confirmTyped wants the expected answer typed out, for what cannot be undone.
func (r *runner) confirmTyped(prompt, expected string) {
	if r.yes {
		return
	}
	if !r.interactive {
		die("%s: no terminal to ask on, pass --yes to skip the confirmation", prompt)
	}
	var answer string
	ask(huh.NewInput().Title(prompt).Description(fmt.Sprintf("Type %q to continue.", expected)).Value(&answer))
	if answer != expected {
		die("aborted")
	}
}
