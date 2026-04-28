package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/cost"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
)

func (e *DefaultPipelineExecutor) executeMatrixStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	matrixExecutor := NewMatrixExecutor(e)
	err := matrixExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Track deliverables from completed matrix step
	e.trackStepDeliverables(execution, step)

	// Extract declared outcomes from matrix step artifacts
	e.processStepOutcomes(execution, step)

	return nil
}

// executeConcurrentStep handles steps with concurrency > 1 using parallel agent execution.
func (e *DefaultPipelineExecutor) executeConcurrentStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	concurrencyExecutor := NewConcurrencyExecutor(e)
	err := concurrencyExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Track deliverables from completed concurrent step
	e.trackStepDeliverables(execution, step)

	// Extract declared outcomes from concurrent step artifacts
	e.processStepOutcomes(execution, step)

	return nil
}

// runStepExecution orchestrates the four phases of a single step run:
// resource resolution → config assembly → adapter dispatch → result processing.
func (e *DefaultPipelineExecutor) runStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	// Phase A: Resolve persona, adapter, workspace, model, artifacts, and build base prompt
	res, err := e.resolveStepResources(ctx, execution, step)
	if err != nil {
		return err
	}

	// Phase B: Build AdapterRunConfig (timeout, system prompt, sandbox, skills, contract prompt)
	cfg, err := e.buildStepAdapterConfig(ctx, execution, step, res)
	if err != nil {
		return err
	}

	// Emit step progress: executing
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    res.pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       res.resolvedPersona,
		Progress:      25,
		CurrentAction: "Executing agent",
	})

	// Iron Rule: estimate prompt size and check against context window
	promptBytes := len(cfg.Prompt) + len(cfg.OntologySection)
	if promptBytes > 0 && res.resolvedModel != "" {
		ironStatus, ironMsg := cost.CheckIronRule(res.resolvedModel, promptBytes)
		switch ironStatus {
		case cost.IronRuleWarning:
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: res.pipelineID,
				StepID:     step.ID,
				State:      "iron_rule_warning",
				Message:    ironMsg,
			})
		case cost.IronRuleFail:
			return fmt.Errorf("iron rule: %s", ironMsg)
		}
	}

	// Phase C: Dispatch to adapter
	stepStart := time.Now()
	e.trace("adapter_start", step.ID, 0, map[string]string{
		"persona": res.resolvedPersona,
		"adapter": res.resolvedAdapterName,
		"model":   res.resolvedModel,
	})
	result, adapterErr := res.stepRunner.Run(ctx, cfg)
	adapterDurationMs := time.Since(stepStart).Milliseconds()

	if adapterErr != nil {
		e.trace("adapter_end", step.ID, adapterDurationMs, map[string]string{
			"status": stateFailed,
			"error":  adapterErr.Error(),
		})
		if e.logger != nil {
			_ = e.logger.LogStepEnd(res.pipelineID, step.ID, stateFailed, time.Since(stepStart), 0, 0, 0, adapterErr.Error())
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        res.pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      res.resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				Success:      false,
				ErrorMessage: adapterErr.Error(),
			})
		}
		return fmt.Errorf("adapter execution failed: %w", adapterErr)
	}

	// Warn on non-zero exit code — adapter process may have crashed, but
	// work may still have been completed (e.g. Claude Code JS error after
	// tool calls finished). Let contract validation decide the outcome.
	if result.ExitCode != 0 {
		msg := fmt.Sprintf("adapter exited with code %d", result.ExitCode)
		if result.FailureReason != "" {
			msg += ": " + result.FailureReason
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: res.pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    msg,
		})
	}

	// Fail immediately on rate limit — the result content is an error message,
	// not useful work product. Proceeding would write the error as an artifact.
	if result.FailureReason == adapter.FailureReasonRateLimit {
		if e.logger != nil {
			_ = e.logger.LogStepEnd(res.pipelineID, step.ID, stateFailed, time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, "rate limited: "+result.ResultContent)
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        res.pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      res.resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				TokensUsed:   result.TokensUsed,
				Success:      false,
				ErrorMessage: "rate limited: " + result.ResultContent,
			})
		}
		return fmt.Errorf("adapter rate limited: %s", result.ResultContent)
	}

	e.trace("adapter_end", step.ID, adapterDurationMs, map[string]string{
		"status":      "success",
		"exit_code":   fmt.Sprintf("%d", result.ExitCode),
		"tokens_used": fmt.Sprintf("%d", result.TokensUsed),
	})

	// Phase D: Process adapter result (stdout, tokens, cost, artifacts, contracts, hooks)
	return e.processAdapterResult(ctx, execution, step, res, result, stepStart)
}

