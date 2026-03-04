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
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/platform"
	"github.com/recinq/wave/internal/tui"
)

// WaveOptions holds options for the `wave run wave` meta-orchestrator.
type WaveOptions struct {
	Manifest string
	Proposal string // --proposal flag: auto-select by index or name
	Mock     bool
	Output   OutputConfig
	Model    string
}

// runWave is the meta-orchestrator for `wave run wave`.
// It runs health checks, generates proposals, and dispatches to pipeline execution.
func runWave(opts WaveOptions, debug bool) error {
	// Gate on onboarding completion
	if err := checkOnboarding(); err != nil {
		return err
	}

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
	// ignore error — just use PlatformUnknown on failure

	// Set up emitter for meta events
	emitterResult := CreateEmitter(opts.Output, "wave", "wave", nil, nil)
	defer emitterResult.Cleanup()
	emitter := emitterResult.Emitter

	// Run health checks
	emitMetaEvent(emitter, "meta.health_started", "Starting health checks")

	checker := meta.NewHealthChecker(
		meta.WithManifestPath(opts.Manifest),
		meta.WithVersion(getWaveVersion()),
		meta.WithPlatformProfile(profile),
	)
	report, err := checker.RunHealthChecks(ctx, meta.DefaultHealthCheckConfig())
	if err != nil {
		return fmt.Errorf("health checks failed: %w", err)
	}

	emitMetaEvent(emitter, "meta.health_completed", "Health checks completed")

	interactive := isInteractive()

	// Non-interactive mode: if no --proposal flag, serialize HealthReport as JSON to stdout and exit
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
			// Build install commands from pipeline skill configs
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

	emitMetaEvent(emitter, "meta.proposals_generated", fmt.Sprintf("Generated %d proposal(s)", len(proposals)))

	// Handle no proposals
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
			// Also match by pipeline name
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

	emitMetaEvent(emitter, "meta.proposal_selected", fmt.Sprintf("Selected proposal: %s", selection.Proposals[0].ID))

	// Dispatch sequence of pipelines if applicable
	if selection.Proposals[0].Type == meta.ProposalSequence && len(selection.Proposals[0].Pipelines) > 1 {
		emitMetaEvent(emitter, "meta.sequence_started", fmt.Sprintf("Starting sequence: %v", selection.Proposals[0].Pipelines))

		// Each pipeline in the sequence is dispatched individually via runRun
		for i, pName := range selection.Proposals[0].Pipelines {
			pInput := selection.ModifiedInputs[pName]
			emitMetaEvent(emitter, "meta.pipeline_dispatched", fmt.Sprintf("Dispatching pipeline %d/%d: %s", i+1, len(selection.Proposals[0].Pipelines), pName))

			runOpts := RunOptions{
				Pipeline: pName,
				Input:    pInput,
				Manifest: opts.Manifest,
				Mock:     opts.Mock,
				Output:   opts.Output,
				Model:    opts.Model,
			}
			if err := runRun(runOpts, debug); err != nil {
				return fmt.Errorf("sequence failed at pipeline %s: %w", pName, err)
			}
		}

		emitMetaEvent(emitter, "meta.sequence_completed", "Sequence completed successfully")
		return nil
	}

	// Dispatch single pipeline
	selectedName := selection.Proposals[0].Pipelines[0]
	input := selection.ModifiedInputs[selectedName]

	emitMetaEvent(emitter, "meta.pipeline_dispatched", fmt.Sprintf("Dispatching pipeline: %s", selectedName))

	runOpts := RunOptions{
		Pipeline: selectedName,
		Input:    input,
		Manifest: opts.Manifest,
		Mock:     opts.Mock,
		Output:   opts.Output,
		Model:    opts.Model,
	}
	return runRun(runOpts, debug)
}

// loadManifestForWave loads and parses the manifest for the wave orchestrator.
func loadManifestForWave(manifestPath string) (*manifest.Manifest, error) {
	return manifest.Load(manifestPath)
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
