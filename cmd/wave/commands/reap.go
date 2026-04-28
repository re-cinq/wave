package commands

import (
	"fmt"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// NewReapCmd creates the reap command.
func NewReapCmd() *cobra.Command {
	var staleAfter time.Duration

	cmd := &cobra.Command{
		Use:   "reap",
		Short: "Mark stale 'running' pipeline runs as failed",
		Long: `Scan the state store for runs stuck at status='running' whose owning
process is no longer alive (no recent heartbeat) and mark them failed
with reason "orphaned (no heartbeat for Ns)".

Runs are considered orphaned when:
  - status = 'running'
  - started_at is older than --stale-after (default 5m)
  - last_heartbeat is older than --stale-after, or never reported

The same logic runs automatically at 'wave serve' startup, but this
command lets you flush the table without restarting the dashboard.

Heartbeats fire every 30s on a healthy run, so the default 5m threshold
provides a 10x safety margin against false positives.`,
		Example: `  wave reap                          # Reap with default 5m staleness
  wave reap --stale-after 30m        # Stricter threshold for batch cleanup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := state.NewStateStore(".agents/state.db")
			if err != nil {
				return fmt.Errorf("open state store: %w", err)
			}
			defer store.Close()

			reaped, err := store.ReapOrphans(staleAfter)
			if err != nil {
				return fmt.Errorf("reap: %w", err)
			}
			if reaped == 0 {
				fmt.Println("No orphaned runs found.")
				return nil
			}
			fmt.Printf("Reaped %d orphaned run(s).\n", reaped)
			return nil
		},
	}

	cmd.Flags().DurationVar(&staleAfter, "stale-after", 5*time.Minute,
		"Treat 'running' runs as orphaned when their last heartbeat is older than this")

	return cmd
}
