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
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"golang.org/x/sync/errgroup"
)

// SubPipelineLoader loads a pipeline by name from a directory.
// This avoids a circular dependency on the tui package.
type SubPipelineLoader func(dir, name string) (*Pipeline, error)

// CompositionExecutor interprets composition step primitives (iterate, branch,
// gate, loop, aggregate, sub-pipeline) and delegates actual pipeline execution
// to SequenceExecutor / DefaultPipelineExecutor.
type CompositionExecutor struct {
	emitterMixin
	seqExecutor    *SequenceExecutor
	store          state.StateStore
	tmplCtx        *TemplateContext
	manifest       *manifest.Manifest
	pipelinesDir   string
	pipelineLoader SubPipelineLoader
	debug          bool
	gateHandler    GateHandler // Interactive handler for approval gates
}

// NewCompositionExecutor creates a composition executor.
func NewCompositionExecutor(
	seqExecutor *SequenceExecutor,
	emitter event.EventEmitter,
	store state.StateStore,
	m *manifest.Manifest,
	input string,
	pipelinesDir string,
	debug bool,
) *CompositionExecutor {
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}
	return &CompositionExecutor{
		emitterMixin: emitterMixin{emitter: emitter},
		seqExecutor:  seqExecutor,
		store:        store,
		tmplCtx:      NewTemplateContext(input, wsRoot),
		manifest:     m,
		pipelinesDir: pipelinesDir,
		debug:        debug,
	}
}

// SetPipelineLoader sets the function used to load sub-pipelines by name.
func (c *CompositionExecutor) SetPipelineLoader(loader SubPipelineLoader) {
	c.pipelineLoader = loader
}

// Execute runs a composition pipeline -- a pipeline whose steps are composition
// primitives rather than persona-driven steps.
func (c *CompositionExecutor) Execute(ctx context.Context, p *Pipeline, input string) error {
	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		State:      event.StateStarted,
		Message:    fmt.Sprintf("composition: %s (%d steps)", p.Metadata.Name, len(p.Steps)),
	})

	for i, step := range p.Steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     p.Metadata.Name,
			StepID:         step.ID,
			State:          event.StateRunning,
			Message:        fmt.Sprintf("composition step %d/%d: %s", i+1, len(p.Steps), step.ID),
			TotalSteps:     len(p.Steps),
			CompletedSteps: i,
		})

		if err := c.executeCompositionStep(ctx, p, &step); err != nil {
			c.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: p.Metadata.Name,
				StepID:     step.ID,
				State:      event.StateFailed,
				Message:    err.Error(),
			})
			return fmt.Errorf("composition step %q failed: %w", step.ID, err)
		}
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("composition completed: %d steps", len(p.Steps)),
	})

	return nil
}

// executeCompositionStep dispatches to the appropriate handler based on step type.
func (c *CompositionExecutor) executeCompositionStep(ctx context.Context, p *Pipeline, step *Step) error {
	switch {
	case step.Iterate != nil:
		return c.executeIterate(ctx, p, step)
	case step.Branch != nil:
		return c.executeBranch(ctx, p, step)
	case step.Gate != nil:
		return c.executeGate(ctx, step)
	case step.Loop != nil:
		return c.executeLoop(ctx, p, step)
	case step.Aggregate != nil:
		return c.executeAggregate(step)
	case step.SubPipeline != "":
		return c.executeSubPipeline(ctx, step)
	default:
		return fmt.Errorf("step %q is not a composition step", step.ID)
	}
}

// executeIterate runs a sub-pipeline for each item in a collection.
func (c *CompositionExecutor) executeIterate(ctx context.Context, p *Pipeline, step *Step) error {
	// Resolve the items array
	itemsJSON, err := ResolveTemplate(step.Iterate.Over, c.tmplCtx)
	if err != nil {
		return fmt.Errorf("failed to resolve iterate.over: %w", err)
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return fmt.Errorf("iterate.over did not resolve to a JSON array: %w", err)
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		StepID:     step.ID,
		State:      event.StateIterationStarted,
		Message:    fmt.Sprintf("iterating over %d items (mode: %s)", len(items), step.Iterate.Mode),
	})

	pipelineName := step.SubPipeline
	if pipelineName == "" {
		return fmt.Errorf("iterate step %q must specify a pipeline", step.ID)
	}

	if step.Iterate.Mode == "parallel" {
		return c.executeIterateParallel(ctx, p, step, pipelineName, items)
	}
	return c.executeIterateSequential(ctx, p, step, pipelineName, items)
}

