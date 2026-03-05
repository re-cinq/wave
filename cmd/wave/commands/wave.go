package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/platform"
	"github.com/recinq/wave/internal/tui"
	"github.com/recinq/wave/internal/tui/mission"
	"github.com/spf13/cobra"
)

// WaveOptions holds options for the `wave run wave` meta-orchestrator.
type WaveOptions struct {
	Manifest string
	Proposal string // --proposal flag: auto-select by index or name
	Mock     bool
	Output   OutputConfig
	Model    string
}

// RunMissionControl launches the fullscreen mission control TUI.
// Exported for use by rootCmd.RunE in main.go.
func RunMissionControl(cmd *cobra.Command) error {
	manifestPath, _ := cmd.Flags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}
	debugMode, _ := cmd.Flags().GetBool("debug")

	return mission.Run(mission.Options{
		ManifestPath:  manifestPath,
		Debug:         debugMode,
		Mock:          false,
		ModelOverride: "",
	})
}

// runWave is the meta-orchestrator for `wave run wave`.
// In interactive mode (no --proposal), it launches the fullscreen mission control TUI.
// With --proposal or non-interactive, it runs the legacy health+proposal+dispatch flow.
func runWave(opts WaveOptions, debugMode bool) error {
	// Interactive mode without --proposal: launch mission control TUI
	if isInteractive() && opts.Proposal == "" {
		return mission.Run(mission.Options{
			ManifestPath:  opts.Manifest,
			Debug:         debugMode,
			Mock:          opts.Mock,
			ModelOverride: opts.Model,
		})
	}

	// Non-interactive or --proposal: legacy flow
	return runWaveLegacy(opts, debugMode)
}

// runWaveLegacy is the original health+proposal+dispatch flow for non-interactive
// or --proposal modes.
func runWaveLegacy(opts WaveOptions, debugMode bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Load manifest
	_, err := os.ReadFile(opts.Manifest)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("No wave.yaml found. Run `wave init` to initialize.")
		}
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Detect platform
	profile, _ := platform.DetectFromGit()

	// Run health checks
	checker := meta.NewHealthChecker(
		meta.WithManifestPath(opts.Manifest),
		meta.WithVersion(getWaveVersion()),
		meta.WithPlatformProfile(profile),
	)
	report, err := checker.RunHealthChecks(ctx, meta.DefaultHealthCheckConfig())
	if err != nil {
		return fmt.Errorf("health checks failed: %w", err)
	}

	interactive := isInteractive()

	// Non-interactive mode: serialize HealthReport as JSON to stdout
	if !interactive && opts.Proposal == "" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Interactive mode: print the health report to stderr
	if interactive {
		fmt.Fprint(os.Stderr, tui.RenderHealthReport(report))
		fmt.Fprint(os.Stderr, "\n\n")
	}

	// Analyze codebase for auto-tuning context
	codebaseProfile := meta.AnalyzeCodebase(".")
	if codebaseProfile.Language != "unknown" && interactive {
		fmt.Fprintf(os.Stderr, "  Detected: %s project", codebaseProfile.Language)
		if codebaseProfile.Framework != "" {
			fmt.Fprintf(os.Stderr, " (%s)", codebaseProfile.Framework)
		}
		fmt.Fprintf(os.Stderr, " [%s]\n", codebaseProfile.Size)
	}

	// Auto-install missing dependencies if interactive
	if interactive {
		installable := meta.GetInstallable(report.Dependencies)
		if len(installable) > 0 {
			fmt.Fprintf(os.Stderr, "\n  %d auto-installable dependency(s) available\n", len(installable))
			installCmds := make(map[string]string)
			for name, cfg := range collectSkillsFromPipelines() {
				if cfg.Install != "" {
					installCmds[name] = cfg.Install
				}
			}

			installer := meta.NewInstaller()
			results := installer.Install(ctx, installable, installCmds)
			for _, r := range results {
				if r.Success {
					fmt.Fprintf(os.Stderr, "  ✓ Installed %s\n", r.Name)
				} else {
					fmt.Fprintf(os.Stderr, "  ✗ Failed to install %s: %s\n", r.Name, r.Message)
				}
			}
		}
	}

	// Discover pipelines
	pipelineInfos, _ := tui.DiscoverPipelines(pipelinesDir())
	pipelineNames := make([]string, len(pipelineInfos))
	for i, p := range pipelineInfos {
		pipelineNames[i] = p.Name
	}

	// Generate proposals
	engine := meta.NewProposalEngine(report, pipelineNames)
	proposals := engine.GenerateProposals()

	if len(proposals) == 0 {
		fmt.Fprintln(os.Stderr, "No runnable pipelines available")
		return nil
	}

	var selection *meta.ProposalSelection

	if opts.Proposal != "" {
		// Non-interactive proposal selection via --proposal flag
		var selected *meta.PipelineProposal
		for i := range proposals {
			if proposals[i].ID == opts.Proposal || (len(proposals) > 0 && fmt.Sprintf("%d", i+1) == opts.Proposal) {
				selected = &proposals[i]
				break
			}
			for _, pName := range proposals[i].Pipelines {
				if pName == opts.Proposal {
					selected = &proposals[i]
					break
				}
			}
			if selected != nil {
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("proposal %q not found in generated proposals", opts.Proposal)
		}
		inputs := make(map[string]string, len(selected.Pipelines))
		for _, p := range selected.Pipelines {
			inputs[p] = selected.PrefilledInput
		}
		selection = &meta.ProposalSelection{
			Proposals:      []meta.PipelineProposal{*selected},
			ModifiedInputs: inputs,
			ExecutionMode:  selected.Type,
		}
	} else {
		// Interactive proposal selection
		selection, err = tui.RunProposalSelector(proposals)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}
	}

	if selection == nil {
		return nil
	}

	// Dispatch pipelines
	proposal := selection.Proposals[0]
	if proposal.Type == meta.ProposalSequence && len(proposal.Pipelines) > 1 {
		for _, pName := range proposal.Pipelines {
			pInput := selection.ModifiedInputs[pName]
			runOpts := RunOptions{
				Pipeline: pName,
				Input:    pInput,
				Manifest: opts.Manifest,
				Mock:     opts.Mock,
				Output:   opts.Output,
				Model:    opts.Model,
			}
			if err := runRun(runOpts, debugMode); err != nil {
				return fmt.Errorf("sequence failed at pipeline %s: %w", pName, err)
			}
		}
		return nil
	}

	// Single pipeline dispatch
	selectedName := proposal.Pipelines[0]
	input := selection.ModifiedInputs[selectedName]

	runOpts := RunOptions{
		Pipeline: selectedName,
		Input:    input,
		Manifest: opts.Manifest,
		Mock:     opts.Mock,
		Output:   opts.Output,
		Model:    opts.Model,
	}
	return runRun(runOpts, debugMode)
}

// getWaveVersion returns the Wave binary version from build info.
func getWaveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// emitMetaEvent emits a meta-specific event with the given state and message.
func emitMetaEvent(emitter event.EventEmitter, state, message string) {
	if emitter == nil {
		return
	}
	emitter.Emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: "wave",
		State:      state,
		Message:    message,
	})
}
