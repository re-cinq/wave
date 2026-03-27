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

func TestExecuteHTTP(t *testing.T) {
	tests := []struct {
		name             string
		responseCode     int
		responseBody     string
		expectedDecision HookDecision
		expectedReason   string
		checkReason      bool
	}{
		{
			name:             "200 ok=true returns proceed",
			responseCode:     200,
			responseBody:     `{"ok":true}`,
			expectedDecision: DecisionProceed,
		},
		{
			name:             "200 ok=false returns block with reason",
			responseCode:     200,
			responseBody:     `{"ok":false,"reason":"security issue"}`,
			expectedDecision: DecisionBlock,
			expectedReason:   "security issue",
			checkReason:      true,
		},
		{
			name:             "200 ok=true action=skip returns skip",
			responseCode:     200,
			responseBody:     `{"ok":true,"action":"skip"}`,
			expectedDecision: DecisionSkip,
		},
		{
			name:             "200 ok=false action=skip still returns skip",
			responseCode:     200,
			responseBody:     `{"ok":false,"action":"skip","reason":"not needed"}`,
			expectedDecision: DecisionSkip,
			expectedReason:   "not needed",
			checkReason:      true,
		},
		{
			name:             "500 returns block",
			responseCode:     500,
			responseBody:     `internal server error`,
			expectedDecision: DecisionBlock,
			expectedReason:   "status 500",
			checkReason:      true,
		},
		{
			name:             "404 returns block",
			responseCode:     404,
			responseBody:     `not found`,
			expectedDecision: DecisionBlock,
			expectedReason:   "status 404",
			checkReason:      true,
		},
		{
			name:             "200 malformed JSON returns block",
			responseCode:     200,
			responseBody:     `not json at all`,
			expectedDecision: DecisionBlock,
			expectedReason:   "failed to parse response",
			checkReason:      true,
		},
		{
			name:             "200 empty body returns block",
			responseCode:     200,
			responseBody:     ``,
			expectedDecision: DecisionBlock,
			expectedReason:   "failed to parse response",
			checkReason:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			hook := &LifecycleHookDef{
				Name:    "test-http-hook",
				Type:    HookTypeHTTP,
				URL:     server.URL,
				Timeout: "5s",
			}
			evt := HookEvent{
				Type:       EventStepStart,
				PipelineID: "test-pipeline",
				StepID:     "test-step",
			}

			result := executeHTTP(context.Background(), hook, evt)

			assert.Equal(t, tc.expectedDecision, result.Decision)
			assert.Equal(t, "test-http-hook", result.HookName)
			if tc.checkReason {
				assert.Contains(t, result.Reason, tc.expectedReason)
			}
		})
	}
}

func TestExecuteHTTPPostBodyContainsEvent(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	hook := &LifecycleHookDef{
		Name:    "body-check-hook",
		Type:    HookTypeHTTP,
		URL:     server.URL,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "my-pipeline",
		StepID:     "my-step",
		Input:      "test input",
		Workspace:  "/tmp/workspace",
		Artifacts:  []string{"artifact1.json"},
	}

	result := executeHTTP(context.Background(), hook, evt)
	require.Equal(t, DecisionProceed, result.Decision)

	// Parse the received body and verify it contains the event fields
	var receivedEvt HookEvent
	err := json.Unmarshal(receivedBody, &receivedEvt)
	require.NoError(t, err)
	assert.Equal(t, EventStepStart, receivedEvt.Type)
	assert.Equal(t, "my-pipeline", receivedEvt.PipelineID)
	assert.Equal(t, "my-step", receivedEvt.StepID)
	assert.Equal(t, "test input", receivedEvt.Input)
	assert.Equal(t, "/tmp/workspace", receivedEvt.Workspace)
	assert.Equal(t, []string{"artifact1.json"}, receivedEvt.Artifacts)
}

func TestExecuteHTTPConnectionError(t *testing.T) {
	// Use a URL that will fail to connect
	hook := &LifecycleHookDef{
		Name:    "conn-err-hook",
		Type:    HookTypeHTTP,
		URL:     "http://127.0.0.1:1", // port 1 should be unreachable
		Timeout: "1s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeHTTP(context.Background(), hook, evt)

	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
	assert.Contains(t, result.Reason, "HTTP request failed")
}
