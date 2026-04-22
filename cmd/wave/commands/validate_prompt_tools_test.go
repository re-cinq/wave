package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectPromptToolMentions covers the heuristic patterns directly so a
// regression in tool detection is caught without round-tripping through a
// pipeline fixture.
func TestDetectPromptToolMentions(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "empty prompt",
			prompt: "",
			want:   nil,
		},
		{
			name:   "Write tool callout",
			prompt: "Use the Write tool to create the report.",
			want:   []string{"Write"},
		},
		{
			name:   "Write a phrasing",
			prompt: "Write a JSON file called review.json in the workspace.",
			want:   []string{"Write"},
		},
		{
			name:   "Write the phrasing",
			prompt: "Write the triage report to disk.",
			want:   []string{"Write"},
		},
		{
			name:   "Bash with parens",
			prompt: "Run Bash(git push origin main) to push.",
			want:   []string{"Bash"},
		},
		{
			name:   "WebFetch bare mention",
			prompt: "Use WebFetch to download the HTML.",
			want:   []string{"WebFetch"},
		},
		{
			name:   "WebSearch bare mention",
			prompt: "Use WebSearch to discover packages.",
			want:   []string{"WebSearch"},
		},
		{
			name:   "Edit tool",
			prompt: "Use the Edit tool to patch the file.",
			want:   []string{"Edit"},
		},
		{
			name:   "Read tool reference",
			prompt: "Use the Read tool to inspect the artifact.",
			want:   []string{"Read"},
		},
		{
			name:   "multiple tools",
			prompt: "First Read the file, then Write a JSON summary.",
			want:   []string{"Read", "Write"},
		},
		{
			name: "incidental verb 'read carefully' should not match Read",
			// "read carefully" — no follow word from our allowlist; no tool match
			prompt: "You should read carefully through the source before answering.",
			want:   nil,
		},
		{
			name: "Bash mentioned only as shell language should still flag",
			// We accept this as a known false-positive risk for prose mentions —
			// the prose-mention test below documents the same behaviour.
			prompt: "The script is written in Bash(4.0).",
			want:   []string{"Bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPromptToolMentions(tt.prompt)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtractAllowedToolNames verifies that parameterised tool grants such as
// `Bash(git log*)` are reduced to their bare tool name for comparison.
func TestExtractAllowedToolNames(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		want    []string
	}{
		{
			name:    "nil",
			allowed: nil,
			want:    nil,
		},
		{
			name:    "bare tools",
			allowed: []string{"Read", "Glob", "Grep"},
			want:    []string{"Glob", "Grep", "Read"},
		},
		{
			name:    "parameterised tools",
			allowed: []string{"Bash(git log*)", "Bash(git status*)", "Write(.agents/artifact.json)"},
			want:    []string{"Bash", "Write"},
		},
		{
			name:    "mixed plus duplicate",
			allowed: []string{"Read", "Bash(git log*)", "Bash"},
			want:    []string{"Bash", "Read"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAllowedToolNames(manifest.Permissions{AllowedTools: tt.allowed})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidatePromptToolPermissions runs the full per-pipeline check using
// table-driven scenarios. Each case constructs an in-memory manifest plus a
// minimal pipeline.Pipeline and asserts the (empty / non-empty) finding set.
func TestValidatePromptToolPermissions(t *testing.T) {
	craftsman := manifest.Persona{
		Adapter: "claude",
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
		},
	}
	navigator := manifest.Persona{
		Adapter: "claude",
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Read", "Glob", "Grep", "Bash(git log*)", "Bash(git status*)"},
		},
	}

	tests := []struct {
		name       string
		manifest   *manifest.Manifest
		pipeline   *pipeline.Pipeline
		wantTool   []string // expected Tool fields, sorted-by-step-then-tool
		wantStepID []string // expected step IDs in same order
	}{
		{
			name: "ok case — persona has Write",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{"craftsman": craftsman},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "greet",
					Persona: "craftsman",
					Exec: pipeline.ExecConfig{
						Type:   "prompt",
						Source: "Write a JSON file called review.json in the workspace root.",
					},
				}},
			},
			wantTool:   nil,
			wantStepID: nil,
		},
		{
			name: "missing case — navigator + Write prompt",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{"navigator": navigator},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "review",
					Persona: "navigator",
					Exec: pipeline.ExecConfig{
						Type:   "prompt",
						Source: "Write a JSON file called review.json in the workspace root.",
					},
				}},
			},
			wantTool:   []string{"Write"},
			wantStepID: []string{"review"},
		},
		{
			name: "prose mention — Bash(4.0) in unrelated context still flags (documented FP risk)",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{"navigator": navigator},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "explain",
					Persona: "navigator",
					Exec: pipeline.ExecConfig{
						Type: "prompt",
						// "Bash(4.0)" is a prose mention of bash version, not a
						// tool invocation, but it lexically looks identical to
						// Bash(git push). The validator surfaces it; the user
						// either rewrites the prompt or runs with the warn
						// flag. Note: navigator IS granted scoped Bash tools
						// (`Bash(git log*)`, `Bash(git status*)`), so the bare
						// "Bash" base name is in its allowed set and this
						// specific case does NOT flag — see next case for the
						// flagging variant.
						Source: "The script is written in Bash(4.0).",
					},
				}},
			},
			wantTool:   nil,
			wantStepID: nil,
		},
		{
			name: "prose mention with WebFetch — flags despite no real intent",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{"navigator": navigator},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "summarise",
					Persona: "navigator",
					Exec: pipeline.ExecConfig{
						Type: "prompt",
						// Documenting the false-positive risk: any mention of
						// WebFetch in prose, even in an aside, will flag.
						Source: "Note: do NOT use WebFetch here — the codebase is mounted offline.",
					},
				}},
			},
			wantTool:   []string{"WebFetch"},
			wantStepID: []string{"summarise"},
		},
		{
			name: "composition step — gate with no persona is skipped",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:   "gate1",
					Gate: &pipeline.GateConfig{Type: "approval"},
					Exec: pipeline.ExecConfig{
						Type:   "prompt",
						Source: "Write the approval log.",
					},
				}},
			},
			wantTool:   nil,
			wantStepID: nil,
		},
		{
			name: "non-prompt exec type — slash_command is skipped",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{"navigator": navigator},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "slash",
					Persona: "navigator",
					Exec: pipeline.ExecConfig{
						Type:   "slash_command",
						Source: "Write a file please.",
					},
				}},
			},
			wantTool:   nil,
			wantStepID: nil,
		},
		{
			name: "persona missing in manifest — silent skip (reported elsewhere)",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "ghost",
					Persona: "ghost-persona",
					Exec: pipeline.ExecConfig{
						Type:   "prompt",
						Source: "Write the report.",
					},
				}},
			},
			wantTool:   nil,
			wantStepID: nil,
		},
		{
			name: "forge template persona resolves",
			manifest: &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"github-analyst": {
						Adapter: "claude",
						Permissions: manifest.Permissions{
							AllowedTools: []string{"Read", "Bash(gh issue view*)"},
						},
					},
				},
			},
			pipeline: &pipeline.Pipeline{
				Steps: []pipeline.Step{{
					ID:      "inventory",
					Persona: "{{ forge.type }}-analyst",
					Exec: pipeline.ExecConfig{
						Type:   "prompt",
						Source: "Use the Write tool to write the inventory.",
					},
				}},
			},
			wantTool:   []string{"Write"},
			wantStepID: []string{"inventory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := validatePromptToolPermissions("test-pipeline", tt.pipeline, tt.manifest)
			require.Len(t, findings, len(tt.wantTool), "unexpected finding count: %v", findings)
			for i, want := range tt.wantTool {
				assert.Equal(t, want, findings[i].Tool, "finding[%d].Tool", i)
				assert.Equal(t, tt.wantStepID[i], findings[i].StepID, "finding[%d].StepID", i)
				assert.Equal(t, "test-pipeline", findings[i].Pipeline)
				assert.NotEmpty(t, findings[i].Persona)
			}
		})
	}
}

