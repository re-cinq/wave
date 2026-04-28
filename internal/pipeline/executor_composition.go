package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
	"golang.org/x/sync/errgroup"
)

func (e *DefaultPipelineExecutor) executeCompositionStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	// Route gate steps to the gate executor
	if step.Gate != nil {
		return e.executeGateInDAG(ctx, execution, step)
	}

	// Route composition primitives
	if step.Iterate != nil {
		return e.executeIterateInDAG(ctx, execution, step)
	}
	if step.Aggregate != nil {
		return e.executeAggregateInDAG(ctx, execution, step)
	}
	if step.Branch != nil {
		return e.executeBranchInDAG(ctx, execution, step)
	}
	if step.Loop != nil {
		return e.executeLoopInDAG(ctx, execution, step)
	}

	// Fall through: bare sub-pipeline step
	input := e.resolveSubPipelineInput(execution, step)
	return e.runNamedSubPipeline(ctx, execution, step, step.SubPipeline, input, compositionLaunchInfo{kind: "sub_pipeline_child"})
}

// resolveSubPipelineInput resolves the input string for a composition step,
// using the step's SubInput template or falling back to the pipeline input.
func (e *DefaultPipelineExecutor) resolveSubPipelineInput(execution *PipelineExecution, step *Step) string {
	input := execution.Input
	if step.SubInput != "" {
		// Resolve {{ input }} directly (not handled by PipelineContext.ResolvePlaceholders)
		resolved := strings.ReplaceAll(step.SubInput, "{{ input }}", execution.Input)
		resolved = strings.ReplaceAll(resolved, "{{input}}", execution.Input)
		// Then resolve remaining pipeline context variables
		if execution.Context != nil {
			resolved = execution.Context.ResolvePlaceholders(resolved)
		}
		if resolved != "" {
			input = resolved
		}
	}
	return input
}

// compositionLaunchInfo carries the metadata needed to populate the
// run_kind / iterate_index / iterate_total / iterate_mode columns
// (issue #1450) for a child run launched by a composition primitive.
// Zero-value defaults to "sub_pipeline_child" with no iterate metadata,
// matching bare sub-pipeline launches.
type compositionLaunchInfo struct {
	kind         string // "sub_pipeline_child" | "iterate_child" | "branch_arm" | "loop_iteration"
	iterateIndex *int   // 0-based index within iterate.over (nil for non-iterate launches)
	iterateTotal *int   // total items in iterate.over (nil for non-iterate launches)
	iterateMode  string // "parallel" or "serial"; empty for non-iterate launches
}

func (info compositionLaunchInfo) effectiveKind() string {
	if info.kind != "" {
		return info.kind
	}
	return "sub_pipeline_child"
}

