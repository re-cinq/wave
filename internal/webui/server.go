package webui

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/recinq/wave/internal/attention"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
)

// AuthMode determines how the server authenticates requests.
type AuthMode string

const (
	AuthModeNone   AuthMode = "none"
	AuthModeBearer AuthMode = "bearer"
	AuthModeJWT    AuthMode = "jwt"
	AuthModeMTLS   AuthMode = "mtls"
)

// serverTransport groups HTTP-only fields: the underlying server and its
// bind address.
type serverTransport struct {
	httpServer *http.Server
	bind       string
	port       int
}

// serverAuth groups credentials and TLS material used to authenticate and
// secure incoming requests.
type serverAuth struct {
	token     string
	authMode  AuthMode
	jwtSecret string
	tlsCert   string
	tlsKey    string
	tlsCA     string
	csrfToken string
}

// serverRuntime groups the long-lived runtime collaborators: state stores,
// manifest, workspace manager, forge client, and pipeline scheduler.
type serverRuntime struct {
	store       state.StateStore
	rwStore     state.StateStore // read-write store for execution control
	manifest    *manifest.Manifest
	wsManager   workspace.WorkspaceManager
	forgeClient forge.Client
	repoSlug    string // "owner/repo"
	repoDir     string // git repository root directory
	scheduler   *Scheduler
}

// serverRealtime groups the realtime/eventing collaborators: SSE broker,
// gate registry, attention broker, and the live run/pipeline tracking maps.
type serverRealtime struct {
	broker            *SSEBroker
	gateRegistry      *GateRegistry
	attention         *attention.Broker
	activeRuns        map[string]context.CancelFunc // runID -> cancel
	disabledPipelines map[string]bool               // pipeline name -> disabled
}

// serverAssets groups templates, the in-memory API cache, and the feature
// registry — the read-mostly assets used to render responses.
type serverAssets struct {
	templates map[string]*template.Template
	cache     *apiCache
	features  *FeatureRegistry
}

// Server is the HTTP server for the Wave dashboard.
type Server struct {
	transport serverTransport
	auth      serverAuth
	runtime   serverRuntime
	realtime  serverRealtime
	assets    serverAssets
	mu        sync.Mutex
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
	// Features is the optional feature registry. When nil, NewServer
	// constructs one via NewFeatureRegistry(), which selects the appropriate
	// per-feature implementations based on build tags.
	Features *FeatureRegistry
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

	// Reclaim zombie runs left over from previously-killed wave run processes
	// (terminal close, OOM, signal). Without this, the run list accumulates
	// "running" rows forever — see fix/zombie-reconciliation for context.
	if reclaimed := state.ReconcileZombies(rwStore, 0); reclaimed > 0 {
		log.Printf("[webui] reconciled %d zombie run(s) on boot", reclaimed)
	}

	// Generate a per-session CSRF token
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		roStore.Close()
		rwStore.Close()
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	features := cfg.Features
	if features == nil {
		features = NewFeatureRegistry()
	}

	tmpl, err := parseTemplates(template.FuncMap{
		"csrfToken": func() string { return csrfToken },
		"featureEnabled": func(name string) bool {
			switch name {
			case "metrics":
				return features.Features.Metrics
			case "analytics":
				return features.Features.Analytics
			case "webhooks":
				return features.Features.Webhooks
			case "ontology":
				return features.Features.Ontology
			case "retros":
				return features.Features.Retros
			default:
				return false
			}
		},
	})
	if err != nil {
		roStore.Close()
		rwStore.Close()
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Initialize workspace manager
	wsRoot := ".agents/workspaces"
	if cfg.Manifest != nil && cfg.Manifest.Runtime.WorkspaceRoot != "" {
		wsRoot = cfg.Manifest.Runtime.WorkspaceRoot
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		log.Printf("[webui] failed to initialize workspace manager: %v", err)
	}

	// Detect forge and initialize client. A non-nil error here means
	// construction itself failed; treat the client as unconfigured and log
	// so the dashboard still renders.
	forgeInfo, _ := forge.DetectFromGitRemotes()
	forgeClient, err := forge.NewClient(forgeInfo)
	if err != nil {
		log.Printf("[webui] forge client init failed: %v", err)
		forgeClient = nil
	}
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
		transport: serverTransport{
			bind: cfg.Bind,
			port: cfg.Port,
		},
		auth: serverAuth{
			token:     cfg.Token,
			authMode:  authMode,
			jwtSecret: cfg.JWTSecret,
			tlsCert:   cfg.TLSCert,
			tlsKey:    cfg.TLSKey,
			tlsCA:     cfg.TLSCA,
			csrfToken: csrfToken,
		},
		runtime: serverRuntime{
			store:       roStore,
			rwStore:     rwStore,
			manifest:    cfg.Manifest,
			wsManager:   wsManager,
			forgeClient: forgeClient,
			repoSlug:    repoSlug,
			repoDir:     repoDir,
			scheduler:   NewScheduler(cfg.MaxConcurrent),
		},
		realtime: serverRealtime{
			broker:            NewSSEBroker(),
			gateRegistry:      NewGateRegistry(),
			attention:         attention.NewBroker(),
			activeRuns:        make(map[string]context.CancelFunc),
			disabledPipelines: make(map[string]bool),
		},
		assets: serverAssets{
			templates: tmpl,
			cache:     newAPICache(5 * time.Minute),
			features:  features,
		},
	}

	// Wire attention broker into the SSE broker so pipeline events
	// are automatically forwarded to the attention classifier.
	s.realtime.broker.attentionSink = s.realtime.attention

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Port)
	s.transport.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.applyMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for SSE
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// pollAttention periodically queries the DB for active runs and updates the
// attention broker. This is needed because detached subprocess runs don't emit
// events through the in-memory SSE broker — they write directly to the DB.
func (s *Server) pollAttention(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.syncAttentionFromDB()
		case <-ctx.Done():
			return
		}
	}
}

