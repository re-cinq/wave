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

// TypedWiringCheck verifies cross-pipeline type compatibility. Given a parent
// pipeline and a loader for child pipelines, it walks every sub-pipeline step
// with a typed InputRef and confirms the produced output type matches the
// child pipeline's declared input type.
//
// If childLoader is nil, this returns nil (runtime will catch unresolved refs).
// Unknown-child errors are non-fatal here — they already surface via
// detectSubPipelineCycles and the runtime loader.
func TypedWiringCheck(p *Pipeline, childLoader SubPipelineLoader, pipelinesDir string) error {
	if p == nil || childLoader == nil {
		return nil
	}

	// First pass: collect child pipelines and their declared input types.
	childInputType := map[string]string{}                   // step id -> child input type
	childSourceOutputType := map[string]map[string]string{} // step id -> (output name -> type)

	for _, step := range p.Steps {
		if step.SubPipeline == "" {
			continue
		}
		child, err := childLoader(pipelinesDir, step.SubPipeline)
		if err != nil {
			// Tolerate missing child pipelines here — treat as soft check.
			continue
		}
		childInputType[step.ID] = child.Input.EffectiveType()
	}

	// For from-wiring, we need to know each step's produced outputs by name.
	// Build a map of parent-step-id -> (output_name -> type) by inspecting the
	// child pipelines' PipelineOutputs for each composition step.
	for _, step := range p.Steps {
		if step.SubPipeline == "" {
			continue
		}
		child, err := childLoader(pipelinesDir, step.SubPipeline)
		if err != nil {
			continue
		}
		out := make(map[string]string, len(child.PipelineOutputs))
		for name, po := range child.PipelineOutputs {
			out[name] = po.EffectiveType()
		}
		childSourceOutputType[step.ID] = out
	}

	// Second pass: validate each InputRef.from binding.
	for _, step := range p.Steps {
		if step.InputRef == nil || step.InputRef.From == "" {
			continue
		}
		srcStep, remainder, ok := splitDot(step.InputRef.From)
		if !ok {
			return fmt.Errorf("pipeline %q step %q: input_ref.from %q must be '<step>.<output>[.<field>...]'",
				p.Metadata.Name, step.ID, step.InputRef.From)
		}
		// Rule 7: input_ref.from may be <step>.<output> (whole value) or
		// <step>.<output>.<path...> (navigate into the JSON value).
		srcField, fieldPath, hasPath := splitDot(remainder)
		if !hasPath {
			srcField = remainder
			fieldPath = ""
		}

		srcOutputs, ok := childSourceOutputType[srcStep]
		if !ok {
			// Source step is not a sub-pipeline step — may be a direct
			// sibling step within this pipeline; skip for now (covered by
			// composition template validation).
			continue
		}
		sourceType, ok := srcOutputs[srcField]
		if !ok {
			return fmt.Errorf("pipeline %q step %q: input_ref.from %q references unknown output %q of step %q",
				p.Metadata.Name, step.ID, step.InputRef.From, srcField, srcStep)
		}
		// When navigating inside the output JSON, the effective type at the
		// binding site is untyped (depends on the field). Skip type-match.
		if fieldPath != "" {
			continue
		}

		// If this step is itself a sub-pipeline step, compare against the child's input.
		if childTy, ok := childInputType[step.ID]; ok {
			if childTy != sourceType {
				return fmt.Errorf(
					"pipeline %q step %q: typed I/O mismatch — source %q produces %q but child pipeline expects %q",
					p.Metadata.Name, step.ID, step.InputRef.From, sourceType, childTy,
				)
			}
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

// splitDot splits "a.b" into ("a", "b", true); returns (_, _, false) if
// the input does not contain exactly one dot.
func splitDot(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			if i == 0 || i == len(s)-1 {
				return "", "", false
			}
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}