func (c *CompositionExecutor) executeIterateSequential(ctx context.Context, p *Pipeline, step *Step, pipelineNameTmpl string, items []json.RawMessage) error {
	resolvedNames := make([]string, 0, len(items))

	for i, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: p.Metadata.Name,
			StepID:     step.ID,
			State:      event.StateIterationProgress,
			Message:    fmt.Sprintf("item %d/%d", i+1, len(items)),
			Progress:   ((i + 1) * 100) / len(items),
		})

		// Set item in template context
		c.tmplCtx.Item = item

		// Resolve pipeline name per item (e.g. "{{ item }}" → "audit-security")
		resolvedName, err := ResolveTemplate(pipelineNameTmpl, c.tmplCtx)
		if err != nil {
			return fmt.Errorf("item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames = append(resolvedNames, resolvedName)

		// Resolve input template
		input, err := c.resolveStepInput(step)
		if err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}

		// Load and execute the sub-pipeline (iterate: key by pipelineName; step.ID is aggregated later)
		if err := c.runSubPipeline(ctx, "", resolvedName, input); err != nil {
			return fmt.Errorf("item %d: pipeline %q failed: %w", i, resolvedName, err)
		}
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	c.collectIterateOutputs(step, resolvedNames)

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		StepID:     step.ID,
		State:      event.StateIterationCompleted,
		Message:    fmt.Sprintf("all %d items completed", len(items)),
	})

	return nil
}

func (c *CompositionExecutor) executeIterateParallel(ctx context.Context, p *Pipeline, step *Step, pipelineNameTmpl string, items []json.RawMessage) error {
	maxConcurrent := step.Iterate.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(items)
	}

	// Pre-resolve all pipeline names so we can track them for output collection.
	resolvedNames := make([]string, len(items))
	resolvedInputs := make([]string, len(items))
	for i, item := range items {
		localCtx := NewTemplateContext(c.tmplCtx.Input, c.tmplCtx.WorkspaceRoot)
		for k, v := range c.tmplCtx.StepOutputs {
			localCtx.StepOutputs[k] = v
		}
		localCtx.Item = item

		name, err := ResolveTemplate(pipelineNameTmpl, localCtx)
		if err != nil {
			return fmt.Errorf("item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames[i] = name

		input := c.tmplCtx.Input
		if step.SubInput != "" {
			input, err = ResolveTemplate(step.SubInput, localCtx)
			if err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}
		resolvedInputs[i] = input
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i := range items {
		resolvedName := resolvedNames[i]
		input := resolvedInputs[i]

		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: p.Metadata.Name,
			StepID:     step.ID,
			State:      event.StateIterationProgress,
			Message:    fmt.Sprintf("parallel item %d/%d: %s", i+1, len(items), resolvedName),
		})

		g.Go(func() error {
			return c.runSubPipeline(gctx, "", resolvedName, input)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	c.collectIterateOutputs(step, resolvedNames)

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		StepID:     step.ID,
		State:      event.StateIterationCompleted,
		Message:    fmt.Sprintf("all %d items completed (parallel)", len(items)),
	})

	return nil
}

// collectIterateOutputs assembles the outputs from all child sub-pipelines into
// a JSON array and registers it under the iterate step's own ID. This allows
// downstream steps to reference {{ stepID.output }} to get the collected result.
func (c *CompositionExecutor) collectIterateOutputs(step *Step, resolvedNames []string) {
	collected := make([]json.RawMessage, 0, len(resolvedNames))
	for _, name := range resolvedNames {
		data, ok := c.tmplCtx.StepOutputs[name]
		if !ok || len(data) == 0 {
			collected = append(collected, json.RawMessage("null"))
			continue
		}
		if json.Valid(data) {
			collected = append(collected, json.RawMessage(data))
		} else {
			quoted, _ := json.Marshal(string(data))
			collected = append(collected, json.RawMessage(quoted))
		}
	}

	arrayBytes, err := json.Marshal(collected)
	if err != nil {
		return
	}

	c.tmplCtx.SetStepOutput(step.ID, arrayBytes)
}

// executeBranch evaluates a condition and runs the matching pipeline.
func (c *CompositionExecutor) executeBranch(ctx context.Context, p *Pipeline, step *Step) error {
	value, err := ResolveTemplate(step.Branch.On, c.tmplCtx)
	if err != nil {
		return fmt.Errorf("failed to resolve branch.on: %w", err)
	}

	pipelineName, ok := step.Branch.Cases[value]
	if !ok {
		// Check for default case
		pipelineName, ok = step.Branch.Cases["default"]
		if !ok {
			return fmt.Errorf("branch value %q has no matching case and no default", value)
		}
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		StepID:     step.ID,
		State:      event.StateBranchEvaluated,
		Message:    fmt.Sprintf("branch %q -> %s", value, pipelineName),
	})

	if pipelineName == "skip" {
		return nil
	}

	input, err := c.resolveStepInput(step)
	if err != nil {
		return err
	}

	return c.runSubPipeline(ctx, step.ID, pipelineName, input)
}

