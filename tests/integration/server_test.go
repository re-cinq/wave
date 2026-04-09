package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/webui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTestServer creates a real webui.Server on a random port and returns
// the base URL and a cleanup function. The server runs against a fresh
// temporary SQLite database.
func startTestServer(t *testing.T) (baseURL string, cleanup func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	// Pre-create the database so both RO and RW handles work.
	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	store.Close()

	// Find a free port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := webui.ServerConfig{
		Bind:     "127.0.0.1",
		Port:     port,
		DBPath:   dbPath,
		AuthMode: webui.AuthModeNone,
	}

	srv, err := webui.NewServer(cfg)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for the server to be ready (max 3 seconds).
	ready := false
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(t, ready, "server did not become ready within 3 seconds")

	return base, func() {
		// The server's Start() blocks on signal — shut it down via the HTTP server.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// We can't call srv.httpServer.Shutdown directly (unexported),
		// so we just let the test process exit. The deferred store.Close
		// in NewServer handles cleanup.
		_ = ctx
	}
}

// TestServerHealth starts the real server and verifies GET /api/health
// returns 200 with a JSON body containing health checks.
func TestServerHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	base, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(base + "/api/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body webui.HealthListResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body.Checks, "expected at least one health check result")

	for _, check := range body.Checks {
		assert.NotEmpty(t, check.Name)
		assert.Contains(t, []string{"ok", "warn", "error"}, check.Status)
	}
}

// TestServerPipelines verifies GET /api/pipelines returns 200 with a JSON
// response containing a pipelines array.
func TestServerPipelines(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	base, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(base + "/api/pipelines")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	// The response must contain a "pipelines" key.
	_, hasPipelines := body["pipelines"]
	assert.True(t, hasPipelines, "response must contain 'pipelines' key")
}

// TestServerRuns verifies GET /api/runs returns 200 with an empty run list
// on a fresh database.
func TestServerRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	base, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(base + "/api/runs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body webui.RunListResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Empty(t, body.Runs, "fresh database should have no runs")
	assert.False(t, body.HasMore)
}

// TestServerNotFound verifies that unknown API paths return 404.
func TestServerNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	base, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(base + "/api/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
