package hooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func disableSSRFValidation(t *testing.T) {
	t.Helper()
	orig := urlValidator
	urlValidator = func(string) error { return nil }
	t.Cleanup(func() { urlValidator = orig })
}

func TestExecuteHTTP(t *testing.T) {
	disableSSRFValidation(t)
	tests := []struct {
		name             string
		responseCode     int
		responseBody     string
		expectedDecision HookDecision
		expectedReason   string
		checkReason      bool
	}{
		{name: "200 ok=true", responseCode: 200, responseBody: `{"ok":true}`, expectedDecision: DecisionProceed},
		{name: "200 ok=false", responseCode: 200, responseBody: `{"ok":false,"reason":"security issue"}`, expectedDecision: DecisionBlock, expectedReason: "security issue", checkReason: true},
		{name: "200 action=skip", responseCode: 200, responseBody: `{"ok":true,"action":"skip"}`, expectedDecision: DecisionSkip},
		{name: "200 ok=false action=skip", responseCode: 200, responseBody: `{"ok":false,"action":"skip","reason":"not needed"}`, expectedDecision: DecisionSkip, expectedReason: "not needed", checkReason: true},
		{name: "500", responseCode: 500, responseBody: `error`, expectedDecision: DecisionBlock, expectedReason: "status 500", checkReason: true},
		{name: "404", responseCode: 404, responseBody: `not found`, expectedDecision: DecisionBlock, expectedReason: "status 404", checkReason: true},
		{name: "malformed JSON", responseCode: 200, responseBody: `not json`, expectedDecision: DecisionBlock, expectedReason: "failed to parse response", checkReason: true},
		{name: "empty body", responseCode: 200, responseBody: ``, expectedDecision: DecisionBlock, expectedReason: "failed to parse response", checkReason: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()
			hook := &LifecycleHookDef{Name: "test-http-hook", Type: HookTypeHTTP, URL: server.URL, Timeout: "5s"}
			result := executeHTTP(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "test-pipeline", StepID: "test-step"})
			assert.Equal(t, tc.expectedDecision, result.Decision)
			if tc.checkReason {
				assert.Contains(t, result.Reason, tc.expectedReason)
			}
		})
	}
}

func TestExecuteHTTPPostBodyContainsEvent(t *testing.T) {
	disableSSRFValidation(t)
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	hook := &LifecycleHookDef{Name: "body-hook", Type: HookTypeHTTP, URL: server.URL, Timeout: "5s"}
	evt := HookEvent{Type: EventStepStart, PipelineID: "my-pipeline", StepID: "my-step", Input: "test input", Workspace: "/tmp/ws", Artifacts: []string{"a.json"}}
	result := executeHTTP(context.Background(), hook, evt)
	require.Equal(t, DecisionProceed, result.Decision)
	var got HookEvent
	require.NoError(t, json.Unmarshal(receivedBody, &got))
	assert.Equal(t, "my-pipeline", got.PipelineID)
	assert.Equal(t, "my-step", got.StepID)
}

func TestExecuteHTTPSSRFBlocked(t *testing.T) {
	for _, tc := range []struct{ name, url string }{
		{"localhost", "http://localhost:8080/hook"},
		{"loopback", "http://127.0.0.1:8080/hook"},
		{"loopback6", "http://[::1]:8080/hook"},
		{"link-local", "http://169.254.169.254/latest/meta-data/"},
		{"10.x", "http://10.0.0.1:8080/hook"},
		{"172.16.x", "http://172.16.0.1:8080/hook"},
		{"192.168.x", "http://192.168.1.1:8080/hook"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executeHTTP(context.Background(), &LifecycleHookDef{Name: "ssrf", Type: HookTypeHTTP, URL: tc.url, Timeout: "1s"}, HookEvent{Type: EventStepStart, PipelineID: "test"})
			assert.Equal(t, DecisionBlock, result.Decision)
			assert.NotNil(t, result.Err)
			assert.Contains(t, result.Reason, "blocked URL")
		})
	}
}

func TestValidateHTTPTarget(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		wantErr   bool
	}{
		{"public IP", "https://8.8.8.8/webhook", false},
		{"localhost", "http://localhost/hook", true},
		{"localhost.", "http://localhost./hook", true},
		{"loopback", "http://127.0.0.1/hook", true},
		{"10.x", "http://10.0.0.1/hook", true},
		{"172.16.x", "http://172.16.0.1/hook", true},
		{"192.168.x", "http://192.168.1.1/hook", true},
		{"link-local", "http://169.254.169.254/m", true},
		{"ipv6 loopback", "http://[::1]/hook", true},
		{"empty host", "http:///hook", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHTTPTarget(tc.url)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteHTTPLimitedResponseBody(t *testing.T) {
	disableSSRFValidation(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		buf := make([]byte, maxResponseBodySize+1024)
		for i := range buf {
			buf[i] = 'x'
		}
		_, _ = w.Write(buf)
	}))
	defer server.Close()
	result := executeHTTP(context.Background(), &LifecycleHookDef{Name: "large", Type: HookTypeHTTP, URL: server.URL, Timeout: "5s"}, HookEvent{Type: EventStepStart, PipelineID: "test"})
	assert.Equal(t, DecisionBlock, result.Decision)
	assert.Contains(t, result.Reason, "failed to parse response")
}
