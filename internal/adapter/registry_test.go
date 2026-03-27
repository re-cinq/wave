package adapter

import (
	"context"
	"sync"
	"testing"
)

func TestAdapterRegistry_ResolveRegistered(t *testing.T) {
	reg := NewAdapterRegistry()
	mock := NewMockAdapter()
	reg.Register("claude", mock)

	runner, ok := reg.Resolve("claude")
	if !ok {
		t.Fatal("expected to find registered adapter 'claude'")
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestAdapterRegistry_ResolveUnregistered(t *testing.T) {
	reg := NewAdapterRegistry()
	runner, ok := reg.Resolve("nonexistent")
	if ok {
		t.Fatal("expected ok=false for unregistered adapter")
	}
	if runner != nil {
		t.Fatal("expected nil runner for unregistered adapter")
	}
}

func TestAdapterRegistry_Names(t *testing.T) {
	reg := NewAdapterRegistry()
	reg.Register("claude", NewMockAdapter())
	reg.Register("codex", NewMockAdapter())

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["claude"] || !nameSet["codex"] {
		t.Errorf("expected claude and codex, got %v", names)
	}
}

func TestAdapterRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewAdapterRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		name := "adapter"
		go func() {
			defer wg.Done()
			reg.Register(name, NewMockAdapter())
		}()
		go func() {
			defer wg.Done()
			reg.Resolve(name)
		}()
	}
	wg.Wait()
}

func TestAdapterRegistry_RegisterOverwrite(t *testing.T) {
	reg := NewAdapterRegistry()
	mock1 := NewMockAdapter()
	mock2 := NewMockAdapter(WithExitCode(42))

	reg.Register("claude", mock1)
	reg.Register("claude", mock2)

	runner, ok := reg.Resolve("claude")
	if !ok {
		t.Fatal("expected to find adapter")
	}
	// Verify it's the second mock (exit code 42)
	result, err := runner.Run(context.Background(), AdapterRunConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42 from overwritten adapter, got %d", result.ExitCode)
	}
}
