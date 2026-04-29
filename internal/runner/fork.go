// fork.go exposes a thin adapter around pipeline.ForkManager so callers
// outside the pipeline package (notably internal/webui) can drive fork and
// fork-point operations without taking a direct dependency on the executor
// package. The runner package already imports internal/pipeline and is the
// canonical bridge between transport layers and the executor.
package runner

import (
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
)

// ForkPoint mirrors pipeline.ForkPoint with the field set webui handlers
// expose to clients. Defined here so internal/webui can keep its DTO
// translation entirely in terms of runner types.
type ForkPoint struct {
	StepID    string
	StepIndex int
	HasSHA    bool
}

// ForkController wraps pipeline.ForkManager so transport-layer callers can
// fork runs and list fork points without importing internal/pipeline.
// Construct one per RW state store; the underlying ForkManager is stateless.
type ForkController struct {
	store state.RunStore
}

// NewForkController creates a fork controller bound to the given state store.
// The store must support both reads and (for Fork) writes.
func NewForkController(store state.RunStore) *ForkController {
	return &ForkController{store: store}
}

// ListForkPoints returns the steps in runID that have a recorded checkpoint
// and are therefore valid fork sources.
func (c *ForkController) ListForkPoints(runID string) ([]ForkPoint, error) {
	fm := pipeline.NewForkManager(c.store)
	raw, err := fm.ListForkPoints(runID)
	if err != nil {
		return nil, err
	}
	out := make([]ForkPoint, len(raw))
	for i, p := range raw {
		out[i] = ForkPoint{StepID: p.StepID, StepIndex: p.StepIndex, HasSHA: p.HasSHA}
	}
	return out, nil
}

// Fork creates a new run that branches from a completed step of an existing
// run. pipelineName is loaded from on-disk YAML via the same loader the
// webui's start path uses; allowFailed mirrors the pipeline.ForkManager
// option for forking non-completed runs.
func (c *ForkController) Fork(sourceRunID, fromStep, pipelineName string, allowFailed bool) (string, error) {
	p, err := LoadPipelineByName(pipelineName)
	if err != nil {
		return "", err
	}
	fm := pipeline.NewForkManager(c.store)
	return fm.Fork(sourceRunID, fromStep, p, allowFailed)
}

// ResumeStepAfter returns the ID of the step immediately following fromStep
// in the named pipeline, or an empty string when fromStep is the final step.
// Webui handlers use this to decide whether a fork should re-execute or
// short-circuit to a "completed" status.
func (c *ForkController) ResumeStepAfter(pipelineName, fromStep string) (string, error) {
	p, err := LoadPipelineByName(pipelineName)
	if err != nil {
		return "", err
	}
	for i, step := range p.Steps {
		if step.ID == fromStep && i+1 < len(p.Steps) {
			return p.Steps[i+1].ID, nil
		}
	}
	return "", nil
}

// RewindPlan describes the effect of rewinding a run to toStep: the index
// of toStep in the pipeline (-1 if not found) and the IDs of every step
// that follows it (and would therefore be discarded by a rewind). Returned
// to webui handlers so they can validate the request and report the deleted
// step list to clients without ever touching pipeline domain types.
type RewindPlan struct {
	StepIndex    int
	StepsDeleted []string
}

// PlanRewind resolves toStep against the named pipeline and returns the
// rewind index plus the list of steps that would be discarded. Index -1
// means toStep was not found.
func (c *ForkController) PlanRewind(pipelineName, toStep string) (RewindPlan, error) {
	p, err := LoadPipelineByName(pipelineName)
	if err != nil {
		return RewindPlan{StepIndex: -1}, err
	}
	plan := RewindPlan{StepIndex: -1}
	for i, step := range p.Steps {
		if step.ID == toStep {
			plan.StepIndex = i
			break
		}
	}
	if plan.StepIndex == -1 {
		return plan, nil
	}
	for i, step := range p.Steps {
		if i > plan.StepIndex {
			plan.StepsDeleted = append(plan.StepsDeleted, step.ID)
		}
	}
	return plan, nil
}