// executeGate blocks until a gate condition is met.
func (c *CompositionExecutor) executeGate(ctx context.Context, step *Step) error {
	gate := NewGateExecutor(c.emitter, c.store, &c.manifest.Runtime.Timeouts)
	if c.gateHandler != nil {
		gate.handler = c.gateHandler
	}
	return gate.Execute(ctx, step.Gate, c.tmplCtx)
}

// executeLoop runs sub-steps repeatedly until a condition is met or max iterations reached.
func (c *CompositionExecutor) executeLoop(ctx context.Context, p *Pipeline, step *Step) error {
	if step.Loop.MaxIterations <= 0 {
		return fmt.Errorf("loop step %q: max_iterations must be > 0", step.ID)
	}

	for i := 0; i < step.Loop.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		c.tmplCtx.Iteration = i

		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: p.Metadata.Name,
			StepID:     step.ID,
			State:      event.StateLoopIteration,
			Message:    fmt.Sprintf("loop iteration %d/%d", i+1, step.Loop.MaxIterations),
		})

		// Execute sub-pipeline if specified at the loop step level
		if step.SubPipeline != "" {
			input, err := c.resolveStepInput(step)
			if err != nil {
				return err
			}
			if err := c.runSubPipeline(ctx, step.ID, step.SubPipeline, input); err != nil {
				return fmt.Errorf("loop iteration %d: %w", i, err)
			}
		}

		for j := range step.Loop.Steps {
			subStep := &step.Loop.Steps[j]
			if subStep.IsCompositionStep() {
				if err := c.executeCompositionStep(ctx, p, subStep); err != nil {
					return fmt.Errorf("loop iteration %d, step %q: %w", i, subStep.ID, err)
				}
			} else if subStep.SubPipeline != "" {
				input, err := c.resolveStepInput(subStep)
				if err != nil {
					return err
				}
				if err := c.runSubPipeline(ctx, subStep.ID, subStep.SubPipeline, input); err != nil {
					return fmt.Errorf("loop iteration %d, sub-pipeline %q: %w", i, subStep.SubPipeline, err)
				}
			}
		}

		// Check loop termination condition
		if step.Loop.Until != "" {
			condResult, err := ResolveTemplate(step.Loop.Until, c.tmplCtx)
			if err != nil {
				return fmt.Errorf("loop until condition: %w", err)
			}
			condResult = strings.TrimSpace(condResult)
			if condResult == "true" || condResult == "done" || condResult == "yes" {
				c.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: p.Metadata.Name,
					StepID:     step.ID,
					State:      event.StateLoopCompleted,
					Message:    fmt.Sprintf("loop terminated: condition met at iteration %d", i+1),
				})
				return nil
			}
		}
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: p.Metadata.Name,
		StepID:     step.ID,
		State:      event.StateLoopCompleted,
		Message:    fmt.Sprintf("loop completed: max iterations (%d) reached", step.Loop.MaxIterations),
	})

	return nil
}

