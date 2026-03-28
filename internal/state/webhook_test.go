package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWebhook(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:    "deploy-notifier",
		URL:     "https://example.com/webhook",
		Events:  []string{"run_completed", "step_failed"},
		Matcher: "^deploy-.*",
		Headers: map[string]string{"X-Custom": "value"},
		Secret:  "s3cret",
		Active:  true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)
	assert.NotZero(t, id)
	assert.Equal(t, id, wh.ID)
	assert.False(t, wh.CreatedAt.IsZero())
	assert.False(t, wh.UpdatedAt.IsZero())
}

func TestCreateWebhook_MinimalFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:   "minimal",
		URL:    "https://example.com/hook",
		Active: true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)
	assert.NotZero(t, id)

	got, err := store.GetWebhook(id)
	require.NoError(t, err)
	assert.Equal(t, "minimal", got.Name)
	assert.Equal(t, "https://example.com/hook", got.URL)
	assert.Empty(t, got.Events)
	assert.Empty(t, got.Matcher)
	assert.Empty(t, got.Headers)
	assert.Empty(t, got.Secret)
	assert.True(t, got.Active)
}

func TestGetWebhook(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:    "test-hook",
		URL:     "https://example.com/wh",
		Events:  []string{"run_completed"},
		Matcher: "",
		Headers: map[string]string{"Authorization": "Bearer tok"},
		Secret:  "hmac-secret",
		Active:  true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	got, err := store.GetWebhook(id)
	require.NoError(t, err)

	assert.Equal(t, id, got.ID)
	assert.Equal(t, "test-hook", got.Name)
	assert.Equal(t, "https://example.com/wh", got.URL)
	assert.Equal(t, []string{"run_completed"}, got.Events)
	assert.Empty(t, got.Matcher)
	assert.Equal(t, map[string]string{"Authorization": "Bearer tok"}, got.Headers)
	assert.Equal(t, "hmac-secret", got.Secret)
	assert.True(t, got.Active)
}

func TestGetWebhook_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.GetWebhook(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook not found")
}

func TestListWebhooks(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Empty list initially
	webhooks, err := store.ListWebhooks()
	require.NoError(t, err)
	assert.Empty(t, webhooks)

	// Create several webhooks
	for _, name := range []string{"hook-a", "hook-b", "hook-c"} {
		_, err := store.CreateWebhook(&Webhook{
			Name:   name,
			URL:    "https://example.com/" + name,
			Events: []string{"run_completed"},
			Active: true,
		})
		require.NoError(t, err)
	}

	webhooks, err = store.ListWebhooks()
	require.NoError(t, err)
	require.Len(t, webhooks, 3)

	// Verify ordering by ID ascending
	assert.Equal(t, "hook-a", webhooks[0].Name)
	assert.Equal(t, "hook-b", webhooks[1].Name)
	assert.Equal(t, "hook-c", webhooks[2].Name)
}

func TestUpdateWebhook(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:   "original",
		URL:    "https://example.com/v1",
		Events: []string{"run_completed"},
		Active: true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Update the webhook
	wh.Name = "updated"
	wh.URL = "https://example.com/v2"
	wh.Events = []string{"step_failed", "step_completed"}
	wh.Headers = map[string]string{"X-New": "header"}
	wh.Secret = "new-secret"
	wh.Active = false

	err = store.UpdateWebhook(wh)
	require.NoError(t, err)

	// Verify the update
	got, err := store.GetWebhook(id)
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, "https://example.com/v2", got.URL)
	assert.Equal(t, []string{"step_failed", "step_completed"}, got.Events)
	assert.Equal(t, map[string]string{"X-New": "header"}, got.Headers)
	assert.Equal(t, "new-secret", got.Secret)
	assert.False(t, got.Active)
}

func TestUpdateWebhook_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		ID:     999,
		Name:   "ghost",
		URL:    "https://example.com/ghost",
		Active: true,
	}

	err := store.UpdateWebhook(wh)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook not found")
}

