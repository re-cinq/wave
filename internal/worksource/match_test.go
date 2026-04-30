package worksource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatches(t *testing.T) {
	base := BindingRecord{
		Forge:       "github",
		RepoPattern: "owner/repo",
		Trigger:     TriggerOnLabel,
		Active:      true,
	}
	ref := WorkItemRef{
		Forge:  "github",
		Repo:   "owner/repo",
		Kind:   "issue",
		ID:     "42",
		Labels: []string{"bug", "ready-for-impl"},
		State:  "open",
	}

	cases := []struct {
		name string
		mut  func(*BindingRecord, *WorkItemRef)
		want bool
	}{
		{name: "exact match", mut: nil, want: true},
		{
			name: "inactive excluded",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.Active = false },
			want: false,
		},
		{
			name: "forge mismatch",
			mut:  func(_ *BindingRecord, r *WorkItemRef) { r.Forge = "gitea" },
			want: false,
		},
		{
			name: "glob star matches any single segment",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.RepoPattern = "owner/*" },
			want: true,
		},
		{
			name: "glob star does not cross slash",
			mut: func(b *BindingRecord, r *WorkItemRef) {
				b.RepoPattern = "owner/*"
				r.Repo = "owner/sub/repo"
			},
			want: false,
		},
		{
			name: "glob ? matches one rune",
			mut: func(b *BindingRecord, r *WorkItemRef) {
				b.RepoPattern = "owner/rep?"
				r.Repo = "owner/repx"
			},
			want: true,
		},
		{
			name: "label any-of matches when at least one label intersects",
			mut: func(b *BindingRecord, _ *WorkItemRef) {
				b.LabelFilter = []string{"ready-for-impl", "blocked"}
			},
			want: true,
		},
		{
			name: "label any-of mismatch",
			mut: func(b *BindingRecord, r *WorkItemRef) {
				b.LabelFilter = []string{"won't-fix"}
				r.Labels = []string{"bug"}
			},
			want: false,
		},
		{
			name: "empty label filter accepts any labels",
			mut: func(b *BindingRecord, r *WorkItemRef) {
				b.LabelFilter = nil
				r.Labels = nil
			},
			want: true,
		},
		{
			name: "state filter match",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.State = "open" },
			want: true,
		},
		{
			name: "state filter mismatch",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.State = "closed" },
			want: false,
		},
		{
			name: "state filter any accepts any",
			mut: func(b *BindingRecord, r *WorkItemRef) {
				b.State = "any"
				r.State = "closed"
			},
			want: true,
		},
		{
			name: "kind filter match",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.Kinds = []string{"issue", "pull_request"} },
			want: true,
		},
		{
			name: "kind filter mismatch",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.Kinds = []string{"pull_request"} },
			want: false,
		},
		{
			name: "repo mismatch (literal)",
			mut:  func(_ *BindingRecord, r *WorkItemRef) { r.Repo = "other/repo" },
			want: false,
		},
		{
			name: "malformed glob never matches",
			mut:  func(b *BindingRecord, _ *WorkItemRef) { b.RepoPattern = "[" },
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, r := base, ref
			if c.mut != nil {
				c.mut(&b, &r)
			}
			assert.Equal(t, c.want, matches(b, r))
		})
	}
}

func TestMatches_MultipleMismatchesAllReturnFalse(t *testing.T) {
	rec := BindingRecord{
		Forge: "github", RepoPattern: "a/b", Active: true,
		LabelFilter: []string{"bug"}, State: "open",
	}
	ref := WorkItemRef{
		Forge: "gitea", Repo: "x/y",
		Labels: []string{"docs"}, State: "closed",
	}
	assert.False(t, matches(rec, ref))
}
