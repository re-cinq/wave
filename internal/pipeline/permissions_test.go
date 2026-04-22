package pipeline

import (
	"reflect"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// TestResolveStepPermissions covers the merge precedence:
// step.Permissions ∪ persona.Permissions ∪ adapter.DefaultPermissions.
// AllowedTools is additive (a step may add tools the persona lacks); Deny
// is also unioned, and the underlying PermissionChecker enforces deny-first
// precedence so persona-level deny rules win at runtime.
func TestResolveStepPermissions(t *testing.T) {
	tests := []struct {
		name       string
		step       *Step
		persona    *manifest.Persona
		adapterDef *manifest.Adapter
		wantAllow  []string
		wantDeny   []string
	}{
		{
			name: "empty step inherits persona permissions",
			step: &Step{ID: "scan"},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
				},
			},
			adapterDef: &manifest.Adapter{},
			wantAllow:  []string{"Read", "Glob", "Grep"},
			wantDeny:   nil,
		},
		{
			name: "step adds Write to read-only persona",
			step: &Step{
				ID: "scan",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Write", "Edit"},
				},
			},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
				},
			},
			adapterDef: &manifest.Adapter{},
			// Step entries appear first (highest precedence ordering),
			// then persona, with duplicates collapsed.
			wantAllow: []string{"Write", "Edit", "Read", "Glob", "Grep"},
			wantDeny:  nil,
		},
		{
			name: "step deny is preserved alongside persona deny",
			step: &Step{
				ID: "scan",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Bash"},
					Deny:         []string{"Bash(curl*)"},
				},
			},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read"},
					Deny:         []string{"Bash(rm *)"},
				},
			},
			adapterDef: &manifest.Adapter{},
			wantAllow:  []string{"Bash", "Read"},
			wantDeny:   []string{"Bash(curl*)", "Bash(rm *)"},
		},
		{
			name: "step adds tool that persona explicitly denies — deny still wins at check time",
			step: &Step{
				ID: "scan",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Write"},
				},
			},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
					Deny:         []string{"Write(*)"},
				},
			},
			adapterDef: &manifest.Adapter{},
			wantAllow:  []string{"Write", "Read", "Glob", "Grep"},
			// Deny survives — runtime enforcement happens in adapter.PermissionChecker
			wantDeny: []string{"Write(*)"},
		},
		{
			name: "adapter defaults fill in when persona and step are silent",
			step: &Step{ID: "scan"},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{},
			},
			adapterDef: &manifest.Adapter{
				DefaultPermissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob"},
					Deny:         []string{"Bash(sudo *)"},
				},
			},
			wantAllow: []string{"Read", "Glob"},
			wantDeny:  []string{"Bash(sudo *)"},
		},
		{
			name: "duplicate patterns across layers collapse, first wins",
			step: &Step{
				ID: "scan",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write"},
				},
			},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob"},
				},
			},
			adapterDef: &manifest.Adapter{
				DefaultPermissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Grep"},
				},
			},
			wantAllow: []string{"Read", "Write", "Glob", "Grep"},
			wantDeny:  nil,
		},
		{
			name: "nil step is safe (interactive/adhoc invocation)",
			step: nil,
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read"},
					Deny:         []string{"Bash(*)"},
				},
			},
			adapterDef: &manifest.Adapter{},
			wantAllow:  []string{"Read"},
			wantDeny:   []string{"Bash(*)"},
		},
		{
			name: "nil adapterDef is safe",
			step: &Step{
				ID: "scan",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Write"},
				},
			},
			persona: &manifest.Persona{
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read"},
				},
			},
			adapterDef: nil,
			wantAllow:  []string{"Write", "Read"},
			wantDeny:   nil,
		},
		{
			name:       "all-empty resolves to zero-value Permissions",
			step:       &Step{ID: "scan"},
			persona:    &manifest.Persona{},
			adapterDef: &manifest.Adapter{},
			wantAllow:  nil,
			wantDeny:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveStepPermissions(tc.step, tc.persona, tc.adapterDef)
			if !reflect.DeepEqual(got.AllowedTools, tc.wantAllow) {
				t.Errorf("AllowedTools mismatch\n  got:  %v\n  want: %v", got.AllowedTools, tc.wantAllow)
			}
			if !reflect.DeepEqual(got.Deny, tc.wantDeny) {
				t.Errorf("Deny mismatch\n  got:  %v\n  want: %v", got.Deny, tc.wantDeny)
			}
		})
	}
}

// TestResolveStepPermissions_DenyEnforcedAtCheckTime asserts the documented
// invariant: even when a step adds Write to a persona that denies Write(*),
// the adapter.PermissionChecker still blocks the operation. The resolver is
// not the place where deny-first precedence is enforced — it is the merge
// site that hands a unified Permissions value to the runtime checker.
func TestResolveStepPermissions_DenyEnforcedAtCheckTime(t *testing.T) {
	step := &Step{
		ID: "scan",
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Write"},
		},
	}
	persona := &manifest.Persona{
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Read"},
			Deny:         []string{"Write(*)"},
		},
	}
	resolved := ResolveStepPermissions(step, persona, &manifest.Adapter{})

	checker := adapter.NewPermissionChecker("test", resolved.AllowedTools, resolved.Deny)

	if err := checker.CheckPermission("Read", "src/main.go"); err != nil {
		t.Errorf("Read should still succeed, got: %v", err)
	}
	if err := checker.CheckPermission("Write", "src/main.go"); err == nil {
		t.Error("Write should be denied because persona's Write(*) deny rule survives the merge")
	}
}
