package onboarding

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	wsroot "github.com/recinq/wave/internal/workspace"
)

// ValidateHint returns the `wave validate` invocation for outputPath,
// adding --manifest when the manifest is not the default wave.yaml.
func ValidateHint(outputPath string) string {
	if outputPath == "" || filepath.Clean(outputPath) == "wave.yaml" {
		return "wave validate"
	}
	return fmt.Sprintf("wave validate --manifest %s", outputPath)
}

// PrintInitSuccess writes the cold-start success banner for `wave init`.
func PrintInitSuccess(out io.Writer, outputPath, workspace string, res *GreenfieldResult) {
	assets := res.Assets
	pipelineNames := make([]string, 0, len(assets.Pipelines))
	for name := range assets.Pipelines {
		pipelineNames = append(pipelineNames, strings.TrimSuffix(name, ".yaml"))
	}
	sort.Strings(pipelineNames)

	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ╦ ╦╔═╗╦  ╦╔═╗\n")
	fmt.Fprintf(out, "  ║║║╠═╣╚╗╔╝║╣ \n")
	fmt.Fprintf(out, "  ╚╩╝╩ ╩ ╚╝ ╚═╝\n")
	fmt.Fprintf(out, "  Multi-Agent Pipeline Orchestrator\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Project initialized successfully!\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Adapter: %s (model %s for every tier)\n", res.Adapter, res.Model)
	if root := filepath.Dir(outputPath); root != "." && root != "" {
		fmt.Fprintf(out, "  Project root: %s (paths below are relative to it)\n", root)
	}
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Created:\n")
	fmt.Fprintf(out, "    %-26s Main manifest\n", filepath.Base(outputPath))
	fmt.Fprintf(out, "    %-26s %d personas\n", ".agents/personas/", len(assets.PersonaConfigs))
	fmt.Fprintf(out, "    %-26s %d pipelines\n", ".agents/pipelines/", len(assets.Pipelines))
	fmt.Fprintf(out, "    %-26s %d contracts\n", ".agents/contracts/", len(assets.Contracts))
	fmt.Fprintf(out, "    %-26s %d prompt templates\n", ".agents/prompts/", len(assets.Prompts))
	if workspace != "" {
		fmt.Fprintf(out, "    %-26s Ephemeral workspace root\n", strings.TrimSuffix(workspace, "/")+"/")
	}
	fmt.Fprintf(out, "    %-26s Audit log directory\n", ".agents/traces/")
	for _, f := range res.InstructionFiles {
		desc := "Project instructions copied into every agent workspace"
		if f == "CLAUDE.md" {
			desc = "Imports AGENTS.md for Claude Code"
		}
		fmt.Fprintf(out, "    %-26s %s\n", f, desc)
	}
	if res.GitignoreUpdated {
		fmt.Fprintf(out, "    %-26s Ignores Wave runtime state\n", ".gitignore")
	}
	if res.Committed {
		fmt.Fprintf(out, "    %-26s chore: initialize wave project\n", "git commit")
	}
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Pipelines: %s\n", strings.Join(pipelineNames, ", "))
	if workspace == "" {
		fmt.Fprintf(out, "  Workspaces: %s/ (outside the checkout; set runtime.workspace_root to move them)\n", wsroot.DefaultRoot())
	}
	fmt.Fprintf(out, "\n")

	if missing := MissingProjectCommands(res.Project); len(missing) > 0 {
		fmt.Fprintf(out, "  Missing project commands: %s\n", strings.Join(missing, ", "))
		fmt.Fprintf(out, "    Pipelines such as impl-issue run project.test_command. Set them in %s\n", outputPath)
		fmt.Fprintf(out, "    (project.test_command / lint_command / build_command), re-run with\n")
		fmt.Fprintf(out, "    --test-command/--lint-command/--build-command, or run 'wave doctor --optimize'.\n")
		fmt.Fprintf(out, "\n")
	}

	if !res.Committed {
		files := append([]string{filepath.Base(outputPath), ".agents"}, res.InstructionFiles...)
		if res.GitignoreUpdated {
			files = append(files, ".gitignore")
		}
		fmt.Fprintf(out, "  Commit the new files to share the setup: git add %s\n", strings.Join(files, " "))
		fmt.Fprintf(out, "\n")
	}

	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run '%s' to check configuration\n", ValidateHint(outputPath))
	fmt.Fprintf(out, "    2. Run 'wave run ops-hello-world \"test\"' to verify setup\n")
	fmt.Fprintf(out, "    3. Run 'wave run plan-task \"your feature\"' to plan a task\n")
	fmt.Fprintf(out, "    Switch adapter or model later with 'wave init --reconfigure --adapter <claude|codex> --model <id>'.\n")
	fmt.Fprintf(out, "\n")
}

// PrintMergeSuccess writes the success banner for `wave init --merge`.
func PrintMergeSuccess(out io.Writer, outputPath string) {
	printMergeBanner(out, outputPath, "Configuration merged successfully!")
}

// PrintReconfigureSuccess writes the success banner for `wave init --reconfigure`.
func PrintReconfigureSuccess(out io.Writer, outputPath, backupPath, adapter, model string) {
	printMergeBanner(out, outputPath, "Configuration updated successfully!")
	fmt.Fprintf(out, "  Adapter: %s (model %s for every tier)\n", adapter, model)
	if backupPath != "" {
		fmt.Fprintf(out, "  Previous manifest saved to %s\n", backupPath)
	}
	fmt.Fprintf(out, "\n")
}

func printMergeBanner(out io.Writer, outputPath, headline string) {
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ╦ ╦╔═╗╦  ╦╔═╗\n")
	fmt.Fprintf(out, "  ║║║╠═╣╚╗╔╝║╣ \n")
	fmt.Fprintf(out, "  ╚╩╝╩ ╩ ╚╝ ╚═╝\n")
	fmt.Fprintf(out, "  Multi-Agent Pipeline Orchestrator\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  %s\n", headline)
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Updated:\n")
	fmt.Fprintf(out, "    %s       Preserved your settings\n", outputPath)
	fmt.Fprintf(out, "    Added missing default personas and settings\n")
	fmt.Fprintf(out, "    Created missing .agents/ directories and files\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    Run '%s' to check configuration\n", ValidateHint(outputPath))
	fmt.Fprintf(out, "\n")
}

// SuggestFirstRun prints a suggestion for what to run after init based on the
// detected flavour.
func SuggestFirstRun(w io.Writer, flavour *FlavourInfo) {
	if flavour == nil || flavour.SourceGlob == "" {
		fmt.Fprintf(w, "  Suggestion: Run 'wave meta \"<task>\"' to generate a pipeline for your project\n")
		return
	}

	fmt.Fprintf(w, "  Suggestion: Run 'wave run audit-architecture' to analyze your codebase\n")
}
