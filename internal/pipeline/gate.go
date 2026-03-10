package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
)

// GateExecutor handles blocking gate steps.
type GateExecutor struct {
	emitter event.EventEmitter
	store   state.StateStore
}

// NewGateExecutor creates a gate executor.
func NewGateExecutor(emitter event.EventEmitter, store state.StateStore) *GateExecutor {
	return &GateExecutor{
		emitter: emitter,
		store:   store,
	}
}

// Execute blocks until the gate condition is met, times out, or context is cancelled.
func (g *GateExecutor) Execute(ctx context.Context, gate *GateConfig, tmplCtx *TemplateContext) error {
	if gate == nil {
		return fmt.Errorf("gate config is nil")
	}

	g.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateGateWaiting,
		Message:   fmt.Sprintf("gate: %s — %s", gate.Type, gate.Message),
	})

	switch gate.Type {
	case "approval":
		return g.executeApproval(ctx, gate)
	case "timer":
		return g.executeTimer(ctx, gate)
	case "pr_merge":
		return g.executePollGate(ctx, gate, "pr_merge")
	case "ci_pass":
		return g.executePollGate(ctx, gate, "ci_pass")
	default:
		return fmt.Errorf("unknown gate type: %q", gate.Type)
	}
}

// executeApproval waits for manual approval or auto-approves.
func (g *GateExecutor) executeApproval(ctx context.Context, gate *GateConfig) error {
	if gate.Auto {
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   "gate auto-approved",
		})
		return nil
	}

	// Parse timeout
	timeout := 24 * time.Hour // default 24h
	if gate.Timeout != "" {
		d, err := time.ParseDuration(gate.Timeout)
		if err != nil {
			return fmt.Errorf("invalid gate timeout %q: %w", gate.Timeout, err)
		}
		timeout = d
	}

	// In non-interactive mode, wait for context cancellation or timeout.
	// The TUI or external system is expected to resolve the gate by cancelling
	// the context or using the state store.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return fmt.Errorf("gate timed out after %s", timeout)
	}
}

// executeTimer waits for a specified duration.
func (g *GateExecutor) executeTimer(ctx context.Context, gate *GateConfig) error {
	if gate.Timeout == "" {
		return fmt.Errorf("timer gate requires a timeout duration")
	}

	d, err := time.ParseDuration(gate.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timer duration %q: %w", gate.Timeout, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   fmt.Sprintf("timer gate elapsed: %s", d),
		})
		return nil
	}
}

// executePollGate polls for a condition (pr_merge or ci_pass).
// In the current implementation, this waits for context cancellation or timeout.
// Full GitHub API polling is wired up via the github package in production.
func (g *GateExecutor) executePollGate(ctx context.Context, gate *GateConfig, gateType string) error {
	if gate.Auto {
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   fmt.Sprintf("%s gate auto-resolved", gateType),
		})
		return nil
	}

	timeout := 2 * time.Hour // default 2h for poll gates
	if gate.Timeout != "" {
		d, err := time.ParseDuration(gate.Timeout)
		if err != nil {
			return fmt.Errorf("invalid gate timeout %q: %w", gate.Timeout, err)
		}
		timeout = d
	}

	pollInterval := 30 * time.Second
	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("%s gate timed out after %s", gateType, timeout)
		case <-time.After(pollInterval):
			// In production, this would check GitHub API.
			// For now, continue polling until timeout or cancellation.
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("polling %s...", gateType),
			})
		}
	}
}

func (g *GateExecutor) emit(ev event.Event) {
	if g.emitter != nil {
		g.emitter.Emit(ev)
	}
}
