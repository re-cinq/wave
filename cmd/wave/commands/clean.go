package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	// Manifest is the manifest whose runtime.workspace_root is honoured;
	// empty means wave.yaml.
	Manifest string
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
			opts.Manifest, _ = cmd.Root().PersistentFlags().GetString("manifest")
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

// getWorkspacesWithStatus returns workspace names that match the given status from the database
func getWorkspacesWithStatus(status string) (map[string]bool, error) {
	result := make(map[string]bool)
	dbPath := ".agents/state.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return result, nil
	}

	store, err := state.NewReadOnlyStateStore(dbPath)
	if err != nil {
		return result, nil // Fail silently, return empty set
	}
	defer store.Close()

	names, err := store.ListPipelineNamesByStatus(status)
	if err != nil {
		return result, nil
	}
	for _, name := range names {
		result[name] = true
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

// isTTY checks if stdin is a terminal
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func runClean(opts CleanOptions) error {
	olderThan, err := validateCleanOptions(opts)
	if err != nil {
		return err
	}

	targets, err := collectCleanTargets(opts, olderThan)
	if err != nil {
		return err
	}

	existingTargets := existingPaths(targets)
	if len(existingTargets) == 0 {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Nothing to clean\n")
		}
		return nil
	}

	// Calculate total size and count for confirmation
	var totalSize int64
	for _, target := range existingTargets {
		size, _ := calculateDirectorySize(target)
		totalSize += size
	}

	if !opts.Force && !opts.DryRun && !confirmClean(opts, len(existingTargets), totalSize) {
		return nil
	}

	if opts.DryRun {
		if !opts.Quiet {
			printCleanDryRun(existingTargets, totalSize)
		}
		return nil
	}

	cleaned, failed := removeCleanTargets(existingTargets, opts.Quiet)
	if !opts.Quiet {
		switch {
		case cleaned == 0 && failed == 0:
			fmt.Fprintf(os.Stderr, "Nothing to clean\n")
		case failed > 0:
			fmt.Fprintf(os.Stderr, "\nCleaned %d item(s), failed to clean %d item(s)\n", cleaned, failed)
		default:
			fmt.Fprintf(os.Stderr, "\nCleaned %d item(s)\n", cleaned)
		}
	}

	return nil
}

// validateCleanOptions checks the flag combination and returns the parsed
// --older-than duration (zero when unset).
func validateCleanOptions(opts CleanOptions) (time.Duration, error) {
	if !opts.All && opts.Pipeline == "" && opts.OlderThan == "" && opts.Status == "" {
		return 0, NewCLIError(CodeInvalidArgs, "specify --all or --pipeline <name> or --older-than <duration> or --status <status>", "Use --all, --pipeline <name>, --older-than <duration>, or --status <status>")
	}

	if opts.Status != "" {
		validStatuses := map[string]bool{"completed": true, "failed": true, "running": true, "cancelled": true, "pending": true}
		if !validStatuses[strings.ToLower(opts.Status)] {
			return 0, NewCLIError(CodeInvalidArgs, fmt.Sprintf("invalid status: %s", opts.Status), "Valid statuses: completed, failed, running, cancelled, pending")
		}
	}

	if opts.OlderThan == "" {
		return 0, nil
	}
	olderThan, err := parseDuration(opts.OlderThan)
	if err != nil {
		return 0, NewCLIError(CodeInvalidArgs, fmt.Sprintf("invalid --older-than duration: %s", err), "Use a duration like '7d', '24h', or '1h30m'").WithCause(err)
	}
	return olderThan, nil
}

