package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/recinq/wave/internal/dashboard"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// NewServeCmd creates the serve command for the web dashboard.
func NewServeCmd() *cobra.Command {
	var port int
	var bind string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard server",
		Long: `Start the Wave web dashboard server for monitoring pipeline executions.

The dashboard provides a read-only web interface for viewing pipeline runs,
step progress, events, and artifacts. Real-time updates are delivered via
Server-Sent Events (SSE).

Examples:
  wave serve                    # Start on localhost:8080
  wave serve --port 3000        # Start on localhost:3000
  wave serve --bind 0.0.0.0     # Listen on all interfaces`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(port, bind)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "Address to bind to")

	return cmd
}

func runServe(port int, bind string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", port)
	}

	dbPath := ".wave/state.db"

	// Check if state database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("state database not found at %s — run a pipeline first", dbPath)
	}

	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer store.Close()

	config := dashboard.ServerConfig{
		Port:   port,
		Bind:   bind,
		DBPath: dbPath,
	}

	srv := dashboard.NewServer(config, store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down dashboard server...")
		cancel()
	}()

	return srv.Start(ctx)
}
