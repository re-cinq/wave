package pipeline

import (
	"fmt"
	"strings"

	"github.com/recinq/wave/internal/contract/schemas/shared"
)

// ValidatePipelineIOTypes checks that all type names declared on a pipeline's
// Input and PipelineOutputs resolve against the shared schema registry (or are
// the "string" sentinel). This runs at load time so misspelled type names
// fail fast, before any step executes.
//
// See docs/adr/010-pipeline-io-protocol.md.
func ValidatePipelineIOTypes(p *Pipeline) error {
	if p == nil {
		return nil
	}

	// Eagerly load the embedded shared-schema registry so that an FS read
	// failure surfaces here as a structured error rather than a stale empty
	// registry. LoadSchemas is idempotent (sync.Once) so repeated calls are
	// cheap.
	if err := shared.LoadSchemas(); err != nil {
		return fmt.Errorf("pipeline %q: shared schema registry failed to load: %w", p.Metadata.Name, err)
	}

	if t := p.Input.EffectiveType(); !shared.Exists(t) {
		return fmt.Errorf("pipeline %q: input.type %q is not a registered shared schema (known: %v, or use %q)",
			p.Metadata.Name, t, shared.Names(), shared.TypeString)
	}

	stepIDs := make(map[string]bool, len(p.Steps))
	for _, s := range p.Steps {
		stepIDs[s.ID] = true
	}

	for name, out := range p.PipelineOutputs {
		if t := out.EffectiveType(); !shared.Exists(t) {
			return fmt.Errorf("pipeline %q: pipeline_outputs[%q].type %q is not a registered shared schema (known: %v, or use %q)",
				p.Metadata.Name, name, t, shared.Names(), shared.TypeString)
		}
		if out.Step != "" && !stepIDs[out.Step] {
			return fmt.Errorf("pipeline %q: pipeline_outputs[%q].step %q is not a declared step",
				p.Metadata.Name, name, out.Step)
		}
	}

	// Validate composition steps: InputRef must set exactly one of From/Literal.
	for _, step := range p.Steps {
		if step.InputRef == nil {
			continue
		}
		ir := step.InputRef
		if ir.From == "" && ir.Literal == "" {
			return fmt.Errorf("pipeline %q step %q: input_ref must specify either 'from' or 'literal'",
				p.Metadata.Name, step.ID)
		}
		if ir.From != "" && ir.Literal != "" {
			return fmt.Errorf("pipeline %q step %q: input_ref.from and input_ref.literal are mutually exclusive",
				p.Metadata.Name, step.ID)
		}
	}

	return nil
}

// CollectWLPLoadErrors returns a list of Wave Lego Protocol (ADR-011) load-time
// violations. The returned slice is nil when the pipeline is WLP-clean. The
// loader (dag.go) treats a non-empty slice as a hard error.
//
// Violation categories (ADR-011 rules):
//
//   - Rule 5: contract `on_failure: retry` — deterministic on_failure values
//     must be fail/skip/continue/rework (+rework_step)/warn/rejected.
//     `retry` is forbidden because retries belong to the step-level retry
//     policy, not contracts. `rejected` is a terminal "design rejection"
//     outcome where the persona deliberately reports a non-actionable
//     verdict (e.g. issue already implemented).
//   - Rule 3: pipeline_outputs entries without an explicit `type:` — every
//     declared output must carry a semantic type so consumers can type-check
//     cross-pipeline wiring at load time.
//
// This function is intentionally separate from ValidatePipelineIOTypes, which
// covers shape/type-registry checks for input.type, pipeline_outputs[*].type,
// and step input_ref bindings.
func CollectWLPLoadErrors(p *Pipeline) []string {
	if p == nil {
		return nil
	}
	var warnings []string

	// Rule 3: nudge pipeline authors to declare output types explicitly.
	for name, out := range p.PipelineOutputs {
		if strings.TrimSpace(out.Type) == "" {
			warnings = append(warnings,
				fmt.Sprintf("pipeline %q: pipeline_outputs[%q] has no explicit type — defaulting to %q. Declare a type for cross-pipeline type-checking (ADR-011 rule 3).",
					p.Metadata.Name, name, shared.TypeString))
		}
	}

	// Rule 5: deterministic contract on_failure values. `retry` is deprecated.
	for _, step := range p.Steps {
		for i, c := range step.Handover.EffectiveContracts() {
			if c.OnFailure == OnFailureRetry {
				warnings = append(warnings,
					fmt.Sprintf("pipeline %q step %q contract[%d] (%s): on_failure=%q is deprecated — use step-level retry.max_attempts for retries, or a deterministic contract outcome (fail, skip, continue, rework, warn, rejected). See ADR-011 rule 5.",
						p.Metadata.Name, step.ID, i, c.Type, OnFailureRetry))
			}
		}
	}

	return warnings
}
