package event

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Event struct {
	Timestamp  time.Time `json:"timestamp"`
	PipelineID string    `json:"pipeline_id"`
	StepID     string    `json:"step_id,omitempty"`
	State      string    `json:"state"`
	DurationMs int64     `json:"duration_ms"`
	Message    string    `json:"message,omitempty"`
	Persona    string    `json:"persona,omitempty"`
	Artifacts  []string  `json:"artifacts,omitempty"`
	TokensUsed int       `json:"tokens_used,omitempty"`
}

type EventEmitter interface {
	Emit(event Event)
}

type NDJSONEmitter struct {
	encoder       *json.Encoder
	humanReadable bool
}

func NewNDJSONEmitter() *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:       json.NewEncoder(os.Stdout),
		humanReadable: false,
	}
}

func NewNDJSONEmitterWithHumanReadable() *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:       json.NewEncoder(os.Stdout),
		humanReadable: true,
	}
}

func (e *NDJSONEmitter) Emit(event Event) {
	if e.humanReadable {
		stateColors := map[string]string{
			"started":   "\033[36m",
			"running":   "\033[33m",
			"completed": "\033[32m",
			"failed":    "\033[31m",
			"retrying":  "\033[35m",
		}
		color := stateColors[event.State]
		if color == "" {
			color = "\033[0m"
		}
		reset := "\033[0m"

		ts := event.Timestamp.Format("15:04:05")
		if event.StepID != "" {
			fmt.Printf("%s[%s]%s %s%-10s%s %s", "\033[90m", ts, reset, color, event.State, reset, event.StepID)
			if event.Persona != "" {
				fmt.Printf(" (%s)", event.Persona)
			}
			if event.DurationMs > 0 {
				secs := float64(event.DurationMs) / 1000.0
				fmt.Printf(" %.1fs", secs)
			}
			if event.TokensUsed > 0 {
				fmt.Printf(" %dk tokens", event.TokensUsed/1000)
			}
			if len(event.Artifacts) > 0 {
				fmt.Printf(" → %v", event.Artifacts)
			}
			if event.Message != "" {
				fmt.Printf(" %s", event.Message)
			}
			fmt.Println()
		} else {
			fmt.Printf("%s[%s]%s %s%-10s%s %s %s\n", "\033[90m", ts, reset, color, event.State, reset, event.PipelineID, event.Message)
		}
	} else {
		e.encoder.Encode(event)
	}
}
