package webui

import (
	"fmt"
	"os"
	"regexp"

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
