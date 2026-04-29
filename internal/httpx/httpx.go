// Package httpx provides a single, shared HTTP Client wrapper used across
// Wave for outbound HTTP traffic (forge probes, GitHub API, LLM judge,
// onboarding probes). Centralizing the client unifies the timeout, retry,
// and audit policy so transport behaviour is consistent and observable
// regardless of caller.
//
// httpx is intentionally a leaf package: it has no internal Wave imports
// other than standard library types so it cannot create import cycles. All
// callers consume *httpx.Client; the underlying *http.Client is never
// constructed directly by feature packages.
//
// The shared Client supports retries with exponential backoff for transient
// failures (network errors and 5xx responses). Set MaxRetries to 0 to opt
// out of retries entirely (used by fast-fail probes). A pluggable
// AuditEmitter hook receives one Event per request attempt for observability
// and credential-scrubbed logging.
package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Defaults reflect sane values picked for general API traffic. Individual
// callers override via Config.
const (
	DefaultTimeout       = 30 * time.Second
	DefaultMaxRetries    = 3
	DefaultRetryBaseWait = 500 * time.Millisecond
	DefaultRetryMaxWait  = 30 * time.Second
)

// Event describes a single HTTP request attempt. AuditEmitter hooks receive
// one Event per attempt (so a request that retries twice produces three
// events, one per attempt). Err is non-nil for network-level failures;
// StatusCode is zero in that case. URL has any query string preserved but
// callers should scrub credentials before logging.
type Event struct {
	URL        string
	Method     string
	StatusCode int
	Attempt    int
	Duration   time.Duration
	Err        error
}

// Config configures a Client. Zero values fall back to package defaults.
type Config struct {
	// Timeout is the per-request timeout applied via http.Client.Timeout.
	// It bounds the total time including connection + body read.
	Timeout time.Duration

	// MaxRetries is the maximum number of additional attempts after the
	// initial request. Zero disables retries entirely (single-shot mode).
	// Negative values are clamped to zero.
	MaxRetries int

	// RetryBaseWait is the starting backoff delay between retry attempts.
	// Backoff doubles each attempt, capped at RetryMaxWait.
	RetryBaseWait time.Duration

	// RetryMaxWait caps the backoff delay between retries.
	RetryMaxWait time.Duration

	// Transport overrides the underlying RoundTripper. When nil, the
	// standard http.DefaultTransport is used. Callers with special TLS
	// requirements (e.g. self-hosted forge probes) supply a custom
	// transport here.
	Transport http.RoundTripper

	// CheckRedirect mirrors http.Client.CheckRedirect for callers (e.g.
	// forge probes) that need to disable redirect-following.
	CheckRedirect func(req *http.Request, via []*http.Request) error

	// AuditEmitter receives one Event per attempt. Optional; when nil no
	// audit events are emitted. Implementations must be cheap and
	// thread-safe.
	AuditEmitter func(Event)
}

// Client is the shared HTTP client wrapping *http.Client with retry and
// audit instrumentation. Exposed methods are concurrency-safe.
type Client struct {
	httpClient    *http.Client
	maxRetries    int
	retryBaseWait time.Duration
	retryMaxWait  time.Duration
	emitAudit     func(Event)
}

// New constructs a Client with the supplied configuration, applying
// package defaults for any zero-valued fields.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	// Caller may explicitly set MaxRetries = 0 for single-shot probes; only
	// substitute the default when MaxRetries was left at zero AND the caller
	// did not explicitly want a single-shot. We distinguish by checking
	// RetryBaseWait/RetryMaxWait — if those are also zero we apply full
	// defaults including DefaultMaxRetries. If the caller set retry waits
	// but left MaxRetries at zero we treat it as single-shot intent.
	applyRetryDefault := cfg.MaxRetries == 0 && cfg.RetryBaseWait == 0 && cfg.RetryMaxWait == 0
	if applyRetryDefault {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.RetryBaseWait <= 0 {
		cfg.RetryBaseWait = DefaultRetryBaseWait
	}
	if cfg.RetryMaxWait <= 0 {
		cfg.RetryMaxWait = DefaultRetryMaxWait
	}

	hc := &http.Client{
		Timeout:       cfg.Timeout,
		Transport:     cfg.Transport,
		CheckRedirect: cfg.CheckRedirect,
	}

	return &Client{
		httpClient:    hc,
		maxRetries:    cfg.MaxRetries,
		retryBaseWait: cfg.RetryBaseWait,
		retryMaxWait:  cfg.RetryMaxWait,
		emitAudit:     cfg.AuditEmitter,
	}
}

