package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewAppliesDefaults(t *testing.T) {
	c := New(Config{})
	if c.maxRetries != DefaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, DefaultMaxRetries)
	}
	if c.retryBaseWait != DefaultRetryBaseWait {
		t.Errorf("retryBaseWait = %v, want %v", c.retryBaseWait, DefaultRetryBaseWait)
	}
	if c.retryMaxWait != DefaultRetryMaxWait {
		t.Errorf("retryMaxWait = %v, want %v", c.retryMaxWait, DefaultRetryMaxWait)
	}
	if c.httpClient.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", c.httpClient.Timeout, DefaultTimeout)
	}
}

func TestNewSingleShotPreservedWhenRetryWaitsSet(t *testing.T) {
	// Caller setting retry waits but leaving MaxRetries=0 expresses
	// single-shot intent — the default-application heuristic must not
	// override that.
	c := New(Config{
		MaxRetries:    0,
		RetryBaseWait: 100 * time.Millisecond,
	})
	if c.maxRetries != 0 {
		t.Errorf("maxRetries = %d, want 0 (single-shot)", c.maxRetries)
	}
}

func TestGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "hello")
	}))
	defer srv.Close()

	c := New(Config{Timeout: 2 * time.Second, MaxRetries: 1})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}
}

func TestRetryOn5xxThenSuccess(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    3,
		RetryBaseWait: 1 * time.Millisecond,
		RetryMaxWait:  10 * time.Millisecond,
	})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3", got)
	}
}

func TestRetriesExhaustedReturns5xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    2,
		RetryBaseWait: 1 * time.Millisecond,
		RetryMaxWait:  5 * time.Millisecond,
	})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("expected resp returned with 5xx, got err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3 (1 initial + 2 retries)", got)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    3,
		RetryBaseWait: 1 * time.Millisecond,
	})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("hits = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestSingleShotMode(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    0,
		RetryBaseWait: 1 * time.Millisecond,
	})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("hits = %d, want 1 (single-shot)", got)
	}
}

func TestPostBodyReplayedAcrossRetries(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "payload" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    2,
		RetryBaseWait: 1 * time.Millisecond,
	})
	resp, err := c.Post(context.Background(), srv.URL, "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAuditEmitterReceivesEventPerAttempt(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var events []Event
	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    3,
		RetryBaseWait: 1 * time.Millisecond,
		AuditEmitter: func(e Event) {
			events = append(events, e)
		},
	})
	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()

	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].Attempt != 1 || events[1].Attempt != 2 {
		t.Errorf("attempts = %d,%d, want 1,2", events[0].Attempt, events[1].Attempt)
	}
	if events[0].StatusCode != http.StatusInternalServerError {
		t.Errorf("event[0].StatusCode = %d, want 500", events[0].StatusCode)
	}
	if events[1].StatusCode != http.StatusOK {
		t.Errorf("event[1].StatusCode = %d, want 200", events[1].StatusCode)
	}
	if events[0].Method != http.MethodGet {
		t.Errorf("event[0].Method = %q, want GET", events[0].Method)
	}
}

func TestContextCancellationStopsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{
		Timeout:       2 * time.Second,
		MaxRetries:    5,
		RetryBaseWait: 50 * time.Millisecond,
		RetryMaxWait:  100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}

func TestCustomTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Custom RoundTripper that rewrites every request to the test server.
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})

	c := New(Config{
		Timeout:    2 * time.Second,
		MaxRetries: 0,
		Transport:  transport,
	})
	resp, err := c.Get(context.Background(), "https://example.invalid/")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestCheckRedirectHonoured(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := New(Config{
		Timeout:    2 * time.Second,
		MaxRetries: 0,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})
	resp, err := c.Get(context.Background(), redirector.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want 302 (redirect not followed)", resp.StatusCode)
	}
}

func TestHTTPClientAccessor(t *testing.T) {
	c := New(Config{Timeout: 5 * time.Second})
	hc := c.HTTPClient()
	if hc == nil {
		t.Fatal("HTTPClient() returned nil")
	}
	if hc.Timeout != 5*time.Second {
		t.Errorf("underlying timeout = %v, want 5s", hc.Timeout)
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