func TestDeleteWebhook(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:   "to-delete",
		URL:    "https://example.com/delete",
		Active: true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	err = store.DeleteWebhook(id)
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.GetWebhook(id)
	assert.Error(t, err)

	// Verify list is empty
	webhooks, err := store.ListWebhooks()
	require.NoError(t, err)
	assert.Empty(t, webhooks)
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.DeleteWebhook(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook not found")
}

func TestDeleteWebhook_CascadesDeliveries(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a run first (deliveries reference run_id)
	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "cascade-test",
		URL:    "https://example.com/cascade",
		Active: true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Record a delivery
	err = store.RecordWebhookDelivery(&WebhookDelivery{
		WebhookID:      id,
		RunID:          runID,
		Event:          "run_completed",
		StatusCode:     200,
		ResponseTimeMs: 42,
	})
	require.NoError(t, err)

	// Delete the webhook — deliveries should be cascade-deleted
	err = store.DeleteWebhook(id)
	require.NoError(t, err)

	// Verify deliveries are gone
	deliveries, err := store.GetWebhookDeliveries(id, 10)
	require.NoError(t, err)
	assert.Empty(t, deliveries)
}

func TestRecordWebhookDelivery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "delivery-test",
		URL:    "https://example.com/deliver",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	delivery := &WebhookDelivery{
		WebhookID:      whID,
		RunID:          runID,
		Event:          "run_completed",
		StatusCode:     200,
		ResponseTimeMs: 150,
	}

	err = store.RecordWebhookDelivery(delivery)
	require.NoError(t, err)
	assert.NotZero(t, delivery.ID)
	assert.False(t, delivery.DeliveredAt.IsZero())
}

func TestRecordWebhookDelivery_WithError(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "error-test",
		URL:    "https://example.com/error",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	delivery := &WebhookDelivery{
		WebhookID:      whID,
		RunID:          runID,
		Event:          "step_failed",
		StatusCode:     0,
		ResponseTimeMs: 5000,
		Error:          "connection refused",
	}

	err = store.RecordWebhookDelivery(delivery)
	require.NoError(t, err)
	assert.NotZero(t, delivery.ID)
}

func TestGetWebhookDeliveries(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "deliveries-test",
		URL:    "https://example.com/deliveries",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Record multiple deliveries
	events := []string{"run_completed", "step_failed", "step_completed"}
	for _, evt := range events {
		err := store.RecordWebhookDelivery(&WebhookDelivery{
			WebhookID:      whID,
			RunID:          runID,
			Event:          evt,
			StatusCode:     200,
			ResponseTimeMs: 50,
		})
		require.NoError(t, err)
	}

	deliveries, err := store.GetWebhookDeliveries(whID, 10)
	require.NoError(t, err)
	require.Len(t, deliveries, 3)

	// Verify most recent first ordering
	assert.Equal(t, "step_completed", deliveries[0].Event)
	assert.Equal(t, "step_failed", deliveries[1].Event)
	assert.Equal(t, "run_completed", deliveries[2].Event)
}

func TestGetWebhookDeliveries_Limit(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "limit-test",
		URL:    "https://example.com/limit",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Record 5 deliveries
	for i := 0; i < 5; i++ {
		err := store.RecordWebhookDelivery(&WebhookDelivery{
			WebhookID:      whID,
			RunID:          runID,
			Event:          "run_completed",
			StatusCode:     200,
			ResponseTimeMs: 10,
		})
		require.NoError(t, err)
	}

	// Request only 2
	deliveries, err := store.GetWebhookDeliveries(whID, 2)
	require.NoError(t, err)
	assert.Len(t, deliveries, 2)
}

func TestGetWebhookDeliveries_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:   "empty-test",
		URL:    "https://example.com/empty",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	deliveries, err := store.GetWebhookDeliveries(whID, 10)
	require.NoError(t, err)
	assert.Empty(t, deliveries)
}

