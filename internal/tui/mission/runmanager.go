package mission

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"gopkg.in/yaml.v3"
)

// RunManagerConfig holds configuration for the RunManager.
type RunManagerConfig struct {
	ManifestPath  string
	Mock          bool
	Debug         bool
	ModelOverride string
}

// StartResult carries metadata from pipeline launch back to the model.
type StartResult struct {
	RunID        string
	StepOrder    []string
	StepPersonas map[string]string
}

// RunManager spawns and tracks pipeline runs as goroutines.
type RunManager struct {
	config     RunManagerConfig
	bus        *EventBus
	store      state.StateStore
	mu         sync.Mutex
	cancels    map[string]context.CancelFunc // runID → cancel
	pipeNames  map[string]string             // runID → pipeline name
}

// NewRunManager creates a new RunManager.
func NewRunManager(config RunManagerConfig, bus *EventBus, store state.StateStore) *RunManager {
	return &RunManager{
		config:    config,
		bus:       bus,
		store:     store,
		cancels:   make(map[string]context.CancelFunc),
		pipeNames: make(map[string]string),
	}
}

// preparedRun holds all the resources needed to execute a pipeline synchronously.
type preparedRun struct {
	runID    string
	executor *pipeline.DefaultPipelineExecutor
	pipeline *pipeline.Pipeline
	manifest *manifest.Manifest
	logger   audit.AuditLogger
	cancel   context.CancelFunc
	ctx      context.Context
	input    string
}

// preparePipeline loads and sets up everything needed to run a pipeline,
// but does not start execution. This is shared between StartPipeline and StartSequence.
func (rm *RunManager) preparePipeline(pipelineName, input string) (*preparedRun, *StartResult, error) {
	// Load manifest
	manifestData, err := os.ReadFile(rm.config.ManifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Load pipeline
	p, err := loadPipelineYAML(pipelineName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load pipeline: %w", err)
	}

	// Extract step order and personas
	stepOrder := make([]string, len(p.Steps))
	stepPersonas := make(map[string]string, len(p.Steps))
	for i, s := range p.Steps {
		stepOrder[i] = s.ID
		stepPersonas[s.ID] = s.Persona
	}

	// Resolve adapter
	var runner adapter.AdapterRunner
	if rm.config.Mock {
		runner = adapter.NewMockAdapter(
			adapter.WithSimulatedDelay(5 * time.Second),
		)
	} else {
		for name := range m.Adapters {
			runner = adapter.ResolveAdapter(name)
			break
		}
		if runner == nil {
			runner = adapter.ResolveAdapter("claude-code")
		}
	}

	// Create run record in state store
	var runID string
	if rm.store != nil {
		runID, err = rm.store.CreateRun(pipelineName, input)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create run: %w", err)
		}
	}
	if runID == "" {
		runID = pipeline.GenerateRunID(pipelineName, m.Runtime.PipelineIDHashLength)
	}

	// Build emitter chain
	busEmitter := NewBusEmitter(rm.bus, runID)
	throttled := display.NewThrottledProgressEmitter(busEmitter)
	emitter := event.NewProgressOnlyEmitter(throttled)

	// Workspace manager
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Audit logger
	var logger audit.AuditLogger
	if m.Runtime.Audit.LogAllToolCalls {
		traceDir := m.Runtime.Audit.LogDir
		if traceDir == "" {
			traceDir = ".wave/traces"
		}
		if l, lErr := audit.NewTraceLoggerWithDir(traceDir); lErr == nil {
			logger = l
		}
	}

	// Build executor
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
		pipeline.WithRunID(runID),
		pipeline.WithDebug(rm.config.Debug),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	if rm.store != nil {
		execOpts = append(execOpts, pipeline.WithStateStore(rm.store))
	}
	if logger != nil {
		execOpts = append(execOpts, pipeline.WithAuditLogger(logger))
	}
	if rm.config.ModelOverride != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(rm.config.ModelOverride))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Track the run
	ctx, cancel := context.WithCancel(context.Background())
	rm.mu.Lock()
	rm.cancels[runID] = cancel
	rm.pipeNames[runID] = pipelineName
	rm.mu.Unlock()

	return &preparedRun{
		runID:    runID,
		executor: executor,
		pipeline: p,
		manifest: &m,
		logger:   logger,
		cancel:   cancel,
		ctx:      ctx,
		input:    input,
	}, &StartResult{
		RunID:        runID,
		StepOrder:    stepOrder,
		StepPersonas: stepPersonas,
	}, nil
}