// runNamedSubPipeline loads a pipeline by name from disk and executes it as a
// child of the current execution. It handles timeout, artifact injection/extraction,
// context merging, and parent-child state linking.
func (e *DefaultPipelineExecutor) runNamedSubPipeline(ctx context.Context, execution *PipelineExecution, step *Step, pipelineName, input string, info compositionLaunchInfo) error {
	pipelineID := execution.Status.ID

	// Apply lifecycle timeout from sub-pipeline config
	execCtx, cancel := subPipelineTimeout(ctx, step.Config)
	defer cancel()

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("composition: loading sub-pipeline %q", pipelineName),
	})

	// Load the sub-pipeline from disk
	loader := &YAMLPipelineLoader{}
	subPipelinePath := filepath.Join(".agents", "pipelines", pipelineName+".yaml")
	subPipeline, err := loader.Load(subPipelinePath)
	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return fmt.Errorf("failed to load sub-pipeline %q: %w", pipelineName, err)
	}

	// Build executor options for the child pipeline, inheriting configuration
	// from the parent but generating a fresh run ID.
	childOpts := []ExecutorOption{
		WithDebug(e.debug),
	}
	if e.emitter != nil {
		childOpts = append(childOpts, WithEmitter(e.emitter))
	}
	if e.store != nil {
		childOpts = append(childOpts, WithStateStore(e.store))
	}
	if e.modelOverride != "" {
		childOpts = append(childOpts, WithModelOverride(e.modelOverride))
	}
	if e.adapterOverride != "" {
		childOpts = append(childOpts, WithAdapterOverride(e.adapterOverride))
	}
	if e.stepTimeoutOverride > 0 {
		childOpts = append(childOpts, WithStepTimeout(e.stepTimeoutOverride))
	}
	if e.debugTracer != nil {
		childOpts = append(childOpts, WithDebugTracer(e.debugTracer))
	}
	if e.logger != nil {
		childOpts = append(childOpts, WithAuditLogger(e.logger))
	}
	if e.wsManager != nil {
		childOpts = append(childOpts, WithWorkspaceManager(e.wsManager))
	}
	if e.relayMonitor != nil {
		childOpts = append(childOpts, WithRelayMonitor(e.relayMonitor))
	}
	if e.skillStore != nil {
		childOpts = append(childOpts, withSkillStore(e.skillStore))
	}

	// Inject parent artifacts into child executor when config specifies injection.
	//
	// Two registration paths produce parent artifacts the child may inject:
	//   - Persona/command step outputs land in execution.ArtifactPaths under
	//     keys of the form "<step.ID>:<artifact.Name>" (executor.go:4460,4472).
	//   - Composition aggregate/iterate outputs ALSO call
	//     execution.Context.SetArtifactPath(<artifact.Name>) so they show up
	//     in the bare-name namespace used by templating.
	// The lookup walks the bare-name space first (the natural author-facing
	// API), then falls back to the dep-scoped ArtifactPaths map keyed by
	// "<dep>:<name>" for any declared step dependency. Without this fallback,
	// injecting a persona-step output (e.g. fetch-pr/pr-context) into a
	// downstream iterate step fails with "artifact … not found in parent
	// context for sub-pipeline injection" even though the artifact exists.
	// Auto-derived parent artifact paths from declared step.Dependencies
	// (issue #1452). Every upstream artifact a composition step depends
	// on becomes visible to its child sub-pipeline as
	// {{ artifacts.<name> }} without an explicit step.Config.Inject
	// entry. Explicit Inject still works and overrides on conflict.
	parentPaths := make(map[string]string)
	if autoResolved, err := e.ResolveDependencyArtifacts(execution, step); err == nil {
		for _, art := range autoResolved {
			parentPaths[art.Name] = art.Path
		}
	}
	if step.Config != nil && len(step.Config.Inject) > 0 {
		for _, name := range step.Config.Inject {
			path := execution.Context.GetArtifactPath(name)
			if path == "" {
				execution.mu.Lock()
				for _, dep := range step.Dependencies {
					if p, ok := execution.ArtifactPaths[dep+":"+name]; ok {
						path = p
						break
					}
				}
				execution.mu.Unlock()
			}
			if path == "" {
				return fmt.Errorf("artifact %q not found in parent context for sub-pipeline injection", name)
			}
			parentPaths[name] = path
		}
	}
	if len(parentPaths) > 0 {
		childOpts = append(childOpts, WithParentArtifactPaths(parentPaths))
	}

	// Propagate env: inherit parent's env first, then overlay step.Config.Env.
	// Result is seeded into the child executor's PipelineContext as
	// {{ env.<key> }} variables.
	stepEnv := map[string]string{}
	if step.Config != nil {
		stepEnv = step.Config.Env
	}
	if len(e.parentEnv) > 0 || len(stepEnv) > 0 {
		merged := make(map[string]string, len(e.parentEnv)+len(stepEnv))
		for k, v := range e.parentEnv {
			merged[k] = v
		}
		for k, v := range stepEnv {
			merged[k] = v
		}
		childOpts = append(childOpts, WithParentEnv(merged))
	}

	// Pass parent workspace path if this step has a resolved workspace
	execution.mu.Lock()
	if ws, ok := execution.WorkspacePaths[step.ID]; ok && ws != "" {
		childOpts = append(childOpts, WithParentWorkspacePath(ws))
	}
	execution.mu.Unlock()

	// Propagate max_cycles to child pipeline's loop config
	if step.Config != nil && step.Config.MaxCycles > 0 {
		for i := range subPipeline.Steps {
			if subPipeline.Steps[i].Loop != nil && subPipeline.Steps[i].Loop.MaxIterations == 0 {
				subPipeline.Steps[i].Loop.MaxIterations = step.Config.MaxCycles
			}
		}
	}

	// Create a run record for the child pipeline so it appears in the dashboard.
	// Link to parent immediately so the WebUI can nest it from the start.
	if e.store != nil {
		childRunID := e.createRunID(pipelineName, 4, input)
		childOpts = append(childOpts, WithRunID(childRunID))
		_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
		// Issue #1450 — record composition metadata so iterate progress
		// + run-kind chips render without re-deriving from event_log.
		_ = e.store.SetRunComposition(childRunID, info.effectiveKind(), pipelineName, info.iterateMode, info.iterateIndex, info.iterateTotal)
	}

	childOpts = append(childOpts, WithRegistry(e.registry))

	// Propagate step-level adapter override to child sub-pipeline
	if step.Adapter != "" {
		childOpts = append(childOpts, WithAdapterOverride(step.Adapter))
	} else if e.adapterOverride != "" {
		// Propagate CLI --adapter flag to sub-pipelines
		childOpts = append(childOpts, WithAdapterOverride(e.adapterOverride))
	}

	childExecutor := NewDefaultPipelineExecutor(e.runner, childOpts...)

	// Mark child run as running in the dashboard
	childRunID := childExecutor.runID
	if e.store != nil && childRunID != "" {
		_ = e.store.UpdateRunStatus(childRunID, "running", "", 0)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("composition: executing sub-pipeline %q", pipelineName),
	})

	if err := childExecutor.Execute(execCtx, subPipeline, execution.Manifest, input); err != nil {
		// Link parent-child state and update status even on failure
		if e.store != nil && childRunID != "" {
			_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
			_ = e.store.UpdateRunStatus(childRunID, "failed", err.Error(), childExecutor.GetTotalTokens())
		}
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return fmt.Errorf("sub-pipeline %q failed: %w", pipelineName, err)
	}

	// Link parent-child state and mark completed
	if e.store != nil && childRunID != "" {
		_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
		_ = e.store.UpdateRunStatus(childRunID, "completed", "", childExecutor.GetTotalTokens())

		// Propagate child's worktree branch to parent so the diff endpoint works
		if childRun, err := e.store.GetRun(childRunID); err == nil && childRun.BranchName != "" {
			_ = e.store.UpdateRunBranch(pipelineID, childRun.BranchName)
		}
	}

	// Extract child artifacts and merge context variables
	childExec := childExecutor.LastExecution()
	if childExec != nil {
		if step.Config != nil {
			// Determine parent workspace for artifact extraction
			parentWorkspace := "."
			execution.mu.Lock()
			if ws, ok := execution.WorkspacePaths[step.ID]; ok && ws != "" {
				parentWorkspace = ws
			}
			execution.mu.Unlock()

			if len(step.Config.Extract) > 0 {
				if err := extractSubPipelineArtifacts(step.Config, childExec.Context, pipelineName, execution.Context, parentWorkspace); err != nil {
					return fmt.Errorf("failed to extract sub-pipeline artifacts: %w", err)
				}
			}

			// Evaluate stop condition if configured
			if step.Config.StopCondition != "" {
				if evaluateStopCondition(step.Config.StopCondition, childExec.Context) {
					execution.Context.SetCustomVariable("sub_pipeline_stop", "true")
				}
			}
		}

		// Merge child context variables into parent (last-writer-wins)
		execution.Context.MergeFrom(childExec.Context, pipelineName)

		// Propagate child execution-level artifact paths into the parent context
		// so that iterate steps can collect outputs from all child sub-pipelines.
		childExec.mu.Lock()
		childArtifacts := make(map[string]string, len(childExec.ArtifactPaths))
		for k, v := range childExec.ArtifactPaths {
			childArtifacts[k] = v
		}
		childExec.mu.Unlock()

		execution.mu.Lock()
		for key, path := range childArtifacts {
			// Namespaced key for iterate/aggregate: "audit-security.scan:findings"
			nsKey := pipelineName + "." + key
			execution.Context.SetArtifactPath(nsKey, path)

			// Register under composition step ID so injectArtifacts can find
			// "audit-security:findings" when a persona step references this
			// sub-pipeline's output. Only for bare sub-pipeline steps (not
			// iterate/aggregate which have their own output collection).
			if step.Iterate == nil && step.Aggregate == nil {
				if _, artName, ok := strings.Cut(key, ":"); ok {
					parentKey := step.ID + ":" + artName
					execution.ArtifactPaths[parentKey] = path
				}
			}
		}
		execution.mu.Unlock()
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("composition: sub-pipeline %q completed", pipelineName),
	})

	return nil
}

