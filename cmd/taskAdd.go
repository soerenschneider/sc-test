package cmd

// task_add.go is the interactive UI layer for `sc task add`. It owns the
// cobra command, the huh form (project/depends pickers, context & sizing,
// tags), the option builders that shape those pickers, the lipgloss theme,
// and the confirmation preview.
//
// Everything that talks to taskwarrior — loading the YAML taxonomy,
// fetching projects/tags/tasks, validating a due date, building and running
// the final `task add` — lives in the internal/taskwarrior package, which
// this file calls into.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/soerenschneider/sc/internal/taskwarrior"
	"github.com/soerenschneider/sc/internal/tui"
)

// ---------------------------------------------------------------------------
// cobra wiring
// ---------------------------------------------------------------------------

var taskAddConfigPath string

var taskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Interactively create a taskwarrior task",
	Long: `Interactive alternative to 'task add' with picker-driven project,
tag, and attribute selection.

Loads the taxonomy of known projects and tag categories from a YAML file
(default: ~/.sc/taskwarrior.yaml, override with --config). When the file
is absent, a small generic GTD taxonomy is used.

The Project step is a filterable picker with the configured
'default_project' pre-selected (falls back to "inbox" when the projects
list contains it). Press "/" to filter, esc/enter to exit filter mode,
↑↓ to navigate, enter to pick. To create a new project (or subproject
like "myproject.sub"), pick "＋ create new…" at the bottom of the list
and type the full dotted name in the follow-up input.

An optional Depends-on step lists pending tasks; "(none)" is the default,
so enter skips it. Pick a task to append depends:<ID> to the invocation.

The project and tags are pre-selected based on the description, marked
with ✨ and the reason. They are deduced from, in this order:
  - rules in the taxonomy YAML, mapping keywords to a project and/or tags:
      suggestions:
        - keywords: [zahnarzt, "tax return"]
          project: health
          tags: ["@phone"]
  - a description word matching the last part of a project name,
    e.g. "openbao" → homelab.openbao
  - the projects and tags of existing tasks with similar descriptions`,
	RunE: runTaskAdd,
}

func init() {
	taskAddCmd.Flags().StringVarP(&taskAddConfigPath, "config", "c", "",
		"Path to taxonomy YAML (default ~/.sc/taskwarrior.yaml; falls back to generic GTD)")
	// Wire taskCmd into your root command:
	//   rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
}

// sentinelNewProject is the Select value for the "＋ create new…" option
// in the Project picker. NUL-wrapped so it can't collide with a real
// project name. When selected, the follow-up Input group is revealed so
// the user can type a new project name.
const sentinelNewProject = "\x00__new__\x00"

// ---------------------------------------------------------------------------
// entry point
// ---------------------------------------------------------------------------

func runTaskAdd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	cfg, err := taskwarrior.LoadConfig(taskAddConfigPath)
	if err != nil {
		return err
	}

	projects, err := taskwarrior.Projects(ctx, cfg)
	if err != nil {
		return fmt.Errorf("fetching taskwarrior projects: %w", err)
	}
	tags, err := taskwarrior.Tags(ctx, cfg)
	if err != nil {
		return fmt.Errorf("fetching taskwarrior tags: %w", err)
	}
	tasks, err := taskwarrior.PendingTasks(ctx)
	if err != nil {
		return fmt.Errorf("fetching taskwarrior tasks for depends-on picker: %w", err)
	}

	// Suggestions are a convenience, so failing to fetch the history is not
	// fatal: they still work off the configured rules and project names.
	history, _ := taskwarrior.History(ctx)
	suggester := taskwarrior.NewSuggester(cfg, projects, tags, history)

	opts, ok, err := runAddForm(projects, tags, tasks, cfg, suggester)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return confirmAndCreate(ctx, opts)
}

// ---------------------------------------------------------------------------
// form
// ---------------------------------------------------------------------------

