package webui

import (
	"log"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
)

// RunOptions is the CLI-parity option set forwarded from the webui start
// form to internal/runner. Aliased so webui handlers and request DTOs keep
// their existing field names while sharing one canonical shape with the cmd
// path.
type RunOptions = runner.Options

// launchPipelineExecution starts pipeline execution as a detached subprocess
// via internal/runner. The subprocess is fully independent of the server
// process — server shutdown does not cancel runs. Dry-run mode short-circuits
// to a synchronous status update because validation completes instantly.
//
// This helper is shared by handleStartPipeline, handleRetryRun, handleResumeRun,
// and handleForkRun. When fromStep is non-empty the subprocess resumes from
// that step.
func (s *Server) launchPipelineExecution(runID, pipelineName, input string, opts RunOptions, fromStep ...string) {
	// Dry-run: handle in-process (instant, no subprocess needed).
	if opts.DryRun {
		if err := s.runtime.rwStore.UpdateRunStatus(runID, "completed", "dry run (validation only)", 0); err != nil {
			log.Printf("Warning: failed to update run %s status for dry-run: %v", runID, err)
		}
		return
	}

	// Spawn a detached subprocess via the shared runner. Concurrency is
	// enforced atomically at CreateRunWithLimit by the calling handler.
	if err := s.spawnDetachedRun(runID, pipelineName, input, opts, fromStep...); err != nil {
		log.Printf("Error: failed to spawn detached run %s: %v — falling back to in-process", runID, err)
		s.launchInProcess(runID, pipelineName, input, opts, fromStep...)
	}
}

// spawnDetachedRun delegates to runner.Detach, reusing the run ID the handler
// already created in the state DB. The runner consumes the same flag-spec
// table the CLI uses, so flag changes only need to land in one place.
func (s *Server) spawnDetachedRun(runID, pipelineName, input string, opts RunOptions, fromStep ...string) error {
	opts.Pipeline = pipelineName
	opts.Input = input
	opts.RunID = runID
	if len(fromStep) > 0 && fromStep[0] != "" {
		opts.FromStep = fromStep[0]
	}
	// Never recurse into detached mode in the subprocess — runner.Detach
	// is already producing a Setsid'd child.
	opts.Detach = false
	// Force --debug for visibility into server-launched runs (matches the
	// pre-extraction behaviour where buildDetachedArgs always appended --debug).
	opts.Output.Verbose = true

	cfg := runner.DetachConfig{
		WorkDir:  s.runtime.repoDir,
		ExtraEnv: []string{"GH_TOKEN", "GITHUB_TOKEN"},
	}
	// runner.Detach reuses the pre-created run row when opts.RunID exists
	// in the store, so no extra coordination is needed.
	if _, err := runner.Detach(opts, s.runtime.rwStore, 0, cfg); err != nil {
		return err
	}
	log.Printf("Pipeline %s (%s) launched as detached process", pipelineName, runID)
	return nil
}

// launchInProcess runs the pipeline inside the server process via
// internal/runner. This is the fallback path when subprocess spawning fails;
// the server-shutdown path will cancel these via activeRuns.
func (s *Server) launchInProcess(runID, pipelineName, input string, opts RunOptions, fromStep ...string) {
	resolvedFromStep := ""
	if len(fromStep) > 0 {
		resolvedFromStep = fromStep[0]
	}

	emitter := &event.DBLoggingEmitter{
		Inner: s.realtime.broker,
		Store: s.runtime.rwStore,
		RunID: runID,
		OnError: func(rid string, err error) {
			log.Printf("Warning: failed to log event for run %s: %v", rid, err)
		},
	}

	var gateHandler pipeline.GateHandler
	if s.realtime.gateRegistry != nil {
		gateHandler = runner.NewWebUIGateHandler(runID, s.realtime.gateRegistry)
	}

	cancel := runner.LaunchInProcess(runner.InProcessConfig{
		RunID:            runID,
		PipelineName:     pipelineName,
		Input:            input,
		Manifest:         s.runtime.manifest,
		Store:            s.runtime.rwStore,
		Emitter:          emitter,
		WorkspaceManager: s.runtime.wsManager,
		GateHandler:      gateHandler,
		FromStep:         resolvedFromStep,
		Options:          opts,
		OnComplete: func(string, error) {
			// Invalidate issue/PR caches so fresh data shows after pipeline completion.
			s.assets.cache.InvalidatePrefix("issues:")
			s.assets.cache.InvalidatePrefix("prs:")

			s.mu.Lock()
			delete(s.realtime.activeRuns, runID)
			s.mu.Unlock()
		},
	})

	s.mu.Lock()
	s.realtime.activeRuns[runID] = cancel
	s.mu.Unlock()
}