// collectCleanTargets returns the paths runClean should remove for the given
// options. Filtered cleanups (--keep-last, --older-than, --status) only touch
// workspaces; a bare --all also removes state.db, traces and retros.
func collectCleanTargets(opts CleanOptions, olderThan time.Duration) ([]string, error) {
	wsDir := manifestWorkspaceRoot(opts.Manifest)

	if opts.All || opts.OlderThan != "" || opts.Status != "" {
		if opts.KeepLast < 0 && opts.OlderThan == "" && opts.Status == "" {
			// --all without filters: remove everything, including
			// workspaces left in the checkout by older versions.
			targets := []string{
				".agents/state.db",
				".agents/traces",
				".agents/retros",
			}
			for _, root := range workspaceRootsToSweep(opts.Manifest) {
				if containsCwd(root) {
					continue // never delete the project itself
				}
				targets = append(targets, root)
			}
			return targets, nil
		}
		return filteredWorkspaceTargets(opts, wsDir, olderThan)
	}

	if opts.Pipeline != "" {
		// Validate pipeline name to prevent path traversal
		if strings.Contains(opts.Pipeline, "..") || filepath.IsAbs(opts.Pipeline) || strings.ContainsAny(opts.Pipeline, `/\`) {
			return nil, NewCLIError(CodeSecurityViolation, fmt.Sprintf("invalid pipeline name: %s", opts.Pipeline), "Pipeline names must not contain path separators or '..' sequences")
		}
		return []string{filepath.Join(wsDir, opts.Pipeline)}, nil
	}
	return nil, nil
}

// filteredWorkspaceTargets applies --older-than, --status and --keep-last to
// the workspaces under wsDir and returns the paths to remove.
func filteredWorkspaceTargets(opts CleanOptions, wsDir string, olderThan time.Duration) ([]string, error) {
	var statusWorkspaces map[string]bool
	if opts.Status != "" {
		var err error
		statusWorkspaces, err = getWorkspacesWithStatus(opts.Status)
		if err != nil {
			return nil, NewCLIError(CodeInternalError, fmt.Sprintf("failed to get workspaces by status: %s", err), "Check .agents/state.db is readable").WithCause(err)
		}
	}

	workspaces, err := workspace.ListWorkspacesSortedByTime(wsDir)
	if err != nil {
		return nil, NewCLIError(CodeInternalError, fmt.Sprintf("failed to list workspaces: %s", err), fmt.Sprintf("Check %s directory permissions", wsDir)).WithCause(err)
	}

	cutoffTime := time.Now().Add(-olderThan)
	var candidates []workspace.WorkspaceInfo
	for _, ws := range workspaces {
		if opts.OlderThan != "" && time.Unix(0, ws.ModTime).After(cutoffTime) {
			continue
		}
		if opts.Status != "" && len(statusWorkspaces) > 0 && !statusWorkspaces[ws.Name] {
			continue
		}
		candidates = append(candidates, ws)
	}

	// Apply keep-last (only if --all is also specified)
	if opts.KeepLast >= 0 && opts.All {
		allWorkspaces, _ := workspace.ListWorkspacesSortedByTime(wsDir)
		keepSet := newestWorkspaceNames(allWorkspaces, opts.KeepLast)
		var filtered []workspace.WorkspaceInfo
		for _, ws := range candidates {
			if !keepSet[ws.Name] {
				filtered = append(filtered, ws)
			}
		}
		candidates = filtered
	}

	targets := make([]string, 0, len(candidates))
	for _, ws := range candidates {
		targets = append(targets, ws.Path)
	}
	return targets, nil
}

// newestWorkspaceNames returns the names of the keep most recent workspaces
// (all of them when there are fewer). workspaces must be sorted oldest first.
func newestWorkspaceNames(workspaces []workspace.WorkspaceInfo, keep int) map[string]bool {
	keepSet := make(map[string]bool)
	start := 0
	if len(workspaces) > keep {
		start = len(workspaces) - keep
	}
	for _, ws := range workspaces[start:] {
		keepSet[ws.Name] = true
	}
	return keepSet
}

// existingPaths filters targets down to the paths that exist on disk.
func existingPaths(targets []string) []string {
	var existing []string
	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			existing = append(existing, target)
		}
	}
	return existing
}

// confirmClean asks the operator to confirm the removal. Without a TTY it
// refuses (and tells the user to pass --force).
func confirmClean(opts CleanOptions, count int, totalSize int64) bool {
	if !isTTY() {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Stdin is not a TTY. Use --force to proceed with cleanup.\n")
			fmt.Fprintf(os.Stderr, "Would clean %d item(s), total size: %s\n", count, formatSize(totalSize))
		}
		return false
	}

	fmt.Fprintf(os.Stderr, "About to remove %d item(s), total size: %s\n", count, formatSize(totalSize))
	fmt.Fprintf(os.Stderr, "Continue? [y/N] ")

	var response string
	_, _ = fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Aborted\n")
		}
		return false
	}
	return true
}

func printCleanDryRun(targets []string, totalSize int64) {
	fmt.Fprintf(os.Stderr, "(dry-run) The following would be removed:\n")
	fmt.Fprintf(os.Stderr, "  Total: %d item(s), %s\n", len(targets), formatSize(totalSize))
	for _, target := range targets {
		size, _ := calculateDirectorySize(target)
		fmt.Fprintf(os.Stderr, "  Would remove %s (%s)\n", target, formatSize(size))
	}
}

// removeCleanTargets deletes targets in batches of 100, printing progress for
// large cleanups, and returns how many were removed and how many failed.
func removeCleanTargets(targets []string, quiet bool) (cleaned, failed int) {
	const batchSize = 100
	showProgress := len(targets) > 10 && !quiet

	for i := 0; i < len(targets); i += batchSize {
		end := min(i+batchSize, len(targets))
		for _, target := range targets[i:end] {
			if err := removeAllWritable(target); err != nil {
				failed++
				if !quiet {
					fmt.Fprintf(os.Stderr, "  Failed to remove %s: %s\n", target, err)
				}
				continue
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "  Removed %s\n", target)
			}
			cleaned++
		}

		if showProgress && end < len(targets) {
			fmt.Fprintf(os.Stderr, "  Progress: %d/%d items processed\n", end, len(targets))
		}
	}
	return cleaned, failed
}

// removeAllWritable makes every directory under target writable (readonly
// mounts block removal) and then removes it.
func removeAllWritable(target string) error {
	_ = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			_ = os.Chmod(path, 0755)
		}
		return nil
	})
	return os.RemoveAll(target)
}
