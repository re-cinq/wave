package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/cmd/wave/commands"
	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if shouldLaunchTUI(cmd) {
			return tui.RunTUI()
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

	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewValidateCmd())
	rootCmd.AddCommand(commands.NewRunCmd())
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
}

// shouldLaunchTUI determines whether to launch the Bubble Tea TUI.
func shouldLaunchTUI(cmd *cobra.Command) bool {
	noTUI, _ := cmd.Root().PersistentFlags().GetBool("no-tui")
	if noTUI {
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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
