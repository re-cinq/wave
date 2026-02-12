package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
		// Auto-select if exactly one match — skip the select field.
		if len(pipelines) == 1 {
			fmt.Println(WaveLogo())
			return runInputAndFlags(pipelines[0])
		}
	}

	// Print the Wave logo before the form.
	fmt.Println(WaveLogo())

	// Build all fields for a single unified form.
	var selectedPipeline string
	var input string
	var selectedFlags []string

	options := buildPipelineOptions(pipelines)
	flags := DefaultFlags()
	flagOptions := buildFlagOptions(flags)

	selectField := huh.NewSelect[string]().
		Title("Select pipeline").
		Options(options...).
		Height(8).
		Value(&selectedPipeline)

	inputField := huh.NewInput().
		Title("Input (optional)").
		Value(&input)

	multiSelect := huh.NewMultiSelect[string]().
		Title("Options").
		Options(flagOptions...).
		Value(&selectedFlags)

	// All fields in one group: shift+tab navigates back to pipeline selection.
	form := huh.NewForm(
		huh.NewGroup(selectField, inputField, multiSelect),
	).WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Set placeholder dynamically (couldn't do it before knowing the selection).
	// The input field doesn't support dynamic placeholder, so this is a no-op note.

	return confirmAndReturn(selectedPipeline, input, selectedFlags)
}

// runInputAndFlags runs the input prompt, flag selection, and confirmation
// when the pipeline is already known (auto-selected via preFilter).
func runInputAndFlags(selected PipelineInfo) (*Selection, error) {
	var input string
	var selectedFlags []string

	flags := DefaultFlags()
	flagOptions := buildFlagOptions(flags)

	// Show selected pipeline as a blurred-style static line.
	pipelineLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Pipeline:")
	pipelineName := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true).Render(selected.Name)
	fmt.Printf("  %s %s\n\n", pipelineLabel, pipelineName)

	inputField := huh.NewInput().
		Title("Input (optional)").
		Placeholder(selected.InputExample).
		Value(&input)

	multiSelect := huh.NewMultiSelect[string]().
		Title("Options").
		Options(flagOptions...).
		Value(&selectedFlags)

	form := huh.NewForm(
		huh.NewGroup(inputField, multiSelect),
	).WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	return confirmAndReturn(selected.Name, input, selectedFlags)
}

// confirmAndReturn shows the composed command, asks for confirmation, and returns the selection.
func confirmAndReturn(pipeline, input string, selectedFlags []string) (*Selection, error) {
	var confirmed bool
	cmdStr := ComposeCommand(pipeline, input, selectedFlags)

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

	// Print the composed command so it stays in scrollback for debugging.
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("$")
	cmdText := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render(cmdStr)
	fmt.Printf("  %s %s\n\n", cmdStyle, cmdText)

	return &Selection{
		Pipeline: pipeline,
		Input:    input,
		Flags:    selectedFlags,
	}, nil
}

// buildPipelineOptions creates huh options from pipeline info.
func buildPipelineOptions(pipelines []PipelineInfo) []huh.Option[string] {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	options := make([]huh.Option[string], len(pipelines))
	for i, p := range pipelines {
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%-20s %s", p.Name, dimStyle.Render(p.Description))
		}
		options[i] = huh.NewOption(label, p.Name)
	}
	return options
}

// buildFlagOptions creates huh options from flags.
func buildFlagOptions(flags []Flag) []huh.Option[string] {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	options := make([]huh.Option[string], len(flags))
	for i, f := range flags {
		label := fmt.Sprintf("%-16s %s", f.Name, dimStyle.Render(f.Description))
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
