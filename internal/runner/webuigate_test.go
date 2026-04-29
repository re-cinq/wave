package runner

import (
	"context"
	"testing"
	"time"

	"github.com/recinq/wave/internal/pipeline"
)

// fakeRegistrar captures the gate that was registered and exposes a channel
// the test can use to deliver a decision. Mirrors the runtime contract the
// real webui.GateRegistry implements.
type fakeRegistrar struct {
	registered  *WebUIGate
	registeredR string
	registeredS string
	resp        chan *WebUIGateDecision
	removed     []string
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{resp: make(chan *WebUIGateDecision, 1)}
}

func (r *fakeRegistrar) Register(runID, stepID string, gate *WebUIGate) chan *WebUIGateDecision {
	r.registered = gate
	r.registeredR = runID
	r.registeredS = stepID
	return r.resp
}

func (r *fakeRegistrar) Remove(runID string) { r.removed = append(r.removed, runID) }

func TestWebUIGate_FindChoiceByKey(t *testing.T) {
	g := &WebUIGate{
		Choices: []WebUIGateChoice{
			{Key: "a", Label: "Approve", Target: "next"},
			{Key: "r", Label: "Reject", Target: "_fail"},
		},
	}
	if c := g.FindChoiceByKey("a"); c == nil || c.Label != "Approve" || c.Target != "next" {
		t.Errorf("FindChoiceByKey(a) = %+v, want Approve/next", c)
	}
	if c := g.FindChoiceByKey("r"); c == nil || c.Target != "_fail" {
		t.Errorf("FindChoiceByKey(r) = %+v, want target _fail", c)
	}
	if c := g.FindChoiceByKey("z"); c != nil {
		t.Errorf("FindChoiceByKey(z) = %+v, want nil", c)
	}
	var empty *WebUIGate
	if c := empty.FindChoiceByKey("a"); c != nil {
		t.Errorf("nil receiver FindChoiceByKey = %+v, want nil", c)
	}
}

func TestWebUIGateHandler_Prompt_DeliversDecision(t *testing.T) {
	reg := newFakeRegistrar()
	h := NewWebUIGateHandler("run-1", reg)

	gate := &pipeline.GateConfig{
		Type:          "approval",
		Message:       "Proceed?",
		Prompt:        "Choose one:",
		RuntimeStepID: "review",
		Choices: []pipeline.GateChoice{
			{Key: "a", Label: "Approve", Target: "implement"},
			{Key: "r", Label: "Reject", Target: "_fail"},
		},
		Default:  "a",
		Freeform: true,
	}

	// Deliver a decision before Prompt blocks.
	reg.resp <- &WebUIGateDecision{Choice: "a", Label: "Approve", Target: "implement", Text: "lgtm"}

	got, err := h.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("Prompt returned err: %v", err)
	}
	if got.Choice != "a" || got.Label != "Approve" || got.Target != "implement" || got.Text != "lgtm" {
		t.Errorf("Prompt decision = %+v, missing fields", got)
	}
	if got.Timestamp.IsZero() {
		t.Errorf("Prompt did not stamp Timestamp")
	}

	// The registrar saw the projected gate.
	if reg.registeredR != "run-1" || reg.registeredS != "review" {
		t.Errorf("registrar got run=%q step=%q, want run-1/review", reg.registeredR, reg.registeredS)
	}
	if reg.registered == nil || reg.registered.Type != "approval" || reg.registered.Default != "a" || !reg.registered.Freeform {
		t.Errorf("projected gate missing fields: %+v", reg.registered)
	}
	if len(reg.registered.Choices) != 2 || reg.registered.Choices[1].Target != "_fail" {
		t.Errorf("projected gate choices wrong: %+v", reg.registered.Choices)
	}
}

func TestWebUIGateHandler_Prompt_ContextCancelled(t *testing.T) {
	reg := newFakeRegistrar()
	h := NewWebUIGateHandler("run-2", reg)

	ctx, cancel := context.WithCancel(context.Background())
	gate := &pipeline.GateConfig{Type: "approval", RuntimeStepID: "review"}

	done := make(chan struct{})
	go func() {
		_, err := h.Prompt(ctx, gate)
		if err == nil {
			t.Errorf("Prompt should error when context is cancelled")
		}
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Prompt did not return after context cancellation")
	}
	if len(reg.removed) != 1 || reg.removed[0] != "run-2" {
		t.Errorf("expected Remove(run-2) on cancellation, got %v", reg.removed)
	}
}

func TestWebUIGateHandler_Prompt_ChannelClosed(t *testing.T) {
	reg := newFakeRegistrar()
	h := NewWebUIGateHandler("run-3", reg)

	gate := &pipeline.GateConfig{Type: "approval", RuntimeStepID: "review"}
	close(reg.resp) // closed channel returns zero value (nil pointer)

	_, err := h.Prompt(context.Background(), gate)
	if err == nil {
		t.Fatal("Prompt should error when registrar channel returns nil decision")
	}
}
