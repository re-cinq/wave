package onboarding

import (
	"bufio"
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/timeouts"
	"github.com/recinq/wave/internal/tui"
	"os"
	"os/exec"
	"strings"
)

// skillInstallTimeout is the maximum duration for skill installation operations.
// Configured via runtime.timeouts.skill_install_seconds in wave.yaml.
var skillInstallTimeout = timeouts.SkillInstall

// lookPathFunc is a function type for looking up executables on PATH.
// Defaults to exec.LookPath but can be overridden for testing.
type lookPathFunc func(string) (string, error)

// commandRunner is a function type for running external commands and capturing stdout.
// Defaults to running via exec.CommandContext but can be overridden for testing.
type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// skillSearchResult represents a single tessl search result for display in the wizard.
type skillSearchResult struct {
	Name        string
	Description string
}

// EcosystemDef defines a skill ecosystem that can be selected during onboarding.
type EcosystemDef struct {
	Name        string              // Display name shown in the selection form
	Value       string              // Value used in form selection
	Prefix      string              // SourceRouter prefix for installation
	Dep         skill.CLIDependency // CLI dependency required by this ecosystem
	InstallAll  bool                // true if the ecosystem installs all skills at once
	Description string              // Human-readable description
}

// ecosystems defines the available skill ecosystems for the onboarding wizard.
var ecosystems = []EcosystemDef{
	{
		Name:        "Tessl",
		Value:       "tessl",
		Prefix:      "tessl",
		Dep:         skill.CLIDependency{Binary: "tessl", Instructions: "npm i -g @tessl/cli"},
		InstallAll:  false,
		Description: "Browse and select individual skills from the Tessl registry",
	},
	{
		Name:        "BMAD",
		Value:       "bmad",
		Prefix:      "bmad",
		Dep:         skill.CLIDependency{Binary: "npx", Instructions: "npm i -g npx (comes with npm)"},
		InstallAll:  true,
		Description: "Install all BMAD method skills for Claude Code",
	},
	{
		Name:        "OpenSpec",
		Value:       "openspec",
		Prefix:      "openspec",
		Dep:         skill.CLIDependency{Binary: "openspec", Instructions: "npm i -g @openspec/cli"},
		InstallAll:  true,
		Description: "Install all OpenSpec skills",
	},
	{
		Name:        "Spec-Kit",
		Value:       "speckit",
		Prefix:      "speckit",
		Dep:         skill.CLIDependency{Binary: "specify", Instructions: "npm i -g @speckit/cli"},
		InstallAll:  true,
		Description: "Install all Spec-Kit skills",
	},
}

// SkillSelectionStep handles ecosystem selection and skill installation during onboarding.
type SkillSelectionStep struct {
	LookPath   lookPathFunc
	RunCommand commandRunner
}

// Name returns the display name for this wizard step.
func (s *SkillSelectionStep) Name() string { return "Skill Selection" }

