package dashboard

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/state"
)

// ServerConfig holds configuration for the dashboard HTTP server.
type ServerConfig struct {
	Port    int
	Bind    string
	DBPath  string
}

// Server is the dashboard HTTP server.
type Server struct {
	config     ServerConfig
	store      state.StateStore
	broker     *SSEBroker
	httpServer *http.Server
}

// NewServer creates a new dashboard server.
func NewServer(config ServerConfig, store state.StateStore) *Server {
	s := &Server{
		config: config,
		store:  store,
		broker: NewSSEBroker(),
	}
	return s
}

// Start starts the HTTP server and blocks until it's shut down.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf("%s:%d", s.config.Bind, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disabled for SSE
		IdleTimeout:  60 * time.Second,
	}

	// Start SSE broker
	go s.broker.Start(ctx)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Printf("Dashboard server listening on http://%s", addr)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Broker returns the SSE broker for publishing events.
func (s *Server) Broker() *SSEBroker {
	return s.broker
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	mux.HandleFunc("GET /api/runs/{id}/events", s.handleGetRunEvents)
	mux.HandleFunc("GET /api/runs/{id}/steps", s.handleGetRunSteps)
	mux.HandleFunc("GET /api/runs/{id}/artifacts", s.handleGetRunArtifacts)
	mux.HandleFunc("GET /api/runs/{id}/progress", s.handleGetRunProgress)

	// SSE endpoint
	mux.HandleFunc("GET /api/events", s.handleSSE)

	// Static files (embedded frontend)
	mux.Handle("GET /", staticHandler())
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers for local development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
