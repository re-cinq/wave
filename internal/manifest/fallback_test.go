package manifest

import (
	"testing"
)

func TestValidateFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		wantErrs int
	}{
		{
			name: "empty fallbacks is valid",
			manifest: &Manifest{
				Adapters: map[string]Adapter{"claude": {Binary: "claude"}},
				Runtime:  Runtime{},
			},
			wantErrs: 0,
		},
		{
			name: "valid fallback chain",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
					"codex":  {Binary: "codex"},
					"gemini": {Binary: "gemini"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"claude": {"codex", "gemini"},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "self-reference rejected",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"claude": {"claude"},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "unknown primary adapter",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"unknown": {"claude"},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "unknown fallback adapter",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"claude": {"nonexistent"},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "duplicate fallback rejected",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
					"codex":  {Binary: "codex"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"claude": {"codex", "codex"},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "multiple errors accumulated",
			manifest: &Manifest{
				Adapters: map[string]Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: Runtime{
					Fallbacks: map[string][]string{
						"claude": {"claude", "missing"},
					},
				},
			},
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateFallbacks(tt.manifest)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateFallbacks() returned %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}