func runAddForm(projects, tags []string, tasks []taskwarrior.Task, cfg *taskwarrior.Config, suggester *taskwarrior.Suggester) (taskwarrior.AddOptions, bool, error) {
	var (
		opts       taskwarrior.AddOptions
		newProject string
		newTagsRaw string
	)

	// 1. Description. Asked in a form of its own, as the project and tags
	// pre-selected in the main form are deduced from it.
	descriptionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Description").
				Description("Required — the task itself").
				Placeholder("e.g. Ship the quarterly report").
				Value(&opts.Description).
				Validate(nonEmpty("description")),
		).Title("✎  New task"),
	).WithShowHelp(true).WithTheme(huh.ThemeFunc(taskAddTheme))

	if err := descriptionForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return opts, false, nil
		}
		return opts, false, err
	}

	// Pre-fill the Project field with the suggested project, falling back to
	// the configured default (or "inbox" if the projects list contains it,
	// or empty otherwise). When the picker opens, the cursor lands on this
	// pre-selection.
	opts.Project = cfg.ResolveDefaultProject()
	locationTags, otherTags := taskwarrior.SplitLocationTags(tags, cfg)
	view := applySuggestion(suggester.Suggest(opts.Description), &opts, projects, locationTags, otherTags,
		"↑↓ navigate · enter picks · / filters (esc exits) · pick ＋ to create new")
	projects = view.Projects
	newTagsRaw = view.ExtraTags

	// Sort projects for a stable suggestion order.
	sort.Strings(projects)

	locationOpts := buildLocationOptions(locationTags)
	tagOpts := buildCategorizedTagOptions(otherTags, cfg)

	// Project picker options with the pre-filled project pre-selected.
	// opts.Project was set to the same value above, so huh's cursor lands
	// on it. See projectPickerOptions for the ordering.
	projectOpts := projectPickerOptions(projects, cfg, opts.Project)

	// Depends-on picker options: "(none)" sentinel first (value ""),
	// then each pending task labelled "[ID] description (project)".
	depOpts := make([]huh.Option[string], 0, len(tasks)+1)
	depOpts = append(depOpts, huh.NewOption("(none)", ""))
	for _, t := range tasks {
		label := fmt.Sprintf("[%d] %s", t.ID, t.Description)
		if t.Project != "" {
			label += fmt.Sprintf(" (%s)", t.Project)
		}
		depOpts = append(depOpts, huh.NewOption(label, strconv.Itoa(t.ID)))
	}

	groups := []*huh.Group{
		// 2. Project picker. Filterable Select over the project list with
		// the configured default pre-selected; huh's built-in "/" key
		// activates filter mode (esc/enter exits filter). The last
		// option is a sentinel that reveals group 2b for typing a new
		// project name.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Description(view.ProjectDescription).
				Options(projectOpts...).
				Value(&opts.Project).
				Height(pickerHeight(len(projectOpts))),
		).Title("📁  Project"),

		// 2b. New project name. Conditional Input — shown only when the
		// user picks "＋ create new…" in the picker above. Autocomplete
		// suggestions from the existing project list help with typing
		// subprojects (e.g. autocomplete "selfhost.k8s." then add a new
		// leaf). Post-processing below replaces the sentinel in
		// opts.Project with whatever the user types here.
		huh.NewGroup(
			huh.NewInput().
				Title("New project name").
				Description("Type · ↓↑ cycle suggestions · ctrl+e completes · . = subproject").
				Placeholder("e.g. work.acme · selfhost.k8s.cluster1").
				Suggestions(projects).
				Value(&newProject).
				Validate(nonEmpty("new project name")),
		).Title("📁  New project").WithHideFunc(func() bool {
			return opts.Project != sentinelNewProject
		}),

		// 3. Context & sizing.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Location").
				Description("Physical-access constraint · leave (none) if it doesn't matter").
				Options(locationOpts...).
				Value(&opts.Location),
			huh.NewSelect[string]().
				Title("Priority").
				Description("Importance").
				Options(priorityOptions()...).
				Value(&opts.Priority),
			huh.NewSelect[string]().
				Title("Size").
				Description("Effort estimate (requires uda.size in .taskrc)").
				Options(sizeOptions()...).
				Value(&opts.Size),
			huh.NewInput().
				Title("Due").
				Description("Anything taskwarrior parses — leave empty for none").
				Placeholder("e.g. tomorrow · friday · 2026-06-15 · +2d").
				Value(&opts.Due).
				Validate(taskwarrior.ValidateDue),
		).Title("🎯  Context & sizing"),

		// 3b. Depends on. Optional single-task picker over pending
		// tasks. First option is a "(none)" sentinel with value "",
		// which is what opts.DependsID starts as — so the cursor lands
		// on "(none)" by default and enter accepts it with zero extra
		// keystrokes. Hidden entirely if the task database has no
		// pending tasks to depend on.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Depends on").
				Description("↑↓ navigate · enter picks · / filters · (none) for no dependency").
				Options(depOpts...).
				Value(&opts.DependsID).
				Height(pickerHeight(len(depOpts))),
		).Title("🔗  Depends on").WithHideFunc(func() bool {
			return len(tasks) == 0
		}),

		// 4. Tags.
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Tags").
				Description(view.TagsDescription).
				Options(tagOpts...).
				Value(&opts.Tags).
				Height(pickerHeight(len(tagOpts))),
			huh.NewInput().
				Title("Additional tags").
				Description("Space-separated · for anything not in the list").
				Placeholder("e.g. urgent q3-review").
				Value(&newTagsRaw),
		).Title("🏷  Tags"),
	}

	form := huh.NewForm(groups...).
		WithShowHelp(true).
		WithTheme(huh.ThemeFunc(taskAddTheme))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return opts, false, nil
		}
		return opts, false, err
	}

	// Post-process.
	if opts.Project == sentinelNewProject {
		opts.Project = strings.TrimSpace(newProject)
	}
	opts.Project = strings.TrimSpace(opts.Project)

	// Resolve DependsID to a human label for the preview.
	if opts.DependsID != "" {
		for _, t := range tasks {
			if strconv.Itoa(t.ID) == opts.DependsID {
				label := fmt.Sprintf("[%d] %s", t.ID, t.Description)
				if t.Project != "" {
					label += fmt.Sprintf(" (%s)", t.Project)
				}
				opts.DependsLabel = label
				break
			}
		}
	}
	if opts.Location != "" {
		opts.Tags = append([]string{opts.Location}, opts.Tags...)
	}
	for _, t := range strings.Fields(newTagsRaw) {
		t = strings.TrimPrefix(t, "+")
		if t != "" {
			opts.Tags = append(opts.Tags, t)
		}
	}
	opts.Tags = dedupe(opts.Tags)
	return opts, true, nil
}

