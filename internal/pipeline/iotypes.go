package pipeline

import (
	"fmt"

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

	outputsByStep := map[string]*Pipeline{} // parent step id -> child pipeline
	// First pass: collect child pipelines and their declared input types.
	childInputType := map[string]string{} // step id -> child input type
	childSourceOutputType := map[string]map[string]string{} // step id -> (output name -> type)
	_ = outputsByStep

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
		srcStep, srcField, ok := splitDot(step.InputRef.From)
		if !ok {
			return fmt.Errorf("pipeline %q step %q: input_ref.from %q must be '<step>.<output>'",
				p.Metadata.Name, step.ID, step.InputRef.From)
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