// executePipelineSync runs a prepared pipeline synchronously. Returns nil on success.
func (rm *RunManager) executePipelineSync(prep *preparedRun) error {
	defer func() {
		if prep.logger != nil {
			if cl, ok := prep.logger.(interface{ Close() error }); ok {
				cl.Close()
			}
		}
		prep.cancel()
		rm.mu.Lock()
		delete(rm.cancels, prep.runID)
		rm.mu.Unlock()
	}()

	// Update status to running
	if rm.store != nil {
		rm.store.UpdateRunStatus(prep.runID, "running", "", 0)
	}

	execErr := prep.executor.Execute(prep.ctx, prep.pipeline, prep.manifest, prep.input)

	// Update final status
	if rm.store != nil {
		tokens := prep.executor.GetTotalTokens()
		if execErr != nil {
			rm.store.UpdateRunStatus(prep.runID, "failed", execErr.Error(), tokens)
		} else {
			rm.store.UpdateRunStatus(prep.runID, "completed", "", tokens)
		}
	}

	return execErr
}

// StartPipeline loads and executes a pipeline in a background goroutine.
func (rm *RunManager) StartPipeline(pipelineName, input string) (*StartResult, error) {
	prep, result, err := rm.preparePipeline(pipelineName, input)
	if err != nil {
		return nil, err
	}

	go rm.executePipelineSync(prep)

	return result, nil
}

// StartSequence launches pipelines sequentially in a background goroutine.
// Each pipeline appears as a separate run in the fleet view.
// If any pipeline fails, remaining pipelines are not started (fail-fast).
//
// All pipelines in the sequence are returned as StartResults upfront so the
// fleet view can show queued placeholders for subsequent pipelines.
func (rm *RunManager) StartSequence(pipelineNames []string, input string) ([]*StartResult, error) {
	if len(pipelineNames) == 0 {
		return nil, fmt.Errorf("no pipelines specified for sequence")
	}

	// Validate all pipelines exist before starting
	for _, name := range pipelineNames {
		if _, err := loadPipelineYAML(name); err != nil {
			return nil, fmt.Errorf("pipeline %q not found: %w", name, err)
		}
	}

	// Prepare the first pipeline immediately so we can return its StartResult
	firstPrep, firstResult, err := rm.preparePipeline(pipelineNames[0], input)
	if err != nil {
		return nil, err
	}

	results := []*StartResult{firstResult}

	// Create queued placeholders for remaining pipelines
	type queuedRun struct {
		runID        string
		pipelineName string
	}
	var queued []queuedRun

	for _, name := range pipelineNames[1:] {
		p, _ := loadPipelineYAML(name)
		stepOrder := make([]string, len(p.Steps))
		stepPersonas := make(map[string]string, len(p.Steps))
		for i, s := range p.Steps {
			stepOrder[i] = s.ID
			stepPersonas[s.ID] = s.Persona
		}

		runID := pipeline.GenerateRunID(name, 8)
		if rm.store != nil {
			if storeID, storeErr := rm.store.CreateRun(name, input); storeErr == nil && storeID != "" {
				runID = storeID
			}
		}

		rm.mu.Lock()
		rm.pipeNames[runID] = name
		rm.mu.Unlock()

		queued = append(queued, queuedRun{runID: runID, pipelineName: name})
		results = append(results, &StartResult{
			RunID:        runID,
			StepOrder:    stepOrder,
			StepPersonas: stepPersonas,
		})
	}

	// TODO(#248): Wire meta.SequenceExecutor for cross-pipeline artifact handoff.
	// Currently executes sequentially without artifact copying between pipelines.
	go func() {
		// Execute first pipeline
		if execErr := rm.executePipelineSync(firstPrep); execErr != nil {
			// Mark remaining as cancelled
			for _, q := range queued {
				if rm.store != nil {
					rm.store.UpdateRunStatus(q.runID, "cancelled", "sequence failed at earlier pipeline", 0)
				}
				rm.bus.Send(RunEvent{
					RunID: q.runID,
					Event: event.Event{
						Timestamp:  time.Now(),
						PipelineID: q.pipelineName,
						State:      "cancelled",
						Message:    "sequence failed at earlier pipeline",
					},
				})
			}
			return
		}

		// Execute remaining pipelines sequentially
		for i, q := range queued {
			prep, _, prepErr := rm.preparePipeline(q.pipelineName, input)
			if prepErr != nil {
				// Mark this and remaining as failed/cancelled
				if rm.store != nil {
					rm.store.UpdateRunStatus(q.runID, "failed", prepErr.Error(), 0)
				}
				for _, remaining := range queued[i+1:] {
					if rm.store != nil {
						rm.store.UpdateRunStatus(remaining.runID, "cancelled", "sequence failed", 0)
					}
					rm.bus.Send(RunEvent{
						RunID: remaining.runID,
						Event: event.Event{
							Timestamp:  time.Now(),
							PipelineID: remaining.pipelineName,
							State:      "cancelled",
							Message:    "sequence failed",
						},
					})
				}
				return
			}

			if execErr := rm.executePipelineSync(prep); execErr != nil {
				// Mark remaining as cancelled
				for _, remaining := range queued[i+1:] {
					if rm.store != nil {
						rm.store.UpdateRunStatus(remaining.runID, "cancelled", "sequence failed at earlier pipeline", 0)
					}
					rm.bus.Send(RunEvent{
						RunID: remaining.runID,
						Event: event.Event{
							Timestamp:  time.Now(),
							PipelineID: remaining.pipelineName,
							State:      "cancelled",
							Message:    "sequence failed at earlier pipeline",
						},
					})
				}
				return
			}
		}
	}()

	return results, nil
}

