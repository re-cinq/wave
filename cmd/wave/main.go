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
}

func init() {
	rootCmd.SetVersionTemplate("wave version {{.Version}}\n")

	rootCmd.PersistentFlags().StringP("manifest", "m", "wave.yaml", "Path to manifest file")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringP("output", "o", "auto", "Output format: auto, json, text, quiet")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Include real-time tool activity")

	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewValidateCmd())
	rootCmd.AddCommand(commands.NewRunCmd())
	rootCmd.AddCommand(commands.NewDoCmd())
	rootCmd.AddCommand(commands.NewMetaCmd())
	rootCmd.AddCommand(commands.NewResumeCmd())
	rootCmd.AddCommand(commands.NewCleanCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewStatusCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())
	rootCmd.AddCommand(commands.NewCancelCmd())
	rootCmd.AddCommand(commands.NewArtifactsCmd())
	rootCmd.AddCommand(commands.NewMigrateCmd())
	rootCmd.AddCommand(commands.NewServeCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
