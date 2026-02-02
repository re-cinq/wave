package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type CleanOptions struct {
	Pipeline string
	All      bool
	Force    bool
}

func NewCleanCmd() *cobra.Command {
	var opts CleanOptions

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up project artifacts",
		Long: `Remove generated workspaces, state files, and cached artifacts.
Use --all to remove everything, or --pipeline to target a specific run.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Clean specific pipeline workspace")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Clean all workspaces and state")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip confirmation")

	return cmd
}

func runClean(opts CleanOptions) error {
	if !opts.All && opts.Pipeline == "" {
		return fmt.Errorf("specify --all or --pipeline <name>")
	}

	targets := []string{}

	if opts.All {
		targets = append(targets,
			".wave/state.db",
			".wave/traces",
			".wave/workspaces",
		)
	} else if opts.Pipeline != "" {
		targets = append(targets,
			filepath.Join(".wave", "workspaces", opts.Pipeline),
		)
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
			fmt.Printf("  ✗ Failed to remove %s: %s\n", target, err)
			continue
		}
		fmt.Printf("  ✓ Removed %s\n", target)
		cleaned++
	}

	if cleaned == 0 {
		fmt.Printf("Nothing to clean\n")
	} else {
		fmt.Printf("\n✓ Cleaned %d item(s)\n", cleaned)
	}

	return nil
}