// HTTPClient returns the underlying *http.Client. Provided so callers that
// must hand a raw *http.Client to a third-party API (or to httptest server
// helpers) can do so without duplicating transport configuration.
func (c *Client) HTTPClient() *http.Client {
	if c == nil {
		return http.DefaultClient
	}
	return c.httpClient
}

// Do executes the request honouring the supplied context, the client
// timeout, and the retry policy. The request body must be replayable
// (bytes.Reader, strings.Reader, or nil) for retries to function; requests
// with a non-replayable body are sent once regardless of MaxRetries.
//
// Successful responses (2xx, 3xx, 4xx) are returned to the caller. 5xx
// responses are retried up to MaxRetries times with exponential backoff
// before being returned. Network errors and timeouts are likewise retried.
//
// Callers are responsible for closing resp.Body when err is nil.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("httpx: nil request")
	}
	// Capture body bytes for replay across retries when possible.
	bodyBytes, replayable, err := snapshotBody(req)
	if err != nil {
		return nil, fmt.Errorf("httpx: snapshot body: %w", err)
	}

	attempts := c.maxRetries + 1
	if !replayable && req.Body != nil {
		// Non-replayable body: fall back to single-shot.
		attempts = 1
	}

	var lastResp *http.Response
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			delay := c.backoff(attempt - 1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Re-arm body for replay attempts.
		if replayable && bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
		}

		// Honour caller context on every attempt.
		reqWithCtx := req.WithContext(ctx)
		start := time.Now()
		resp, err := c.httpClient.Do(reqWithCtx)
		dur := time.Since(start)

		c.audit(Event{
			URL:        reqWithCtx.URL.String(),
			Method:     reqWithCtx.Method,
			StatusCode: statusCodeOf(resp),
			Attempt:    attempt,
			Duration:   dur,
			Err:        err,
		})

		if err != nil {
			lastErr = err
			lastResp = nil
			if !shouldRetryErr(err) || attempt == attempts {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode <= 599 && attempt < attempts {
			// Drain and close so the connection can be reused for retry.
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			lastResp = nil
			lastErr = fmt.Errorf("httpx: server returned %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, errors.New("httpx: request failed with no response")
}

// Get is a convenience wrapper for GET requests.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post is a convenience wrapper for POST requests with an explicit
// Content-Type. Body may be nil.
func (c *Client) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(ctx, req)
}

func (c *Client) audit(e Event) {
	if c == nil || c.emitAudit == nil {
		return
	}
	c.emitAudit(e)
}

// backoff returns the wait before the (attempt+1)-th retry, doubling each
// step up to retryMaxWait.
func (c *Client) backoff(retry int) time.Duration {
	d := c.retryBaseWait
	for i := 1; i < retry; i++ {
		d *= 2
		if d >= c.retryMaxWait {
			return c.retryMaxWait
		}
	}
	if d > c.retryMaxWait {
		return c.retryMaxWait
	}
	return d
}

// snapshotBody reads the request body into memory so it can be replayed
// across retries. Returns (bytes, replayable, err). For nil bodies returns
// (nil, true, nil).
//
// http.NewRequestWithContext sets GetBody for known seekable readers
// (bytes.Reader, bytes.Buffer, strings.Reader); we always read into memory
// here so the retry loop can re-arm req.Body uniformly without depending on
// transport-level redirect machinery.
func snapshotBody(req *http.Request) ([]byte, bool, error) {
	if req.Body == nil {
		return nil, true, nil
	}
	data, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func statusCodeOf(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// shouldRetryErr identifies transient transport errors worth retrying.
// Conservatively we retry context-independent errors only — context.Canceled
// and context.DeadlineExceeded short-circuit because the caller has already
// given up.
func shouldRetryErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}