// tagsFieldDescription is the help text of the Tags multiselect.
const tagsFieldDescription = `Grouped by purpose · space to toggle · "/" to filter`

// suggestionView is how a suggestion is presented in the form.
type suggestionView struct {
	// Projects is the project list, extended by the suggested project if a
	// rule named one that has no tasks yet.
	Projects []string
	// ProjectDescription and TagsDescription are the help texts of the
	// Project and Tags fields, prefixed with what was suggested and why.
	ProjectDescription string
	TagsDescription    string
	// ExtraTags are suggested tags unknown so far, for the free-form input.
	ExtraTags string
}

// applySuggestion pre-selects the suggested project and tags in opts: the
// project in the Project picker, the first location tag in the Location
// Select, known tags in the multiselect, and tags unknown so far (only
// possible via rules) in the free-form input, so all of them stay visible
// and can be deselected. Fields without a suggestion keep their value.
func applySuggestion(suggestion taskwarrior.Suggestion, opts *taskwarrior.AddOptions, projects, locationTags, otherTags []string, projectDescription string) suggestionView {
	view := suggestionView{
		Projects:           projects,
		ProjectDescription: projectDescription,
		TagsDescription:    tagsFieldDescription,
	}

	if suggestion.Project != "" {
		opts.Project = suggestion.Project
		if !slices.Contains(projects, suggestion.Project) {
			view.Projects = append(slices.Clone(projects), suggestion.Project)
		}
		view.ProjectDescription = fmt.Sprintf("✨ suggested %s (%s)\n%s", suggestion.Project, suggestion.ProjectReason, projectDescription)
	}

	var unknownTags []string
	for _, tag := range suggestion.Tags {
		switch {
		case slices.Contains(locationTags, tag) && opts.Location == "":
			opts.Location = tag
		case slices.Contains(otherTags, tag):
			opts.Tags = append(opts.Tags, tag)
		case !slices.Contains(locationTags, tag):
			unknownTags = append(unknownTags, tag)
		}
	}
	view.ExtraTags = strings.Join(unknownTags, " ")

	if len(suggestion.Tags) > 0 {
		suggested := make([]string, len(suggestion.Tags))
		for i, tag := range suggestion.Tags {
			suggested[i] = fmt.Sprintf("+%s (%s)", tag, suggestion.TagReasons[tag])
		}
		view.TagsDescription = "✨ pre-selected " + strings.Join(suggested, ", ") + "\n" + tagsFieldDescription
	}

	return view
}

