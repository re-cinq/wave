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
			block := BuildRecoveryBlock(tt.pipelineName, tt.input, tt.stepID, tt.runID, "", tt.errClass, nil)

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
	block := BuildRecoveryBlock("feature", "", "implement", "feature-abc123", "", ClassRuntimeError, nil)

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
	block := BuildRecoveryBlock("feature", "it's a test & more", "implement", "feature-abc123", "", ClassRuntimeError, nil)

	for _, hint := range block.Hints {
		if hint.Type == HintResume {
			expected := "wave run feature --input 'it'\\''s a test & more' --from-step implement"
			if hint.Command != expected {
				t.Errorf("resume command = %q, want %q", hint.Command, expected)
			}
		}
	}
}

func TestBuildRecoveryBlock_ForceLabel(t *testing.T) {
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", "", ClassContractValidation, nil)

	for _, hint := range block.Hints {
		if hint.Type == HintForce {
			if hint.Label != "Resume and skip validation checks" {
				t.Errorf("force hint label = %q, want %q", hint.Label, "Resume and skip validation checks")
			}
		}
	}
}

func TestBuildRecoveryBlock_CustomWorkspaceRoot(t *testing.T) {
	block := BuildRecoveryBlock("feature", "", "implement", "feature-abc", "/tmp/ws", ClassRuntimeError, nil)

	if block.WorkspacePath != "/tmp/ws/feature-abc/implement/" {
		t.Errorf("WorkspacePath = %q, want %q", block.WorkspacePath, "/tmp/ws/feature-abc/implement/")
	}
}

func TestBuildRecoveryBlock_InputFlag(t *testing.T) {
	// Input starting with "--" must use --input flag to avoid cobra misparse.
	// "--help" contains no shell metacharacters so ShellEscape returns it as-is,
	// but the --input flag prevents cobra from interpreting it as a flag.
	block := BuildRecoveryBlock("feature", "--help", "implement", "run-abc", "", ClassRuntimeError, nil)

	for _, hint := range block.Hints {
		if hint.Type == HintResume {
			expected := "wave run feature --input --help --from-step implement"
			if hint.Command != expected {
				t.Errorf("resume command = %q, want %q", hint.Command, expected)
			}
		}
	}
}

func TestBuildRecoveryBlock_EmptyStepID(t *testing.T) {
	// When stepID is unknown, resume/force/debug hints should be omitted
	block := BuildRecoveryBlock("feature", "test", "", "run-abc", "", ClassRuntimeError, nil)

	// Verify workspace path does not contain double trailing slash
	expectedPath := ".wave/workspaces/run-abc/"
	if block.WorkspacePath != expectedPath {
		t.Errorf("WorkspacePath = %q, want %q", block.WorkspacePath, expectedPath)
	}

	for _, hint := range block.Hints {
		switch hint.Type {
		case HintResume, HintForce, HintDebug:
			t.Errorf("unexpected hint type %q when stepID is empty", hint.Type)
		}
	}
	// Should still have workspace hint
	found := false
	for _, hint := range block.Hints {
		if hint.Type == HintWorkspace {
			found = true
		}
	}
	if !found {
		t.Error("expected workspace hint even with empty stepID")
	}
}

func TestBuildRecoveryBlock_PreflightNoDoubleSlash(t *testing.T) {
	// Preflight failures typically have empty stepID, verify no double slash
	tests := []struct {
		name             string
		pipelineName     string
		runID            string
		stepID           string
		workspaceRoot    string
		expectedPath     string
	}{
		{
			name:          "preflight with empty stepID and default workspace root",
			pipelineName:  "speckit-flow",
			runID:         "speckit-flow-20260223-114229-0e8a",
			stepID:        "",
			workspaceRoot: "",
			expectedPath:  ".wave/workspaces/speckit-flow-20260223-114229-0e8a/",
		},
		{
			name:          "preflight with custom workspace root",
			pipelineName:  "feature",
			runID:         "run-abc",
			stepID:        "",
			workspaceRoot: "/tmp/custom-ws",
			expectedPath:  "/tmp/custom-ws/run-abc/",
		},
		{
			name:          "normal step with stepID present",
			pipelineName:  "feature",
			runID:         "run-abc",
			stepID:        "implement",
			workspaceRoot: "",
			expectedPath:  ".wave/workspaces/run-abc/implement/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := BuildRecoveryBlock(tt.pipelineName, "", tt.stepID, tt.runID, tt.workspaceRoot, ClassPreflight, nil)

			if block.WorkspacePath != tt.expectedPath {
				t.Errorf("WorkspacePath = %q, want %q", block.WorkspacePath, tt.expectedPath)
			}

			// Verify no double slashes anywhere in the path
			if containsDoubleSlash(block.WorkspacePath) {
				t.Errorf("WorkspacePath contains double slash: %q", block.WorkspacePath)
			}
		})
	}
}

