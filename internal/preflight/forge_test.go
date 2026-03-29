package preflight

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/forge"
)

func TestCheckForgeSteps_LocalForge_DetectsAllowedTools(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "create-pr",
			PersonaTools: []string{"Read", "Write", "Bash(gh pr create *)"},
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr == nil {
		t.Fatal("expected ForgeError, got nil")
	}
	if len(ferr.Steps) != 1 {
		t.Fatalf("expected 1 failed step, got %d", len(ferr.Steps))
	}
	if ferr.Steps[0].StepID != "create-pr" {
		t.Errorf("step ID = %q, want %q", ferr.Steps[0].StepID, "create-pr")
	}
	if ferr.Steps[0].Tool != "gh" {
		t.Errorf("tool = %q, want %q", ferr.Steps[0].Tool, "gh")
	}
	if !strings.Contains(ferr.Error(), "create-pr") {
		t.Errorf("error message should mention step ID, got: %s", ferr.Error())
	}
}

func TestCheckForgeSteps_LocalForge_DetectsGlab(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "merge-request",
			PersonaTools: []string{"Bash(glab mr create)"},
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr == nil {
		t.Fatal("expected ForgeError, got nil")
	}
	if ferr.Steps[0].Tool != "glab" {
		t.Errorf("tool = %q, want %q", ferr.Steps[0].Tool, "glab")
	}
}

func TestCheckForgeSteps_LocalForge_DetectsTea(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "open-issue",
			PersonaTools: []string{"Bash(tea issues create *)"},
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr == nil {
		t.Fatal("expected ForgeError, got nil")
	}
	if ferr.Steps[0].Tool != "tea" {
		t.Errorf("tool = %q, want %q", ferr.Steps[0].Tool, "tea")
	}
}

func TestCheckForgeSteps_LocalForge_DetectsPromptTemplate(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "publish",
			PromptSource: "Run {{ forge.cli_tool }} {{ forge.pr_command }} create to publish the PR.",
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr == nil {
		t.Fatal("expected ForgeError, got nil")
	}
	if !strings.Contains(ferr.Steps[0].Reason, "{{ forge.cli_tool }}") {
		t.Errorf("reason should mention template var, got: %s", ferr.Steps[0].Reason)
	}
}

func TestCheckForgeSteps_LocalForge_NoForgeDeps(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "analyze",
			PersonaTools: []string{"Read", "Glob", "Grep"},
			PromptSource: "Analyze the codebase for quality issues.",
		},
		{
			StepID:       "implement",
			PersonaTools: []string{"Read", "Write", "Bash(go test *)"},
			PromptSource: "Implement the changes.",
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr != nil {
		t.Fatalf("expected nil, got: %s", ferr.Error())
	}
}

func TestCheckForgeSteps_GitHubForge_SkipsCheck(t *testing.T) {
	info := forge.ForgeInfo{
		Type:    forge.ForgeGitHub,
		CLITool: "gh",
	}
	steps := []ForgeStepInput{
		{
			StepID:       "create-pr",
			PersonaTools: []string{"Bash(gh pr create *)"},
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr != nil {
		t.Fatalf("expected nil for GitHub forge, got: %s", ferr.Error())
	}
}

func TestCheckForgeSteps_MultipleFailedSteps(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}
	steps := []ForgeStepInput{
		{
			StepID:       "fetch-issue",
			PersonaTools: []string{"Bash(gh issue view *)"},
		},
		{
			StepID:       "create-pr",
			PromptSource: "Use {{ forge.cli_tool }} to create the PR.",
		},
	}

	ferr := CheckForgeSteps(info, steps)
	if ferr == nil {
		t.Fatal("expected ForgeError, got nil")
	}
	if len(ferr.Steps) != 2 {
		t.Fatalf("expected 2 failed steps, got %d", len(ferr.Steps))
	}
}

func TestCheckForgePipelineName_LocalForge_GHPrefix(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}

	ferr := CheckForgePipelineName(info, "gh-implement")
	if ferr == nil {
		t.Fatal("expected ForgeError for gh- prefix, got nil")
	}
	if !strings.Contains(ferr.Error(), "gh-implement") {
		t.Errorf("error should mention pipeline name, got: %s", ferr.Error())
	}
}

func TestCheckForgePipelineName_LocalForge_GLPrefix(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}

	ferr := CheckForgePipelineName(info, "gl-merge-request")
	if ferr == nil {
		t.Fatal("expected ForgeError for gl- prefix, got nil")
	}
}

func TestCheckForgePipelineName_LocalForge_BBPrefix(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}

	ferr := CheckForgePipelineName(info, "bb-deploy")
	if ferr == nil {
		t.Fatal("expected ForgeError for bb- prefix, got nil")
	}
}

func TestCheckForgePipelineName_LocalForge_GTPrefix(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}

	ferr := CheckForgePipelineName(info, "gt-pr")
	if ferr == nil {
		t.Fatal("expected ForgeError for gt- prefix, got nil")
	}
}

func TestCheckForgePipelineName_LocalForge_NoPrefix(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeLocal}

	ferr := CheckForgePipelineName(info, "impl-issue")
	if ferr != nil {
		t.Fatalf("expected nil for non-forge pipeline name, got: %s", ferr.Error())
	}
}

func TestCheckForgePipelineName_GitHubForge_SkipsCheck(t *testing.T) {
	info := forge.ForgeInfo{Type: forge.ForgeGitHub}

	ferr := CheckForgePipelineName(info, "gh-implement")
	if ferr != nil {
		t.Fatalf("expected nil for GitHub forge, got: %s", ferr.Error())
	}
}

func TestDetectForgeToolInAllowedTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    []string
		wantTool string
		wantOK   bool
	}{
		{name: "gh with args", tools: []string{"Bash(gh pr create *)"}, wantTool: "gh", wantOK: true},
		{name: "gh bare", tools: []string{"Bash(gh)"}, wantTool: "gh", wantOK: true},
		{name: "glab with args", tools: []string{"Bash(glab mr create)"}, wantTool: "glab", wantOK: true},
		{name: "tea with args", tools: []string{"Bash(tea issues list *)"}, wantTool: "tea", wantOK: true},
		{name: "bb with args", tools: []string{"Bash(bb pr list)"}, wantTool: "bb", wantOK: true},
		{name: "no forge tools", tools: []string{"Read", "Write", "Bash(go test *)"}, wantTool: "", wantOK: false},
		{name: "empty tools", tools: nil, wantTool: "", wantOK: false},
		{name: "gh in mixed tools", tools: []string{"Read", "Bash(go test *)", "Bash(gh issue list *)"}, wantTool: "gh", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := detectForgeToolInAllowedTools(tt.tools)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tool != tt.wantTool {
				t.Errorf("tool = %q, want %q", tool, tt.wantTool)
			}
		})
	}
}

func TestDetectForgeTemplateVar(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{name: "cli_tool var", prompt: "Use {{ forge.cli_tool }} to create PR", want: "{{ forge.cli_tool }}"},
		{name: "pr_command var", prompt: "Run gh {{ forge.pr_command }} create", want: "{{ forge.pr_command }}"},
		{name: "both vars", prompt: "{{ forge.cli_tool }} {{ forge.pr_command }}", want: "{{ forge.cli_tool }}"},
		{name: "no vars", prompt: "Just a normal prompt", want: ""},
		{name: "empty prompt", prompt: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectForgeTemplateVar(tt.prompt)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