// executeIterateInDAG fans out over a JSON array, running a sub-pipeline per item.
func (e *DefaultPipelineExecutor) executeIterateInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the items array — may contain step output references like
	// {{ fetch-children.output.child_urls }} or pipeline context vars.
	itemsExpr := step.Iterate.Over
	itemsExpr = e.resolveStepOutputRef(itemsExpr, execution)
	if execution.Context != nil {
		itemsExpr = execution.Context.ResolvePlaceholders(itemsExpr)
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(itemsExpr), &items); err != nil {
		return fmt.Errorf("iterate.over did not resolve to a JSON array: %w (raw: %s)", err, itemsExpr)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("iterate: %d items (mode: %s)", len(items), step.Iterate.Mode),
	})

	pipelineNameTmpl := step.SubPipeline

	if step.Iterate.Mode == "parallel" {
		return e.executeIterateParallelInDAG(ctx, execution, step, pipelineNameTmpl, items)
	}

	// Sequential iterate — track resolved pipeline names so we can collect outputs.
	resolvedNames := make([]string, 0, len(items))

	for i, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tmplCtx := NewTemplateContext(execution.Input, "")
		tmplCtx.Item = item

		// Resolve pipeline name (e.g. "{{ item }}" → "audit-security")
		resolvedName, err := ResolveTemplate(pipelineNameTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("iterate item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames = append(resolvedNames, resolvedName)

		// Resolve input. Pre-resolve step-output references against the
		// DAG executor's ArtifactPaths first (same path iterate.over uses),
		// then fall through to the per-item template context for {{item}} /
		// {{iteration}}. Without this, references like
		// {{scope.output.parent_issue.repository}} inside SubInput cannot
		// find the sub-pipeline step's output because each iteration gets
		// a fresh TemplateContext without StepOutputs populated.
		input := execution.Input
		if step.SubInput != "" {
			prepared := e.resolveStepOutputRef(step.SubInput, execution)
			if execution.Context != nil {
				prepared = execution.Context.ResolvePlaceholders(prepared)
			}
			input, err = ResolveTemplate(prepared, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}

		e.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     pipelineID,
			StepID:         step.ID,
			State:          event.StateIterationProgress,
			Message:        fmt.Sprintf("iterate item %d/%d: %s", i+1, len(items), resolvedName),
			TotalSteps:     len(items),
			CompletedSteps: i,
		})

		idx := i
		total := len(items)
		info := compositionLaunchInfo{
			kind:         "iterate_child",
			iterateIndex: &idx,
			iterateTotal: &total,
			iterateMode:  "serial",
		}
		if err := e.runNamedSubPipeline(ctx, execution, step, resolvedName, input, info); err != nil {
			return fmt.Errorf("iterate item %d (%s): %w", i, resolvedName, err)
		}
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	if err := e.collectIterateOutputs(execution, step, resolvedNames); err != nil {
		return fmt.Errorf("iterate: failed to collect outputs: %w", err)
	}

	e.emit(event.Event{
		Timestamp:      time.Now(),
		PipelineID:     pipelineID,
		StepID:         step.ID,
		State:          event.StateIterationCompleted,
		Message:        fmt.Sprintf("iterate: all %d items completed", len(items)),
		TotalSteps:     len(items),
		CompletedSteps: len(items),
	})

	return nil
}

