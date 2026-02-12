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
		{Name: "--output text", Description: "Plain text output"},
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

	// Print the Wave logo before the form.
	fmt.Println(WaveLogo())

	// Step 1: Pipeline selection
	options := buildPipelineOptions(pipelines)
	var selectedPipeline string
	selectField := huh.NewSelect[string]().
		Title("Select pipeline").
		Options(options...).
		Value(&selectedPipeline)

	form := huh.NewForm(huh.NewGroup(selectField)).
		WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
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

// runInputAndFlags runs the input prompt, flag selection, and confirmation as a single form.
func runInputAndFlags(selected PipelineInfo) (*Selection, error) {
	var input string
	var selectedFlags []string
	var confirmed bool

	flags := DefaultFlags()
	flagOptions := buildFlagOptions(flags)

	inputField := huh.NewInput().
		Title("Input (optional)").
		Placeholder(selected.InputExample).
		Value(&input)

	multiSelect := huh.NewMultiSelect[string]().
		Title("Options").
		Options(flagOptions...).
		Value(&selectedFlags)

	// Build command string dynamically for confirmation.
	// We use a Note to preview the composed command, then confirm.
	form := huh.NewForm(
		huh.NewGroup(inputField, multiSelect),
	).WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Confirmation step with the composed command.
	cmdStr := ComposeCommand(selected.Name, input, selectedFlags)
	confirm := huh.NewConfirm().
		Title(cmdStr).
		Description("Run this command?").
		Affirmative("Run").
		Negative("Cancel").
		Value(&confirmed)

	confirmForm := huh.NewForm(huh.NewGroup(confirm)).
		WithTheme(WaveTheme())

	if err := confirmForm.Run(); err != nil {
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
