package cmd

// task_groom.go adds `sc task groom` (alias `inbox`): an interactive
// GTD-style inbox sweep. Quick-capture — e.g. a phone alias that runs
// `task add project:inbox …` — drops bare tasks into the inbox; this command
// walks each pending project:inbox task and runs a grooming form: assign a
// real project (which removes the inbox project), plus the usual tags /
// location / priority / size / due, then applies it with `task <id> modify`.
// Per task you can Apply, Skip (leave it in the inbox), or Quit.
//
// It reuses the form helpers, theme and preview from task_add.go, and the
// internal/taskwarrior package for all `task` I/O.

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/soerenschneider/sc/internal/taskwarrior"
)

var taskGroomConfigPath string

var taskGroomCmd = &cobra.Command{
	Use:     "groom",
	Aliases: []string{"inbox"},
	Short:   "Interactively groom inbox tasks (assign project, tags, …)",
	Long: `Walk every pending task with project:inbox and groom it: pick a real
project (which removes the inbox project), add tags/location/priority/size/
due, then apply with 'task <id> modify'.

Uses the same taxonomy YAML as 'task add' (default ~/.sc/taskwarrior.yaml,
override with --config). Tasks are processed oldest-first. For each one you
can Apply the changes, Skip it (leave it in the inbox), or Quit the sweep.

Like 'task add', the project and tags are pre-selected based on the task's
description, marked with ✨ and the reason. See 'sc task add --help'.`,
	RunE: runTaskGroom,
}

func init() {
	taskGroomCmd.Flags().StringVarP(&taskGroomConfigPath, "config", "c", "",
		"Path to taxonomy YAML (default ~/.sc/taskwarrior.yaml; falls back to generic GTD)")
	taskCmd.AddCommand(taskGroomCmd)
}

// groomAction is the per-task disposition chosen at the confirm step.
type groomAction int

const (
	groomApply groomAction = iota // apply the modify; the task leaves the inbox
	groomSkip                     // leave the task in the inbox, move on
	groomQuit                     // stop the sweep
)

func runTaskGroom(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	cfg, err := taskwarrior.LoadConfig(taskGroomConfigPath)
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
	inbox, err := taskwarrior.InboxTasks(ctx)
	if err != nil {
		return fmt.Errorf("fetching inbox tasks: %w", err)
	}

	if len(inbox) == 0 {
		fmt.Println("Inbox is empty — nothing to groom. 🎉")
		return nil
	}

	// Offering "inbox" as a grooming target makes no sense — the whole
	// point is to move tasks out of it.
	targetProjects := make([]string, 0, len(projects))
	for _, p := range projects {
		if p != "inbox" {
			targetProjects = append(targetProjects, p)
		}
	}
	sort.Strings(targetProjects)

	// Suggestions are a convenience, so failing to fetch the history is not
	// fatal: they still work off the configured rules and project names.
	history, _ := taskwarrior.History(ctx)

	groomed := 0
groomLoop:
	for i, t := range inbox {
		// The groomed task itself is part of the history and would vote for
		// its own tags, so it is left out.
		others := slices.DeleteFunc(slices.Clone(history), func(h taskwarrior.HistoryTask) bool {
			return h.ID == t.ID
		})
		suggestion := taskwarrior.NewSuggester(cfg, targetProjects, tags, others).Suggest(t.Description)

		opts, ok, err := runGroomForm(t, targetProjects, tags, cfg, suggestion, i+1, len(inbox))
		if err != nil {
			return err
		}
		if !ok { // aborted the field form (esc) → treat as quit
			break groomLoop
		}

		action, err := confirmGroom(opts, i+1, len(inbox))
		if err != nil {
			return err
		}
		switch action {
		case groomQuit:
			break groomLoop
		case groomSkip:
			continue
		case groomApply:
			if err := taskwarrior.Run(ctx, taskwarrior.BuildModifyArgs(t.ID, opts)); err != nil {
				return fmt.Errorf("modifying task %d: %w", t.ID, err)
			}
			groomed++
		}
	}

	fmt.Printf("\nGroomed %d task(s); %d still in the inbox.\n", groomed, len(inbox)-groomed)
	return nil
}

// runGroomForm runs the per-task grooming form. It mirrors runAddForm but
// seeds the description from the captured task and pre-selects the suggested
// project, or starts the project picker empty if there is none (so the user
// must choose a real project). Returns ok=false if the user aborts the form
// (esc), which the caller treats as quitting the sweep.
func runGroomForm(t taskwarrior.Task, projects, tags []string, cfg *taskwarrior.Config, suggestion taskwarrior.Suggestion, idx, total int) (taskwarrior.AddOptions, bool, error) {
	var (
		opts       taskwarrior.AddOptions
		newProject string
		newTagsRaw string
	)

	// Seed the description from the captured task; leave the project empty
	// unless one is suggested, so the picker forces a real choice (that's
	// the point of grooming).
	opts.Description = t.Description
	opts.Project = ""

	locationTags, otherTags := taskwarrior.SplitLocationTags(tags, cfg)
	view := applySuggestion(suggestion, &opts, projects, locationTags, otherTags,
		"↑↓ navigate · enter picks · / filters · ＋ to create new · (none) removes the project")
	projects = view.Projects
	sort.Strings(projects)
	newTagsRaw = view.ExtraTags

	locationOpts := buildLocationOptions(locationTags)
	tagOpts := buildCategorizedTagOptions(otherTags, cfg)
	projectOpts := projectPickerOptions(projects, cfg, opts.Project)

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewInput().
				Title("Description").
				Description("Captured from the inbox — edit if needed").
				Value(&opts.Description).
				Validate(nonEmpty("description")),
		).Title(fmt.Sprintf("🧹  Groom %d/%d", idx, total)),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Description(view.ProjectDescription).
				Options(projectOpts...).
				Value(&opts.Project).
				Height(pickerHeight(len(projectOpts))),
		).Title("📁  Project"),

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

	// Post-process — mirror of the add flow.
	if opts.Project == sentinelNewProject {
		opts.Project = strings.TrimSpace(newProject)
	}
	opts.Project = strings.TrimSpace(opts.Project)
	opts.Description = strings.TrimSpace(opts.Description)

	if opts.Location != "" {
		opts.Tags = append([]string{opts.Location}, opts.Tags...)
	}
	for _, tag := range strings.Fields(newTagsRaw) {
		tag = strings.TrimPrefix(tag, "+")
		if tag != "" {
			opts.Tags = append(opts.Tags, tag)
		}
	}
	opts.Tags = dedupe(opts.Tags)
	return opts, true, nil
}

// confirmGroom shows the groomed result and asks what to do with it. Esc at
// this step stops the sweep (returns groomQuit).
func confirmGroom(opts taskwarrior.AddOptions, idx, total int) (groomAction, error) {
	action := groomApply
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[groomAction]().
				Title(fmt.Sprintf("Apply changes to task %d/%d?", idx, total)).
				Description(renderPreview(opts)).
				Options(
					huh.NewOption("Apply — groom & remove from inbox", groomApply),
					huh.NewOption("Skip — leave it in the inbox", groomSkip),
					huh.NewOption("Quit grooming", groomQuit),
				).
				Value(&action),
		).Title("✓  Confirm"),
	).WithTheme(huh.ThemeFunc(taskAddTheme))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return groomQuit, nil // esc at confirm → stop the sweep
		}
		return groomApply, err
	}
	return action, nil
}
