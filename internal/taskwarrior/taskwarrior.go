// Package taskwarrior is the integration layer for the `task` binary and
// the ~/.sc/taskwarrior.yaml taxonomy. It owns config loading (with a GTD
// fallback), the task model types shared with callers (AddOptions, Task),
// shelling out to `task` (fetching projects/tags/pending tasks, validating
// a due date, executing `task add`), and building the `task add` argument
// vector.
//
// It contains no interactive UI — the huh form and cobra wiring live in the
// cmd package, which depends on this one.
package taskwarrior

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// locationCategoryLabel is the label of the category whose members become
// the Location Select in the caller's form.
const locationCategoryLabel = "loc"

// ---------------------------------------------------------------------------
// config
// ---------------------------------------------------------------------------

type Config struct {
	Projects          []string      `yaml:"projects"`
	DefaultProject    string        `yaml:"default_project"`
	AllowEmptyProject bool          `yaml:"allow_empty_project"`
	TagCategories     []TagCategory `yaml:"tag_categories"`

	// SuggestionRules map keywords in task descriptions to a project and/or
	// tags, see Suggester.
	SuggestionRules []SuggestionRule `yaml:"suggestions"`
}

type TagCategory struct {
	Label       string   `yaml:"label"`
	Members     []string `yaml:"members,omitempty"`
	MatchPrefix string   `yaml:"match_prefix,omitempty"`
}

// Categorize returns the label of the category a tag belongs to, or "other".
func (c *Config) Categorize(tag string) string {
	for _, tc := range c.TagCategories {
		for _, m := range tc.Members {
			if tag == m {
				return tc.Label
			}
		}
		if tc.MatchPrefix != "" && strings.HasPrefix(tag, tc.MatchPrefix) {
			return tc.Label
		}
	}
	return "other"
}

func (c *Config) seedTags() []string {
	var out []string
	for _, tc := range c.TagCategories {
		out = append(out, tc.Members...)
	}
	return out
}

// ResolveDefaultProject returns the project the Project field is pre-filled
// with. Priority: explicit `default_project` from YAML → "inbox" if it's in
// the projects list → empty (caller starts with a blank field).
//
// (Named ResolveDefaultProject rather than DefaultProject because the latter
// is already the name of the YAML-backed struct field above.)
func (c *Config) ResolveDefaultProject() string {
	if c.DefaultProject != "" {
		return c.DefaultProject
	}
	if slices.Contains(c.Projects, "inbox") {
		return "inbox"
	}
	return ""
}

// LoadConfig reads the taxonomy from explicit, or from ~/.sc/taskwarrior.yaml
// when explicit is empty. A missing default path yields the generic GTD
// fallback so the tool works with no config file; a missing explicit path is
// an error.
func LoadConfig(explicit string) (*Config, error) {
	path := explicit
	usingDefault := explicit == ""
	if usingDefault {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolving home directory: %w", err)
		}
		path = filepath.Join(home, ".sc", "taskwarrior.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && usingDefault {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Projects: []string{
			"inbox", "personal", "work", "home", "errands",
		},
		DefaultProject:    "inbox",
		AllowEmptyProject: true,
		TagCategories: []TagCategory{
			{Label: "state", Members: []string{"next", "waiting", "someday", "quick", "deep"}},
			{
				Label:       "context",
				Members:     []string{"@computer", "@phone", "@home", "@online"},
				MatchPrefix: "@",
			},
			{Label: "energy", Members: []string{"lowenergy", "highenergy"}},
		},
	}
}

// ---------------------------------------------------------------------------
// task model
// ---------------------------------------------------------------------------

// AddOptions is the contract between the caller's form (which fills it in)
// and this package (which turns it into a `task add` invocation).
type AddOptions struct {
	Description  string
	Project      string
	Location     string
	Tags         []string
	Priority     string
	Size         string
	Due          string
	DependsID    string // task ID as string; "" means no dependency
	DependsLabel string // human label like "[5] Fix login bug", for preview
}

// Task is a minimal projection of a taskwarrior task, enough to render a
// picker entry and shell out with depends:ID.
type Task struct {
	ID          int
	Description string
	Project     string
}

// ---------------------------------------------------------------------------
// taskwarrior I/O
// ---------------------------------------------------------------------------

// Projects returns the union of the seed projects from cfg and the projects
// taskwarrior already knows about (`task _projects`), deduplicated.
func Projects(ctx context.Context, cfg *Config) ([]string, error) {
	out, err := output(ctx, "_projects")
	if err != nil {
		return nil, err
	}
	seed := append([]string{}, cfg.Projects...)
	return dedupe(append(seed, splitLines(out)...)), nil
}

// Tags returns the union of the seed tags from cfg and the user tags
// taskwarrior already knows about (`task _tags`), deduplicated. Virtual tags
// (those with any uppercase, e.g. +OVERDUE) are dropped.
func Tags(ctx context.Context, cfg *Config) ([]string, error) {
	out, err := output(ctx, "_tags")
	if err != nil {
		return nil, err
	}
	fetched := splitLines(out)
	userTags := make([]string, 0, len(fetched))
	for _, t := range fetched {
		if t != strings.ToLower(t) {
			continue // virtual tag
		}
		userTags = append(userTags, t)
	}
	return dedupe(append(cfg.seedTags(), userTags...)), nil
}