// reconcileZombiesLoop periodically scans the DB for "running" runs whose
// owning process is gone and marks them failed. This complements the one-shot
// reconcile that runs at server boot, catching runs that died after boot.
func (s *Server) reconcileZombiesLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if reclaimed := state.ReconcileZombies(s.runtime.rwStore, 0); reclaimed > 0 {
				log.Printf("[webui] reconciled %d zombie run(s)", reclaimed)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) syncAttentionFromDB() {
	if s.realtime.attention == nil {
		return
	}
	// Only track running runs. Completed and failed runs are done —
	// they don't need live attention, the run detail page shows their status.
	running, err := s.runtime.store.ListRuns(state.ListRunsOptions{Status: "running"})
	if err != nil {
		log.Printf("[attention] failed to list running runs: %v", err)
		return
	}

	now := time.Now()
	seen := make(map[string]bool)

	for _, r := range running {
		seen[r.RunID] = true
		s.realtime.attention.UpdateWithName(r.RunID, r.PipelineName, event.Event{
			PipelineID: r.RunID,
			State:      "running",
			StepID:     r.CurrentStep,
			Timestamp:  now,
		})
	}

	// Clear completed runs that the attention broker still tracks.
	summary := s.realtime.attention.Summary()
	for _, ra := range summary.Runs {
		if !seen[ra.RunID] {
			s.realtime.attention.Update(event.Event{
				PipelineID: ra.RunID,
				State:      "completed",
				Timestamp:  now,
			})
		}
	}
}

// Start starts the HTTP server and blocks until shutdown.
func (s *Server) Start() error {
	go s.realtime.broker.Start()
	attentionCtx, attentionCancel := context.WithCancel(context.Background())
	defer attentionCancel()
	go s.pollAttention(attentionCtx)
	go s.reconcileZombiesLoop(attentionCtx)

	addr := fmt.Sprintf("%s:%d", s.transport.bind, s.transport.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Configure TLS if enabled
	if s.auth.tlsCert != "" && s.auth.tlsKey != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// Load server certificate
		cert, err := tls.LoadX509KeyPair(s.auth.tlsCert, s.auth.tlsKey)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		// Configure mTLS if auth mode is mtls
		if s.auth.authMode == AuthModeMTLS && s.auth.tlsCA != "" {
			caCert, err := os.ReadFile(s.auth.tlsCA)
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

	if s.auth.token != "" && s.transport.bind != "127.0.0.1" && s.transport.bind != "localhost" {
		fmt.Fprintf(os.Stderr, "Dashboard token: %s\n", s.auth.token)
	}

	// Issue #1467 — reap any "running" rows whose owning process died
	// without writing the deferred UpdateRunStatus (host sleep, sandbox
	// cycle, SIGKILL). Heartbeats fire every 30s; treat anything stale
	// for > 5 minutes as orphaned.
	if s.runtime.store != nil {
		if reaped, err := s.runtime.store.ReapOrphans(5 * time.Minute); err != nil {
			fmt.Fprintf(os.Stderr, "warning: orphan reap failed: %v\n", err)
		} else if reaped > 0 {
			fmt.Fprintf(os.Stderr, "Reaped %d orphaned run(s) on startup\n", reaped)
		}
	}

	// Graceful shutdown on interrupt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.transport.httpServer.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		log.Println("Shutting down dashboard server...")

		// Cancel in-process fallback runs only (detached runs are independent processes)
		s.mu.Lock()
		for runID, cancelFn := range s.realtime.activeRuns {
			log.Printf("Cancelling in-process run %s", runID)
			cancelFn()
		}
		s.mu.Unlock()

		// Drain scheduler queue
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer drainCancel()
		if err := s.runtime.scheduler.Shutdown(drainCtx); err != nil {
			log.Printf("Warning: scheduler drain timed out: %v", err)
		}

		// Shutdown HTTP server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.transport.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	s.realtime.broker.Stop()
	s.runtime.store.Close()
	s.runtime.rwStore.Close()

	return nil
}

// GetBroker returns the SSE broker for external event integration.
func (s *Server) GetBroker() *SSEBroker {
	return s.realtime.broker
}