// StartParallel launches multiple pipelines concurrently.
// Each pipeline runs in its own goroutine via StartPipeline.
func (rm *RunManager) StartParallel(pipelineNames []string, inputs map[string]string) ([]*StartResult, error) {
	if len(pipelineNames) == 0 {
		return nil, fmt.Errorf("no pipelines specified for parallel launch")
	}

	var results []*StartResult
	for _, name := range pipelineNames {
		input := ""
		if inputs != nil {
			input = inputs[name]
		}
		result, err := rm.StartPipeline(name, input)
		if err != nil {
			return results, fmt.Errorf("failed to start %s: %w", name, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CancelRun cancels a running pipeline.
func (rm *RunManager) CancelRun(runID string) bool {
	rm.mu.Lock()
	cancel, ok := rm.cancels[runID]
	rm.mu.Unlock()

	if ok {
		cancel()
		if rm.store != nil {
			rm.store.RequestCancellation(runID, true)
		}
		return true
	}

	// Try cancelling via state store (for external runs)
	if rm.store != nil {
		if err := rm.store.RequestCancellation(runID, false); err == nil {
			return true
		}
	}
	return false
}

// ActiveRunCount returns the number of locally managed active runs.
func (rm *RunManager) ActiveRunCount() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return len(rm.cancels)
}

// Shutdown cancels all active runs.
func (rm *RunManager) Shutdown() {
	rm.mu.Lock()
	for _, cancel := range rm.cancels {
		cancel()
	}
	rm.mu.Unlock()
}

// loadPipelineYAML loads a pipeline from the standard location.
func loadPipelineYAML(name string) (*pipeline.Pipeline, error) {
	candidates := []string{
		".wave/pipelines/" + name + ".yaml",
		".wave/pipelines/" + name,
		name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline '%s' not found (searched .wave/pipelines/)", name)
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline: %w", err)
	}

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline: %w", err)
	}

	return &p, nil
}
