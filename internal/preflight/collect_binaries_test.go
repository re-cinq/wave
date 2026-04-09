package preflight

import (
	"sort"
	"testing"
)

func TestCollectAdapterBinaries(t *testing.T) {
	tests := []struct {
		name     string
		personas map[string]Persona
		adapters map[string]AdapterDef
		steps    []StepRef
		want     []string
	}{
		{
			name: "empty steps returns empty",
			personas: map[string]Persona{
				"nav": {Adapter: "claude"},
			},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
			},
			steps: nil,
			want:  nil,
		},
		{
			name: "persona adapter resolved",
			personas: map[string]Persona{
				"nav": {Adapter: "claude"},
			},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
			},
			steps: []StepRef{
				{Persona: "nav"},
			},
			want: []string{"claude"},
		},
		{
			name: "step adapter overrides persona",
			personas: map[string]Persona{
				"nav": {Adapter: "claude"},
			},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
				"codex":  {Binary: "codex"},
			},
			steps: []StepRef{
				{Persona: "nav", Adapter: "codex"},
			},
			want: []string{"codex"},
		},
		{
			name: "deduplicates binaries",
			personas: map[string]Persona{
				"nav": {Adapter: "claude"},
				"dev": {Adapter: "claude"},
			},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
			},
			steps: []StepRef{
				{Persona: "nav"},
				{Persona: "dev"},
			},
			want: []string{"claude"},
		},
		{
			name: "multiple different binaries",
			personas: map[string]Persona{
				"nav": {Adapter: "claude"},
				"dev": {Adapter: "codex"},
			},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
				"codex":  {Binary: "codex"},
			},
			steps: []StepRef{
				{Persona: "nav"},
				{Persona: "dev"},
			},
			want: []string{"claude", "codex"},
		},
		{
			name:     "unknown persona skipped",
			personas: map[string]Persona{},
			adapters: map[string]AdapterDef{
				"claude": {Binary: "claude"},
			},
			steps: []StepRef{
				{Persona: "unknown"},
			},
			want: nil,
		},
		{
			name: "unknown adapter skipped",
			personas: map[string]Persona{
				"nav": {Adapter: "missing"},
			},
			adapters: map[string]AdapterDef{},
			steps: []StepRef{
				{Persona: "nav"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollectAdapterBinaries(tt.personas, tt.adapters, tt.steps)
			// Sort both for deterministic comparison
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("CollectAdapterBinaries() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("CollectAdapterBinaries()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
