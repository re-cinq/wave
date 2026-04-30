package proposals

import (
	"errors"
	"testing"

	"github.com/recinq/wave/internal/state"
)

func newRollbackStore(t *testing.T) state.StateStore {
	t.Helper()
	store, err := state.NewStateStore(":memory:")
	if err != nil {
		t.Fatalf("NewStateStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestPriorVersion_Found(t *testing.T) {
	store := newRollbackStore(t)
	for _, v := range []int{1, 2, 3} {
		if err := store.CreatePipelineVersion(state.PipelineVersionRecord{
			PipelineName: "p", Version: v, SHA256: "x", YAMLPath: "x.yaml", Active: v == 3,
		}); err != nil {
			t.Fatalf("seed v%d: %v", v, err)
		}
	}
	prior, current, err := PriorVersion(store, "p")
	if err != nil {
		t.Fatalf("PriorVersion: %v", err)
	}
	if current.Version != 3 {
		t.Errorf("expected current=3, got %d", current.Version)
	}
	if prior.Version != 2 {
		t.Errorf("expected prior=2, got %d", prior.Version)
	}
}

func TestPriorVersion_NoActive(t *testing.T) {
	store := newRollbackStore(t)
	_, _, err := PriorVersion(store, "ghost")
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("expected ErrNoActiveVersion, got %v", err)
	}
}

func TestPriorVersion_NoPrior(t *testing.T) {
	store := newRollbackStore(t)
	if err := store.CreatePipelineVersion(state.PipelineVersionRecord{
		PipelineName: "p", Version: 1, SHA256: "x", YAMLPath: "x.yaml", Active: true,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, _, err := PriorVersion(store, "p")
	if !errors.Is(err, ErrNoPriorVersion) {
		t.Fatalf("expected ErrNoPriorVersion, got %v", err)
	}
}

func TestPriorVersion_GapAware(t *testing.T) {
	// Versions 1, 2 exist but only 1 active. Rolling back is impossible.
	// Versions 1, 5 active=5: prior should be 1, not 4 (gap).
	store := newRollbackStore(t)
	for _, vr := range []state.PipelineVersionRecord{
		{PipelineName: "p", Version: 1, SHA256: "a", YAMLPath: "1.yaml", Active: false},
		{PipelineName: "p", Version: 5, SHA256: "b", YAMLPath: "5.yaml", Active: true},
	} {
		if err := store.CreatePipelineVersion(vr); err != nil {
			t.Fatalf("seed v%d: %v", vr.Version, err)
		}
	}
	prior, current, err := PriorVersion(store, "p")
	if err != nil {
		t.Fatalf("PriorVersion: %v", err)
	}
	if current.Version != 5 || prior.Version != 1 {
		t.Errorf("expected 5->1, got %d->%d", current.Version, prior.Version)
	}
}
