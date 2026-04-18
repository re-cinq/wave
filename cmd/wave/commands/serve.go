package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/webui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewServeCmd creates the serve command for the dashboard server.
func NewServeCmd() *cobra.Command {
	var (
		port          int
		bind          string
		token         string
		dbPath        string
		manifestPath  string
		maxConcurrent int
		authMode      string
		tlsCert       string
		tlsKey        string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard server",
		Long: `Start an HTTP server that serves the Wave operations dashboard.

The dashboard provides real-time pipeline monitoring, execution control,
DAG visualization, and artifact browsing through a web interface.

By default, the server binds to localhost:8080. When binding to a
non-localhost address, authentication is required via bearer token.

Authentication modes:
  none    - No authentication (default for localhost)
  bearer  - Bearer token authentication (default for non-localhost)
  jwt     - JWT token authentication (requires WAVE_JWT_SECRET)
  mtls    - Mutual TLS client certificate authentication`,
		Example: `  wave serve
  wave serve --port 9090
  wave serve --bind 0.0.0.0 --token mysecret
  wave serve --max-concurrent 10
  wave serve --auth-mode jwt
  wave serve --tls-cert cert.pem --tls-key key.pem
  wave serve --db .agents/state.db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load manifest if available
			var m *manifest.Manifest
			manifestData, err := os.ReadFile(manifestPath)
			if err == nil {
				var parsed manifest.Manifest
				if err := yaml.Unmarshal(manifestData, &parsed); err == nil {
					m = &parsed
				}
			}
			// Manifest is optional - server can start without it

			// Build config from manifest defaults, then override with CLI flags
			cfg := buildServerConfig(m, port, bind, token, dbPath, maxConcurrent, authMode, tlsCert, tlsKey, cmd)

			srv, err := webui.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("failed to create dashboard server: %w", err)
			}

			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "Address to bind to")
	cmd.Flags().StringVar(&token, "token", "", "Authentication token (required for non-localhost binding)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to state database (default: .agents/state.db)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 0, "Maximum concurrent pipeline runs (default: 5)")
	cmd.Flags().StringVar(&authMode, "auth-mode", "", "Authentication mode: none, bearer, jwt, mtls")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "Path to TLS certificate file")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "Path to TLS key file")

	return cmd
}

// buildServerConfig merges manifest server config with CLI flags.
// CLI flags take precedence over manifest values.
func buildServerConfig(m *manifest.Manifest, port int, bind, token, dbPath string, maxConcurrent int, authMode, tlsCert, tlsKey string, cmd *cobra.Command) webui.ServerConfig {
	// Start with defaults
	cfg := webui.ServerConfig{
		Bind:     bind,
		Port:     port,
		Manifest: m,
	}

	// Apply manifest server config as base (if present)
	if m != nil && m.Server != nil {
		sc := m.Server
		if sc.Bind != "" && !cmd.Flags().Changed("bind") {
			cfg.Bind = sc.Bind
			// Parse host:port if bind contains a colon with port
			if parts := strings.SplitN(sc.Bind, ":", 2); len(parts) == 2 {
				cfg.Bind = parts[0]
				var p int
				if n, _ := fmt.Sscanf(parts[1], "%d", &p); n == 1 && !cmd.Flags().Changed("port") {
					cfg.Port = p
				}
			}
		}
		if sc.MaxConcurrent > 0 && !cmd.Flags().Changed("max-concurrent") {
			maxConcurrent = sc.MaxConcurrent
		}
		if sc.Auth.Mode != "" && !cmd.Flags().Changed("auth-mode") {
			authMode = sc.Auth.Mode
		}
		if sc.Auth.JWTSecret != "" {
			cfg.JWTSecret = expandEnvVars(sc.Auth.JWTSecret)
		}
		if sc.TLS.Cert != "" && !cmd.Flags().Changed("tls-cert") {
			tlsCert = sc.TLS.Cert
		}
		if sc.TLS.Key != "" && !cmd.Flags().Changed("tls-key") {
			tlsKey = sc.TLS.Key
		}
		if sc.TLS.CA != "" {
			cfg.TLSCA = sc.TLS.CA
		}
	}

	// Apply CLI overrides
	cfg.MaxConcurrent = maxConcurrent
	cfg.TLSCert = tlsCert
	cfg.TLSKey = tlsKey

	// Resolve database path
	if dbPath == "" {
		dbPath = ".agents/state.db"
	}
	cfg.DBPath = dbPath

	// Resolve auth mode
	if authMode != "" {
		cfg.AuthMode = webui.AuthMode(authMode)
	}

	// JWT secret from env (CLI flag override of manifest)
	if cfg.JWTSecret == "" {
		if envSecret := os.Getenv("WAVE_JWT_SECRET"); envSecret != "" {
			cfg.JWTSecret = envSecret
		}
	}

	// Resolve token
	cfg.Token = resolveToken(token, cfg.Bind)

	return cfg
}

// resolveToken determines the auth token based on flag, env, or auto-generation.
func resolveToken(flagToken, bind string) string {
	// If token provided via flag, use it
	if flagToken != "" {
		return flagToken
	}

	// Check environment variable
	if envToken := os.Getenv("WAVE_SERVE_TOKEN"); envToken != "" {
		return envToken
	}

	// Auto-generate for non-localhost binding
	if bind != "127.0.0.1" && bind != "localhost" && bind != "" {
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err == nil {
			return hex.EncodeToString(tokenBytes)
		}
	}

	return ""
}

// expandEnvVars expands ${VAR} patterns in a string.
func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}
