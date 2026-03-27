package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"
)

// HookRunner executes lifecycle hooks for pipeline events.
type HookRunner interface {
	RunHooks(ctx context.Context, evt HookEvent) ([]HookResult, error)
}

// DefaultHookRunner filters and executes hooks sequentially in definition order.
type DefaultHookRunner struct {
	hooks    []LifecycleHookDef
	matchers []*Matcher
	emitter  event.EventEmitter
}

// NewHookRunner creates a HookRunner from a list of hook definitions.
// Matchers are pre-compiled at construction time.
func NewHookRunner(hooks []LifecycleHookDef, emitter event.EventEmitter) (*DefaultHookRunner, error) {
	matchers := make([]*Matcher, len(hooks))
	for i, h := range hooks {
		m, err := NewMatcher(h.Matcher)
		if err != nil {
			return nil, fmt.Errorf("hook %q: invalid matcher %q: %w", h.Name, h.Matcher, err)
		}
		matchers[i] = m
	}
	return &DefaultHookRunner{
		hooks:    hooks,
		matchers: matchers,
		emitter:  emitter,
	}, nil
}

// RunHooks executes all matching hooks for the given event.
// For blocking hooks, returns an error on first failure.
// For non-blocking hooks, failures are logged but do not affect the result.
func (r *DefaultHookRunner) RunHooks(ctx context.Context, evt HookEvent) ([]HookResult, error) {
	var results []HookResult

	for i, hook := range r.hooks {
		// Filter by event type
		if hook.Event != evt.Type {
			continue
		}

		// Filter by step matcher (only for step-level events)
		if evt.StepID != "" && !r.matchers[i].Match(evt.StepID) {
			continue
		}

		// Emit hook started event
		r.emitHookEvent(evt.PipelineID, evt.StepID, event.StateHookStarted, hook.Name, "")

		start := time.Now()
		result := r.executeHook(ctx, &hook, evt)
		result.Duration = time.Since(start)
		results = append(results, result)

		if result.Decision == DecisionProceed || result.Decision == DecisionSkip {
			r.emitHookEvent(evt.PipelineID, evt.StepID, event.StateHookPassed, hook.Name, "")
			continue
		}

		// Hook failed or blocked
		if hook.IsBlocking() {
			// Check fail-open behavior
			if result.Err != nil && hook.IsFailOpen() {
				r.emitHookEvent(evt.PipelineID, evt.StepID, event.StateHookPassed, hook.Name,
					fmt.Sprintf("fail-open: %s", result.Reason))
				continue
			}
			r.emitHookEvent(evt.PipelineID, evt.StepID, event.StateHookFailed, hook.Name, result.Reason)
			return results, fmt.Errorf("blocking hook %q failed: %s", hook.Name, result.Reason)
		}

		// Non-blocking: log and continue
		r.emitHookEvent(evt.PipelineID, evt.StepID, event.StateHookFailed, hook.Name,
			fmt.Sprintf("non-blocking: %s", result.Reason))
	}

	return results, nil
}

// executeHook dispatches to the appropriate hook type executor.
func (r *DefaultHookRunner) executeHook(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	switch hook.Type {
	case HookTypeCommand:
		return executeCommand(ctx, hook, evt)
	case HookTypeHTTP:
		return executeHTTP(ctx, hook, evt)
	case HookTypeLLMJudge:
		return executeLLMJudge(ctx, hook, evt)
	case HookTypeScript:
		return executeScript(ctx, hook, evt)
	default:
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("unknown hook type: %s", hook.Type),
			Err:      fmt.Errorf("unknown hook type: %s", hook.Type),
		}
	}
}

func (r *DefaultHookRunner) emitHookEvent(pipelineID, stepID, state, hookName, message string) {
	if r.emitter == nil {
		return
	}
	msg := fmt.Sprintf("hook=%s", hookName)
	if message != "" {
		msg = fmt.Sprintf("hook=%s %s", hookName, message)
	}
	r.emitter.Emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     stepID,
		State:      state,
		Message:    msg,
	})
}
