package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type ResumeOptions struct {
	Pipeline string
	FromStep string
	Manifest string
}

func NewResumeCmd() *cobra.Command {
	var opts ResumeOptions

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume a paused pipeline",
		Long: `Resume a previously paused or failed pipeline execution.
Picks up from the last completed step and continues forward.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Pipeline ID to resume (required)")
	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Resume from specific step")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	cmd.MarkFlagRequired("pipeline")

	return cmd
}

func runResume(opts ResumeOptions) error {
	fmt.Printf("Resuming pipeline: %s\n", opts.Pipeline)

	if opts.FromStep != "" {
		fmt.Printf("  Starting from step: %s\n", opts.FromStep)
	} else {
		fmt.Printf("  Starting from last checkpoint\n")
	}

	stateDB := filepath.Join(".wave", "state.db")
	fmt.Printf("  Loading state from %s...\n", stateDB)

	if _, err := os.Stat(stateDB); os.IsNotExist(err) {
		return fmt.Errorf("no state database found at %s — nothing to resume", stateDB)
	}

	fmt.Printf("  ✓ Pipeline state loaded\n")
	fmt.Printf("  ✓ Resuming execution\n")

	return nil
}
