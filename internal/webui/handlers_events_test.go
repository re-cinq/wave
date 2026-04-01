package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAPIStepEvents_Basic(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a run and add events
	runID, err := rwStore.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)
	require.NoError(t, rwStore.UpdateRunStatus(runID, "running", "step1", 0))

	for i := 0; i < 5; i++ {
		require.NoError(t, rwStore.LogEvent(runID, "step1", "stream_activity", "navigator", "message "+string(rune('A'+i)), 0, 0, "", ""))
	}
	for i := 0; i < 3; i++ {
		require.NoError(t, rwStore.LogEvent(runID, "step2", "stream_activity", "craftsman", "step2 msg "+string(rune('A'+i)), 0, 0, "", ""))
	}

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	t.Run("all events for run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/step-events", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StepEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 8, len(resp.Events))
		assert.False(t, resp.HasMore)
	})

	t.Run("filter by step", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/step-events?step=step1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StepEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 5, len(resp.Events))
		assert.False(t, resp.HasMore)
		for _, ev := range resp.Events {
			assert.Equal(t, "step1", ev.StepID)
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/step-events?limit=3", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StepEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 3, len(resp.Events))
		assert.True(t, resp.HasMore)
		assert.Equal(t, 0, resp.Offset)
		assert.Equal(t, 3, resp.Limit)
	})

	t.Run("pagination with offset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/step-events?offset=5&limit=10", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StepEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 3, len(resp.Events))
		assert.False(t, resp.HasMore)
		assert.Equal(t, 5, resp.Offset)
	})

	t.Run("run not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/nonexistent/step-events", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("limit capped at 5000", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/step-events?limit=99999", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StepEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 5000, resp.Limit)
	})
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		param    string
		def      int
		expected int
	}{
		{"empty", "", "limit", 200, 200},
		{"valid", "?limit=50", "limit", 200, 50},
		{"invalid", "?limit=abc", "limit", 200, 200},
		{"negative", "?offset=-5", "offset", 0, -5},
		{"zero", "?limit=0", "limit", 200, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tc.query, nil)
			result := parseIntParam(req, tc.param, tc.def)
			assert.Equal(t, tc.expected, result)
		})
	}
}
