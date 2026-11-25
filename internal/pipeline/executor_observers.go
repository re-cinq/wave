package pipeline

import (
	"context"
	"time"

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
