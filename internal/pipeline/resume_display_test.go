package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// capturingEmitter records all emitted events for test assertions.
type capturingEmitter struct {
	mu     sync.Mutex
	events []event.Event
}

func (e *capturingEmitter) Emit(evt event.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, evt)
}

func (e *capturingEmitter) Events() []event.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	copied := make([]event.Event, len(e.events))
	copy(copied, e.events)
	return copied
}

func TestResumeFromStep_SyntheticCompletionEvents(t *testing.T) {
	tests := []struct {
		name              string
		steps             []Step
		fromStep          string
		setupWorkspace    func(t *testing.T, tempDir string)
		wantSynthetic     int      // expected number of synthetic completion events
		wantSyntheticIDs  []string // expected step IDs in synthetic events
		wantPersonas      []string // expected personas in synthetic events
	}{
		{
			name: "resume from step 3 of 5 emits 2 synthetic completions",
			steps: []Step{
				{ID: "step-1", Persona: "analyst", OutputArtifacts: []ArtifactDef{{Name: "out", Path: "artifact.json"}}},
				{ID: "step-2", Persona: "researcher", Dependencies: []string{"step-1"}, OutputArtifacts: []ArtifactDef{{Name: "out", Path: "artifact.json"}}},
				{ID: "step-3", Persona: "writer", Dependencies: []string{"step-2"}, Exec: ExecConfig{Source: "write"}},
				{ID: "step-4", Persona: "reviewer", Dependencies: []string{"step-3"}, Exec: ExecConfig{Source: "review"}},
				{ID: "step-5", Persona: "publisher", Dependencies: []string{"step-4"}, Exec: ExecConfig{Source: "publish"}},
			},
			fromStep: "step-3",
			setupWorkspace: func(t *testing.T, tempDir string) {
				for _, sid := range []string{"step-1", "step-2"} {
					dir := filepath.Join(tempDir, ".wave/workspaces/test-pipeline", sid)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "artifact.json"), []byte("{}"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			},
			wantSynthetic:    2,
			wantSyntheticIDs: []string{"step-1", "step-2"},
			wantPersonas:     []string{"analyst", "researcher"},
		},
		{
			name: "resume from step 1 emits 0 synthetic completions",
			steps: []Step{
				{ID: "step-1", Persona: "analyst", Exec: ExecConfig{Source: "analyze"}},
				{ID: "step-2", Persona: "researcher", Dependencies: []string{"step-1"}, Exec: ExecConfig{Source: "research"}},
			},
			fromStep: "step-1",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// No prior workspaces to set up
			},
			wantSynthetic:    0,
			wantSyntheticIDs: nil,
			wantPersonas:     nil,
		},
		{
			name: "resume from step 2 of 3 emits 1 synthetic completion with correct persona",
			steps: []Step{
				{ID: "gather", Persona: "github-analyst", OutputArtifacts: []ArtifactDef{{Name: "data", Path: "data.json"}}},
				{ID: "process", Persona: "data-engineer", Dependencies: []string{"gather"}, Exec: ExecConfig{Source: "process"}},
				{ID: "report", Persona: "writer", Dependencies: []string{"process"}, Exec: ExecConfig{Source: "report"}},
			},
			fromStep: "process",
			setupWorkspace: func(t *testing.T, tempDir string) {
				dir := filepath.Join(tempDir, ".wave/workspaces/test-pipeline/gather")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantSynthetic:    1,
			wantSyntheticIDs: []string{"gather"},
			wantPersonas:     []string{"github-analyst"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}

			tt.setupWorkspace(t, tempDir)

			emitter := &capturingEmitter{}
			mockAdapter := adapter.NewMockAdapter()
			executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(emitter))
			manager := NewResumeManager(executor)

			p := &Pipeline{
				Metadata: PipelineMetadata{Name: "test-pipeline"},
				Steps:    tt.steps,
			}
			m := &manifest.Manifest{
				Metadata: manifest.Metadata{Name: "test"},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude", Mode: "headless"},
				},
				Personas: map[string]manifest.Persona{},
				Runtime: manifest.Runtime{
					WorkspaceRoot:     tempDir,
					DefaultTimeoutMin: 5,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// ResumeFromStep will emit synthetic events then try to execute steps.
			// Execution will fail (mock adapter), but we only care about the synthetic events.
			_ = manager.ResumeFromStep(ctx, p, m, "test-input", tt.fromStep, true)

			// Filter for synthetic completion events (completed events with "Completed in prior run" message)
			events := emitter.Events()
			var syntheticEvents []event.Event
			for _, evt := range events {
				if evt.State == "completed" && evt.Message == "Completed in prior run" {
					syntheticEvents = append(syntheticEvents, evt)
				}
			}

			if len(syntheticEvents) != tt.wantSynthetic {
				t.Errorf("expected %d synthetic completion events, got %d", tt.wantSynthetic, len(syntheticEvents))
				for i, evt := range syntheticEvents {
					t.Logf("  synthetic[%d]: stepID=%s persona=%s", i, evt.StepID, evt.Persona)
				}
			}

			for i, wantID := range tt.wantSyntheticIDs {
				if i >= len(syntheticEvents) {
					break
				}
				if syntheticEvents[i].StepID != wantID {
					t.Errorf("synthetic event %d: expected stepID %q, got %q", i, wantID, syntheticEvents[i].StepID)
				}
			}

			for i, wantPersona := range tt.wantPersonas {
				if i >= len(syntheticEvents) {
					break
				}
				if syntheticEvents[i].Persona != wantPersona {
					t.Errorf("synthetic event %d: expected persona %q, got %q", i, wantPersona, syntheticEvents[i].Persona)
				}
			}
		})
	}
}

func TestLookupStepPersona(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "navigator"},
			{ID: "step-b", Persona: "auditor"},
			{ID: "step-c", Persona: ""},
		},
	}

	tests := []struct {
		stepID      string
		wantPersona string
	}{
		{"step-a", "navigator"},
		{"step-b", "auditor"},
		{"step-c", ""},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		got := manager.lookupStepPersona(p, tt.stepID)
		if got != tt.wantPersona {
			t.Errorf("lookupStepPersona(%q) = %q, want %q", tt.stepID, got, tt.wantPersona)
		}
	}
}
