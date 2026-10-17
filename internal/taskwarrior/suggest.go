package taskwarrior

// suggest.go deduces a project and tags for a new task from its description.
// It is a heuristic built from three sources of evidence, in order of
// precedence:
//
//  1. Rules from the taxonomy YAML (`suggestions:`), mapping keywords to a
//     project and/or tags. Explicit, so they always win.
//  2. Project names: a description word that equals the leaf of a project
//     ("homelab.openbao" ← "openbao"). Not done for tags: words like
//     "investigate" turned out to appear in descriptions far more often
//     than the equally named tag is applied.
//  3. History: the most similar existing tasks, by shared words weighted by
//     how rare the word is (idf). Words are compared by their first stemLen
//     runes, so inflections and compounds match. Their projects and tags are
//     voted on, weighted by similarity. Votes for a project also count for
//     its parent projects, so sibling subprojects reinforce their common
//     parent, and the deepest project with enough support wins.
//
// The thresholds favour precision over recall: a wrong pre-selection costs
// more attention than a missing one.
//
// Everything here is pure: callers fetch the history with History and feed
// it into NewSuggester.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// minTokenLen is the minimum length in runes of a word to be considered.
	minTokenLen = 3
	// maxNeighbors is the number of most similar tasks that vote.
	maxNeighbors = 10
	// minSimilarity is the minimum similarity for a task to vote at all.
	minSimilarity = 0.3
	// minShare is the minimum share of the neighbors' summed similarity a
	// project or tag needs to be suggested.
	minShare = 0.5
	// stemLen is the number of runes words are reduced to when comparing
	// them with existing tasks.
	stemLen = 6
	// minSupporters is the number of similar tasks that need to agree on a
	// project or tag for it to be suggested.
	minSupporters = 2
)

// stopwords are frequent English and German words that carry no meaning for
// the classification.
var stopwords = map[string]bool{
	// English
	"the": true, "and": true, "for": true, "with": true, "from": true, "about": true,
	"into": true, "onto": true, "that": true, "this": true, "these": true, "those": true,
	"are": true, "was": true, "were": true, "been": true, "have": true, "has": true,
	"not": true, "but": true, "all": true, "any": true, "some": true, "new": true,
	"get": true, "set": true, "use": true, "via": true, "out": true, "off": true,
	"our": true, "your": true, "you": true, "his": true, "her": true, "its": true,
	"next": true, "maybe": true, "check": true, "make": true, "more": true,
	// German
	"der": true, "die": true, "das": true, "den": true, "dem": true, "des": true,
	"ein": true, "eine": true, "einen": true, "einem": true, "einer": true,
	"und": true, "oder": true, "mit": true, "für": true, "von": true, "vom": true,
	"zum": true, "zur": true, "auf": true, "aus": true, "bei": true, "nach": true,
	"über": true, "unter": true, "ist": true, "sind": true, "nicht": true, "noch": true,
	"auch": true, "wie": true, "wer": true, "neue": true, "neuen": true,
}

// SuggestionRule maps keywords found in a task description to a project
// and/or tags. A single-word keyword matches any word starting with it, so
// "zahnarzt" also matches "Zahnarzttermin". A multi-word keyword matches the
// exact phrase.
type SuggestionRule struct {
	Keywords []string `yaml:"keywords"`
	Project  string   `yaml:"project,omitempty"`
	Tags     []string `yaml:"tags,omitempty"`
}

// HistoryTask is an existing task used as evidence for suggestions.
type HistoryTask struct {
	// ID is the task's working-set ID, 0 for completed tasks.
	ID          int
	Description string
	Project     string
	Tags        []string
}

