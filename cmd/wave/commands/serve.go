//go:build webui

package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/webui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewServeCmd creates the serve command for the dashboard server.
func NewServeCmd() *cobra.Command {
	var (
		port         int
		bind         string
		token        string
		dbPath       string
		manifestPath string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard server",
		Long: `Start an HTTP server that serves the Wave operations dashboard.

The dashboard provides real-time pipeline monitoring, execution control,
DAG visualization, and artifact browsing through a web interface.

By default, the server binds to localhost:8080. When binding to a
non-localhost address, authentication is required via bearer token.`,
		Example: `  wave serve
  wave serve --port 9090
  wave serve --bind 0.0.0.0 --token mysecret
  wave serve --db .wave/state.db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve token for non-localhost binding
			resolvedToken := resolveToken(token, bind)

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

			// Resolve database path
			if dbPath == "" {
				dbPath = ".wave/state.db"
			}

			cfg := webui.ServerConfig{
				Bind:     bind,
				Port:     port,
				DBPath:   dbPath,
				Manifest: m,
				Token:    resolvedToken,
			}

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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to state database (default: .wave/state.db)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
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
