package onboarding

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// PrintInitSuccess writes the cold-start success banner for `wave init`.
func PrintInitSuccess(out io.Writer, outputPath string, assets *AssetSet) {
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
	fmt.Fprintf(out, "  Created:\n")
	fmt.Fprintf(out, "    %-24s Main manifest\n", outputPath)
	fmt.Fprintf(out, "    .agents/personas/          %d persona archetypes\n", len(assets.Personas))
	fmt.Fprintf(out, "    .agents/pipelines/         %d pipelines\n", len(assets.Pipelines))
	fmt.Fprintf(out, "    .agents/contracts/         %d JSON schema validators\n", len(assets.Contracts))
	fmt.Fprintf(out, "    .agents/prompts/           %d prompt templates\n", len(assets.Prompts))
	fmt.Fprintf(out, "    .agents/workspaces/        Ephemeral workspace root\n")
	fmt.Fprintf(out, "    .agents/traces/            Audit log directory\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Pipelines: %s\n", strings.Join(pipelineNames, ", "))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "    2. Run 'wave run ops-hello-world \"test\"' to verify setup\n")
	fmt.Fprintf(out, "    3. Run 'wave run plan-task \"your feature\"' to plan a task\n")
	fmt.Fprintf(out, "\n")
}

// PrintMergeSuccess writes the success banner for `wave init --merge`.
func PrintMergeSuccess(out io.Writer, outputPath string) {
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ╦ ╦╔═╗╦  ╦╔═╗\n")
	fmt.Fprintf(out, "  ║║║╠═╣╚╗╔╝║╣ \n")
	fmt.Fprintf(out, "  ╚╩╝╩ ╩ ╚╝ ╚═╝\n")
	fmt.Fprintf(out, "  Multi-Agent Pipeline Orchestrator\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Configuration merged successfully!\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Updated:\n")
	fmt.Fprintf(out, "    %s       Preserved your settings\n", outputPath)
	fmt.Fprintf(out, "    Added missing default adapters and personas\n")
	fmt.Fprintf(out, "    Created missing .agents/ directories and files\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave migrate up' to apply pending migrations\n")
	fmt.Fprintf(out, "    2. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "\n")
}

// SuggestFirstRun prints a suggestion for what to run after init based on the
// detected flavour.
func SuggestFirstRun(w io.Writer, flavour *FlavourInfo) {
	if flavour == nil || flavour.SourceGlob == "" {
		fmt.Fprintf(w, "  Suggestion: Run 'wave run ops-bootstrap' to scaffold your project\n")
		return
	}

	fmt.Fprintf(w, "  Suggestion: Run 'wave run audit-architecture' to analyze your codebase\n")
}