func (e *DefaultPipelineExecutor) executeIterateParallelInDAG(ctx context.Context, execution *PipelineExecution, step *Step, pipelineNameTmpl string, items []json.RawMessage) error {
	pipelineID := execution.Status.ID

	maxConcurrent := step.Iterate.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(items)
	}

	// Pre-resolve all pipeline names so we can track them for output collection.
	resolvedNames := make([]string, len(items))
	resolvedInputs := make([]string, len(items))
	for i, item := range items {
		tmplCtx := NewTemplateContext(execution.Input, "")
		tmplCtx.Item = item

		name, err := ResolveTemplate(pipelineNameTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("iterate item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames[i] = name

		input := execution.Input
		if step.SubInput != "" {
			prepared := e.resolveStepOutputRef(step.SubInput, execution)
			if execution.Context != nil {
				prepared = execution.Context.ResolvePlaceholders(prepared)
			}
			input, err = ResolveTemplate(prepared, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}
		resolvedInputs[i] = input
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i := range items {
		resolvedName := resolvedNames[i]
		input := resolvedInputs[i]

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateRunning,
			Message:    fmt.Sprintf("iterate parallel item %d/%d: %s", i+1, len(items), resolvedName),
		})

		idx := i
		total := len(items)
		info := compositionLaunchInfo{
			kind:         "iterate_child",
			iterateIndex: &idx,
			iterateTotal: &total,
			iterateMode:  "parallel",
		}
		g.Go(func() error {
			return e.runNamedSubPipeline(gctx, execution, step, resolvedName, input, info)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	if err := e.collectIterateOutputs(execution, step, resolvedNames); err != nil {
		return fmt.Errorf("iterate: failed to collect outputs: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("iterate: all %d items completed (parallel)", len(items)),
	})

	return nil
}

// collectIterateOutputs scans the parent execution's context for artifacts
// merged from child sub-pipelines (via MergeFrom) and assembles them into a
// JSON array. The collected output is written to .agents/output/<stepID>-collected.json
// and registered in execution.ArtifactPaths so {{ stepID.output }} resolves to
// the combined result for downstream aggregate steps.
func (e *DefaultPipelineExecutor) collectIterateOutputs(execution *PipelineExecution, step *Step, resolvedNames []string) error {
	if execution.Context == nil {
		return nil
	}

	// For each child pipeline name, find the first artifact path that was
	// merged under its namespace (e.g. "audit-alpha.scan:output") and read
	// its content. Order matches the original items array.
	collected := make([]json.RawMessage, 0, len(resolvedNames))
	execution.Context.mu.Lock()
	artifactSnapshot := make(map[string]string, len(execution.Context.ArtifactPaths))
	for k, v := range execution.Context.ArtifactPaths {
		artifactSnapshot[k] = v
	}
	execution.Context.mu.Unlock()

	for _, name := range resolvedNames {
		prefix := name + "."
		var artPath string
		for key, path := range artifactSnapshot {
			if strings.HasPrefix(key, prefix) {
				artPath = path
				break
			}
		}

		if artPath == "" {
			// No artifact found for this child — include null placeholder
			// to keep the array aligned with the items array.
			collected = append(collected, json.RawMessage("null"))
			continue
		}

		data, err := os.ReadFile(artPath)
		if err != nil {
			collected = append(collected, json.RawMessage("null"))
			continue
		}

		// If the content is valid JSON, include it raw; otherwise wrap as a string.
		if json.Valid(data) {
			collected = append(collected, json.RawMessage(data))
		} else {
			quoted, _ := json.Marshal(string(data))
			collected = append(collected, json.RawMessage(quoted))
		}
	}

	arrayBytes, err := json.Marshal(collected)
	if err != nil {
		return fmt.Errorf("failed to marshal collected outputs: %w", err)
	}

	// Write to .agents/output/<stepID>-collected.json
	outputDir := filepath.Join(".agents", "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, step.ID+"-collected.json")
	if err := os.WriteFile(outputPath, arrayBytes, 0644); err != nil {
		return fmt.Errorf("failed to write collected output: %w", err)
	}

	// Register in execution.ArtifactPaths so resolveStepOutputRef can find it
	// via the standard "stepID:<artifactName>" key convention.
	execution.mu.Lock()
	execution.ArtifactPaths[step.ID+":collected-output"] = outputPath
	execution.mu.Unlock()

	// Register in DB so resume preflight and inject_artifacts can find the
	// collected output. Without this, downstream steps depending on iterate
	// output fail to resume.
	if e.store != nil {
		var size int64
		if info, statErr := os.Stat(outputPath); statErr == nil {
			size = info.Size()
		}
		_ = e.store.RegisterArtifact(execution.Status.ID, step.ID, "collected-output", outputPath, "json", size)
	}

	return nil
}

// executeAggregateInDAG collects outputs from prior steps and writes them to a file.
func (e *DefaultPipelineExecutor) executeAggregateInDAG(_ context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the source expression — step output references like
	// {{ run-audits.output }} must be resolved before context placeholders.
	sourceExpr := step.Aggregate.From
	sourceExpr = e.resolveStepOutputRef(sourceExpr, execution)
	if execution.Context != nil {
		sourceExpr = execution.Context.ResolvePlaceholders(sourceExpr)
	}

	var result string
	var err error

	switch step.Aggregate.Strategy {
	case "concat":
		result = sourceExpr
	case "merge_arrays":
		result, err = mergeJSONArrays(sourceExpr, step.Aggregate.Key)
		if err != nil {
			return fmt.Errorf("merge_arrays failed: %w", err)
		}
	case "reduce":
		result = sourceExpr
	default:
		return fmt.Errorf("unknown aggregate strategy: %q", step.Aggregate.Strategy)
	}

	// Write result to file
	outputPath := step.Aggregate.Into
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write aggregate output: %w", err)
	}

	// Derive the artifact name from the output filename (without extension)
	// so inject_artifacts can find it via the "stepID:artifactName" key.
	artifactName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	// Register in execution.ArtifactPaths (used by injectArtifacts lookup)
	execution.mu.Lock()
	execution.ArtifactPaths[step.ID+":"+artifactName] = outputPath
	execution.mu.Unlock()

	// Also register in context for template resolution
	if execution.Context != nil {
		execution.Context.SetArtifactPath(artifactName, outputPath)
	}

	// Register in DB so inject_artifacts and resume preflight can find this
	// artifact via the run-scoped artifact table — without this, downstream
	// steps depending on aggregate output fail to resume.
	if e.store != nil {
		var size int64
		if info, statErr := os.Stat(outputPath); statErr == nil {
			size = info.Size()
		}
		_ = e.store.RegisterArtifact(pipelineID, step.ID, artifactName, outputPath, "json", size)
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("aggregate: wrote %s (strategy: %s)", outputPath, step.Aggregate.Strategy),
	})

	return nil
}

// executeBranchInDAG evaluates a condition and runs the matching pipeline.
func (e *DefaultPipelineExecutor) executeBranchInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the branch condition — may reference step outputs
	value := step.Branch.On
	value = e.resolveStepOutputRef(value, execution)
	if execution.Context != nil {
		value = execution.Context.ResolvePlaceholders(value)
	}

	pipelineName, ok := step.Branch.Cases[value]
	if !ok {
		pipelineName, ok = step.Branch.Cases["default"]
		if !ok {
			return fmt.Errorf("branch value %q has no matching case and no default", value)
		}
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("branch %q → %s", value, pipelineName),
	})

	if pipelineName == "skip" {
		execution.mu.Lock()
		execution.States[step.ID] = stateCompleted
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "skipped by branch")
		}
		return nil
	}

	input := e.resolveSubPipelineInput(execution, step)
	return e.runNamedSubPipeline(ctx, execution, step, pipelineName, input, compositionLaunchInfo{kind: "branch_arm"})
}

