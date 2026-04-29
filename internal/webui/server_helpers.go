package webui

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
)

// validPipelineName matches safe pipeline names: alphanumeric, hyphens, underscores, dots.
// Used by loadPipelineYAML to prevent path traversal when resolving pipeline files.
var validPipelineName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// loadPipelineYAML loads a pipeline definition from .agents/pipelines/.
// The name must match [a-zA-Z0-9][a-zA-Z0-9._-]* to prevent path traversal.
func loadPipelineYAML(name string) (*pipeline.Pipeline, error) {
	if !validPipelineName.MatchString(name) {
		return nil, fmt.Errorf("invalid pipeline name")
	}

	candidates := []string{
		".agents/pipelines/" + name + ".yaml",
		".agents/pipelines/" + name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline not found")
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found")
	}

	loader := &pipeline.YAMLPipelineLoader{}
	return loader.Unmarshal(data)
}

// runOptionsFromStartRequest projects an HTTP StartPipelineRequest onto the
// shared runner.Options struct so the launch path is identical regardless of
// which handler triggered the run.
func runOptionsFromStartRequest(req StartPipelineRequest) runner.Options {
	return runner.Options{
		Model:             req.Model,
		Adapter:           req.Adapter,
		DryRun:            req.DryRun,
		FromStep:          req.FromStep,
		Force:             req.Force,
		Detach:            req.Detach,
		Timeout:           req.Timeout,
		Steps:             req.Steps,
		Exclude:           req.Exclude,
		OnFailure:         req.OnFailure,
		Continuous:        req.Continuous,
		Source:            req.Source,
		MaxIterations:     req.MaxIterations,
		Delay:             req.Delay,
		Mock:              req.Mock,
		PreserveWorkspace: req.PreserveWorkspace,
		AutoApprove:       req.AutoApprove,
		NoRetro:           req.NoRetro,
		ForceModel:        req.ForceModel,
	}
}

// getCompositionPipelines reads pipeline YAML files and returns only those
// with composition primitives, along with their structure details. Lives in
// this helper file so the handlers_compose.go transport layer doesn't need
// to import internal/pipeline directly.
func getCompositionPipelines() []CompositionPipeline {
	names := listPipelineNames()
	var result []CompositionPipeline

	for _, name := range names {
		p, err := loadPipelineYAML(name)
		if err != nil {
			continue
		}

		hasComposition := false
		for _, step := range p.Steps {
			if step.IsCompositionStep() {
				hasComposition = true
				break
			}
		}
		if !hasComposition {
			continue
		}

		var steps []CompositionStep
		for _, step := range p.Steps {
			cs := classifyCompositionStep(&step)
			steps = append(steps, cs)
		}

		result = append(result, CompositionPipeline{
			Name:        name,
			Description: p.Metadata.Description,
			Category:    p.Metadata.Category,
			StepCount:   len(p.Steps),
			Steps:       steps,
			Skills:      filterTemplateVars(p.Skills),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// classifyCompositionStep inspects a pipeline step and returns a
// CompositionStep with the appropriate type label and details. Kept in this
// helper file so handlers_compose.go can stay free of pipeline imports.
func classifyCompositionStep(step *pipeline.Step) CompositionStep {
	cs := CompositionStep{
		ID:      step.ID,
		Details: make(map[string]string),
	}

	switch {
	case step.Iterate != nil:
		cs.Type = "iterate"
		cs.SubPipeline = step.SubPipeline
		cs.Details["mode"] = step.Iterate.Mode
		if step.Iterate.Mode == "" {
			cs.Details["mode"] = "sequential"
		}
		if step.Iterate.MaxConcurrent > 0 {
			cs.Details["max_concurrent"] = fmt.Sprintf("%d", step.Iterate.MaxConcurrent)
		}
	case step.Branch != nil:
		cs.Type = "branch"
		var cases []string
		for k, v := range step.Branch.Cases {
			cases = append(cases, k+"->"+v)
		}
		sort.Strings(cases)
		cs.Details["cases"] = strings.Join(cases, ", ")
	case step.Gate != nil:
		cs.Type = "gate"
		cs.Details["gate_type"] = step.Gate.Type
		if step.Gate.Timeout != "" {
			cs.Details["timeout"] = step.Gate.Timeout
		}
		if step.Gate.Message != "" {
			cs.Details["message"] = step.Gate.Message
		}
	case step.Loop != nil:
		cs.Type = "loop"
		cs.Details["max_iterations"] = fmt.Sprintf("%d", step.Loop.MaxIterations)
		if step.Loop.Until != "" {
			cs.Details["until"] = step.Loop.Until
		}
		if len(step.Loop.Steps) > 0 {
			var subIDs []string
			for _, s := range step.Loop.Steps {
				subIDs = append(subIDs, s.ID)
			}
			cs.Details["sub_steps"] = strings.Join(subIDs, ", ")
		}
		cs.SubPipeline = step.SubPipeline
	case step.Aggregate != nil:
		cs.Type = "aggregate"
		cs.Details["strategy"] = step.Aggregate.Strategy
		cs.Details["into"] = step.Aggregate.Into
	case step.SubPipeline != "":
		cs.Type = "sub_pipeline"
		cs.SubPipeline = step.SubPipeline
	default:
		cs.Type = "persona"
		cs.Persona = resolveForgeVars(step.Persona)
	}

	return cs
}

// runOptionsFromSubmitRequest mirrors runOptionsFromStartRequest for the
// /api/runs submit endpoint.
func runOptionsFromSubmitRequest(req SubmitRunRequest) runner.Options {
	return runner.Options{
		Model:             req.Model,
		Adapter:           req.Adapter,
		DryRun:            req.DryRun,
		FromStep:          req.FromStep,
		Force:             req.Force,
		Detach:            req.Detach,
		Timeout:           req.Timeout,
		Steps:             req.Steps,
		Exclude:           req.Exclude,
		OnFailure:         req.OnFailure,
		Continuous:        req.Continuous,
		Source:            req.Source,
		MaxIterations:     req.MaxIterations,
		Delay:             req.Delay,
		Mock:              req.Mock,
		PreserveWorkspace: req.PreserveWorkspace,
		AutoApprove:       req.AutoApprove,
		NoRetro:           req.NoRetro,
		ForceModel:        req.ForceModel,
	}
}
