package main

import (
	"fmt"
	"os"
	"strings"

	"context"

	"github.com/recinq/wave/cmd/wave/commands"
	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/farewell"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/suggest"
	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "wave",
	Short: "Wave multi-agent orchestrator",
	Long: `
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator

  Wave coordinates multiple AI personas through structured pipelines,
  enforcing permissions, contracts, and workspace isolation at every step.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		rf, err := commands.ResolveOutputConfig(cmd)
		if err != nil {
			return err
		}
		commands.StoreResolvedFlags(cmd, rf)

		// --no-color sets NO_COLOR env var for downstream code
		if rf.Output.NoColor {
			_ = os.Setenv("NO_COLOR", "1")
		}

		// TERM=dumb implies --no-color and --no-tui
		if os.Getenv("TERM") == "dumb" {
			_ = os.Setenv("NO_COLOR", "1")
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		suppress := commands.ShouldSuppressOutput(cmd)
		return farewell.WriteFarewell(os.Stdout, os.Getenv("USER"), suppress)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if shouldLaunchTUI(cmd) {
			deps := tui.LaunchDependencies{}

			// Attempt to load manifest for pipeline launching
			manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
			if manifestPath == "" {
				manifestPath = "wave.yaml"
			}
			data, err := os.ReadFile(manifestPath)
			if err == nil {
				var m manifest.Manifest
				if yamlErr := yaml.Unmarshal(data, &m); yamlErr == nil {
					deps.Manifest = &m
				}
			}

			// Attempt to open state store
			store, err := state.NewStateStore(".agents/state.db")
			if err == nil {
				deps.Store = store
				defer store.Close()
			}

			// Determine pipelines directory (default .agents/pipelines)
			deps.PipelinesDir = ".agents/pipelines"

			// Wire suggest provider (constructed here to avoid import cycle tui→doctor→onboarding→tui)
			pipelinesDir := deps.PipelinesDir
			deps.SuggestProvider = &tui.FuncSuggestDataProvider{
				Fn: func() (*tui.SuggestProposal, error) {
					report, err := doctor.RunChecks(context.Background(), doctor.Options{
						PipelinesDir: pipelinesDir,
						SkipCodebase: false,
					})
					if err != nil {
						return nil, err
					}
					proposal, err := suggest.Suggest(suggest.EngineOptions{
						Report:       report,
						PipelinesDir: pipelinesDir,
						Limit:        10,
					})
					if err != nil {
						return nil, err
					}
					// Convert suggest types to TUI types to avoid import cycle
					result := &tui.SuggestProposal{Rationale: proposal.Rationale}
					for _, p := range proposal.Pipelines {
						result.Pipelines = append(result.Pipelines, tui.SuggestProposedPipeline{
							Name:     p.Name,
							Reason:   p.Reason,
							Input:    p.Input,
							Priority: p.Priority,
							Type:     p.Type,
							Sequence: p.Sequence,
						})
					}
					return result, nil
				},
			}

			return tui.RunTUI(deps)
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.SetVersionTemplate("wave version {{.Version}}\n")

	rootCmd.PersistentFlags().StringP("manifest", "m", "wave.yaml", "Path to manifest file")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringP("output", "o", "auto", "Output format: auto, json, text, quiet")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Include real-time tool activity")
	rootCmd.PersistentFlags().Bool("no-tui", false, "Disable TUI and print help text")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format (equivalent to --output json)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output (equivalent to --output quiet)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")

	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewValidateCmd())
	rootCmd.AddCommand(commands.NewRunCmd())
	rootCmd.AddCommand(commands.NewResumeCmd())
	rootCmd.AddCommand(commands.NewDoCmd())
	rootCmd.AddCommand(commands.NewMetaCmd())
	rootCmd.AddCommand(commands.NewCleanCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewStatusCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())
	rootCmd.AddCommand(commands.NewCancelCmd())
	rootCmd.AddCommand(commands.NewArtifactsCmd())
	rootCmd.AddCommand(commands.NewMigrateCmd())
	rootCmd.AddCommand(commands.NewServeCmd())
	rootCmd.AddCommand(commands.NewChatCmd())
	rootCmd.AddCommand(commands.NewComposeCmd())
	rootCmd.AddCommand(commands.NewDoctorCmd())
	rootCmd.AddCommand(commands.NewSuggestCmd())
	rootCmd.AddCommand(commands.NewSkillsCmd())
	rootCmd.AddCommand(commands.NewPostmortemCmd())
	rootCmd.AddCommand(commands.NewAgentCmd())
	rootCmd.AddCommand(commands.NewAnalyzeCmd())
	rootCmd.AddCommand(commands.NewBenchCmd())
	rootCmd.AddCommand(commands.NewForkCmd())
	rootCmd.AddCommand(commands.NewRewindCmd())
	rootCmd.AddCommand(commands.NewRetroCmd())
	rootCmd.AddCommand(commands.NewDecisionsCmd())
	rootCmd.AddCommand(commands.NewPipelineCmd())
	rootCmd.AddCommand(commands.NewPersonaCmd())
	rootCmd.AddCommand(commands.NewCleanupCmd())
	rootCmd.AddCommand(commands.NewMergeCmd())
}

// shouldLaunchTUI determines whether to launch the Bubble Tea TUI.
func shouldLaunchTUI(cmd *cobra.Command) bool {
	noTUI, _ := cmd.Root().PersistentFlags().GetBool("no-tui")
	if noTUI {
		return false
	}

	// --json and --quiet suppress TUI
	jsonFlag, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonFlag {
		return false
	}
	quietFlag, _ := cmd.Root().PersistentFlags().GetBool("quiet")
	if quietFlag {
		return false
	}

	// Check WAVE_FORCE_TTY override
	if forceTTY := os.Getenv("WAVE_FORCE_TTY"); forceTTY != "" {
		switch strings.ToLower(forceTTY) {
		case "1", "true":
			return true
		case "0", "false":
			return false
		}
	}

	// TERM=dumb means non-ANSI terminal
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}

func main() {
	// Wrap cobra flag parse errors as CLIError so JSON mode renders them
	// with code "invalid_args" instead of "internal_error", and suppresses
	// the usage dump that cobra would otherwise print before returning.
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return commands.NewCLIError(commands.CodeInvalidArgs, err.Error(),
			fmt.Sprintf("Run 'wave %s --help' for usage.", cmd.Name()))
	})

	if err := rootCmd.Execute(); err != nil {
		// Determine output mode from root flags for error rendering
		debug, _ := rootCmd.PersistentFlags().GetBool("debug")
		jsonMode, _ := rootCmd.PersistentFlags().GetBool("json")
		outputMode, _ := rootCmd.PersistentFlags().GetString("output")

		if jsonMode || outputMode == "json" {
			commands.RenderJSONError(os.Stderr, err, debug)
		} else {
			commands.RenderTextError(os.Stderr, err, debug)
		}
		os.Exit(1)
	}
}