// Run executes the skill selection wizard step.
func (s *SkillSelectionStep) Run(cfg *WizardConfig) (*StepResult, error) {
	// Non-interactive: preserve existing skills on reconfigure, empty otherwise (FR-006)
	if !cfg.Interactive {
		existing := []string{}
		if cfg.Reconfigure && cfg.Existing != nil {
			existing = cfg.Existing.Skills
		}
		return &StepResult{
			Data: map[string]interface{}{
				"skills": existing,
			},
		}, nil
	}

	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	runCmd := s.RunCommand
	if runCmd == nil {
		runCmd = defaultCommandRunner
	}

	// Show reconfigure context if applicable (FR-009)
	if cfg.Reconfigure && cfg.Existing != nil && len(cfg.Existing.Skills) > 0 {
		fmt.Fprintf(os.Stderr, "\n  Currently installed skills: %s\n\n", strings.Join(cfg.Existing.Skills, ", "))
	}

	// Ecosystem selection form (FR-001)
	selectedEcosystem, err := s.promptEcosystemSelection()
	if err != nil {
		return nil, err
	}

	// Skip selected — return empty skills (FR-001)
	if selectedEcosystem == "skip" {
		return &StepResult{
			Data: map[string]interface{}{
				"skills": []string{},
			},
		}, nil
	}

	// Find the selected ecosystem definition
	eco := findEcosystem(selectedEcosystem)
	if eco == nil {
		return nil, fmt.Errorf("unknown ecosystem: %s", selectedEcosystem)
	}

	// Check CLI dependency (FR-007)
	if _, lookErr := lookPath(eco.Dep.Binary); lookErr != nil {
		shouldSkip, handleErr := s.handleMissingCLI(eco)
		if handleErr != nil {
			return nil, handleErr
		}
		if shouldSkip {
			return &StepResult{
				Data: map[string]interface{}{
					"skills": []string{},
				},
			}, nil
		}
		// User saw instructions — check again
		if _, retryErr := lookPath(eco.Dep.Binary); retryErr != nil {
			fmt.Fprintf(os.Stderr, "  CLI still not found. Skipping skill installation.\n\n")
			return &StepResult{
				Data: map[string]interface{}{
					"skills": []string{},
				},
			}, nil
		}
	}

	// Ensure .wave/skills/ directory exists
	skillsDir := ".wave/skills"
	if cfg.WaveDir != "" {
		skillsDir = cfg.WaveDir + "/skills"
	}
	if mkErr := os.MkdirAll(skillsDir, 0755); mkErr != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", mkErr)
	}

	// Create store and router for installation
	store := skill.NewDirectoryStore(skill.SkillSource{Root: skillsDir, Precedence: 1})
	router := skill.NewDefaultRouter(".")
	ctx, cancel := context.WithTimeout(context.Background(), skillInstallTimeout)
	defer cancel()

	var installedSkills []string

	if eco.InstallAll {
		// Install-all ecosystems: confirm then install (FR-002)
		installed, installErr := s.handleInstallAllEcosystem(ctx, eco, router, store)
		if installErr != nil {
			return nil, installErr
		}
		installedSkills = installed
	} else {
		// Tessl: search, browse, multi-select, then install (FR-002)
		installed, installErr := s.handleTesslEcosystem(ctx, eco, router, store, runCmd)
		if installErr != nil {
			return nil, installErr
		}
		installedSkills = installed
	}

	// Merge with existing skills on reconfigure (deduplicated)
	if cfg.Reconfigure && cfg.Existing != nil && len(cfg.Existing.Skills) > 0 {
		installedSkills = mergeSkills(cfg.Existing.Skills, installedSkills)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"skills": installedSkills,
		},
	}, nil
}

// promptEcosystemSelection shows the ecosystem selection form.
func (s *SkillSelectionStep) promptEcosystemSelection() (string, error) {
	var selected string

	var options []huh.Option[string]
	for _, eco := range ecosystems {
		label := fmt.Sprintf("%-12s %s", eco.Name, eco.Description)
		options = append(options, huh.NewOption(label, eco.Value))
	}
	options = append(options, huh.NewOption("Skip         Skip skill installation", "skip"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a skill ecosystem").
				Options(options...).
				Value(&selected),
		).Title("Step 6 of 8 — Skill Selection").
			Description("Choose a skill ecosystem to install skills from, or skip."),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return "", fmt.Errorf("wizard cancelled by user")
		}
		return "", err
	}

	return selected, nil
}

// handleMissingCLI presents options when a required CLI tool is not found.
// Returns true if the user chooses to skip.
func (s *SkillSelectionStep) handleMissingCLI(eco *EcosystemDef) (bool, error) {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("The %s CLI (%s) was not found on PATH", eco.Name, eco.Dep.Binary)).
				Options(
					huh.NewOption("Skip skill installation", "skip"),
					huh.NewOption("Show install instructions", "instructions"),
				).
				Value(&choice),
		),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return false, fmt.Errorf("wizard cancelled by user")
		}
		return false, err
	}

	if choice == "skip" {
		return true, nil
	}

	// Show install instructions
	fmt.Fprintf(os.Stderr, "\n  Install %s with:\n    %s\n\n", eco.Dep.Binary, eco.Dep.Instructions)
	fmt.Fprintf(os.Stderr, "  After installing, the wizard will check again.\n\n")

	// Wait for user acknowledgment
	var proceed bool
	ackForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Continue after installing?").
				Value(&proceed),
		),
	).WithTheme(tui.WaveTheme())

	if err := ackForm.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return false, fmt.Errorf("wizard cancelled by user")
		}
		return false, err
	}

	if !proceed {
		return true, nil
	}

	return false, nil
}

