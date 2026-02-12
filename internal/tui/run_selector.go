package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// Selection holds the result of the interactive pipeline selection.
type Selection struct {
	Pipeline string
	Input    string
	Flags    []string
}

// Flag represents a toggleable CLI flag shown in the TUI.
type Flag struct {
	Name        string
	Description string
}

// DefaultFlags returns the flags presented in the interactive selector.
func DefaultFlags() []Flag {
	return []Flag{
		{Name: "--verbose", Description: "Real-time tool activity"},
		{Name: "--output json", Description: "JSON output format"},
		{Name: "--dry-run", Description: "Preview without executing"},
		{Name: "--mock", Description: "Use mock adapter"},
		{Name: "--debug", Description: "Debug logging"},
	}
}

// RunPipelineSelector launches the interactive TUI for pipeline selection.
// preFilter narrows the initial pipeline list (e.g. from a partial name argument).
// pipelinesDir is the directory to scan for pipeline YAML files.
func RunPipelineSelector(pipelinesDir, preFilter string) (*Selection, error) {
	pipelines, err := DiscoverPipelines(pipelinesDir)
	if err != nil {
		return nil, fmt.Errorf("discovering pipelines: %w", err)
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipelines found in %s", pipelinesDir)
	}

	// Filter pipelines if a pre-filter is provided.
	if preFilter != "" {
		pipelines = filterPipelines(pipelines, preFilter)
		if len(pipelines) == 0 {
			return nil, fmt.Errorf("no pipelines match %q", preFilter)
		}
		// Auto-select if exactly one match.
		if len(pipelines) == 1 {
			return runInputAndFlags(pipelines[0])
		}
	}

	// Step 1: Pipeline selection
	options := buildPipelineOptions(pipelines)
	var selectedPipeline string
	selectField := huh.NewSelect[string]().
		Title("Select pipeline").
		Options(options...).
		Value(&selectedPipeline)

	if err := huh.Run(selectField); err != nil {
		return nil, err
	}

	// Find the selected pipeline info for input example.
	var selected PipelineInfo
	for _, p := range pipelines {
		if p.Name == selectedPipeline {
			selected = p
			break
		}
	}

	return runInputAndFlags(selected)
}

// runInputAndFlags runs the input prompt, flag selection, and confirmation steps.
func runInputAndFlags(selected PipelineInfo) (*Selection, error) {
	// Step 2: Input prompt
	var input string
	inputField := huh.NewInput().
		Title("Input (optional)").
		Description("Press Enter to confirm, or leave empty for no input").
		Placeholder(selected.InputExample).
		Value(&input)

	if err := huh.Run(inputField); err != nil {
		return nil, err
	}

	// Step 3: Flag selection
	flags := DefaultFlags()
	flagOptions := buildFlagOptions(flags)
	var selectedFlags []string
	multiSelect := huh.NewMultiSelect[string]().
		Title("Options").
		Options(flagOptions...).
		Value(&selectedFlags)

	if err := huh.Run(multiSelect); err != nil {
		return nil, err
	}

	// Step 4: Confirmation
	cmdStr := ComposeCommand(selected.Name, input, selectedFlags)
	var confirmed bool
	confirm := huh.NewConfirm().
		Title(cmdStr).
		Description("Press Enter to run, Esc to cancel").
		Affirmative("Run").
		Negative("Cancel").
		Value(&confirmed)

	if err := huh.Run(confirm); err != nil {
		return nil, err
	}

	if !confirmed {
		return nil, huh.ErrUserAborted
	}

	return &Selection{
		Pipeline: selected.Name,
		Input:    input,
		Flags:    selectedFlags,
	}, nil
}

// buildPipelineOptions creates huh options from pipeline info.
func buildPipelineOptions(pipelines []PipelineInfo) []huh.Option[string] {
	options := make([]huh.Option[string], len(pipelines))
	for i, p := range pipelines {
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%-20s %s", p.Name, p.Description)
		}
		options[i] = huh.NewOption(label, p.Name)
	}
	return options
}

// buildFlagOptions creates huh options from flags.
func buildFlagOptions(flags []Flag) []huh.Option[string] {
	options := make([]huh.Option[string], len(flags))
	for i, f := range flags {
		label := fmt.Sprintf("%-16s %s", f.Name, f.Description)
		options[i] = huh.NewOption(label, f.Name)
	}
	return options
}

// filterPipelines returns pipelines whose names contain the filter string (case-insensitive).
func filterPipelines(pipelines []PipelineInfo, filter string) []PipelineInfo {
	filter = strings.ToLower(filter)
	var matched []PipelineInfo
	for _, p := range pipelines {
		if strings.Contains(strings.ToLower(p.Name), filter) {
			matched = append(matched, p)
		}
	}
	return matched
}

// ComposeCommand builds the command string shown in the confirmation step.
func ComposeCommand(pipeline, input string, flags []string) string {
	parts := []string{"wave run", pipeline}
	if input != "" {
		parts = append(parts, fmt.Sprintf("%q", input))
	}
	parts = append(parts, flags...)
	return strings.Join(parts, " ")
}
