package hooks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebhookRunner_FiresMatchingEvents(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Temporarily allow localhost for test
	origValidator := urlValidator
	urlValidator = func(string) error { return nil }
	defer func() { urlValidator = origValidator }()

	webhooks := []WebhookRecord{
		{ID: 1, Name: "test-hook", URL: srv.URL, Events: []string{"run_completed"}, Active: true},
	}
	runner := NewWebhookRunner(webhooks, nil)

	// Should fire
	runner.FireWebhooks(context.Background(), HookEvent{Type: EventRunCompleted, PipelineID: "run-1"})
	time.Sleep(500 * time.Millisecond) // async delivery

	if received.Load() != 1 {
		t.Errorf("expected 1 delivery, got %d", received.Load())
	}
}

func TestWebhookRunner_SkipsNonMatchingEvents(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	origValidator := urlValidator
	urlValidator = func(string) error { return nil }
	defer func() { urlValidator = origValidator }()

	webhooks := []WebhookRecord{
		{ID: 1, Name: "run-only", URL: srv.URL, Events: []string{"run_completed"}, Active: true},
	}
	runner := NewWebhookRunner(webhooks, nil)

	// Should NOT fire (step event, not run event)
	runner.FireWebhooks(context.Background(), HookEvent{Type: EventStepCompleted, PipelineID: "run-1", StepID: "step-1"})
	time.Sleep(100 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("expected 0 deliveries, got %d", received.Load())
	}
}

func TestWebhookRunner_SkipsInactive(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	origValidator := urlValidator
	urlValidator = func(string) error { return nil }
	defer func() { urlValidator = origValidator }()

	webhooks := []WebhookRecord{
		{ID: 1, Name: "inactive", URL: srv.URL, Events: []string{"run_completed"}, Active: false},
	}
	runner := NewWebhookRunner(webhooks, nil)
	runner.FireWebhooks(context.Background(), HookEvent{Type: EventRunCompleted, PipelineID: "run-1"})
	time.Sleep(100 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("expected 0 deliveries for inactive webhook, got %d", received.Load())
	}
}

func TestWebhookRunner_RateLimiting(t *testing.T) {
	rl := newWebhookRateLimiter()

	// Should allow up to maxDeliveriesPerMinute
	for i := 0; i < maxDeliveriesPerMinute; i++ {
		if !rl.allow(1) {
			t.Fatalf("expected allow at delivery %d", i)
		}
	}

	// Should block the next one
	if rl.allow(1) {
		t.Error("expected rate limit to block delivery")
	}

	// Different webhook should still be allowed
	if !rl.allow(2) {
		t.Error("expected different webhook to be allowed")
	}
}

func TestWebhookRunner_RateLimiterResets(t *testing.T) {
	rl := newWebhookRateLimiter()
	rl.resetAt = time.Now().Add(-time.Second) // force immediate reset

	for i := 0; i < maxDeliveriesPerMinute; i++ {
		rl.allow(1)
	}

	// Force reset by setting reset time in the past
	rl.mu.Lock()
	rl.resetAt = time.Now().Add(-time.Second)
	rl.mu.Unlock()

	// Should allow again after reset
	if !rl.allow(1) {
		t.Error("expected allow after rate limit reset")
	}
}

func TestWebhookRunner_HMACSignature(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Wave-Signature-256")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	origValidator := urlValidator
	urlValidator = func(string) error { return nil }
	defer func() { urlValidator = origValidator }()

	webhooks := []WebhookRecord{
		{ID: 1, Name: "signed", URL: srv.URL, Events: []string{"run_completed"}, Active: true, Secret: "test-secret"},
	}
	runner := NewWebhookRunner(webhooks, nil)
	runner.FireWebhooks(context.Background(), HookEvent{Type: EventRunCompleted, PipelineID: "run-1"})
	time.Sleep(200 * time.Millisecond)

	if gotSig == "" {
		t.Error("expected X-Wave-Signature-256 header")
	}
	if len(gotSig) < 10 || gotSig[:7] != "sha256=" {
		t.Errorf("expected sha256= prefix, got: %s", gotSig)
	}
}