// containsDoubleSlash checks if a string contains "//"
func containsDoubleSlash(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '/' && s[i+1] == '/' {
			return true
		}
	}
	return false
}

func TestBuildRecoveryBlock_PreflightWithSkills(t *testing.T) {
	meta := &PreflightMetadata{
		MissingSkills: []string{"speckit", "testkit"},
	}
	block := BuildRecoveryBlock("speckit-flow", "test input", "", "run-abc", "", ClassPreflight, meta)

	// Should have skill install hints
	skillHintCount := 0
	for _, hint := range block.Hints {
		if hint.Type == HintType("preflight") && hint.Label == "Install missing skill" {
			skillHintCount++
			// Verify command format
			if hint.Command != "wave skill install speckit" && hint.Command != "wave skill install testkit" {
				t.Errorf("unexpected skill install command: %q", hint.Command)
			}
		}
	}
	if skillHintCount != 2 {
		t.Errorf("expected 2 skill install hints, got %d", skillHintCount)
	}

	// Should NOT have resume hints (preflight errors don't have steps yet)
	for _, hint := range block.Hints {
		if hint.Type == HintResume || hint.Type == HintForce || hint.Type == HintDebug {
			t.Errorf("unexpected hint type %q for preflight error", hint.Type)
		}
	}

	// Should still have workspace hint
	hasWorkspaceHint := false
	for _, hint := range block.Hints {
		if hint.Type == HintWorkspace {
			hasWorkspaceHint = true
		}
	}
	if !hasWorkspaceHint {
		t.Error("expected workspace hint for preflight error")
	}
}

func TestBuildRecoveryBlock_PreflightWithTools(t *testing.T) {
	meta := &PreflightMetadata{
		MissingTools: []string{"jq", "yq"},
	}
	block := BuildRecoveryBlock("feature", "", "", "run-xyz", "", ClassPreflight, meta)

	// Should have tool hints
	toolHintCount := 0
	for _, hint := range block.Hints {
		if hint.Type == HintType("preflight") && hint.Label == "Install missing tool" {
			toolHintCount++
			// Verify command contains tool name and guidance
			if hint.Command != "jq is required but not on PATH\nInstall it using your package manager or ensure it's in PATH" &&
				hint.Command != "yq is required but not on PATH\nInstall it using your package manager or ensure it's in PATH" {
				t.Errorf("unexpected tool hint command: %q", hint.Command)
			}
		}
	}
	if toolHintCount != 2 {
		t.Errorf("expected 2 tool hints, got %d", toolHintCount)
	}
}

func TestBuildRecoveryBlock_PreflightMixed(t *testing.T) {
	meta := &PreflightMetadata{
		MissingSkills: []string{"speckit"},
		MissingTools:  []string{"jq"},
	}
	block := BuildRecoveryBlock("feature", "", "", "run-abc", "", ClassPreflight, meta)

	// Should have both skill and tool hints
	hasSkillHint := false
	hasToolHint := false
	for _, hint := range block.Hints {
		if hint.Type == HintType("preflight") {
			if hint.Label == "Install missing skill" {
				hasSkillHint = true
			}
			if hint.Label == "Install missing tool" {
				hasToolHint = true
			}
		}
	}
	if !hasSkillHint {
		t.Error("expected skill install hint")
	}
	if !hasToolHint {
		t.Error("expected tool hint")
	}
}

func TestBuildRecoveryBlock_PreflightNoMetadata(t *testing.T) {
	// When ClassPreflight is used but metadata is nil, should still work gracefully
	block := BuildRecoveryBlock("feature", "", "", "run-abc", "", ClassPreflight, nil)

	// Should not have any preflight hints
	for _, hint := range block.Hints {
		if hint.Type == HintType("preflight") {
			t.Errorf("unexpected preflight hint when metadata is nil: %+v", hint)
		}
	}

	// Should still have workspace hint
	hasWorkspaceHint := false
	for _, hint := range block.Hints {
		if hint.Type == HintWorkspace {
			hasWorkspaceHint = true
		}
	}
	if !hasWorkspaceHint {
		t.Error("expected workspace hint even without metadata")
	}
}
