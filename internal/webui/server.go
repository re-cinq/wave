package webui

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	_ "modernc.org/sqlite"
)

// AuthMode determines how the server authenticates requests.
type AuthMode string

const (
	AuthModeNone   AuthMode = "none"
	AuthModeBearer AuthMode = "bearer"
	AuthModeJWT    AuthMode = "jwt"
	AuthModeMTLS   AuthMode = "mtls"
)

// Server is the HTTP server for the Wave dashboard.
type Server struct {
	httpServer        *http.Server
	store             state.StateStore
	rwStore           state.StateStore // read-write store for execution control
	manifest          *manifest.Manifest
	templates         map[string]*template.Template
	broker            *SSEBroker
	wsManager         workspace.WorkspaceManager
	forgeClient       forge.Client
	repoSlug          string // "owner/repo"
	repoDir           string // git repository root directory
	bind              string
	port              int
	token             string
	authMode          AuthMode
	jwtSecret         string
	scheduler         *Scheduler
	gateRegistry      *GateRegistry
	activeRuns        map[string]context.CancelFunc // runID -> cancel
	disabledPipelines map[string]bool               // pipeline name -> disabled
	mu                sync.Mutex
	tlsCert           string
	tlsKey            string
	tlsCA             string
	csrfToken         string
}

// ServerConfig holds configuration for the dashboard server.
type ServerConfig struct {
	Bind          string
	Port          int
	DBPath        string
	Manifest      *manifest.Manifest
	Token         string
	AuthMode      AuthMode
	JWTSecret     string
	MaxConcurrent int
	TLSCert       string
	TLSKey        string
	TLSCA         string // CA cert for mTLS client verification
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

	// Backfill pipeline_run.total_tokens from event_log for runs that have 0
	backfillRunTokens(cfg.DBPath)

	// Generate a per-session CSRF token
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		roStore.Close()
		rwStore.Close()
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	tmpl, err := parseTemplates(template.FuncMap{
		"csrfToken": func() string { return csrfToken },
	})
	if err != nil {
		roStore.Close()
		rwStore.Close()
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Initialize workspace manager
	wsRoot := ".wave/workspaces"
	if cfg.Manifest != nil && cfg.Manifest.Runtime.WorkspaceRoot != "" {
		wsRoot = cfg.Manifest.Runtime.WorkspaceRoot
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		log.Printf("[webui] failed to initialize workspace manager: %v", err)
	}

	// Detect forge and initialize client
	forgeInfo, _ := forge.DetectFromGitRemotes()
	forgeClient := forge.NewClient(forgeInfo)
	repoSlug := forgeInfo.Slug()

	// Resolve git repo root for safe subprocess execution
	repoDir := "."
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		repoDir = strings.TrimSpace(string(out))
	}

	// Resolve auth mode
	authMode := cfg.AuthMode
	if authMode == "" {
		if cfg.Token != "" {
			authMode = AuthModeBearer
		} else {
			authMode = AuthModeNone
		}
	}

	s := &Server{
		store:             roStore,
		rwStore:           rwStore,
		manifest:          cfg.Manifest,
		templates:         tmpl,
		broker:            NewSSEBroker(),
		wsManager:         wsManager,
		forgeClient:       forgeClient,
		repoSlug:          repoSlug,
		repoDir:           repoDir,
		bind:              cfg.Bind,
		port:              cfg.Port,
		token:             cfg.Token,
		authMode:          authMode,
		jwtSecret:         cfg.JWTSecret,
		scheduler:         NewScheduler(cfg.MaxConcurrent),
		gateRegistry:      NewGateRegistry(),
		activeRuns:        make(map[string]context.CancelFunc),
		disabledPipelines: make(map[string]bool),
		tlsCert:           cfg.TLSCert,
		tlsKey:            cfg.TLSKey,
		tlsCA:             cfg.TLSCA,
		csrfToken:         csrfToken,
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

// backfillRunTokens updates pipeline_run.total_tokens from event_log for runs
// that still have 0 tokens. This fixes a historical bug where tokens were lost
// because the executor cleaned up in-memory state before GetTotalTokens was called.
func backfillRunTokens(dbPath string) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("[webui] backfill tokens: failed to open db: %v", err)
		return
	}
	defer db.Close()

	result, err := db.Exec(`
		UPDATE pipeline_run SET total_tokens = (
			SELECT COALESCE(SUM(el.tokens_used), 0)
			FROM event_log el
			WHERE el.run_id = pipeline_run.run_id AND el.tokens_used > 0
		)
		WHERE total_tokens = 0
		AND status IN ('completed', 'failed', 'cancelled')
	`)
	if err != nil {
		log.Printf("[webui] backfill tokens: %v", err)
		return
	}
	if n, _ := result.RowsAffected(); n > 0 {
		log.Printf("[webui] backfilled tokens for %d runs", n)
	}
}

// Start starts the HTTP server and blocks until shutdown.
func (s *Server) Start() error {
	go s.broker.Start()

	addr := fmt.Sprintf("%s:%d", s.bind, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Configure TLS if enabled
	if s.tlsCert != "" && s.tlsKey != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// Load server certificate
		cert, err := tls.LoadX509KeyPair(s.tlsCert, s.tlsKey)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		// Configure mTLS if auth mode is mtls
		if s.authMode == AuthModeMTLS && s.tlsCA != "" {
			caCert, err := os.ReadFile(s.tlsCA)
			if err != nil {
				return fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.ClientCAs = caCertPool
		}

		listener = tls.NewListener(listener, tlsConfig)
		fmt.Fprintf(os.Stderr, "Wave dashboard running at https://%s\n", addr)
	} else {
		fmt.Fprintf(os.Stderr, "Wave dashboard running at http://%s\n", addr)
	}

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

		// Cancel all active runs
		s.mu.Lock()
		for runID, cancelFn := range s.activeRuns {
			log.Printf("Cancelling active run %s", runID)
			cancelFn()
		}
		s.mu.Unlock()

		// Drain scheduler queue
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer drainCancel()
		if err := s.scheduler.Shutdown(drainCtx); err != nil {
			log.Printf("Warning: scheduler drain timed out: %v", err)
		}

		// Shutdown HTTP server
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