// executeAggregate collects outputs and writes them to a file.
func (c *CompositionExecutor) executeAggregate(step *Step) error {
	sourceData, err := ResolveTemplate(step.Aggregate.From, c.tmplCtx)
	if err != nil {
		return fmt.Errorf("failed to resolve aggregate.from: %w", err)
	}

	var result string
	switch step.Aggregate.Strategy {
	case "concat":
		result = sourceData
	case "merge_arrays":
		result, err = mergeJSONArrays(sourceData, step.Aggregate.Key)
		if err != nil {
			return fmt.Errorf("merge_arrays failed: %w", err)
		}
	case "reduce":
		// reduce just passes through -- actual reduction logic is in the template
		result = sourceData
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

	// Store in template context
	c.tmplCtx.SetStepOutput(step.ID, []byte(result))

	c.emit(event.Event{
		Timestamp: time.Now(),
		StepID:    step.ID,
		State:     event.StateAggregateCompleted,
		Message:   fmt.Sprintf("aggregated to %s (strategy: %s)", outputPath, step.Aggregate.Strategy),
	})

	return nil
}

// executeSubPipeline loads and executes a named sub-pipeline.
// When the step has a Config with lifecycle settings, artifact inject/extract
// and timeout enforcement are applied.
func (c *CompositionExecutor) executeSubPipeline(ctx context.Context, step *Step) error {
	input, err := c.resolveStepInput(step)
	if err != nil {
		return err
	}

	// Apply lifecycle timeout from config
	execCtx, cancel := subPipelineTimeout(ctx, step.Config)
	defer cancel()

	return c.runSubPipeline(execCtx, step.ID, step.SubPipeline, input)
}

// runSubPipeline loads a pipeline by name and executes it.
// stepID is the parent step ID used to key the sub-pipeline's outputs into
// the parent template context. When empty, outputs are keyed by pipelineName.
func (c *CompositionExecutor) runSubPipeline(ctx context.Context, stepID, pipelineName, input string) error {
	if c.pipelineLoader == nil {
		return fmt.Errorf("no pipeline loader configured")
	}

	p, err := c.pipelineLoader(c.pipelinesDir, pipelineName)
	if err != nil {
		return fmt.Errorf("failed to load pipeline %q: %w", pipelineName, err)
	}

	result, err := c.seqExecutor.Execute(ctx, []*Pipeline{p}, c.manifest, input)
	if err != nil {
		return err
	}

	key := stepID
	if key == "" {
		key = pipelineName
	}

	// Register sub-pipeline outputs into parent template context.
	// Use SequenceExecutor's recorded pipeline outputs (keyed by pipeline name
	// -> artifact name) rather than reconstructing filesystem paths, since the
	// worktree layout is managed by the executor and not trivially predictable.
	outputs := c.seqExecutor.GetPipelineOutputs()[pipelineName]
	if len(outputs) > 0 {
		// Pick primary output: walk declared pipeline_outputs in step order so
		// the first declared, load-successful output wins. Fall back to terminal
		// step's first output artifact.
		var primary []byte
		if len(p.PipelineOutputs) > 0 {
			for _, s := range p.Steps {
				if primary != nil {
					break
				}
				for name, po := range p.PipelineOutputs {
					if po.Step == s.ID {
						if data, ok := outputs[name]; ok {
							primary = data
							break
						}
						if data, ok := outputs[po.Artifact]; ok {
							primary = data
							break
						}
					}
				}
			}
		}
		if primary == nil && len(p.Steps) > 0 {
			terminalStep := p.Steps[len(p.Steps)-1]
			for _, art := range terminalStep.OutputArtifacts {
				if data, ok := outputs[art.Name]; ok {
					primary = data
					break
				}
			}
		}
		if primary != nil {
			c.tmplCtx.SetStepOutput(key, primary)
		}
	}

	_ = result
	return nil
}

// resolveStepInput resolves the input template for a step.
//
// Resolution order:
//  1. step.InputRef.From      ("<step_id>.<output_name>") — looked up in
//     the template context's StepOutputs; resolves to the raw JSON value.
//  2. step.InputRef.Literal   (template string)
//  3. step.SubInput           (legacy string template)
//  4. parent input (c.tmplCtx.Input)
func (c *CompositionExecutor) resolveStepInput(step *Step) (string, error) {
	if step.InputRef != nil {
		if step.InputRef.From != "" {
			srcStep, _, ok := splitDot(step.InputRef.From)
			if !ok {
				return "", fmt.Errorf("step %q: input_ref.from %q must be '<step>.<output>'", step.ID, step.InputRef.From)
			}
			raw, has := c.tmplCtx.StepOutputs[srcStep]
			if !has {
				return "", fmt.Errorf("step %q: input_ref.from references step %q which has no recorded output", step.ID, srcStep)
			}
			return string(raw), nil
		}
		if step.InputRef.Literal != "" {
			return ResolveTemplate(step.InputRef.Literal, c.tmplCtx)
		}
	}
	if step.SubInput != "" {
		return ResolveTemplate(step.SubInput, c.tmplCtx)
	}
	return c.tmplCtx.Input, nil
}

// mergeJSONArrays takes a JSON string containing multiple arrays and merges them.
// If the input is an array of arrays, the inner arrays are flattened into one.
// If the input is already a flat array (no inner arrays), it is returned as-is.
//
// When key is non-empty, each element is expected to be a JSON object and the
// value at that key is extracted before merging. This supports the common pattern
// where sub-pipelines produce {"findings": [...], "summary": "..."} envelopes
// and only the array field should be merged.
func mergeJSONArrays(data string, key string) (string, error) {
	var elements []json.RawMessage
	if err := json.Unmarshal([]byte(data), &elements); err != nil {
		return "", fmt.Errorf("cannot parse as JSON array: %w", err)
	}

	// When a key is specified, extract that field from each element first.
	if key != "" {
		extracted := make([]json.RawMessage, 0, len(elements))
		for i, elem := range elements {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(elem, &obj); err != nil {
				return "", fmt.Errorf("element %d: expected JSON object for key extraction, got: %s", i, truncateJSON(elem))
			}
			val, ok := obj[key]
			if !ok {
				return "", fmt.Errorf("element %d: key %q not found in object", i, key)
			}
			// The extracted value should be an array; unwrap its items directly.
			var inner []json.RawMessage
			if err := json.Unmarshal(val, &inner); err != nil {
				return "", fmt.Errorf("element %d: value at key %q is not a JSON array", i, key)
			}
			extracted = append(extracted, inner...)
		}
		result, err := json.Marshal(extracted)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	// Check if any element is itself an array -- if so, flatten all.
	hasSubArray := false
	for _, elem := range elements {
		trimmed := strings.TrimSpace(string(elem))
		if len(trimmed) > 0 && trimmed[0] == '[' {
			hasSubArray = true
			break
		}
	}

	if !hasSubArray {
		// Already a flat array, return as-is
		return data, nil
	}

	// Flatten: each element must be an array
	var merged []json.RawMessage
	for _, elem := range elements {
		var inner []json.RawMessage
		if err := json.Unmarshal(elem, &inner); err != nil {
			// Not an array element -- include it directly
			merged = append(merged, elem)
			continue
		}
		merged = append(merged, inner...)
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// truncateJSON returns a short representation of a JSON value for error messages.
func truncateJSON(data json.RawMessage) string {
	s := string(data)
	if len(s) > 60 {
		return s[:60] + "..."
	}
	return s
}

// ValidateCompositionTemplates checks that all template references in a composition
// pipeline resolve to known step IDs or valid expressions.
func ValidateCompositionTemplates(p *Pipeline) []string {
	var errors []string
	stepIDs := make(map[string]bool)
	for _, step := range p.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range p.Steps {
		if step.Iterate != nil {
			errors = append(errors, validateTemplateRefs(step.Iterate.Over, stepIDs, step.ID, "iterate.over")...)
		}
		if step.Branch != nil {
			errors = append(errors, validateTemplateRefs(step.Branch.On, stepIDs, step.ID, "branch.on")...)
		}
		if step.SubInput != "" {
			errors = append(errors, validateTemplateRefs(step.SubInput, stepIDs, step.ID, "input")...)
		}
		if step.Loop != nil && step.Loop.Until != "" {
			errors = append(errors, validateTemplateRefs(step.Loop.Until, stepIDs, step.ID, "loop.until")...)
		}
		if step.Aggregate != nil {
			errors = append(errors, validateTemplateRefs(step.Aggregate.From, stepIDs, step.ID, "aggregate.from")...)
		}
	}

	return errors
}

// validateTemplateRefs checks that step references in a template exist.
func validateTemplateRefs(tmpl string, stepIDs map[string]bool, stepID, field string) []string {
	var errors []string
	matches := templatePattern.FindAllStringSubmatch(tmpl, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		expr := strings.TrimSpace(match[1])
		// Skip built-in expressions
		if expr == "input" || expr == "item" || expr == "iteration" || strings.HasPrefix(expr, "item.") {
			continue
		}
		// Must be step_id.output or step_id.output.field
		parts := strings.SplitN(expr, ".", 2)
		if len(parts) >= 2 && (parts[1] == "output" || strings.HasPrefix(parts[1], "output.")) {
			if !stepIDs[parts[0]] {
				errors = append(errors, fmt.Sprintf("step %q, %s: references unknown step %q", stepID, field, parts[0]))
			}
		}
	}
	return errors
}
