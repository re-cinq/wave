package recovery

import "testing"

func TestBuildRecoveryBlock(t *testing.T) {
	tests := []struct {
		name           string
		pipelineName   string
		input          string
		stepID         string
		runID          string
		errClass       ErrorClass
		wantHintTypes  []HintType
		wantNoTypes    []HintType
		wantWorkspace  string
	}{
		{
			name:          "runtime error produces resume, workspace, debug",
			pipelineName:  "feature",
			input:         "add auth",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassRuntimeError,
			wantHintTypes: []HintType{HintResume, HintWorkspace, HintDebug},
			wantNoTypes:   []HintType{HintForce},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
		{
			name:          "contract validation produces resume, force, workspace (no debug)",
			pipelineName:  "feature",
			input:         "add auth",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassContractValidation,
			wantHintTypes: []HintType{HintResume, HintForce, HintWorkspace},
			wantNoTypes:   []HintType{HintDebug},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
		{
			name:          "security error produces resume, workspace only",
			pipelineName:  "feature",
			input:         "add auth",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassSecurityViolation,
			wantHintTypes: []HintType{HintResume, HintWorkspace},
			wantNoTypes:   []HintType{HintForce, HintDebug},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
		{
			name:          "unknown error produces resume, workspace, debug",
			pipelineName:  "feature",
			input:         "add auth",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassUnknown,
			wantHintTypes: []HintType{HintResume, HintWorkspace, HintDebug},
			wantNoTypes:   []HintType{HintForce},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
		{
			name:          "empty input omits input from commands",
			pipelineName:  "feature",
			input:         "",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassRuntimeError,
			wantHintTypes: []HintType{HintResume, HintWorkspace, HintDebug},
			wantNoTypes:   []HintType{HintForce},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
		{
			name:          "special characters in input are shell-escaped",
			pipelineName:  "feature",
			input:         "it's a test & more",
			stepID:        "implement",
			runID:         "feature-abc123",
			errClass:      ClassRuntimeError,
			wantHintTypes: []HintType{HintResume, HintWorkspace, HintDebug},
			wantNoTypes:   []HintType{HintForce},
			wantWorkspace: ".wave/workspaces/feature-abc123/implement/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := BuildRecoveryBlock(tt.pipelineName, tt.input, tt.stepID, tt.runID, tt.errClass)

			if block.PipelineName != tt.pipelineName {
				t.Errorf("PipelineName = %q, want %q", block.PipelineName, tt.pipelineName)
			}
			if block.StepID != tt.stepID {
				t.Errorf("StepID = %q, want %q", block.StepID, tt.stepID)
			}
			if block.WorkspacePath != tt.wantWorkspace {
				t.Errorf("WorkspacePath = %q, want %q", block.WorkspacePath, tt.wantWorkspace)
			}
			if block.ErrorClass != tt.errClass {
				t.Errorf("ErrorClass = %q, want %q", block.ErrorClass, tt.errClass)
			}

			// Check expected hint types are present
			for _, wantType := range tt.wantHintTypes {
				found := false
				for _, hint := range block.Hints {
					if hint.Type == wantType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected hint type %q not found in hints", wantType)
				}
			}

			// Check unwanted hint types are absent
			for _, noType := range tt.wantNoTypes {
				for _, hint := range block.Hints {
					if hint.Type == noType {
						t.Errorf("unexpected hint type %q found in hints", noType)
					}
				}
			}
		})
	}
}

func TestBuildRecoveryBlock_EmptyInput(t *testing.T) {
	block := BuildRecoveryBlock("feature", "", "implement", "feature-abc123", ClassRuntimeError)

	for _, hint := range block.Hints {
		if hint.Type == HintResume || hint.Type == HintDebug {
			// Resume and debug commands should not contain empty quotes
			expected := "wave run feature --from-step implement"
			if hint.Type == HintDebug {
				expected += " --debug"
			}
			if hint.Command != expected {
				t.Errorf("hint %q command = %q, want %q", hint.Type, hint.Command, expected)
			}
		}
	}
}

func TestBuildRecoveryBlock_SpecialCharsInput(t *testing.T) {
	block := BuildRecoveryBlock("feature", "it's a test & more", "implement", "feature-abc123", ClassRuntimeError)

	for _, hint := range block.Hints {
		if hint.Type == HintResume {
			expected := "wave run feature 'it'\\''s a test & more' --from-step implement"
			if hint.Command != expected {
				t.Errorf("resume command = %q, want %q", hint.Command, expected)
			}
		}
	}
}

func TestBuildRecoveryBlock_ForceLabel(t *testing.T) {
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", ClassContractValidation)

	for _, hint := range block.Hints {
		if hint.Type == HintForce {
			if hint.Label != "Resume and skip validation checks" {
				t.Errorf("force hint label = %q, want %q", hint.Label, "Resume and skip validation checks")
			}
		}
	}
}
