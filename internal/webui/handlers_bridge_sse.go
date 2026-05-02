package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleBridgeActivitySSE handles GET /api/bridge/stream — a real-time stream
// of pipeline activity events shown in the bridge page's live-activity panel.
func (s *Server) handleBridgeActivitySSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial heartbeat to establish connection.
	fmt.Fprintf(w, "retry: 5000\n\n")
	flusher.Flush()

	// Subscribe to the broker's event channel.
	ch := s.realtime.broker.Subscribe()
	defer s.realtime.broker.Unsubscribe(ch)

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			// Format: "time source message"
			ae := bridgeEventFromSSE(ev)
			data, _ := json.Marshal(ae)
			fmt.Fprintf(w, "event: activity\ndata: %s\n\n", data)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// bridgeActivityEvent is the JSON shape sent to the bridge live-activity panel.
type bridgeActivityEvent struct {
	Time        string `json:"time"`
	Source      string `json:"source"`
	SourceClass string `json:"sourceClass"`
	Message     string `json:"message"`
}

// bridgeEventFromSSE converts an internal SSEEvent to a bridge activity event.
// It parses event.Type and event.Data to produce human-readable activity text
// with color class based on pipeline/step outcome.
func bridgeEventFromSSE(ev SSEEvent) bridgeActivityEvent {
	ae := bridgeActivityEvent{
		Time:   time.Now().Format("15:04:05"),
		Source: "wave",
	}

	switch ev.Event {
	case "run_step_completed":
		// ev.Data is JSON: {"run_id","pipeline","step","outcome"}
		var d struct {
			Pipeline string `json:"pipeline"`
			Step     string `json:"step"`
			Outcome  string `json:"outcome"`
		}
		if json.Unmarshal([]byte(ev.Data), &d) == nil {
			ae.Source = d.Pipeline
			ae.SourceClass = colorForOutcome(d.Outcome)
			ae.Message = fmt.Sprintf("step %s \u2014 %s", d.Step, outcomeLabel(d.Outcome))
		} else {
			ae.Message = ev.Data
		}
	case "run_completed":
		ae.Source = "wave"
		ae.SourceClass = "g"
		ae.Message = ev.Data
	case "contract_passed":
		ae.Source = ev.Source()
		ae.SourceClass = "g"
		ae.Message = fmt.Sprintf("contract %s \u2014 passed", ev.Data)
	case "contract_failed":
		ae.Source = ev.Source()
		ae.SourceClass = "r"
		ae.Message = fmt.Sprintf("contract %s \u2014 %s", ev.Data, outcomeLabel("failed"))
	case "proposal_created":
		ae.Source = "pipeline-evolve"
		ae.SourceClass = "y"
		ae.Message = fmt.Sprintf("proposal created \u2014 %s", ev.Data)
	case "scheduler_event":
		ae.Source = "scheduler"
		ae.SourceClass = "b"
		ae.Message = ev.Data
	default:
		ae.Message = fmt.Sprintf("%s \u2014 %s", ev.Event, ev.Data)
	}
	return ae
}

// Source returns the pipeline name extracted from the event's ID field.
// The SSEEvent.ID field is used as the pipeline/source identifier.
func (ev SSEEvent) Source() string {
	return ev.ID
}

func colorForOutcome(outcome string) string {
	switch outcome {
	case "passed", "success":
		return "g"
	case "failed", "rejected":
		return "r"
	case "rework":
		return "y"
	default:
		return "b"
	}
}

func outcomeLabel(outcome string) string {
	switch outcome {
	case "passed", "success":
		return "passed"
	case "failed", "rejected":
		return "failed"
	case "rework":
		return "rework"
	default:
		return outcome
	}
}