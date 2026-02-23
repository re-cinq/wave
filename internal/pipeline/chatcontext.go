package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/state"
)

// ChatContext holds assembled context for a wave chat session.
type ChatContext struct {
	Run          *state.RunRecord
	Steps        []ChatStepContext
	Pipeline     *Pipeline
	PipelinePath string
	Artifacts    []state.ArtifactRecord
	ProjectRoot  string
}

// ChatStepContext holds context for a single step in the chat session.
type ChatStepContext struct {
	StepID        string
	Persona       string
	State         string
	Duration      time.Duration
	TokensUsed    int
	WorkspacePath string
	Artifacts     []state.ArtifactRecord
	ErrorMessage  string
}

// BuildChatContext assembles the context for a chat session from the state store.
// It gathers run info, step events, artifacts, and workspace paths.
func BuildChatContext(store state.StateStore, runID string, p *Pipeline, projectRoot string) (*ChatContext, error) {
	// 1. Get run record
	run, err := store.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	// 2. Get all events for the run
	events, err := store.GetEvents(runID, state.EventQueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// 3. Get all artifacts for the run
	artifacts, err := store.GetArtifacts(runID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}

	// 4. Build per-step context from pipeline steps + events
	steps := buildStepContexts(p, events, artifacts, run.PipelineName, projectRoot)

	return &ChatContext{
		Run:         run,
		Steps:       steps,
		Pipeline:    p,
		Artifacts:   artifacts,
		ProjectRoot: projectRoot,
	}, nil
}

// buildStepContexts assembles per-step context from events and artifacts.
func buildStepContexts(p *Pipeline, events []state.LogRecord, artifacts []state.ArtifactRecord, pipelineName, projectRoot string) []ChatStepContext {
	// Index events by step ID — track last state, total tokens, duration
	type stepInfo struct {
		state      string
		tokens     int
		durationMs int64
		persona    string
		errMsg     string
	}
	stepEvents := make(map[string]*stepInfo)
	for _, evt := range events {
		if evt.StepID == "" {
			continue
		}
		info, ok := stepEvents[evt.StepID]
		if !ok {
			info = &stepInfo{}
			stepEvents[evt.StepID] = info
		}
		info.state = evt.State
		info.tokens += evt.TokensUsed
		info.durationMs += evt.DurationMs
		if evt.Persona != "" {
			info.persona = evt.Persona
		}
		if evt.State == "failed" && evt.Message != "" {
			info.errMsg = evt.Message
		}
	}

	// Index artifacts by step ID
	artifactsByStep := make(map[string][]state.ArtifactRecord)
	for _, art := range artifacts {
		artifactsByStep[art.StepID] = append(artifactsByStep[art.StepID], art)
	}

	// Build context for each pipeline step
	var steps []ChatStepContext
	for _, step := range p.Steps {
		ctx := ChatStepContext{
			StepID:    step.ID,
			Persona:   step.Persona,
			Artifacts: artifactsByStep[step.ID],
		}

		// Merge in event data
		if info, ok := stepEvents[step.ID]; ok {
			ctx.State = info.state
			ctx.TokensUsed = info.tokens
			ctx.Duration = time.Duration(info.durationMs) * time.Millisecond
			ctx.ErrorMessage = info.errMsg
			if info.persona != "" {
				ctx.Persona = info.persona
			}
		}

		// Check for preserved workspace
		wsPath := filepath.Join(projectRoot, ".wave", "workspaces", pipelineName, step.ID)
		if fi, err := os.Stat(wsPath); err == nil && fi.IsDir() {
			ctx.WorkspacePath = wsPath
		}

		steps = append(steps, ctx)
	}

	return steps
}

// MostRecentCompletedRunID finds the most recent completed run ID from the state store.
func MostRecentCompletedRunID(store state.StateStore) (string, error) {
	runs, err := store.ListRuns(state.ListRunsOptions{
		Limit: 10,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list runs: %w", err)
	}

	// Find most recent completed or failed run (either is valid for analysis)
	for _, run := range runs {
		if run.Status == "completed" || run.Status == "failed" {
			return run.RunID, nil
		}
	}

	// If no completed/failed, return the most recent run
	if len(runs) > 0 {
		return runs[0].RunID, nil
	}

	return "", fmt.Errorf("no pipeline runs found")
}
