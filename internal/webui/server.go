//go:build webui

package webui

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// Server is the HTTP server for the Wave dashboard.
type Server struct {
	httpServer *http.Server
	store      state.StateStore
	rwStore    state.StateStore // read-write store for execution control
	manifest   *manifest.Manifest
	templates  *template.Template
	broker     *SSEBroker
	bind       string
	port       int
	token      string
}

// ServerConfig holds configuration for the dashboard server.
type ServerConfig struct {
	Bind     string
	Port     int
	DBPath   string
	Manifest *manifest.Manifest
	Token    string
}

// NewServer creates a new dashboard server instance.
func NewServer(cfg ServerConfig) (*Server, error) {
	roStore, err := state.NewReadOnlyStateStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open read-only state store: %w", err)
	}

	// Open a read-write store for execution control operations
	rwStore, err := state.NewStateStore(cfg.DBPath)
	if err != nil {
		roStore.Close()
		return nil, fmt.Errorf("failed to open read-write state store: %w", err)
	}

	tmpl, err := parseTemplates()
	if err != nil {
		roStore.Close()
		rwStore.Close()
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	s := &Server{
		store:     roStore,
		rwStore:   rwStore,
		manifest:  cfg.Manifest,
		templates: tmpl,
		broker:    NewSSEBroker(),
		bind:      cfg.Bind,
		port:      cfg.Port,
		token:     cfg.Token,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.applyMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for SSE
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Start starts the HTTP server and blocks until shutdown.
func (s *Server) Start() error {
	go s.broker.Start()

	addr := fmt.Sprintf("%s:%d", s.bind, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	fmt.Fprintf(os.Stderr, "Wave dashboard running at http://%s\n", addr)
	if s.token != "" && s.bind != "127.0.0.1" && s.bind != "localhost" {
		fmt.Fprintf(os.Stderr, "Dashboard token: %s\n", s.token)
	}

	// Graceful shutdown on interrupt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		log.Println("Shutting down dashboard server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	s.broker.Stop()
	s.store.Close()
	s.rwStore.Close()

	return nil
}

// GetBroker returns the SSE broker for external event integration.
func (s *Server) GetBroker() *SSEBroker {
	return s.broker
}
