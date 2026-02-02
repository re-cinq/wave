package main

import (
	"fmt"
	"os"

	"github.com/recinq/wave/cmd/wave/commands"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
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
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringP("manifest", "m", "wave.yaml", "Path to manifest file")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().String("log-format", "text", "Log format (text, json)")

	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewValidateCmd())
	rootCmd.AddCommand(commands.NewRunCmd())
	rootCmd.AddCommand(commands.NewDoCmd())
	rootCmd.AddCommand(commands.NewResumeCmd())
	rootCmd.AddCommand(commands.NewCleanCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewStatusCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())
	rootCmd.AddCommand(commands.NewCancelCmd())
	rootCmd.AddCommand(commands.NewArtifactsCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
