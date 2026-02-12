package dashboard

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*Server, state.StateStore) {
	t.Helper()
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)

	config := ServerConfig{
		Port:   0, // Will be overridden
		Bind:   "127.0.0.1",
		DBPath: ":memory:",
	}

	srv := NewServer(config, store)
	return srv, store
}

func TestNewServer(t *testing.T) {
	srv, store := setupTestServer(t)
	defer store.Close()

	assert.NotNil(t, srv)
	assert.NotNil(t, srv.broker)
	assert.NotNil(t, srv.store)
	assert.Equal(t, "127.0.0.1", srv.config.Bind)
}

func TestServerStartAndShutdown(t *testing.T) {
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	config := ServerConfig{
		Port: 0,
		Bind: "127.0.0.1",
	}

	srv := NewServer(config, store)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServerMiddleware(t *testing.T) {
	srv, store := setupTestServer(t)
	defer store.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.withMiddleware(mux)

	// Test CORS headers
	t.Run("adds CORS headers", func(t *testing.T) {
		w := &testResponseWriter{headers: make(http.Header)}
		r, _ := http.NewRequest("GET", "/test", nil)
		handler.ServeHTTP(w, r)
		assert.Equal(t, "*", w.headers.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, OPTIONS", w.headers.Get("Access-Control-Allow-Methods"))
	})

	// Test OPTIONS preflight
	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		w := &testResponseWriter{headers: make(http.Header)}
		r, _ := http.NewRequest("OPTIONS", "/test", nil)
		handler.ServeHTTP(w, r)
		assert.Equal(t, http.StatusNoContent, w.statusCode)
	})
}

func TestBrokerAccess(t *testing.T) {
	srv, store := setupTestServer(t)
	defer store.Close()

	broker := srv.Broker()
	assert.NotNil(t, broker)
	assert.Same(t, srv.broker, broker)
}

// testResponseWriter is a minimal ResponseWriter for unit tests.
type testResponseWriter struct {
	headers    http.Header
	statusCode int
	body       []byte
}

func (w *testResponseWriter) Header() http.Header {
	return w.headers
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