// TestReadStepPrompt covers the source/source_path resolution with templated
// paths skipped to avoid triggering false errors during validation.
func TestReadStepPrompt(t *testing.T) {
	t.Run("inline source wins", func(t *testing.T) {
		got, err := readStepPrompt(pipeline.Step{
			Exec: pipeline.ExecConfig{Source: "hello"},
		})
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("source_path read", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "p.txt")
		require.NoError(t, os.WriteFile(p, []byte("Write a file"), 0o644))
		got, err := readStepPrompt(pipeline.Step{
			Exec: pipeline.ExecConfig{SourcePath: p},
		})
		require.NoError(t, err)
		assert.Equal(t, "Write a file", got)
	})

	t.Run("templated source_path skipped", func(t *testing.T) {
		got, err := readStepPrompt(pipeline.Step{
			Exec: pipeline.ExecConfig{SourcePath: ".agents/{{ pipeline_id }}/p.md"},
		})
		require.NoError(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("missing file returns empty", func(t *testing.T) {
		got, err := readStepPrompt(pipeline.Step{
			Exec: pipeline.ExecConfig{SourcePath: "/nonexistent/path/file.md"},
		})
		require.NoError(t, err)
		assert.Equal(t, "", got)
	})
}

// TestPromptToolWarnEnabled confirms env / flag wiring.
func TestPromptToolWarnEnabled(t *testing.T) {
	t.Setenv(PromptToolWarnEnv, "")
	assert.False(t, promptToolWarnEnabled(ValidateOptions{}))
	assert.True(t, promptToolWarnEnabled(ValidateOptions{PromptToolsWarn: true}))

	t.Setenv(PromptToolWarnEnv, "1")
	assert.True(t, promptToolWarnEnabled(ValidateOptions{}))
	t.Setenv(PromptToolWarnEnv, "true")
	assert.True(t, promptToolWarnEnabled(ValidateOptions{}))
	t.Setenv(PromptToolWarnEnv, "no")
	assert.False(t, promptToolWarnEnabled(ValidateOptions{}))
}

// TestValidatePromptToolPermissions_HardErrorByDefault end-to-ends through
// the validate command CLI to verify a hard error is produced when prompt /
// tool mismatches exist, and that --prompt-tools-warn downgrades to a warning.
func TestValidatePromptToolPermissions_HardErrorByDefault(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
runtime:
  workspace_root: .agents/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".agents/pipelines/bad.yaml", `kind: WavePipeline
metadata:
  name: bad
steps:
  - id: review
    persona: navigator
    exec:
      type: prompt
      source: |
        Write a JSON file called review.json in the workspace root.
`)

	t.Run("hard error by default", func(t *testing.T) {
		cmd := NewValidateCmd()
		cmd.SetArgs([]string{"--all"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt/tool mismatch")
	})

	t.Run("warn flag downgrades", func(t *testing.T) {
		cmd := NewValidateCmd()
		cmd.SetArgs([]string{"--all", "--prompt-tools-warn"})
		err := cmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("env var downgrades", func(t *testing.T) {
		t.Setenv(PromptToolWarnEnv, "1")
		cmd := NewValidateCmd()
		cmd.SetArgs([]string{"--all"})
		err := cmd.Execute()
		assert.NoError(t, err)
	})
}

// TestValidatePromptToolPermissions_OkPipeline verifies --all passes when the
// persona grants every tool the prompt mentions.
func TestValidatePromptToolPermissions_OkPipeline(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  craftsman:
    adapter: claude
    system_prompt_file: personas/craftsman.md
    permissions:
      allowed_tools:
        - Read
        - Write
        - Edit
        - Bash
runtime:
  workspace_root: .agents/workspaces
`)
	h.writeFile("personas/craftsman.md", "You are a craftsman.")
	h.writeFile(".agents/pipelines/ok.yaml", `kind: WavePipeline
metadata:
  name: ok
steps:
  - id: greet
    persona: craftsman
    exec:
      type: prompt
      source: |
        Write a plain text file called greeting.txt with a greeting.
`)

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--all"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

// TestValidatePromptToolPermissions_PerPipelineErrorOutput exercises the
// --pipeline path so the per-pipeline error formatting is covered.
func TestValidatePromptToolPermissions_PerPipelineErrorOutput(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
    permissions:
      allowed_tools:
        - Read
runtime:
  workspace_root: .agents/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".agents/pipelines/bad.yaml", `kind: WavePipeline
metadata:
  name: bad
steps:
  - id: review
    persona: navigator
    exec:
      type: prompt
      source: |
        Use the Write tool to dump the result.
`)
	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "bad"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "prompt/tool mismatch"),
		"expected mismatch error, got: %s", err.Error(),
	)
}

// TestValidatePipelineWithPromptTools_GracefullyHandlesMissingFile ensures the
// helper does not crash when the YAML cannot be read (the structural
// validator already reports that). We expect findings to be nil.
func TestValidatePipelineWithPromptTools_GracefullyHandlesMissingFile(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"navigator": {Adapter: "claude"},
		},
	}
	structErrs, findings := validatePipelineWithPromptTools(
		"missing-pipeline", m, forge.ForgeInfo{Type: forge.ForgeGitHub},
	)
	assert.NotEmpty(t, structErrs, "structural pass should report missing file")
	assert.Nil(t, findings)
}
