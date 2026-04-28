package pipeline

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/state"
)

const terminalHookTimeout = 30 * time.Second

// runTerminalHooks executes lifecycle hooks with a fresh, detached context.
// Terminal events fire after the pipeline context may already be cancelled.
func (e *DefaultPipelineExecutor) runTerminalHooks(evt hooks.HookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), terminalHookTimeout)
	defer cancel()
	if e.hookRunner != nil {
		e.hookRunner.RunHooks(ctx, evt)
	}
	e.fireWebhooks(ctx, evt)
}

// trace emits a structured NDJSON trace event when debug tracing is enabled.

func (e *DefaultPipelineExecutor) checkRelayCompaction(ctx context.Context, execution *PipelineExecution, step *Step, tokensUsed int, workspacePath string, chatHistory string) error {
	if e.relayMonitor == nil {
		return nil // No relay monitor configured
	}

	relayConfig := execution.Manifest.Runtime.Relay
	thresholdPercent := relayConfig.TokenThresholdPercent

	// Allow step-level override via handover.compaction.trigger
	if step.Handover.Compaction.Trigger != "" {
		// Parse trigger like "token_limit_80%"
		var pct int
		if _, err := fmt.Sscanf(step.Handover.Compaction.Trigger, "token_limit_%d%%", &pct); err == nil {
			thresholdPercent = pct
		}
	}

	if thresholdPercent == 0 {
		// No threshold configured, skip compaction check
		return nil
	}

	// Check if we should compact based on token usage
	if !e.relayMonitor.ShouldCompact(tokensUsed, thresholdPercent) {
		return nil
	}

	pipelineID := execution.Status.ID

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "compacting",
		TokensUsed: tokensUsed,
		Message:    fmt.Sprintf("Token threshold exceeded (%d tokens, %d%% threshold), triggering compaction", tokensUsed, thresholdPercent),
	})

	// Get summarizer persona from config (step-level takes precedence over runtime-level)
	summarizerName := relayConfig.SummarizerPersona
	if step.Handover.Compaction.Persona != "" {
		summarizerName = step.Handover.Compaction.Persona
	}
	if summarizerName == "" {
		summarizerName = "summarizer" // Default fallback
	}

	// Load summarizer persona for system prompt
	summarizerPersona := execution.Manifest.GetPersona(summarizerName)
	systemPrompt := ""
	compactPrompt := "Summarize this conversation history concisely, preserving key context, decisions, and progress:"

	if summarizerPersona != nil {
		if summarizerPersona.SystemPromptFile != "" {
			if data, err := os.ReadFile(summarizerPersona.SystemPromptFile); err == nil {
				systemPrompt = string(data)
			}
		}
	}

	// Trigger compaction
	summary, err := e.relayMonitor.Compact(ctx, chatHistory, systemPrompt, compactPrompt, workspacePath)
	if err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "compacted",
		Message:    fmt.Sprintf("Checkpoint written to %s/checkpoint.md (%d chars)", workspacePath, len(summary)),
	})

	if e.logger != nil {
		_ = e.logger.LogToolCall(pipelineID, step.ID, "relay.Compact", fmt.Sprintf("tokens=%d summary_len=%d persona=%s", tokensUsed, len(summary), summarizerName))
	}

	return nil
}

// trackStepDeliverables automatically tracks deliverables produced by a completed step

func (e *DefaultPipelineExecutor) fireWebhooks(ctx context.Context, evt hooks.HookEvent) {
	if e.webhookRunner != nil {
		e.webhookRunner.FireWebhooks(ctx, evt)
	}
}

// webhookStoreAdapter bridges the hooks.WebhookStore interface to the state store,
// avoiding a direct state→hooks import cycle.
type webhookStoreAdapter struct {
	store state.StateStore
}

func (a *webhookStoreAdapter) RecordWebhookDeliveryResult(d *hooks.WebhookDeliveryRecord) error {
	return a.store.RecordWebhookDelivery(&state.WebhookDelivery{
		WebhookID:      d.WebhookID,
		RunID:          d.RunID,
		Event:          d.Event,
		StatusCode:     d.StatusCode,
		ResponseTimeMs: d.ResponseTimeMs,
		Error:          d.Error,
	})
}

// resolveWorkspaceStepRefs resolves {{ steps.<step-id>.artifacts.<artifact-name>.<json-path> }}
// and {{ steps.<step-id>.output.<field> }} references in a workspace config field.
// This is called just before workspace creation so that branch/base fields can reference
// outputs from prior steps (e.g. a PR's headRefName fetched by a preceding step).
//
// Supported patterns:
//   - {{ steps.STEP_ID.artifacts.ARTIFACT_NAME.json.path }} — read a JSON field from a named artifact
//   - {{ steps.STEP_ID.output.json.path }} — read a JSON field from the first artifact of the step
//
// Returns an error if a referenced step/artifact does not exist or the JSON path fails.