// handleTesslEcosystem handles the tessl ecosystem: search, browse, select, install.
func (s *SkillSelectionStep) handleTesslEcosystem(
	ctx context.Context,
	eco *EcosystemDef,
	router *skill.SourceRouter,
	store skill.Store,
	runCmd commandRunner,
) ([]string, error) {
	// Search for available skills
	fmt.Fprintf(os.Stderr, "  Searching tessl registry...\n")
	output, err := runCmd(ctx, "tessl", "search", "")
	if err != nil {
		// Network failure — offer skip or retry (edge case)
		return s.handleSearchFailure(ctx, eco, router, store, runCmd, err)
	}

	results := parseTesslOutput(string(output))
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "  No skills found in the tessl registry.\n\n")
		return []string{}, nil
	}

	// Build multi-select options
	var options []huh.Option[string]
	for _, r := range results {
		label := r.Name
		if r.Description != "" {
			label = fmt.Sprintf("%-30s %s", r.Name, r.Description)
		}
		options = append(options, huh.NewOption(label, r.Name))
	}

	var selectedSkills []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select skills to install").
				Options(options...).
				Value(&selectedSkills).
				Height(12),
		).Description("Use space to toggle, enter to confirm. Type to filter."),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, fmt.Errorf("wizard cancelled by user")
		}
		return nil, err
	}

	if len(selectedSkills) == 0 {
		return []string{}, nil
	}

	// Install each selected skill (FR-003, FR-004)
	var installed []string
	for _, name := range selectedSkills {
		fmt.Fprintf(os.Stderr, "  Installing %s...\n", name)
		source := eco.Prefix + ":" + name
		result, installErr := router.Install(ctx, source, store)
		if installErr != nil {
			// FR-008: report failure but continue with remaining skills
			fmt.Fprintf(os.Stderr, "  Failed to install %s: %v\n", name, installErr)
			continue
		}
		for _, sk := range result.Skills {
			installed = append(installed, sk.Name)
			fmt.Fprintf(os.Stderr, "  Installed %s\n", sk.Name)
		}
		for _, warn := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  Warning: %s\n", warn)
		}
	}

	return installed, nil
}

// handleSearchFailure presents skip/retry options when tessl search fails.
func (s *SkillSelectionStep) handleSearchFailure(
	ctx context.Context,
	eco *EcosystemDef,
	router *skill.SourceRouter,
	store skill.Store,
	runCmd commandRunner,
	searchErr error,
) ([]string, error) {
	fmt.Fprintf(os.Stderr, "  Failed to search tessl registry: %v\n\n", searchErr)

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Registry search failed").
				Options(
					huh.NewOption("Skip skill installation", "skip"),
					huh.NewOption("Retry search", "retry"),
				).
				Value(&choice),
		),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, fmt.Errorf("wizard cancelled by user")
		}
		return nil, err
	}

	if choice == "skip" {
		return []string{}, nil
	}

	// Retry the search
	fmt.Fprintf(os.Stderr, "  Retrying tessl search...\n")
	output, err := runCmd(ctx, "tessl", "search", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Search failed again: %v. Skipping skill installation.\n\n", err)
		return []string{}, nil
	}

	results := parseTesslOutput(string(output))
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "  No skills found in the tessl registry.\n\n")
		return []string{}, nil
	}

	// Show multi-select for retry results
	var options []huh.Option[string]
	for _, r := range results {
		label := r.Name
		if r.Description != "" {
			label = fmt.Sprintf("%-30s %s", r.Name, r.Description)
		}
		options = append(options, huh.NewOption(label, r.Name))
	}

	var selectedSkills []string
	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select skills to install").
				Options(options...).
				Value(&selectedSkills).
				Height(12),
		).Description("Use space to toggle, enter to confirm. Type to filter."),
	).WithTheme(tui.WaveTheme())

	if err := selectForm.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, fmt.Errorf("wizard cancelled by user")
		}
		return nil, err
	}

	if len(selectedSkills) == 0 {
		return []string{}, nil
	}

	var installed []string
	for _, name := range selectedSkills {
		fmt.Fprintf(os.Stderr, "  Installing %s...\n", name)
		source := eco.Prefix + ":" + name
		result, installErr := router.Install(ctx, source, store)
		if installErr != nil {
			fmt.Fprintf(os.Stderr, "  Failed to install %s: %v\n", name, installErr)
			continue
		}
		for _, sk := range result.Skills {
			installed = append(installed, sk.Name)
			fmt.Fprintf(os.Stderr, "  Installed %s\n", sk.Name)
		}
		for _, warn := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  Warning: %s\n", warn)
		}
	}

	return installed, nil
}

