package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandlers(t *testing.T) (*Server, state.StateStore) {
	t.Helper()
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)

	config := ServerConfig{
		Port: 8080,
		Bind: "127.0.0.1",
	}

	srv := NewServer(config, store)
	return srv, store
}

func TestHandleListRuns(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	t.Run("empty database returns empty list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs", nil)
		w := httptest.NewRecorder()

		srv.handleListRuns(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RunListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Runs)
	})

	t.Run("returns runs from database", func(t *testing.T) {
		_, err := store.CreateRun("test-pipeline", "test input")
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/runs", nil)
		w := httptest.NewRecorder()

		srv.handleListRuns(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RunListResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "test-pipeline", resp.Runs[0].PipelineName)
	})

	t.Run("filters by status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs?status=completed", nil)
		w := httptest.NewRecorder()

		srv.handleListRuns(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RunListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		// No completed runs exist
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs?limit=1", nil)
		w := httptest.NewRecorder()

		srv.handleListRuns(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RunListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.LessOrEqual(t, resp.Total, 1)
	})
}

func TestHandleGetRun(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	t.Run("returns 404 for unknown run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/nonexistent", nil)
		req.SetPathValue("id", "nonexistent")
		w := httptest.NewRecorder()

		srv.handleGetRun(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns run details", func(t *testing.T) {
		runID, err := store.CreateRun("detail-pipeline", "some input")
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/runs/"+runID, nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRun(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RunDetailResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, runID, resp.Run.RunID)
		assert.Equal(t, "detail-pipeline", resp.Run.PipelineName)
		assert.Equal(t, "pending", resp.Run.Status)
	})

	t.Run("returns 400 for missing id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/", nil)
		req.SetPathValue("id", "")
		w := httptest.NewRecorder()

		srv.handleGetRun(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleGetRunEvents(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	runID, err := store.CreateRun("events-pipeline", "input")
	require.NoError(t, err)

	// Log some events
	err = store.LogEvent(runID, "step-1", "started", "navigator", "Starting step", 0, 0)
	require.NoError(t, err)
	err = store.LogEvent(runID, "step-1", "completed", "navigator", "Step done", 1000, 5000)
	require.NoError(t, err)

	t.Run("returns events for run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/events", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp EventListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Events, 2)
		assert.Equal(t, "started", resp.Events[0].State)
		assert.Equal(t, "completed", resp.Events[1].State)
	})

	t.Run("filters by step", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/events?step=step-1", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp EventListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Events, 2)
	})
}

func TestHandleGetRunSteps(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	runID, err := store.CreateRun("steps-pipeline", "input")
	require.NoError(t, err)

	// Create step progress
	err = store.UpdateStepProgress(runID, "step-1", "navigator", "running", 50, "reading files", "Working", 10000, 500)
	require.NoError(t, err)

	t.Run("returns steps for run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/steps", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunSteps(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []StepProgressResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "step-1", resp[0].StepID)
		assert.Equal(t, 50, resp[0].Progress)
		assert.Equal(t, "navigator", resp[0].Persona)
	})
}

func TestHandleGetRunArtifacts(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	runID, err := store.CreateRun("artifacts-pipeline", "input")
	require.NoError(t, err)

	err = store.RegisterArtifact(runID, "step-1", "output.json", "/path/to/output.json", "application/json", 1024)
	require.NoError(t, err)

	t.Run("returns artifacts for run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/artifacts", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunArtifacts(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ArtifactListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Artifacts, 1)
		assert.Equal(t, "output.json", resp.Artifacts[0].Name)
		assert.Equal(t, int64(1024), resp.Artifacts[0].SizeBytes)
	})
}

func TestHandleGetRunProgress(t *testing.T) {
	srv, store := setupTestHandlers(t)
	defer store.Close()

	runID, err := store.CreateRun("progress-pipeline", "input")
	require.NoError(t, err)

	t.Run("returns 404 when no progress exists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/progress", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunProgress(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns progress when available", func(t *testing.T) {
		err := store.UpdatePipelineProgress(runID, 5, 2, 2, 40, 30000)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/runs/"+runID+"/progress", nil)
		req.SetPathValue("id", runID)
		w := httptest.NewRecorder()

		srv.handleGetRunProgress(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp PipelineProgressResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 5, resp.TotalSteps)
		assert.Equal(t, 2, resp.CompletedSteps)
		assert.Equal(t, 40, resp.OverallProgress)
	})
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"key":"value"`)
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad request", "details here")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "bad request", resp.Error)
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "details here", resp.Details)
}
