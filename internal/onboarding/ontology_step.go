package onboarding

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/uitheme"
)

// OntologyStep prompts for the project's telos and bounded context names.
// This is the "first touch" ontology kernel scaffolded during onboarding.
//
// The step compiles unconditionally — feature gating happens at the
// ontology Service layer. If the user supplies neither telos nor contexts
// the step records Skipped and no git hook is installed.
type OntologyStep struct{}

// Name returns the display name for this wizard step.
func (s *OntologyStep) Name() string { return "Project Ontology" }

// Run executes the ontology wizard step.
func (s *OntologyStep) Run(cfg *WizardConfig) (*StepResult, error) {
	var telos string
	var contextsRaw string

	// Pre-fill from existing manifest if reconfiguring
	if cfg.Reconfigure && cfg.Existing != nil && cfg.Existing.Ontology != nil {
		telos = cfg.Existing.Ontology.Telos
		names := make([]string, len(cfg.Existing.Ontology.Contexts))
		for i, ctx := range cfg.Existing.Ontology.Contexts {
			names[i] = ctx.Name
		}
		contextsRaw = strings.Join(names, ", ")
	}

	if cfg.Interactive {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("What is this project's purpose (telos)?").
					Value(&telos).
					Placeholder("e.g. Enable users to query accounting data conversationally").
					CharLimit(500).
					Lines(3),
				huh.NewInput().
					Title("Bounded contexts (comma-separated, optional)").
					Value(&contextsRaw).
					Placeholder("e.g. identity, conversation, analytics"),
			).Title("Step 7 of 8 — Project Ontology").
				Description("Wave works best when it understands your project's domain."),
		).WithTheme(uitheme.WaveTheme())

		if err := form.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil, fmt.Errorf("wizard cancelled by user")
			}
			return nil, err
		}
	}

	contexts := parseContextNames(contextsRaw)

	// If telos is empty and no contexts, skip (optional step)
	if strings.TrimSpace(telos) == "" && len(contexts) == 0 {
		if cfg.Interactive {
			fmt.Fprintf(os.Stderr, "  Skipping ontology — run 'wave analyze' later to generate it.\n\n")
		}
		return &StepResult{
			Skipped: true,
			Data:    map[string]interface{}{},
		}, nil
	}

	// Install git post-merge hook via the ontology package — it owns the
	// sentinel path and hook body convention.
	if err := ontology.InstallStalenessHookAt(".git/hooks"); err != nil && cfg.Interactive {
		fmt.Fprintf(os.Stderr, "  Note: could not install git post-merge hook: %v\n", err)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"telos":    strings.TrimSpace(telos),
			"contexts": contexts,
		},
	}, nil
}

// parseContextNames splits a comma-separated string into trimmed, non-empty context names.
func parseContextNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var names []string
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