// projectPickerOptions builds the Project Select options in a stable order:
// "＋ create new…" first (the escape hatch to typing a new name), then the
// preselected default, then "(none)" when empty projects are allowed, then
// every other project. preselect is the value the caller has pre-set on the
// bound field so huh's cursor lands on it; projects is expected pre-sorted.
// Shared by `task add` (default = configured default) and `task groom`
// (default = "", forcing a real choice out of the inbox).
func projectPickerOptions(projects []string, cfg *taskwarrior.Config, preselect string) []huh.Option[string] {
	def := preselect
	defInList := def != "" && slices.Contains(projects, def)

	opts := make([]huh.Option[string], 0, len(projects)+2)
	opts = append(opts, huh.NewOption("＋ create new…", sentinelNewProject))

	switch {
	case defInList:
		opts = append(opts, huh.NewOption(def, def))
	case cfg.AllowEmptyProject && def == "":
		// Empty is the default AND allowed — (none) is the default slot.
		opts = append(opts, huh.NewOption("(none)", ""))
	}

	if cfg.AllowEmptyProject && def != "" {
		opts = append(opts, huh.NewOption("(none)", ""))
	}

	for _, p := range projects {
		if p != def {
			opts = append(opts, huh.NewOption(p, p))
		}
	}
	return opts
}

func buildLocationOptions(locs []string) []huh.Option[string] {
	sort.Strings(locs)
	opts := make([]huh.Option[string], 0, len(locs)+1)
	opts = append(opts, huh.NewOption("(none)", ""))
	for _, l := range locs {
		opts = append(opts, huh.NewOption(l, l))
	}
	return opts
}

func buildCategorizedTagOptions(tags []string, cfg *taskwarrior.Config) []huh.Option[string] {
	byCategory := map[string][]string{}
	for _, t := range tags {
		byCategory[cfg.Categorize(t)] = append(byCategory[cfg.Categorize(t)], t)
	}
	for _, ts := range byCategory {
		sort.Strings(ts)
	}

	const prefixWidth = 10
	pad := func(label string) string {
		return fmt.Sprintf("%-*s", prefixWidth, "["+label+"]")
	}

	opts := make([]huh.Option[string], 0, len(tags))
	for _, tc := range cfg.TagCategories {
		for _, t := range byCategory[tc.Label] {
			opts = append(opts, huh.NewOption(pad(tc.Label)+" "+t, t))
		}
	}
	for _, t := range byCategory["other"] {
		opts = append(opts, huh.NewOption(pad("other")+" "+t, t))
	}
	return opts
}

func priorityOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("(none)", ""),
		huh.NewOption("High", "H"),
		huh.NewOption("Medium", "M"),
		huh.NewOption("Low", "L"),
	}
}

func sizeOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("(none)", ""),
		huh.NewOption("XS · tiny", "xs"),
		huh.NewOption("S · small", "s"),
		huh.NewOption("M · medium", "m"),
		huh.NewOption("L · large", "l"),
		huh.NewOption("XL · huge", "xl"),
	}
}

func nonEmpty(name string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		return nil
	}
}

func pickerHeight(items int) int {
	h := max(min(items+2, 15), 5)
	return h
}

// dedupe returns xs with duplicates removed, order preserved. Local to the
// cmd package so taskwarrior needn't export a generic slice helper.
func dedupe(xs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

// ---------------------------------------------------------------------------
// preview + confirm
// ---------------------------------------------------------------------------

// Preview colours come from the shared internal/tui palette rather than
// being redefined here: icons/accent → tui.IdentityStyle, structural labels
// → tui.DimStyle, tags → tui.InfoStyle. tui already does the adaptive
// light/dark background detection once at its init.

// taskAddTheme derives from huh.ThemeCharm and adds a colored thick left
// border to the focused field. Blurred fields get a hidden border of the
// same width so cycling focus doesn't cause layout shift. Without this,
// the two-field Tags page is hard to read — huh's default focused vs.
// blurred colors are similar enough on many terminals that the active
// widget isn't obviously distinguishable.
func taskAddTheme(isDark bool) *huh.Styles {
	s := huh.ThemeCharm(isDark)

	// Focus-border accent, sourced from the shared tui palette so it stays
	// in sync with the log-line identity colour. GetForeground pulls the
	// resolved color.Color back out of the exported Style, since tui
	// exports IdentityStyle but not the raw identity colour.
	accent := tui.IdentityStyle.GetForeground()

	s.Focused.Base = s.Focused.Base.
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(accent).
		PaddingLeft(1)

	s.Blurred.Base = s.Blurred.Base.
		BorderStyle(lipgloss.HiddenBorder()).
		BorderLeft(true).
		PaddingLeft(1)

	return s
}

func renderPreview(opts taskwarrior.AddOptions) string {
	const labelWidth = 9

	row := func(icon, label, value string) string {
		return tui.IdentityStyle.Render(icon) + "  " +
			tui.DimStyle.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "  " +
			value
	}

	var b strings.Builder
	b.WriteString(row("✎", "task", opts.Description) + "\n")
	if opts.Project != "" {
		b.WriteString(row("📁", "project", opts.Project) + "\n")
	}
	if len(opts.Tags) > 0 {
		parts := make([]string, len(opts.Tags))
		for i, t := range opts.Tags {
			parts[i] = tui.InfoStyle.Render("+" + t)
		}
		b.WriteString(row("🏷", "tags", strings.Join(parts, " ")) + "\n")
	}
	if opts.Priority != "" {
		b.WriteString(row("⚡", "priority", opts.Priority) + "\n")
	}
	if opts.Size != "" {
		b.WriteString(row("📏", "size", opts.Size) + "\n")
	}
	if opts.Due != "" {
		b.WriteString(row("⏰", "due", opts.Due) + "\n")
	}
	if opts.DependsLabel != "" {
		b.WriteString(row("🔗", "depends", opts.DependsLabel) + "\n")
	}
	return b.String()
}

func confirmAndCreate(ctx context.Context, opts taskwarrior.AddOptions) error {
	confirm := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create this task?").
				Description(renderPreview(opts)).
				Affirmative("Create").
				Negative("Cancel").
				Value(&confirm),
		).Title("✓  Confirm"),
	).WithTheme(huh.ThemeFunc(taskAddTheme))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}
	if !confirm {
		return nil
	}
	return taskwarrior.Run(ctx, taskwarrior.BuildArgs(opts))
}