// handleInstallAllEcosystem handles install-all ecosystems (BMAD, OpenSpec, Spec-Kit).
func (s *SkillSelectionStep) handleInstallAllEcosystem(
	ctx context.Context,
	eco *EcosystemDef,
	router *skill.SourceRouter,
	store skill.Store,
) ([]string, error) {
	// Confirmation prompt (FR-002)
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Install all %s skills?", eco.Name)).
				Description(eco.Description).
				Value(&confirmed),
		),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, fmt.Errorf("wizard cancelled by user")
		}
		return nil, err
	}

	if !confirmed {
		return []string{}, nil
	}

	// Install all skills for this ecosystem (FR-003, FR-004)
	fmt.Fprintf(os.Stderr, "  Installing %s skills...\n", eco.Name)
	source := eco.Prefix + ":"
	result, err := router.Install(ctx, source, store)
	if err != nil {
		// FR-008: report failure gracefully
		fmt.Fprintf(os.Stderr, "  Failed to install %s skills: %v\n", eco.Name, err)
		return []string{}, nil
	}

	var installed []string
	for _, sk := range result.Skills {
		installed = append(installed, sk.Name)
		fmt.Fprintf(os.Stderr, "  Installed %s\n", sk.Name)
	}
	for _, warn := range result.Warnings {
		fmt.Fprintf(os.Stderr, "  Warning: %s\n", warn)
	}

	return installed, nil
}

// parseTesslOutput parses the output from `tessl search` into skill search results.
// Adapted from cmd/wave/commands/skills.go:parseTesslSearchOutput for local types.
func parseTesslOutput(output string) []skillSearchResult {
	var results []skillSearchResult
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Parse tab-separated or space-separated output: name [rating] description
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		result := skillSearchResult{
			Name: fields[0],
		}
		// If the second field looks like a rating (e.g. "4.5", "★★★"), skip it
		if len(fields) >= 3 {
			result.Description = strings.Join(fields[2:], " ")
		} else {
			result.Description = strings.Join(fields[1:], " ")
		}
		results = append(results, result)
	}
	return results
}

// findEcosystem looks up an ecosystem definition by its value identifier.
func findEcosystem(value string) *EcosystemDef {
	for i := range ecosystems {
		if ecosystems[i].Value == value {
			return &ecosystems[i]
		}
	}
	return nil
}

// mergeSkills combines two skill name slices, deduplicating entries.
func mergeSkills(existing, newSkills []string) []string {
	seen := make(map[string]bool, len(existing)+len(newSkills))
	var merged []string

	for _, name := range existing {
		if !seen[name] {
			seen[name] = true
			merged = append(merged, name)
		}
	}
	for _, name := range newSkills {
		if !seen[name] {
			seen[name] = true
			merged = append(merged, name)
		}
	}

	return merged
}

// defaultCommandRunner executes a command and returns its stdout output.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", name, err)
	}
	return output, nil
}