// executeLoopInDAG runs sub-steps/sub-pipelines repeatedly until a condition is
// met or max iterations reached.
func (e *DefaultPipelineExecutor) executeLoopInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	if step.Loop.MaxIterations <= 0 {
		return fmt.Errorf("loop step %q: max_iterations must be > 0", step.ID)
	}

	for i := 0; i < step.Loop.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateRunning,
			Message:    fmt.Sprintf("loop iteration %d/%d", i+1, step.Loop.MaxIterations),
		})

		// Execute sub-pipeline if specified at the loop step level
		if step.SubPipeline != "" {
			input := e.resolveSubPipelineInput(execution, step)
			idx := i
			total := step.Loop.MaxIterations
			info := compositionLaunchInfo{
				kind:         "loop_iteration",
				iterateIndex: &idx,
				iterateTotal: &total,
			}
			if err := e.runNamedSubPipeline(ctx, execution, step, step.SubPipeline, input, info); err != nil {
				return fmt.Errorf("loop iteration %d: %w", i, err)
			}
		}

		// Execute loop sub-steps
		for j := range step.Loop.Steps {
			subStep := &step.Loop.Steps[j]
			if subStep.IsCompositionStep() {
				if err := e.executeCompositionStep(ctx, execution, subStep); err != nil {
					return fmt.Errorf("loop iteration %d, step %q: %w", i, subStep.ID, err)
				}
			}
		}

		// Check loop termination condition
		if step.Loop.Until != "" {
			condResult := step.Loop.Until
			if execution.Context != nil {
				condResult = execution.Context.ResolvePlaceholders(condResult)
			}
			condResult = strings.TrimSpace(condResult)
			if condResult == "true" || condResult == "done" || condResult == "yes" {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateCompleted,
					Message:    fmt.Sprintf("loop terminated: condition met at iteration %d", i+1),
				})
				execution.mu.Lock()
				execution.States[step.ID] = stateCompleted
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "loop condition met")
				}
				return nil
			}
		}
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("loop completed: max iterations (%d) reached", step.Loop.MaxIterations),
	})

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "max iterations reached")
	}

	return nil
}

