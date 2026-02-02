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
	StepID     string    `json:"step_id"`
	State      string    `json:"state"`
	DurationMs int64     `json:"duration_ms"`
	Message    string    `json:"message"`
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
		// Human-readable format with colored output
		fmt.Printf("\033[36m[%s]\033[0m \033[33mPipeline:%s\033[0m \033[32mStep:%s\033[0m \033[35mState:%s\033[0m \033[34mDuration:%dms\033[0m %s\n",
			event.Timestamp.Format("2006-01-02 15:04:05"),
			event.PipelineID,
			event.StepID,
			event.State,
			event.DurationMs,
			event.Message)
	} else {
		e.encoder.Encode(event)
	}
}