// resolveStepResources resolves the persona, adapter, workspace, and model for a step,
// injects dependent artifacts, and builds the base step prompt.
// It is Phase A of runStepExecution.
func (e *DefaultPipelineExecutor) resolveStepResources(ctx context.Context, execution *PipelineExecution, step *Step) (*stepRunResources, error) {
	pipelineID := execution.Status.ID

	resolvedPersona := step.Persona
	if execution.Context != nil {
		resolvedPersona = execution.Context.ResolvePlaceholders(step.Persona)
	}
	persona := execution.Manifest.GetPersona(resolvedPersona)
	if persona == nil {
		return nil, fmt.Errorf("persona %q not found in manifest", resolvedPersona)
	}

	// Resolve adapter name (strongest to weakest):
	// 1. CLI --adapter flag  2. Step-level adapter  3. Persona-level adapter  4. Adapter defaults
	resolvedAdapterName := persona.Adapter
	if step.Adapter != "" {
		resolvedAdapterName = step.Adapter
	}
	if e.adapterOverride != "" {
		resolvedAdapterName = e.adapterOverride
	}

	adapterDef := execution.Manifest.GetAdapter(resolvedAdapterName)
	if adapterDef == nil {
		available := make([]string, 0, len(execution.Manifest.Adapters))
		for name := range execution.Manifest.Adapters {
			available = append(available, name)
		}
		return nil, fmt.Errorf("adapter %q not found in manifest for step %q (available: %v)", resolvedAdapterName, step.ID, available)
	}

	// Resolve adapter runner from registry for per-step dispatch
	stepRunner := e.registry.ResolveWithFallback(resolvedAdapterName)

	// Create workspace under .agents/workspaces/<pipeline>/<step>/
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	execution.mu.Lock()
	execution.WorkspacePaths[step.ID] = workspacePath
	execution.mu.Unlock()

	// Run workspace_created hooks (non-blocking by default)
	if e.hookRunner != nil {
		e.hookRunner.RunHooks(ctx, hooks.HookEvent{
			Type:       hooks.EventWorkspaceCreated,
			PipelineID: pipelineID,
			StepID:     step.ID,
			Workspace:  workspacePath,
		})
	}

	// Pre-create .agents/output/ so personas without Bash can write artifacts
	if len(step.OutputArtifacts) > 0 {
		outputDir := filepath.Join(workspacePath, ".agents", "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	var adapterTierModels map[string]string
	if adapterDef != nil {
		adapterTierModels = adapterDef.TierModels
	}
	resolvedModel := e.resolveModel(step, persona, &execution.Manifest.Runtime.Routing, resolvedPersona, adapterTierModels)

	// When no model was resolved, fall back to adapter's default_model
	if resolvedModel == "" && adapterDef != nil && adapterDef.DefaultModel != "" {
		resolvedModel = adapterDef.DefaultModel
	}

	// When resolved adapter differs from persona's adapter and no explicit model was set,
	// fall back to the target adapter's default model (avoids cross-ecosystem model IDs)
	if resolvedAdapterName != persona.Adapter && e.modelOverride == "" && step.Model == "" {
		if adapterDef != nil && adapterDef.DefaultModel != "" {
			resolvedModel = adapterDef.DefaultModel
		} else {
			resolvedModel = ""
		}
	}

	// Determine the configured (pre-resolution) model for provenance tracking
	configuredModel := step.Model
	if configuredModel == "" {
		configuredModel = persona.Model
	}

	e.emit(event.Event{
		Timestamp:       time.Now(),
		PipelineID:      pipelineID,
		StepID:          step.ID,
		State:           stateRunning,
		Persona:         resolvedPersona,
		Message:         fmt.Sprintf("Starting %s persona in %s", resolvedPersona, workspacePath),
		CurrentAction:   "Initializing",
		Model:           resolvedModel,
		ConfiguredModel: configuredModel,
		Adapter:         resolvedAdapterName,
		Temperature:     persona.Temperature,
	})

	// Record model routing decision
	{
		var rationale string
		switch {
		case e.modelOverride != "":
			rationale = "CLI --model flag override"
		case step.Model != "":
			rationale = "per-step model pinning in pipeline YAML"
		case persona.Model != "":
			rationale = "per-persona model configuration"
		case execution.Manifest.Runtime.Routing.AutoRoute:
			rationale = "auto-routed based on step complexity"
		default:
			rationale = "adapter default (no override)"
		}
		modelDisplay := resolvedModel
		if modelDisplay == "" {
			modelDisplay = "(adapter default)"
		}
		e.recordDecision(pipelineID, step.ID, "model_routing",
			fmt.Sprintf("selected model %s for step %s", modelDisplay, step.ID),
			rationale,
			map[string]interface{}{
				"model":   resolvedModel,
				"persona": resolvedPersona,
				"adapter": resolvedAdapterName,
			},
		)
	}

	// Auto-inject every declared dependency's output artifacts into the
	// canonical .agents/artifacts/<dep>/<name> layout (issue #1452). Runs
	// BEFORE legacy injectArtifacts so manual `as:` renames can still
	// overwrite the canonical copy when desired.
	if _, err := e.injectDependencyArtifacts(execution, step, workspacePath); err != nil {
		return nil, fmt.Errorf("failed to auto-inject dep artifacts: %w", err)
	}

	// Inject artifacts from dependencies (legacy explicit inject_artifacts).
	artifactInjectStart := time.Now()
	if err := e.injectArtifacts(execution, step, workspacePath); err != nil {
		return nil, fmt.Errorf("failed to inject artifacts: %w", err)
	}
	e.trace("artifact_injection", step.ID, time.Since(artifactInjectStart).Milliseconds(), map[string]string{
		"workspace": workspacePath,
		"count":     fmt.Sprintf("%d", len(step.Memory.InjectArtifacts)),
	})

	// Audit: log step start with injected artifact names
	if e.logger != nil {
		var artifactNames []string
		for _, ref := range step.Memory.InjectArtifacts {
			name := ref.As
			if name == "" {
				name = ref.Artifact
			}
			artifactNames = append(artifactNames, name)
		}
		_ = e.logger.LogStepStartWithAdapter(pipelineID, step.ID, resolvedPersona, resolvedAdapterName, resolvedModel, artifactNames)
	}

	prompt := e.buildStepPrompt(execution, step)
	if e.logger != nil {
		_ = e.logger.LogToolCall(pipelineID, step.ID, "adapter.Run", fmt.Sprintf("persona=%s prompt_len=%d", resolvedPersona, len(prompt)))
	}

	return &stepRunResources{
		pipelineID:          pipelineID,
		resolvedPersona:     resolvedPersona,
		persona:             persona,
		adapterDef:          adapterDef,
		resolvedAdapterName: resolvedAdapterName,
		stepRunner:          stepRunner,
		workspacePath:       workspacePath,
		resolvedModel:       resolvedModel,
		configuredModel:     configuredModel,
		prompt:              prompt,
	}, nil
}

// buildStepAdapterConfig assembles the adapter.AdapterRunConfig for a step.
// It resolves timeout, system prompt, sandbox settings, skills, contract prompt,
// and ontology section. It is Phase B of runStepExecution.
func (e *DefaultPipelineExecutor) buildStepAdapterConfig(_ context.Context, execution *PipelineExecution, step *Step, res *stepRunResources) (adapter.AdapterRunConfig, error) {
	pipelineID := res.pipelineID
	prompt := res.prompt

	// Resolve timeout with four-tier precedence:
	// 1. Step-level timeout_minutes (pipeline YAML) — most specific
	// 2. CLI --timeout flag (stepTimeoutOverride)
	// 3. runtime.default_timeout_minutes (manifest)
	// 4. Hardcoded fallback (5 minutes)
	timeout := execution.Manifest.Runtime.GetDefaultTimeout()
	if e.stepTimeoutOverride > 0 {
		timeout = e.stepTimeoutOverride
	}
	if stepTimeout := step.GetTimeout(); stepTimeout > 0 {
		timeout = stepTimeout
	}

	// Load system prompt from persona file
	systemPrompt := ""
	if res.persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(res.persona.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	}

	// Resolve sandbox config using the new backend-aware resolution
	sandboxBackend := execution.Manifest.Runtime.Sandbox.ResolveBackend()
	sandboxEnabled := sandboxBackend != "none"
	var sandboxDomains []string
	var envPassthrough []string
	if sandboxEnabled {
		if res.persona.Sandbox != nil && len(res.persona.Sandbox.AllowedDomains) > 0 {
			sandboxDomains = res.persona.Sandbox.AllowedDomains
		} else if len(execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains) > 0 {
			sandboxDomains = execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains
		}
		envPassthrough = execution.Manifest.Runtime.Sandbox.EnvPassthrough
	}

	// Resolve skills from all three scopes: global, persona, pipeline
	// Pipeline scope includes both pipeline.Skills and requires.skills keys.
	var pipelineSkills []string
	for _, s := range execution.Pipeline.Skills {
		r := execution.Context.ResolvePlaceholders(s)
		if r != "" {
			pipelineSkills = append(pipelineSkills, r)
		}
	}
	if execution.Pipeline.Requires != nil {
		pipelineSkills = append(pipelineSkills, execution.Pipeline.Requires.SkillNames()...)
	}
	// Step-level skills take highest precedence in the merge — they declare
	// the exact skill set this single agent run needs.
	for _, s := range step.Skills {
		r := execution.Context.ResolvePlaceholders(s)
		if r != "" {
			pipelineSkills = append(pipelineSkills, r)
		}
	}
	resolvedSkills := skill.ResolveSkills(execution.Manifest.Skills, res.persona.Skills, pipelineSkills)

	// Provision skill commands from requires.skills (SkillConfig-backed skills)
	var skillCommandsDir string
	if execution.Pipeline.Requires != nil && len(execution.Pipeline.Requires.Skills) > 0 {
		skillNames := execution.Pipeline.Requires.SkillNames()
		provisioner := skill.NewProvisioner(execution.Pipeline.Requires.Skills, "")
		commands, _ := provisioner.DiscoverCommands(skillNames)
		if len(commands) > 0 {
			tmpDir := filepath.Join(res.workspacePath, ".wave-skill-commands")
			if err := provisioner.Provision(tmpDir, skillNames); err != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warning",
					Message:    fmt.Sprintf("skill provisioning failed: %v", err),
				})
			} else {
				skillCommandsDir = filepath.Join(tmpDir, ".claude", "commands")
			}
		}
	}

	// Resolve DirectoryStore skills (name-only references not in requires.skills).
	// Read metadata from the store but do NOT write files here — each adapter's
	// prepareWorkspace handles file provisioning to the correct location
	// (e.g. .claude/skills/ for claude, text injection for others).
	var resolvedSkillRefs []adapter.SkillRef
	if e.skillStore != nil && len(resolvedSkills) > 0 {
		requiresSkills := make(map[string]bool)
		if execution.Pipeline.Requires != nil {
			for name := range execution.Pipeline.Requires.Skills {
				requiresSkills[name] = true
			}
		}
		for _, name := range resolvedSkills {
			if requiresSkills[name] {
				continue
			}
			s, readErr := e.skillStore.Read(name)
			if readErr != nil {
				if errors.Is(readErr, skill.ErrNotFound) {
					// skill not found in store, skip silently
					continue
				}
				return adapter.AdapterRunConfig{}, fmt.Errorf("skill %q: %w", name, readErr)
			}
			resolvedSkillRefs = append(resolvedSkillRefs, adapter.SkillRef{
				Name:        s.Name,
				Description: s.Description,
				SourcePath:  s.SourcePath,
			})
		}
	}

	// Auto-generate contract compliance section. Appended directly to the user prompt
	// so the model sees it alongside the task instructions (system prompt injection was unreliable).
	contractPrompt := e.buildContractPrompt(step, execution.Context)
	if contractPrompt != "" {
		prompt = prompt + "\n\n" + contractPrompt
	}

	// Build ontology section from manifest for AGENTS.md injection.
	// NoOp service returns "" when the feature is disabled.
	ontologySection := e.ontology.BuildStepSection(pipelineID, step.ID, step.Contexts)

	// Resolve effective tool permissions: step overlay ∪ persona ∪ adapter defaults.
	// Step.Permissions can ADD tools (additive); persona-level deny rules still win
	// because PermissionChecker enforces deny-first precedence.
	effectivePerms := ResolveStepPermissions(step, res.persona, res.adapterDef)

	cfg := adapter.AdapterRunConfig{
		Adapter:             res.resolvedAdapterName,
		Persona:             res.resolvedPersona,
		WorkspacePath:       res.workspacePath,
		Prompt:              prompt,
		SystemPrompt:        systemPrompt,
		Timeout:             timeout,
		Temperature:         res.persona.Temperature,
		Model:               res.resolvedModel,
		AllowedTools:        effectivePerms.AllowedTools,
		DenyTools:           effectivePerms.Deny,
		OutputFormat:        res.adapterDef.OutputFormat,
		Debug:               e.debug,
		SandboxEnabled:      sandboxEnabled,
		AllowedDomains:      sandboxDomains,
		EnvPassthrough:      envPassthrough,
		SandboxBackend:      sandboxBackend,
		DockerImage:         execution.Manifest.Runtime.Sandbox.GetDockerImage(),
		SkillCommandsDir:    skillCommandsDir,
		ResolvedSkills:      resolvedSkillRefs,
		OntologySection:     ontologySection,
		MaxConcurrentAgents: step.MaxConcurrentAgents,
		OnStreamEvent: func(evt adapter.StreamEvent) {
			// Reset the activity timer on ANY stream event so a thinking-only
			// loop (no tool_use yet) does not look identical to a wedged
			// subprocess. Only progress events (tool_use on a writing tool)
			// reset the longer progress timer; that is what protects against
			// read-only loops.
			execution.mu.Lock()
			wd := execution.Watchdog
			execution.mu.Unlock()
			if wd != nil {
				wd.NotifyActivity()
			}

			if evt.Type == "tool_use" && evt.ToolName != "" {
				if wd != nil && IsProgressTool(evt.ToolName) {
					wd.NotifyProgress()
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStreamActivity,
					Persona:    res.resolvedPersona,
					ToolName:   evt.ToolName,
					ToolTarget: evt.ToolInput,
				})
			}
		},
	}

	return cfg, nil
}

