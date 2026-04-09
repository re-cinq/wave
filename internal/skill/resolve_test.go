package skill

import (
	"fmt"
	"reflect"
	"testing"
)

// SC-005 Traceability: ResolveSkills test cases map to acceptance criteria as follows:
//
//	"all empty returns nil"                        → US3-4: all nil → nil
//	"global only sorted"                           → US3-1: global-only resolved
//	"all three scopes merged deduped sorted"       → US3-3: all scopes merged sorted
//	"duplicate across scopes appears once"         → US3-2: global+persona deduplicated
//	"empty slices not nil"                         → US3-5: all empty → nil
func TestResolveSkills(t *testing.T) {
	tests := []struct {
		name     string
		global   []string
		persona  []string
		pipeline []string
		want     []string
	}{
		{
			name: "all empty returns nil", // US3-4: all nil → nil
			want: nil,
		},
		{
			name:   "global only sorted", // US3-1: global-only resolved
			global: []string{"zeta", "alpha", "mu"},
			want:   []string{"alpha", "mu", "zeta"},
		},
		{
			name:    "persona only sorted",
			persona: []string{"charlie", "alpha"},
			want:    []string{"alpha", "charlie"},
		},
		{
			name:     "pipeline only sorted",
			pipeline: []string{"beta", "alpha"},
			want:     []string{"alpha", "beta"},
		},
		{
			name:     "all three scopes merged deduped sorted", // US3-3: all scopes merged sorted
			global:   []string{"alpha", "gamma"},
			persona:  []string{"beta", "delta"},
			pipeline: []string{"epsilon", "alpha"},
			want:     []string{"alpha", "beta", "delta", "epsilon", "gamma"},
		},
		{
			name:     "duplicate across scopes appears once", // US3-2: global+persona deduplicated
			global:   []string{"shared", "global-only"},
			persona:  []string{"shared", "persona-only"},
			pipeline: []string{"shared", "pipeline-only"},
			want:     []string{"global-only", "persona-only", "pipeline-only", "shared"},
		},
		{
			name:     "T023 same skill at all three scopes yields one entry",
			global:   []string{"speckit"},
			persona:  []string{"speckit"},
			pipeline: []string{"speckit"},
			want:     []string{"speckit"},
		},
		{
			name:     "large input with many duplicates",
			global:   []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			persona:  []string{"a", "c", "e", "g", "i", "k", "m", "o"},
			pipeline: []string{"b", "d", "f", "h", "j", "l", "n", "p"},
			want:     []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p"},
		},
		{
			name:   "nil persona and pipeline",
			global: []string{"only"},
			want:   []string{"only"},
		},
		{
			name:     "empty slices not nil", // US3-5: all empty → nil
			global:   []string{},
			persona:  []string{},
			pipeline: []string{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSkills(tt.global, tt.persona, tt.pipeline)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveSkills(%v, %v, %v) = %v, want %v",
					tt.global, tt.persona, tt.pipeline, got, tt.want)
			}
		})
	}
}

func TestResolveSkills_Deterministic(t *testing.T) {
	global := []string{"gamma", "alpha", "epsilon"}
	persona := []string{"beta", "alpha", "delta"}
	pipeline := []string{"epsilon", "zeta"}

	first := ResolveSkills(global, persona, pipeline)
	second := ResolveSkills(global, persona, pipeline)

	if !reflect.DeepEqual(first, second) {
		t.Errorf("non-deterministic output:\n  first:  %v\n  second: %v", first, second)
	}
}

func TestResolveSkills_LargeDuplicates(t *testing.T) {
	// Generate 100 skills with heavy overlap across scopes.
	var global, persona, pipeline []string
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("skill-%03d", i)
		global = append(global, name)
		if i%2 == 0 {
			persona = append(persona, name)
		}
		if i%3 == 0 {
			pipeline = append(pipeline, name)
		}
	}

	got := ResolveSkills(global, persona, pipeline)

	if len(got) != 100 {
		t.Fatalf("expected 100 unique skills, got %d", len(got))
	}

	// Verify sorted order.
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("not sorted at index %d: %q < %q", i, got[i], got[i-1])
		}
	}
}