// exportTasks runs `task rc.verbose=nothing [FILTERS...] export` and returns
// the parsed Task projection. Callers apply their own ordering.
func exportTasks(ctx context.Context, filters ...string) ([]Task, error) {
	args := append([]string{"rc.verbose=nothing"}, filters...)
	args = append(args, "export")
	out, err := output(ctx, args...)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
		Project     string `json:"project"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing task export output: %w", err)
	}
	tasks := make([]Task, 0, len(raw))
	for _, r := range raw {
		if r.ID == 0 {
			continue // skip completed/deleted with id=0
		}
		tasks = append(tasks, Task{ID: r.ID, Description: r.Description, Project: r.Project})
	}
	return tasks, nil
}

// PendingTasks returns all pending tasks, newest first — recent tasks are the
// likelier dependency candidates for the Depends-on picker.
func PendingTasks(ctx context.Context) ([]Task, error) {
	tasks, err := exportTasks(ctx, "status:pending")
	if err != nil {
		return nil, err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID > tasks[j].ID })
	return tasks, nil
}

// InboxTasks returns the pending tasks whose project is exactly "inbox",
// oldest first so a grooming sweep processes them in capture order.
// project.is:inbox (not project:inbox) so it can't also match inbox.* .
func InboxTasks(ctx context.Context) ([]Task, error) {
	tasks, err := exportTasks(ctx, "status:pending", "project.is:inbox")
	if err != nil {
		return nil, err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

// output runs `task ARGS...` and returns captured stdout. Used for the
// read-only queries above; use Run for the interactive `task add`.
func output(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "task", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	if execErr := new(exec.Error); errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return nil, errors.New("'task' not found in PATH, install taskwarrior first")
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return nil, fmt.Errorf("task %s: %s", strings.Join(args, " "), msg)
}

func splitLines(b []byte) []string {
	raw := strings.Split(string(b), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// SplitLocationTags partitions the tag list into the location tags (those
// whose category is the location category) and everything else, using the
// taxonomy in cfg. The caller feeds these into the Location Select and the
// categorized tag multiselect respectively.
func SplitLocationTags(all []string, cfg *Config) (locs, rest []string) {
	for _, t := range all {
		if cfg.Categorize(t) == locationCategoryLabel {
			locs = append(locs, t)
		} else {
			rest = append(rest, t)
		}
	}
	return locs, rest
}

// ValidateDue checks whether taskwarrior can parse the input as a due date
// by running "task count due:INPUT" and inspecting the exit status. An empty
// string is allowed (no due date). Its signature matches huh's
// Input.Validate, so callers can pass it directly — on error, huh shows the
// message inline and keeps focus so the user can fix it.
func ValidateDue(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	//#nosec:G204
	cmd := exec.CommandContext(ctx, "task", "rc.verbose=nothing",
		"rc.confirmation=no", "due:"+input, "count")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err == nil {
		return nil
	}

	// task's stderr is the useful diagnostic. Strip whitespace and take
	// the first line so it renders cleanly in a form validator inline.
	msg := strings.TrimSpace(stderr.String())
	if i := strings.IndexByte(msg, '\n'); i > 0 {
		msg = msg[:i]
	}
	if msg == "" {
		return fmt.Errorf("taskwarrior rejected %q as a due date", input)
	}
	return errors.New(msg)
}

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
// building + executing `task add`
// ---------------------------------------------------------------------------

// BuildArgs turns the collected options into a `task add …` argument vector.
func BuildArgs(opts AddOptions) []string {
	args := []string{"add", opts.Description}
	if opts.Project != "" {
		args = append(args, "project:"+opts.Project)
	}
	for _, t := range opts.Tags {
		args = append(args, "+"+t)
	}
	if opts.Priority != "" {
		args = append(args, "priority:"+opts.Priority)
	}
	if opts.Size != "" {
		args = append(args, "size:"+opts.Size)
	}
	if opts.Due != "" {
		args = append(args, "due:"+opts.Due)
	}
	if opts.DependsID != "" {
		args = append(args, "depends:"+opts.DependsID)
	}
	return args
}

// BuildModifyArgs turns groomed options into a `task <id> modify …` argument
// vector. The project is always emitted: a non-empty value replaces the
// task's project (e.g. inbox → work.acme), and an empty value ("project:")
// clears it entirely — so either way the inbox project is removed. The
// description is re-set (harmless if unchanged), tags are added (the caller
// passes the final desired set), and attributes are set when non-empty.
func BuildModifyArgs(id int, opts AddOptions) []string {
	args := []string{strconv.Itoa(id), "modify", "project:" + opts.Project}
	if opts.Description != "" {
		args = append(args, "description:"+opts.Description)
	}
	for _, t := range opts.Tags {
		args = append(args, "+"+t)
	}
	if opts.Priority != "" {
		args = append(args, "priority:"+opts.Priority)
	}
	if opts.Size != "" {
		args = append(args, "size:"+opts.Size)
	}
	if opts.Due != "" {
		args = append(args, "due:"+opts.Due)
	}
	return args
}

// Run executes `task ARGS...` with stdio wired to the current process, so
// taskwarrior's own output (and any prompts) reach the user. Use it with
// BuildArgs to create the task.
func Run(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "task", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
