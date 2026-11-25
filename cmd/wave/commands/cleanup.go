package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
)

// CleanupOptions holds options for the cleanup command.
type CleanupOptions struct {
	DryRun bool
	Force  bool
	// Manifest is the manifest whose runtime.workspace_root is honoured;
	// empty means wave.yaml.
	Manifest string
}

// NewCleanupCmd creates the cleanup command.
func NewCleanupCmd() *cobra.Command {
	var opts CleanupOptions

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove orphaned worktrees from the workspace root",
		Long: `Remove orphaned worktrees that have no corresponding running pipeline.

Uses 'git worktree list' to find all worktrees under the workspace root
(runtime.workspace_root, or the per-project default under the user cache
dir; worktrees left in .agents/workspaces/ by older versions are included),
cross-references with the state store to identify active runs, and removes
worktrees not associated with any running pipeline.

Use --dry-run to preview what would be removed without deleting anything.
Use --force to skip the confirmation prompt.`,
		Example: `  wave cleanup              # Remove orphaned worktrees (with confirmation)
  wave cleanup --dry-run    # Preview what would be removed
  wave cleanup --force      # Remove without confirmation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Manifest, _ = cmd.Root().PersistentFlags().GetString("manifest")
			return runCleanup(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be removed without deleting anything")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip confirmation prompt")

	return cmd
}

// worktreeEntry represents a parsed git worktree.
type worktreeEntry struct {
	Path   string
	Branch string
}

// listGitWorktrees shells out to `git worktree list --porcelain` and parses the output.
func listGitWorktrees() ([]worktreeEntry, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	var entries []worktreeEntry
	var current worktreeEntry

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				entries = append(entries, current)
			}
			current = worktreeEntry{}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		}
		if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}
	// Flush last entry if no trailing blank line.
	if current.Path != "" {
		entries = append(entries, current)
	}

	return entries, nil
}

func runCleanup(opts CleanupOptions) error {
	project := manifestProjectDir(opts.Manifest)
	if err := workspace.CheckRoot(manifestWorkspaceRoot(opts.Manifest), project); err != nil {
		return unsafeRootError(err)
	}

	// Workspace roots that exist and are safe to clean below.
	var roots []string
	sweep, _ := workspaceRootsToSweep(opts.Manifest)
	for _, dir := range sweep {
		if _, err := os.Stat(dir); err != nil || workspace.CheckRoot(dir, project) != nil {
			continue
		}
		roots = append(roots, dir)
	}
	if len(roots) == 0 {
		fmt.Fprintf(os.Stderr, "No workspaces directory found\n")
		return nil
	}

	// List all git worktrees.
	worktrees, err := listGitWorktrees()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to list worktrees: %s", err), "Ensure git is available").WithCause(err)
	}

	// Keep only step worktrees Wave created: <root>/<run dir>/__wt_*,
	// never the main worktree (listed first by git) or the project.
	var waveWorktrees []worktreeEntry
	for i, wt := range worktrees {
		if i == 0 {
			continue // main worktree
		}
		if ownedByRoots(roots, wt.Path, project) {
			waveWorktrees = append(waveWorktrees, wt)
		}
	}

	if len(waveWorktrees) == 0 {
		fmt.Fprintf(os.Stderr, "No worktrees found under %s\n", strings.Join(roots, ", "))
		return nil
	}

	// Build set of branches associated with active (running/pending) runs.
	activeBranches := make(map[string]bool)
	stateDB := ".agents/state.db"
	if _, statErr := os.Stat(stateDB); statErr == nil {
		store, storeErr := state.NewStateStore(stateDB)
		if storeErr == nil {
			defer store.Close()
			runs, runErr := store.GetRunningRuns()
			if runErr == nil {
				for _, run := range runs {
					if run.BranchName != "" {
						activeBranches[run.BranchName] = true
					}
				}
			}
		}
	}

	// Identify orphaned worktrees (not associated with active runs).
	var orphaned []worktreeEntry
	for _, wt := range waveWorktrees {
		if wt.Branch != "" && activeBranches[wt.Branch] {
			continue // active run — keep
		}
		orphaned = append(orphaned, wt)
	}

	if len(orphaned) == 0 {
		fmt.Fprintf(os.Stderr, "No orphaned worktrees found (%d active)\n", len(waveWorktrees))
		return nil
	}

	// Calculate total size of orphaned worktrees.
	var totalSize int64
	for _, wt := range orphaned {
		size, _ := calculateDirectorySize(wt.Path)
		totalSize += size
	}

	// Dry-run: list what would be removed.
	if opts.DryRun {
		fmt.Fprintf(os.Stderr, "(dry-run) Would remove %d orphaned worktree(s), freeing %s:\n", len(orphaned), formatSize(totalSize))
		for _, wt := range orphaned {
			size, _ := calculateDirectorySize(wt.Path)
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			fmt.Fprintf(os.Stderr, "  %s [%s] (%s)\n", wt.Path, branch, formatSize(size))
		}
		return nil
	}

	// Confirmation prompt unless --force.
	if !opts.Force {
		if !isTTY() {
			fmt.Fprintf(os.Stderr, "Stdin is not a TTY. Use --force to proceed.\n")
			fmt.Fprintf(os.Stderr, "Would remove %d orphaned worktree(s), freeing %s\n", len(orphaned), formatSize(totalSize))
			return nil
		}
		fmt.Fprintf(os.Stderr, "About to remove %d orphaned worktree(s), freeing %s\n", len(orphaned), formatSize(totalSize))
		fmt.Fprintf(os.Stderr, "Continue? [y/N] ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Fprintf(os.Stderr, "Aborted\n")
			return nil
		}
	}

	// Remove orphaned worktrees.
	removed := 0
	var freedBytes int64
	for _, wt := range orphaned {
		size, _ := calculateDirectorySize(wt.Path)

		// Try git worktree remove first (cleans up .git/worktrees entry).
		cmd := exec.Command("git", "worktree", "remove", "--force", wt.Path)
		if err := cmd.Run(); err != nil {
			// Fall back to direct removal if git worktree remove fails,
			// re-checking ownership right before deleting anything.
			if !ownedByRoots(roots, wt.Path, project) {
				fmt.Fprintf(os.Stderr, "  Refusing to remove %s: not a Wave worktree\n", wt.Path)
				continue
			}
			// Make dirs writable first (readonly mounts block removal).
			_ = filepath.Walk(wt.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					_ = os.Chmod(path, 0755)
				}
				return nil
			})
			if rmErr := os.RemoveAll(wt.Path); rmErr != nil {
				fmt.Fprintf(os.Stderr, "  Failed to remove %s: %s\n", wt.Path, rmErr)
				continue
			}
		}
		removed++
		freedBytes += size
		fmt.Fprintf(os.Stderr, "  Removed %s\n", wt.Path)
	}

	// Prune stale worktree references after removal.
	_ = exec.Command("git", "worktree", "prune").Run()

	fmt.Fprintf(os.Stderr, "\nRemoved %d orphaned worktree(s), freed %s\n", removed, formatSize(freedBytes))
	return nil
}

// ownedByRoots reports whether path is a step worktree Wave created under
// one of roots (see workspace.OwnedWorktree).
func ownedByRoots(roots []string, path, project string) bool {
	for _, root := range roots {
		if workspace.OwnedWorktree(root, path, project) {
			return true
		}
	}
	return false
}
