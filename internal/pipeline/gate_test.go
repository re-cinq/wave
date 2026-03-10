package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestGateExecutor_Approval_Auto(t *testing.T) {
	emitter := &testEmitter{}
	gate := NewGateExecutor(emitter, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "approval", Auto: true}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.hasState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Timer(t *testing.T) {
	emitter := &testEmitter{}
	gate := NewGateExecutor(emitter, nil)

	ctx := context.Background()
	start := time.Now()
	err := gate.Execute(ctx, &GateConfig{Type: "timer", Timeout: "100ms"}, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 90*time.Millisecond {
		t.Errorf("timer resolved too quickly: %v", elapsed)
	}

	if !emitter.hasState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Timer_MissingTimeout(t *testing.T) {
	gate := NewGateExecutor(nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "timer"}, nil)
	if err == nil {
		t.Fatal("expected error for timer without timeout")
	}
}

func TestGateExecutor_Approval_Timeout(t *testing.T) {
	gate := NewGateExecutor(nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "approval", Timeout: "50ms"}, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestGateExecutor_Approval_ContextCancel(t *testing.T) {
	gate := NewGateExecutor(nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := gate.Execute(ctx, &GateConfig{Type: "approval"}, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestGateExecutor_PollGate_Auto(t *testing.T) {
	emitter := &testEmitter{}
	gate := NewGateExecutor(emitter, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "pr_merge", Auto: true}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.hasState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_UnknownType(t *testing.T) {
	gate := NewGateExecutor(nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "unknown"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown gate type")
	}
}

func TestGateExecutor_NilConfig(t *testing.T) {
	gate := NewGateExecutor(nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}
