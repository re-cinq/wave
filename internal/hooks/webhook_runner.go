package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// maxDeliveriesPerMinute is the rate limit per webhook to prevent runaway delivery loops.
const maxDeliveriesPerMinute = 30

// WebhookRecord mirrors state.Webhook without importing state (avoids cycle).
type WebhookRecord struct {
	ID      int64
	Name    string
	URL     string
	Events  []string
	Matcher string
	Headers map[string]string
	Secret  string
	Active  bool
}

// WebhookDeliveryRecord mirrors state.WebhookDelivery for recording delivery results.
type WebhookDeliveryRecord struct {
	WebhookID      int64
	RunID          string
	Event          string
	StatusCode     int
	ResponseTimeMs int64
	Error          string
}

// WebhookStore is the minimal interface needed by the webhook runner.
// Avoids importing the full state package.
type WebhookStore interface {
	RecordWebhookDeliveryResult(delivery *WebhookDeliveryRecord) error
}

// WebhookRunner fires HTTP POST to registered webhooks for matching events.
type WebhookRunner struct {
	webhooks    []WebhookRecord
	matchers    []*regexp.Regexp
	client      *http.Client
	store       WebhookStore
	rateLimiter *webhookRateLimiter
}

// webhookRateLimiter tracks delivery counts per webhook per minute window.
type webhookRateLimiter struct {
	mu      sync.Mutex
	counts  map[int64]int
	resetAt time.Time
}

func newWebhookRateLimiter() *webhookRateLimiter {
	return &webhookRateLimiter{
		counts:  make(map[int64]int),
		resetAt: time.Now().Add(time.Minute),
	}
}

// allow returns true if the webhook hasn't exceeded the rate limit.
func (rl *webhookRateLimiter) allow(webhookID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if time.Now().After(rl.resetAt) {
		rl.counts = make(map[int64]int)
		rl.resetAt = time.Now().Add(time.Minute)
	}

	if rl.counts[webhookID] >= maxDeliveriesPerMinute {
		return false
	}
	rl.counts[webhookID]++
	return true
}

// NewWebhookRunner creates a runner from a list of webhook records.
func NewWebhookRunner(webhooks []WebhookRecord, store WebhookStore) *WebhookRunner {
	matchers := make([]*regexp.Regexp, len(webhooks))
	for i, wh := range webhooks {
		if wh.Matcher != "" {
			if re, err := regexp.Compile(wh.Matcher); err == nil {
				matchers[i] = re
			}
		}
	}
	return &WebhookRunner{
		webhooks:    webhooks,
		matchers:    matchers,
		client:      &http.Client{Timeout: 10 * time.Second},
		store:       store,
		rateLimiter: newWebhookRateLimiter(),
	}
}

// FireWebhooks sends the event to all matching webhooks.
// Non-blocking — failures are recorded but don't stop execution.
func (r *WebhookRunner) FireWebhooks(ctx context.Context, evt HookEvent) {
	for i, wh := range r.webhooks {
		if !wh.Active {
			continue
		}
		if !r.matchesEvent(wh, evt) {
			continue
		}
		if evt.StepID != "" && r.matchers[i] != nil && !r.matchers[i].MatchString(evt.StepID) {
			continue
		}

		if !r.rateLimiter.allow(wh.ID) {
			r.recordDelivery(&wh, evt, 0, 0, "rate limited: exceeded 30 deliveries/minute")
			continue
		}
		go r.deliver(ctx, &wh, evt)
	}
}

func (r *WebhookRunner) matchesEvent(wh WebhookRecord, evt HookEvent) bool {
	if len(wh.Events) == 0 {
		return true // no filter = all events
	}
	evtStr := string(evt.Type)
	for _, e := range wh.Events {
		if e == evtStr {
			return true
		}
	}
	return false
}

func (r *WebhookRunner) deliver(ctx context.Context, wh *WebhookRecord, evt HookEvent) {
	// SSRF protection: validate webhook URL before delivery
	if err := urlValidator(wh.URL); err != nil {
		r.recordDelivery(wh, evt, 0, 0, fmt.Sprintf("SSRF blocked: %s", err))
		return
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		r.recordDelivery(wh, evt, 0, 0, fmt.Sprintf("marshal error: %s", err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payload))
	if err != nil {
		r.recordDelivery(wh, evt, 0, 0, fmt.Sprintf("request error: %s", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Wave-Webhook/1.0")

	// Custom headers
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	// HMAC signature
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Wave-Signature-256", "sha256="+sig)
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		r.recordDelivery(wh, evt, 0, elapsed, err.Error())
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) //nolint:errcheck

	errMsg := ""
	if resp.StatusCode >= 400 {
		errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	r.recordDelivery(wh, evt, resp.StatusCode, elapsed, errMsg)
}

func (r *WebhookRunner) recordDelivery(wh *WebhookRecord, evt HookEvent, statusCode int, elapsed int64, errMsg string) {
	if r.store == nil {
		return
	}
	r.store.RecordWebhookDeliveryResult(&WebhookDeliveryRecord{
		WebhookID:      wh.ID,
		RunID:          evt.PipelineID,
		Event:          string(evt.Type),
		StatusCode:     statusCode,
		ResponseTimeMs: elapsed,
		Error:          errMsg,
	})
}
