package main

import (
	"fmt"
	"os"

	"github.com/recinq/wave/cmd/wave/commands"
	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
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
	rootCmd.AddCommand(commands.NewReapCmd())
	rootCmd.AddCommand(commands.NewArtifactsCmd())
	rootCmd.AddCommand(commands.NewMigrateCmd())
	rootCmd.AddCommand(commands.NewDoctorCmd())
	rootCmd.AddCommand(commands.NewSuggestCmd())
	rootCmd.AddCommand(commands.NewSkillsCmd())
	rootCmd.AddCommand(commands.NewPostmortemCmd())
	rootCmd.AddCommand(commands.NewAgentCmd())
	rootCmd.AddCommand(commands.NewBenchCmd())
	rootCmd.AddCommand(commands.NewForkCmd())
	rootCmd.AddCommand(commands.NewRewindCmd())
	rootCmd.AddCommand(commands.NewRetroCmd())
	rootCmd.AddCommand(commands.NewDecisionsCmd())
	rootCmd.AddCommand(commands.NewPipelineCmd())
	rootCmd.AddCommand(commands.NewPersonaCmd())
	rootCmd.AddCommand(commands.NewCleanupCmd())
	rootCmd.AddCommand(commands.NewMergeCmd())
	rootCmd.AddCommand(commands.NewProposalsCmd())
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