// executeGateInDAG handles gate steps within a DAG pipeline.
// It delegates to GateExecutor, stores the decision in PipelineContext,
// writes freeform text as an artifact, and returns routing information via error type.
func (e *DefaultPipelineExecutor) executeGateInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("gate step: %s", step.Gate.Type),
	})

	// Select the appropriate handler
	var handler GateHandler
	if e.autoApprove {
		handler = &AutoApproveHandler{}
	} else if e.gateHandler != nil {
		handler = e.gateHandler
	}
	// If no handler and gate has choices, fall back to CLI handler
	if handler == nil && len(step.Gate.Choices) > 0 {
		handler = &CLIGateHandler{}
	}

	gate := NewGateExecutorWithHandler(e.emitter, e.store, &execution.Manifest.Runtime.Timeouts, handler)

	// Annotate the gate config with the step ID so downstream handlers
	// (e.g. WebUI) can associate the pending gate with a specific step.
	step.Gate.RuntimeStepID = step.ID

	decision, err := gate.ExecuteWithDecision(ctx, step.Gate, nil)
	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	// Store the gate decision in the pipeline context for template resolution
	if decision != nil && execution.Context != nil {
		execution.Context.SetGateDecision(step.ID, decision)

		// Write freeform text as artifact if provided
		if decision.Text != "" {
			wsRoot := execution.Manifest.Runtime.WorkspaceRoot
			if wsRoot == "" {
				wsRoot = ".agents/workspaces"
			}
			artifactPath := filepath.Join(wsRoot, e.workspaceRunIDFor(pipelineID), ".agents", "artifacts", fmt.Sprintf("gate-%s-text", step.ID))
			if mkErr := os.MkdirAll(filepath.Dir(artifactPath), 0755); mkErr != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStepProgress,
					Message:    fmt.Sprintf("warning: failed to create artifact directory for gate freeform text: %v", mkErr),
				})
			} else if writeErr := os.WriteFile(artifactPath, []byte(decision.Text), 0644); writeErr != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStepProgress,
					Message:    fmt.Sprintf("warning: failed to write gate freeform text artifact: %v", writeErr),
				})
			}
		}
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    "gate step completed",
	})

	// Handle choice routing
	if decision != nil && decision.Target == "_fail" {
		return &gateAbortError{StepID: step.ID, Choice: decision.Label}
	}

	// If the target is a step ID that has already been completed, re-queue it
	// (revision loop pattern). If the target is a pending/future step, just
	// continue normal DAG flow — the scheduler will reach it naturally.
	if decision != nil && decision.Target != "" && decision.Target != "_fail" {
		execution.mu.Lock()
		targetState := execution.States[decision.Target]
		execution.mu.Unlock()

		if targetState == stateCompleted || targetState == stateFailed || targetState == stateSkipped {
			return e.reQueueStep(execution, decision.Target)
		}
		// Target is pending/running — normal flow continues
	}

	return nil
}

