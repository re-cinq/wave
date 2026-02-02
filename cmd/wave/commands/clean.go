package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
)

type CleanOptions struct {
	Pipeline string
	All      bool
	Force    bool
	KeepLast int
	DryRun   bool
}

func NewCleanCmd() *cobra.Command {
	var opts CleanOptions

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up project artifacts",
		Long: `Remove generated workspaces, state files, and cached artifacts.
Use --all to remove everything, or --pipeline to target a specific run.
Use --keep-last N to retain the N most recent workspaces.
Use --dry-run to preview what would be deleted without actually removing anything.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Clean specific pipeline workspace")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Clean all workspaces and state")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip confirmation")
	cmd.Flags().IntVar(&opts.KeepLast, "keep-last", -1, "Keep the N most recent workspaces (use with --all)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be deleted without removing anything")

	return cmd
}


func runClean(opts CleanOptions) error {
	if !opts.All && opts.Pipeline == "" {
		return fmt.Errorf("specify --all or --pipeline <name>")
	}

	targets := []string{}

	if opts.All {
		// Handle --keep-last for workspaces
		if opts.KeepLast >= 0 {
			// Get workspaces sorted by modification time using workspace package
			wsDir := ".wave/workspaces"
			workspaces, err := workspace.ListWorkspacesSortedByTime(wsDir)
			if err != nil {
				return fmt.Errorf("failed to list workspaces: %w", err)
			}

			// Determine which workspaces to remove (keep the N most recent)
			if len(workspaces) > opts.KeepLast {
				// Remove oldest workspaces, keeping the most recent opts.KeepLast
				toRemove := len(workspaces) - opts.KeepLast
				for i := 0; i < toRemove; i++ {
					targets = append(targets, workspaces[i].Path)
				}
			}
			// When using --keep-last, we only affect workspaces, not state.db or traces
		} else {
			// Default behavior: remove everything
			targets = append(targets,
				".wave/state.db",
				".wave/traces",
				".wave/workspaces",
			)
		}
	} else if opts.Pipeline != "" {
		targets = append(targets,
			filepath.Join(".wave", "workspaces", opts.Pipeline),
		)
	}

	if opts.DryRun {
		fmt.Printf("(dry-run) The following would be removed:\n")
		for _, target := range targets {
			if _, err := os.Stat(target); os.IsNotExist(err) {
				continue
			}
			fmt.Printf("  Would remove %s\n", target)
		}
		if len(targets) == 0 {
			fmt.Printf("Nothing to clean\n")
		}
		return nil
	}

	cleaned := 0
	for _, target := range targets {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			continue
		}
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
			fmt.Printf("  Failed to remove %s: %s\n", target, err)
			continue
		}
		fmt.Printf("  Removed %s\n", target)
		cleaned++
	}

	if cleaned == 0 {
		fmt.Printf("Nothing to clean\n")
	} else {
		fmt.Printf("\nCleaned %d item(s)\n", cleaned)
	}

	return nil
}