// History returns all tasks that have not been deleted, including completed
// ones, as evidence for suggestions.
func History(ctx context.Context) ([]HistoryTask, error) {
	out, err := output(ctx, "rc.verbose=nothing", "status.not:deleted", "export")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID          int      `json:"id"`
		Description string   `json:"description"`
		Project     string   `json:"project"`
		Tags        []string `json:"tags"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing task export output: %w", err)
	}
	tasks := make([]HistoryTask, 0, len(raw))
	for _, r := range raw {
		tasks = append(tasks, HistoryTask{ID: r.ID, Description: r.Description, Project: r.Project, Tags: r.Tags})
	}
	return tasks, nil
}

// Suggestion is the deduced project and tags for a description, each with a
// human-readable reason.
type Suggestion struct {
	Project       string
	ProjectReason string
	Tags          []string
	TagReasons    map[string]string
}

// Suggester deduces projects and tags from task descriptions.
type Suggester struct {
	cfg      *Config
	projects []string
	tags     []string

	history       []HistoryTask
	historyTokens []map[string]bool
	idf           map[string]float64
}

// NewSuggester builds a Suggester from the taxonomy, the known projects and
// tags, and the existing tasks.
func NewSuggester(cfg *Config, projects, tags []string, history []HistoryTask) *Suggester {
	s := &Suggester{
		cfg:           cfg,
		projects:      projects,
		tags:          tags,
		history:       history,
		historyTokens: make([]map[string]bool, len(history)),
		idf:           map[string]float64{},
	}

	docFreq := map[string]int{}
	for i, task := range history {
		s.historyTokens[i] = tokenSet(stems(tokenize(task.Description)))
		for token := range s.historyTokens[i] {
			docFreq[token]++
		}
	}
	for token, df := range docFreq {
		s.idf[token] = math.Log(1 + float64(len(history))/float64(df))
	}

	return s
}

// Suggest returns the project and tags deduced from description. Fields are
// left empty if there is not enough evidence.
func (s *Suggester) Suggest(description string) Suggestion {
	ret := Suggestion{TagReasons: map[string]string{}}
	tokens := tokenize(description)

	addTag := func(tag, reason string) {
		if _, found := ret.TagReasons[tag]; !found {
			ret.Tags = append(ret.Tags, tag)
			ret.TagReasons[tag] = reason
		}
	}

	// 1. Rules. Matched against all words, as the user chose the keywords
	// deliberately, even if they are short or stopwords.
	words := splitWords(description)
	for _, rule := range s.cfg.SuggestionRules {
		keyword, ok := matchRule(rule, words)
		if !ok {
			continue
		}
		reason := fmt.Sprintf("rule %q", keyword)
		if rule.Project != "" && ret.Project == "" {
			ret.Project, ret.ProjectReason = rule.Project, reason
		}
		for _, tag := range rule.Tags {
			addTag(tag, reason)
		}
	}

	neighbors := s.neighbors(tokens)

	// 2. Project names.
	if ret.Project == "" {
		if project := s.matchProjectName(tokens, neighbors); project != "" {
			ret.Project, ret.ProjectReason = project, "matches project name"
		}
	}

	// 3. History.
	if ret.Project == "" {
		ret.Project, ret.ProjectReason = s.voteProject(neighbors)
	}
	for _, tag := range s.voteTags(neighbors) {
		addTag(tag, fmt.Sprintf("similar to %d tasks", countWithTag(neighbors, tag)))
	}

	return ret
}

// neighbor is a similar existing task.
type neighbor struct {
	task       HistoryTask
	similarity float64
}

// neighbors returns the existing tasks most similar to tokens, most similar
// first. Similarity is the idf-weighted share of the description's words
// that also occur in the existing task, between 0 and 1.
func (s *Suggester) neighbors(tokens []string) []neighbor {
	unique := tokenSet(stems(tokens))

	var total float64
	for token := range unique {
		total += s.tokenWeight(token)
	}
	if total == 0 {
		return nil
	}

	var ret []neighbor
	for i, historyTokens := range s.historyTokens {
		var shared float64
		for token := range unique {
			if historyTokens[token] {
				shared += s.tokenWeight(token)
			}
		}
		if similarity := shared / total; similarity >= minSimilarity {
			ret = append(ret, neighbor{task: s.history[i], similarity: similarity})
		}
	}

	sort.SliceStable(ret, func(i, j int) bool { return ret[i].similarity > ret[j].similarity })
	if len(ret) > maxNeighbors {
		ret = ret[:maxNeighbors]
	}
	return ret
}

// tokenWeight returns the idf of token. Words never seen before get the
// highest possible weight, as they are as rare as it gets.
func (s *Suggester) tokenWeight(token string) float64 {
	if idf, found := s.idf[token]; found {
		return idf
	}
	return math.Log(1 + float64(len(s.history)+1))
}

// matchProjectName returns the project whose leaf name occurs in tokens. If
// several projects share that leaf name, the one most similar tasks belong
// to wins.
func (s *Suggester) matchProjectName(tokens []string, neighbors []neighbor) string {
	var candidates []string
	for _, project := range s.projects {
		if s.isDefaultProject(project) {
			continue
		}
		leaf := project[strings.LastIndex(project, ".")+1:]
		if utf8.RuneCountInString(leaf) >= minTokenLen && slices.Contains(tokens, strings.ToLower(leaf)) {
			candidates = append(candidates, project)
		}
	}
	if len(candidates) == 0 {
		return ""
	}

	sort.Strings(candidates)
	best, bestScore := candidates[0], 0.0
	for _, candidate := range candidates {
		var score float64
		for _, n := range neighbors {
			if isSameOrSubproject(n.task.Project, candidate) {
				score += n.similarity
			}
		}
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	return best
}

// voteProject returns the deepest known project with enough support from
// neighbors. Votes for a project also count for its parents.
func (s *Suggester) voteProject(neighbors []neighbor) (string, string) {
	var total float64
	scores := map[string]float64{}
	supporters := map[string]int{}

	for _, n := range neighbors {
		// Tasks without a project or in the inbox have not been sorted yet
		// and say nothing about where a task belongs.
		if n.task.Project == "" || s.isDefaultProject(n.task.Project) {
			continue
		}
		total += n.similarity
		for _, project := range projectAndParents(n.task.Project) {
			scores[project] += n.similarity
			supporters[project]++
		}
	}

	best, bestDepth := "", -1
	for project, score := range scores {
		if !slices.Contains(s.projects, project) || !hasEnoughSupport(score, total, supporters[project]) {
			continue
		}
		depth := strings.Count(project, ".")
		if depth > bestDepth || (depth == bestDepth && score > scores[best]) {
			best, bestDepth = project, depth
		}
	}

	if best == "" {
		return "", ""
	}
	return best, fmt.Sprintf("similar to %d tasks", supporters[best])
}

// voteTags returns the known tags with enough support from neighbors.
func (s *Suggester) voteTags(neighbors []neighbor) []string {
	var total float64
	scores := map[string]float64{}
	supporters := map[string]int{}

	for _, n := range neighbors {
		total += n.similarity
		for _, tag := range n.task.Tags {
			scores[tag] += n.similarity
			supporters[tag]++
		}
	}

	var ret []string
	for tag, score := range scores {
		if slices.Contains(s.tags, tag) && hasEnoughSupport(score, total, supporters[tag]) {
			ret = append(ret, tag)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return scores[ret[i]] > scores[ret[j]] })
	return ret
}

func (s *Suggester) isDefaultProject(project string) bool {
	return project == s.cfg.ResolveDefaultProject()
}

// hasEnoughSupport decides whether a project or tag is suggested: it needs
// the majority of the summed similarity and enough agreeing tasks.
func hasEnoughSupport(score, total float64, supporters int) bool {
	return total > 0 && score/total >= minShare && supporters >= minSupporters
}

func countWithTag(neighbors []neighbor, tag string) int {
	var count int
	for _, n := range neighbors {
		if slices.Contains(n.task.Tags, tag) {
			count++
		}
	}
	return count
}

// matchRule returns the first keyword of rule found in words.
func matchRule(rule SuggestionRule, words []string) (string, bool) {
	joined := " " + strings.Join(words, " ") + " "
	for _, keyword := range rule.Keywords {
		keywordWords := splitWords(keyword)
		switch len(keywordWords) {
		case 0:
			continue
		case 1:
			for _, word := range words {
				if strings.HasPrefix(word, keywordWords[0]) {
					return keyword, true
				}
			}
		default:
			if strings.Contains(joined, " "+strings.Join(keywordWords, " ")+" ") {
				return keyword, true
			}
		}
	}
	return "", false
}

// projectAndParents returns project and all its parent projects, e.g.
// "a.b.c" → ["a.b.c", "a.b", "a"].
func projectAndParents(project string) []string {
	ret := []string{project}
	for i := strings.LastIndex(project, "."); i > 0; i = strings.LastIndex(project, ".") {
		project = project[:i]
		ret = append(ret, project)
	}
	return ret
}

func isSameOrSubproject(project, parent string) bool {
	return project == parent || strings.HasPrefix(project, parent+".")
}

// tokenize splits a description into lowercase words, dropping short words,
// numbers and stopwords.
func tokenize(s string) []string {
	var ret []string
	for _, word := range splitWords(s) {
		if utf8.RuneCountInString(word) < minTokenLen || stopwords[word] || isNumber(word) {
			continue
		}
		ret = append(ret, word)
	}
	return ret
}

func splitWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// stems reduces words to their first stemLen runes, so inflections and
// compounds like "monitor", "monitoring" and "monitors" are treated alike.
func stems(tokens []string) []string {
	ret := make([]string, len(tokens))
	for i, token := range tokens {
		runes := []rune(token)
		ret[i] = string(runes[:min(len(runes), stemLen)])
	}
	return ret
}

func isNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func tokenSet(tokens []string) map[string]bool {
	ret := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		ret[token] = true
	}
	return ret
}
