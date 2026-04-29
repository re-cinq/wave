// webuigate.go bridges the pipeline executor's GateHandler contract to the
// webui's transport-layer GateRegistry without forcing the webui package to
// import internal/pipeline directly.
//
// The runner package already imports both internal/pipeline and is a stable
// neutral point between the HTTP transport (webui) and the domain executor
// (pipeline). Defining the bridge here lets internal/webui depend only on
// internal/runner — which it already does for Options, Detach, and
// LaunchInProcess — instead of pulling in the pipeline package.
//
// Type flow:
//
//	pipeline.GateConfig   ──projected──▶  WebUIGate         (sent to webui)
//	WebUIGateDecision     ──translated──▶ pipeline.GateDecision (returned to executor)
//
// The webui receives plain runner types it can serialize to JSON without ever
// touching pipeline domain shapes.
package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/pipeline"
)

// WebUIGateChoice mirrors pipeline.GateChoice with the subset of fields the
// webui actually presents to clients. Defined here (not in webui) so the
// runner-side adapter can populate it without a webui import cycle.
type WebUIGateChoice struct {
	Key    string
	Label  string
	Target string
}

// WebUIGate carries the data the webui needs to render and resolve a pending
// gate. RuntimeStepID matches pipeline.GateConfig.RuntimeStepID — the
// executor sets it before invoking GateHandler.Prompt so the HTTP endpoint
// can verify the path's {step} segment.
type WebUIGate struct {
	RuntimeStepID string
	Type          string
	Message       string
	Prompt        string
	Choices       []WebUIGateChoice
	Freeform      bool
	Default       string
}

// FindChoiceByKey returns the matching WebUIGateChoice or nil. Mirrors the
// helper on pipeline.GateConfig so webui handlers can keep their existing
// lookup logic without touching pipeline types.
func (g *WebUIGate) FindChoiceByKey(key string) *WebUIGateChoice {
	if g == nil {
		return nil
	}
	for i := range g.Choices {
		if g.Choices[i].Key == key {
			return &g.Choices[i]
		}
	}
	return nil
}

// WebUIGateDecision carries the user's choice back from the webui registry
// to the pipeline executor. NewWebUIGateHandler converts this into a
// pipeline.GateDecision before returning it from Prompt.
type WebUIGateDecision struct {
	Choice string
	Label  string
	Text   string
	Target string
}

// GateChannelRegistrar is the narrow interface webui's GateRegistry exposes
// to the runner-side gate handler. The registrar parks an incoming gate
// against a runID and returns a channel that delivers the human's decision
// once an HTTP client resolves it.
type GateChannelRegistrar interface {
	Register(runID, stepID string, gate *WebUIGate) chan *WebUIGateDecision
	Remove(runID string)
}

// WebUIGateHandler implements pipeline.GateHandler by registering the gate
// in a shared GateRegistry and blocking until the HTTP endpoint resolves it.
// One instance is created per pipeline run.
type WebUIGateHandler struct {
	runID    string
	registry GateChannelRegistrar
}

// NewWebUIGateHandler creates a gate handler for a specific pipeline run
// that delegates to the supplied registrar. The handler converts pipeline
// types to/from the runner's webui-facing types so the webui package never
// imports internal/pipeline.
func NewWebUIGateHandler(runID string, registry GateChannelRegistrar) pipeline.GateHandler {
	return &WebUIGateHandler{runID: runID, registry: registry}
}

// Prompt registers the gate in the registry and blocks until a decision
// arrives from the HTTP endpoint or the context is cancelled.
func (h *WebUIGateHandler) Prompt(ctx context.Context, gate *pipeline.GateConfig) (*pipeline.GateDecision, error) {
	view := projectGateConfig(gate)
	ch := h.registry.Register(h.runID, view.RuntimeStepID, view)

	select {
	case <-ctx.Done():
		h.registry.Remove(h.runID)
		return nil, ctx.Err()
	case decision := <-ch:
		if decision == nil {
			return nil, fmt.Errorf("gate decision channel closed without a decision")
		}
		return &pipeline.GateDecision{
			Choice:    decision.Choice,
			Label:     decision.Label,
			Text:      decision.Text,
			Target:    decision.Target,
			Timestamp: time.Now(),
		}, nil
	}
}

// projectGateConfig copies the subset of pipeline.GateConfig fields the
// webui needs into a transport-friendly WebUIGate.
func projectGateConfig(gate *pipeline.GateConfig) *WebUIGate {
	if gate == nil {
		return &WebUIGate{}
	}
	choices := make([]WebUIGateChoice, len(gate.Choices))
	for i, c := range gate.Choices {
		choices[i] = WebUIGateChoice{Key: c.Key, Label: c.Label, Target: c.Target}
	}
	return &WebUIGate{
		RuntimeStepID: gate.RuntimeStepID,
		Type:          gate.Type,
		Message:       gate.Message,
		Prompt:        gate.Prompt,
		Choices:       choices,
		Freeform:      gate.Freeform,
		Default:       gate.Default,
	}
}