// processAdapterResult handles the result from a successful adapter run:
// reads stdout, accumulates tokens and cost, writes artifacts, runs relay compaction,
// validates contracts, fires completion hooks, and records performance metrics.
// It is Phase D of runStepExecution.
func (e *DefaultPipelineExecutor) processAdapterResult(
	ctx context.Context,
	execution *PipelineExecution,
	step *Step,
	res *stepRunResources,
	result *adapter.AdapterResult,
	stepStart time.Time,
) error {
	pipelineID := res.pipelineID
	stepDuration := time.Since(stepStart).Milliseconds()

	// Emit step progress: processing results
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       res.resolvedPersona,
		Progress:      75,
		CurrentAction: "Processing results",
		TokensUsed:    result.TokensUsed,
	})

	output := make(map[string]interface{})
	stdoutData, err := io.ReadAll(result.Stdout)
	if err == nil {
		output["stdout"] = string(stdoutData)
	}
	output["exit_code"] = result.ExitCode
	output["tokens_used"] = result.TokensUsed
	output["workspace"] = res.workspacePath

	execution.mu.Lock()
	execution.Results[step.ID] = output
	execution.mu.Unlock()

	// Append step output to thread transcript when the step is part of a thread group
	if step.Thread != "" && execution.ThreadManager != nil {
		content := result.ResultContent
		if content == "" {
			content = string(stdoutData)
		}
		if content != "" {
			execution.ThreadManager.AppendTranscript(step.Thread, step.ID, content)
			e.trace(audit.TraceThreadAppend, step.ID, int64(len(content)), map[string]string{
				"thread": step.Thread,
				"size":   fmt.Sprintf("%d", len(content)),
			})
		}
	}

	// Accumulate tokens at executor level (survives pipeline cleanup)
	if result.TokensUsed > 0 {
		e.mu.Lock()
		e.totalTokens += result.TokensUsed
		e.mu.Unlock()
	}

	// Record cost and enforce budget
	if e.costLedger != nil && (result.TokensIn > 0 || result.TokensOut > 0) {
		_, budgetStatus := e.costLedger.Record(pipelineID, step.ID, res.resolvedModel, result.TokensIn, result.TokensOut, result.TokensUsed)
		switch budgetStatus {
		case cost.BudgetWarning:
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "budget_warning",
				Message:    fmt.Sprintf("Cost warning: %s", e.costLedger.Summary()),
			})
		case cost.BudgetExceeded:
			return fmt.Errorf("budget exceeded: %s", e.costLedger.Summary())
		}
	}

	// Check for stdout artifacts and validate size limits
	hasStdoutArtifacts := false
	for _, art := range step.OutputArtifacts {
		if art.IsStdoutArtifact() {
			hasStdoutArtifacts = true
			break
		}
	}

	if hasStdoutArtifacts {
		maxSize := execution.Manifest.Runtime.Artifacts.GetMaxStdoutSize()
		if int64(len(stdoutData)) > maxSize {
			return fmt.Errorf("stdout artifact size (%d bytes) exceeds limit (%d bytes); consider reducing output or increasing runtime.artifacts.max_stdout_size",
				len(stdoutData), maxSize)
		}
		e.writeOutputArtifacts(execution, step, res.workspacePath, stdoutData)
	}

	// Write file-based output artifacts
	// Use ResultContent if available; don't fall back to raw stdout (contains JSON wrapper)
	if result.ResultContent != "" && !hasStdoutArtifacts {
		e.writeOutputArtifacts(execution, step, res.workspacePath, []byte(result.ResultContent))
	} else if !hasStdoutArtifacts {
		// ResultContent is empty — check whether the persona wrote artifact files to disk
		// (e.g. via Write/Bash). Without this, persona-written files are never registered
		// in ArtifactPaths and contract validation fails on missing files.
		e.writeOutputArtifacts(execution, step, res.workspacePath, nil)
	}

	// Check relay/compaction threshold (FR-009)
	if cErr := e.checkRelayCompaction(ctx, execution, step, result.TokensUsed, res.workspacePath, string(stdoutData)); cErr != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    fmt.Sprintf("relay compaction failed: %v", cErr),
		})
	}

	// Validate handover contracts — iterate over EffectiveContracts() which returns
	// the plural 'contracts' list if set, otherwise wraps the singular 'contract' field.
	if err := e.validateStepContracts(ctx, execution, step, res.workspacePath, res.stepRunner, pipelineID, res.resolvedPersona, stepStart, result); err != nil {
		return err
	}

	// Populate artifact paths from step's OutputArtifacts when the adapter
	// doesn't report them (e.g. Claude adapter never populates Artifacts).
	stepArtifacts := result.Artifacts
	if len(stepArtifacts) == 0 && len(step.OutputArtifacts) > 0 && execution.Context != nil {
		for _, art := range step.OutputArtifacts {
			stepArtifacts = append(stepArtifacts, execution.Context.ResolveArtifactPath(art))
		}
	}

	// Run artifact_created hooks (non-blocking by default)
	if e.hookRunner != nil && len(stepArtifacts) > 0 {
		e.hookRunner.RunHooks(ctx, hooks.HookEvent{
			Type:       hooks.EventArtifactCreated,
			PipelineID: pipelineID,
			StepID:     step.ID,
			Workspace:  res.workspacePath,
			Artifacts:  stepArtifacts,
		})
	}

	// Run step_completed hooks (blocking by default)
	stepCompletedEvt := hooks.HookEvent{
		Type:       hooks.EventStepCompleted,
		PipelineID: pipelineID,
		StepID:     step.ID,
		Input:      execution.Input,
		Workspace:  res.workspacePath,
	}
	if e.hookRunner != nil {
		if _, err := e.hookRunner.RunHooks(ctx, stepCompletedEvt); err != nil {
			return fmt.Errorf("step_completed hook failed: %w", err)
		}
	}
	e.fireWebhooks(ctx, stepCompletedEvt)

	// Detect zero-diff worktree steps: step completed but produced no code changes.
	// This gives the UI an honest signal — "completed_empty" with warning colors
	// instead of a misleading green "completed" checkmark.
	finalState := stateCompleted
	if step.Workspace.Type == "worktree" && isWorktreeClean(res.workspacePath) {
		finalState = stateCompletedEmpty
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      finalState,
		Persona:    res.resolvedPersona,
		DurationMs: stepDuration,
		TokensUsed: result.TokensUsed,
		Artifacts:  stepArtifacts,
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
	})

	if e.logger != nil {
		_ = e.logger.LogStepEnd(pipelineID, step.ID, finalState, time.Since(stepStart), result.ExitCode, len(stdoutData), result.TokensUsed, "")
	}

	// Record performance metric for TUI step breakdown
	if e.store != nil {
		completedAt := time.Now()
		e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
			RunID:              pipelineID,
			StepID:             step.ID,
			PipelineName:       execution.Status.PipelineName,
			Persona:            res.resolvedPersona,
			StartedAt:          stepStart,
			CompletedAt:        &completedAt,
			DurationMs:         stepDuration,
			TokensUsed:         result.TokensUsed,
			ArtifactsGenerated: len(stepArtifacts),
			Success:            true,
		})
	}

	return nil
}

// isWorktreeClean checks whether a worktree workspace has uncommitted or
// unstaged changes. Returns true when the worktree is identical to its HEAD
// (zero diff) — meaning the agent produced no code changes.