// reQueueStep resets a step and all its dependents to pending state,
// allowing the main loop to re-execute them. This implements the revision loop pattern.
func (e *DefaultPipelineExecutor) reQueueStep(execution *PipelineExecution, targetStepID string) error {
	pipelineID := execution.Status.ID

	// Find the target step
	var targetStep *Step
	for i := range execution.Pipeline.Steps {
		if execution.Pipeline.Steps[i].ID == targetStepID {
			targetStep = &execution.Pipeline.Steps[i]
			break
		}
	}
	if targetStep == nil {
		return fmt.Errorf("gate routing: target step %q not found in pipeline", targetStepID)
	}

	// Reset the target step and all steps that depend on it (transitively)
	toReset := map[string]bool{targetStepID: true}
	changed := true
	for changed {
		changed = false
		for i := range execution.Pipeline.Steps {
			s := &execution.Pipeline.Steps[i]
			if toReset[s.ID] {
				continue
			}
			for _, dep := range s.Dependencies {
				if toReset[dep] {
					toReset[s.ID] = true
					changed = true
					break
				}
			}
		}
	}

	execution.mu.Lock()
	for stepID := range toReset {
		execution.States[stepID] = statePending
	}
	execution.mu.Unlock()

	for stepID := range toReset {
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, stepID, state.StatePending, "re-queued by gate routing")
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     stepID,
			State:      statePending,
			Message:    fmt.Sprintf("re-queued by gate routing to %q", targetStepID),
		})
	}

	return &reQueueError{TargetStepID: targetStepID, ResetSteps: toReset}
}

// reQueueError signals that steps have been re-queued via gate routing.
// The main executor loop should handle this by re-entering the scheduling loop.
type reQueueError struct {
	TargetStepID string
	ResetSteps   map[string]bool
}

func (e *reQueueError) Error() string {
	return fmt.Sprintf("gate routing: re-queued step %q and %d dependents", e.TargetStepID, len(e.ResetSteps)-1)
}

// cleanupCompletedPipeline removes a completed or failed pipeline from in-memory storage
// to prevent memory leaks. This is safe to call because completed pipeline status
