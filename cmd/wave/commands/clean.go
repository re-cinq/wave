package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	_ "modernc.org/sqlite"
)

type CleanOptions struct {
	Pipeline  string
	All       bool
	Force     bool
	KeepLast  int
	DryRun    bool
	OlderThan string
	Status    string
	Quiet     bool
}

func NewCleanCmd() *cobra.Command {
	var opts CleanOptions

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up project artifacts",
		Long: `Remove generated workspaces, state files, and cached artifacts.
Use --all to remove everything, or --pipeline to target a specific run.
Use --keep-last N to retain the N most recent workspaces.
Use --dry-run to preview what would be deleted without actually removing anything.
Use --older-than to remove workspaces older than a specified duration (e.g., "7d", "24h", "1h30m").
Use --status to only clean workspaces for pipelines with a given status (completed, failed).
Use --quiet to suppress output for scripting (clean exit when nothing to clean).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Clean specific pipeline workspace")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Clean all workspaces and state")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip confirmation")
	cmd.Flags().IntVar(&opts.KeepLast, "keep-last", -1, "Keep the N most recent workspaces (use with --all)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be deleted without removing anything")
	cmd.Flags().StringVar(&opts.OlderThan, "older-than", "", "Remove workspaces older than specified duration (e.g., \"7d\", \"24h\", \"1h30m\")")
	cmd.Flags().StringVar(&opts.Status, "status", "", "Only clean workspaces for pipelines with given status (completed, failed)")
	cmd.Flags().BoolVar(&opts.Quiet, "quiet", false, "Suppress output for scripting")

	return cmd
}

// parseDuration parses duration strings like "7d", "24h", "1h30m"
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Check for day suffix (not supported by time.ParseDuration)
	dayRegex := regexp.MustCompile(`^(\d+)d(.*)$`)
	if matches := dayRegex.FindStringSubmatch(s); len(matches) == 3 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", matches[1])
		}
		remaining := matches[2]
		var extraDuration time.Duration
		if remaining != "" {
			var err error
			extraDuration, err = time.ParseDuration(remaining)
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %s", s)
			}
		}
		return time.Duration(days)*24*time.Hour + extraDuration, nil
	}

	return time.ParseDuration(s)
}

// getWorkspacesWithStatus returns workspace names that match the given status from the database
func getWorkspacesWithStatus(status string) (map[string]bool, error) {
	result := make(map[string]bool)
	dbPath := ".wave/state.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return result, nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return result, nil // Fail silently, return empty set
	}
	defer db.Close()

	// Query from pipeline_run table for status
	rows, err := db.Query(`
		SELECT DISTINCT pipeline_name FROM pipeline_run
		WHERE LOWER(status) = LOWER(?)
	`, status)
	if err != nil {
		// Try fallback to pipeline_state table
		rows, err = db.Query(`
			SELECT DISTINCT pipeline_name FROM pipeline_state
			WHERE LOWER(status) = LOWER(?)
		`, status)
		if err != nil {
			return result, nil
		}
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			result[name] = true
		}
	}

	return result, nil
}

// calculateDirectorySize calculates the total size of files in a directory
func calculateDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// isTTY checks if stdin is a terminal
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}


func runClean(opts CleanOptions) error {
	if !opts.All && opts.Pipeline == "" && opts.OlderThan == "" && opts.Status == "" {
		return fmt.Errorf("specify --all or --pipeline <name> or --older-than <duration> or --status <status>")
	}

	// Validate status if provided
	if opts.Status != "" {
		validStatuses := map[string]bool{"completed": true, "failed": true, "running": true, "cancelled": true, "pending": true}
		if !validStatuses[strings.ToLower(opts.Status)] {
			return fmt.Errorf("invalid status: %s (valid: completed, failed, running, cancelled, pending)", opts.Status)
		}
	}

	// Parse older-than duration
	var olderThanDuration time.Duration
	if opts.OlderThan != "" {
		var err error
		olderThanDuration, err = parseDuration(opts.OlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than duration: %w", err)
		}
	}

	targets := []string{}
	wsDir := ".wave/workspaces"

	// Get workspaces with matching status (if status filter is provided)
	var statusWorkspaces map[string]bool
	if opts.Status != "" {
		var err error
		statusWorkspaces, err = getWorkspacesWithStatus(opts.Status)
		if err != nil {
			return fmt.Errorf("failed to get workspaces by status: %w", err)
		}
	}

	if opts.All || opts.OlderThan != "" || opts.Status != "" {
		// Handle workspace-based filtering
		if opts.KeepLast >= 0 || opts.OlderThan != "" || opts.Status != "" {
			// Get workspaces sorted by modification time using workspace package
			workspaces, err := workspace.ListWorkspacesSortedByTime(wsDir)
			if err != nil {
				return fmt.Errorf("failed to list workspaces: %w", err)
			}

			cutoffTime := time.Now().Add(-olderThanDuration)

			// Filter workspaces
			var candidatesForRemoval []workspace.WorkspaceInfo
			for _, ws := range workspaces {
				wsTime := time.Unix(0, ws.ModTime)

				// Apply older-than filter
				if opts.OlderThan != "" && wsTime.After(cutoffTime) {
					continue
				}

				// Apply status filter
				if opts.Status != "" && len(statusWorkspaces) > 0 {
					if !statusWorkspaces[ws.Name] {
						continue
					}
				}

				candidatesForRemoval = append(candidatesForRemoval, ws)
			}

			// Apply keep-last (only if --all is also specified)
			if opts.KeepLast >= 0 && opts.All {
				// Sort all workspaces and keep the most recent ones
				allWorkspaces, _ := workspace.ListWorkspacesSortedByTime(wsDir)
				keepSet := make(map[string]bool)
				if len(allWorkspaces) > opts.KeepLast {
					// Mark the most recent ones to keep
					for i := len(allWorkspaces) - opts.KeepLast; i < len(allWorkspaces); i++ {
						keepSet[allWorkspaces[i].Name] = true
					}
				} else {
					// Keep all if fewer than KeepLast
					for _, ws := range allWorkspaces {
						keepSet[ws.Name] = true
					}
				}

				// Filter out the ones we need to keep
				var filtered []workspace.WorkspaceInfo
				for _, ws := range candidatesForRemoval {
					if !keepSet[ws.Name] {
						filtered = append(filtered, ws)
					}
				}
				candidatesForRemoval = filtered
			}

			for _, ws := range candidatesForRemoval {
				targets = append(targets, ws.Path)
			}

			// When using filters, we only affect workspaces, not state.db or traces
		} else if opts.All {
			// Default behavior: remove everything
			targets = append(targets,
				".wave/state.db",
				".wave/traces",
				".wave/workspaces",
			)
		}
	} else if opts.Pipeline != "" {
		// Validate pipeline name to prevent path traversal
		if strings.Contains(opts.Pipeline, "..") || filepath.IsAbs(opts.Pipeline) || strings.ContainsAny(opts.Pipeline, `/\`) {
			return fmt.Errorf("invalid pipeline name: %s", opts.Pipeline)
		}
		targets = append(targets,
			filepath.Join(".wave", "workspaces", opts.Pipeline),
		)
	}

	// Filter to existing targets
	var existingTargets []string
	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			existingTargets = append(existingTargets, target)
		}
	}

	if len(existingTargets) == 0 {
		if !opts.Quiet {
			fmt.Printf("Nothing to clean\n")
		}
		return nil
	}

	// Calculate total size and count for confirmation
	var totalSize int64
	workspaceCount := len(existingTargets)
	for _, target := range existingTargets {
		size, _ := calculateDirectorySize(target)
		totalSize += size
	}

	// TTY detection and confirmation
	if !opts.Force && !opts.DryRun {
		if !isTTY() {
			if !opts.Quiet {
				fmt.Printf("Stdin is not a TTY. Use --force to proceed with cleanup.\n")
				fmt.Printf("Would clean %d item(s), total size: %s\n", workspaceCount, formatSize(totalSize))
			}
			return nil
		}

		// Show confirmation prompt
		fmt.Printf("About to remove %d item(s), total size: %s\n", workspaceCount, formatSize(totalSize))
		fmt.Printf("Continue? [y/N] ")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			if !opts.Quiet {
				fmt.Printf("Aborted\n")
			}
			return nil
		}
	}

	if opts.DryRun {
		if !opts.Quiet {
			fmt.Printf("(dry-run) The following would be removed:\n")
			fmt.Printf("  Total: %d item(s), %s\n", workspaceCount, formatSize(totalSize))
			for _, target := range existingTargets {
				size, _ := calculateDirectorySize(target)
				fmt.Printf("  Would remove %s (%s)\n", target, formatSize(size))
			}
		}
		return nil
	}

	// Batch processing for large cleanups with progress indicator
	batchSize := 100
	cleaned := 0
	failed := 0
	showProgress := len(existingTargets) > 10 && !opts.Quiet

	for i := 0; i < len(existingTargets); i += batchSize {
		end := i + batchSize
		if end > len(existingTargets) {
			end = len(existingTargets)
		}
		batch := existingTargets[i:end]

		for _, target := range batch {
			// Ensure all dirs are writable before removal (readonly mounts block removal)
			filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					os.Chmod(path, 0755)
				}
				return nil
			})
			if err := os.RemoveAll(target); err != nil {
				failed++
				if !opts.Quiet {
					fmt.Printf("  Failed to remove %s: %s\n", target, err)
				}
				continue
			}
			if !opts.Quiet {
				fmt.Printf("  Removed %s\n", target)
			}
			cleaned++
		}

		// Show progress for large cleanups
		if showProgress && end < len(existingTargets) {
			fmt.Printf("  Progress: %d/%d items processed\n", end, len(existingTargets))
		}
	}

	if !opts.Quiet {
		if cleaned == 0 && failed == 0 {
			fmt.Printf("Nothing to clean\n")
		} else if failed > 0 {
			fmt.Printf("\nCleaned %d item(s), failed to clean %d item(s)\n", cleaned, failed)
		} else {
			fmt.Printf("\nCleaned %d item(s)\n", cleaned)
		}
	}

	return nil
}
