package taskwarrior

import (
	"slices"
	"testing"
)

func TestSuggester_Suggest(t *testing.T) {
	cfg := &Config{
		DefaultProject: "inbox",
		SuggestionRules: []SuggestionRule{
			{Keywords: []string{"zahnarzt"}, Project: "health", Tags: []string{"@phone"}},
			{Keywords: []string{"tax return"}, Project: "finance.de"},
		},
	}
	projects := []string{"inbox", "errands", "finance.de", "health", "homelab", "homelab.openbao", "homelab.storage", "homelab.storage.hw"}
	tags := []string{"@computer", "@dus", "@phone", "next"}
	history := []HistoryTask{
		{Description: "backup nas photos", Project: "homelab.storage", Tags: []string{"@computer"}},
		{Description: "nas disk replacement", Project: "homelab.storage.hw", Tags: []string{"@dus"}},
		{Description: "nas firmware update", Project: "homelab.storage", Tags: []string{"@computer"}},
		{Description: "buy milk", Project: "errands", Tags: []string{"@dus"}},
		{Description: "groceries milk eggs", Project: "inbox", Tags: []string{"@dus"}},
		{Description: "milk the cow", Project: "inbox", Tags: []string{"@dus"}},
	}

	tests := []struct {
		name        string
		description string
		wantProject string
		wantReason  string
		wantTags    []string
	}{
		{
			name:        "rule matches word prefix",
			description: "Zahnarzttermin vereinbaren",
			wantProject: "health",
			wantReason:  `rule "zahnarzt"`,
			wantTags:    []string{"@phone"},
		},
		{
			name:        "rule matches phrase",
			description: "File the tax return",
			wantProject: "finance.de",
			wantReason:  `rule "tax return"`,
		},
		{
			name:        "project name matches leaf",
			description: "Upgrade OpenBao to 2.1",
			wantProject: "homelab.openbao",
			wantReason:  "matches project name",
		},
		{
			name:        "history votes for the deepest project with enough support",
			description: "nas firmware",
			wantProject: "homelab.storage",
			wantReason:  "similar to 3 tasks",
			wantTags:    []string{"@computer"},
		},
		{
			name:        "inbox tasks do not vote for a project but for tags",
			description: "milk",
			wantProject: "",
			wantTags:    []string{"@dus"},
		},
		{
			name:        "no evidence",
			description: "something completely different",
		},
		{
			name:        "empty description",
			description: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSuggester(cfg, projects, tags, history).Suggest(tt.description)
			if got.Project != tt.wantProject {
				t.Errorf("Project = %q, want %q", got.Project, tt.wantProject)
			}
			if tt.wantReason != "" && got.ProjectReason != tt.wantReason {
				t.Errorf("ProjectReason = %q, want %q", got.ProjectReason, tt.wantReason)
			}
			if !slices.Equal(got.Tags, tt.wantTags) {
				t.Errorf("Tags = %v, want %v", got.Tags, tt.wantTags)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []string
	}{
		{
			name: "lowercases and splits on punctuation",
			s:    "Fix OpenBao-Cluster, now!",
			want: []string{"fix", "openbao", "cluster", "now"},
		},
		{
			name: "drops short words, numbers and stopwords",
			s:    "Check the k8s update for 2026 in DE",
			want: []string{"k8s", "update"},
		},
		{
			name: "keeps umlauts",
			s:    "Bücher zurückgeben",
			want: []string{"bücher", "zurückgeben"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokenize(tt.s); !slices.Equal(got, tt.want) {
				t.Errorf("tokenize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectAndParents(t *testing.T) {
	tests := []struct {
		project string
		want    []string
	}{
		{project: "a", want: []string{"a"}},
		{project: "a.b.c", want: []string{"a.b.c", "a.b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			if got := projectAndParents(tt.project); !slices.Equal(got, tt.want) {
				t.Errorf("projectAndParents() = %v, want %v", got, tt.want)
			}
		})
	}
}