func TestGetWebhookDeliveries_DefaultLimit(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &Webhook{
		Name:   "default-limit",
		URL:    "https://example.com/default",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Passing 0 should use default limit of 50 (not fail)
	deliveries, err := store.GetWebhookDeliveries(whID, 0)
	require.NoError(t, err)
	assert.Empty(t, deliveries)
}

func TestWebhook_JSONSerialization(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Test with complex events and headers
	wh := &Webhook{
		Name:    "json-test",
		URL:     "https://example.com/json",
		Events:  []string{"run_completed", "step_failed", "step_start", "contract_validated"},
		Matcher: "^(implement|test)$",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "custom-value",
			"Content-Type":  "application/json",
		},
		Secret: "webhook-signing-secret",
		Active: true,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	got, err := store.GetWebhook(id)
	require.NoError(t, err)

	assert.Equal(t, wh.Events, got.Events)
	assert.Equal(t, wh.Headers, got.Headers)
	assert.Equal(t, wh.Matcher, got.Matcher)
	assert.Equal(t, wh.Secret, got.Secret)
}

func TestWebhook_ActiveFlag(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create inactive webhook
	wh := &Webhook{
		Name:   "inactive",
		URL:    "https://example.com/inactive",
		Active: false,
	}

	id, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	got, err := store.GetWebhook(id)
	require.NoError(t, err)
	assert.False(t, got.Active)

	// Toggle to active
	got.Active = true
	err = store.UpdateWebhook(got)
	require.NoError(t, err)

	got2, err := store.GetWebhook(id)
	require.NoError(t, err)
	assert.True(t, got2.Active)
}

func TestWebhook_DeliveryFieldsRoundTrip(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh := &Webhook{
		Name:   "roundtrip",
		URL:    "https://example.com/roundtrip",
		Active: true,
	}

	whID, err := store.CreateWebhook(wh)
	require.NoError(t, err)

	// Record delivery with all fields
	delivery := &WebhookDelivery{
		WebhookID:      whID,
		RunID:          runID,
		Event:          "step_failed",
		StatusCode:     502,
		ResponseTimeMs: 3500,
		Error:          "upstream timeout",
	}

	err = store.RecordWebhookDelivery(delivery)
	require.NoError(t, err)

	deliveries, err := store.GetWebhookDeliveries(whID, 1)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)

	got := deliveries[0]
	assert.Equal(t, whID, got.WebhookID)
	assert.Equal(t, runID, got.RunID)
	assert.Equal(t, "step_failed", got.Event)
	assert.Equal(t, 502, got.StatusCode)
	assert.Equal(t, int64(3500), got.ResponseTimeMs)
	assert.Equal(t, "upstream timeout", got.Error)
	assert.False(t, got.DeliveredAt.IsZero())
}

func TestWebhook_IsolatedByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	wh1 := &Webhook{Name: "hook-1", URL: "https://a.com", Active: true}
	wh2 := &Webhook{Name: "hook-2", URL: "https://b.com", Active: true}

	id1, err := store.CreateWebhook(wh1)
	require.NoError(t, err)
	id2, err := store.CreateWebhook(wh2)
	require.NoError(t, err)

	// Record deliveries for each webhook
	err = store.RecordWebhookDelivery(&WebhookDelivery{
		WebhookID: id1, RunID: runID, Event: "run_completed", StatusCode: 200,
	})
	require.NoError(t, err)

	err = store.RecordWebhookDelivery(&WebhookDelivery{
		WebhookID: id2, RunID: runID, Event: "step_failed", StatusCode: 500,
	})
	require.NoError(t, err)

	// Each webhook should only see its own deliveries
	d1, err := store.GetWebhookDeliveries(id1, 10)
	require.NoError(t, err)
	require.Len(t, d1, 1)
	assert.Equal(t, "run_completed", d1[0].Event)

	d2, err := store.GetWebhookDeliveries(id2, 10)
	require.NoError(t, err)
	require.Len(t, d2, 1)
	assert.Equal(t, "step_failed", d2[0].Event)
}
