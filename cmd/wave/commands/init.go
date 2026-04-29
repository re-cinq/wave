package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/onboarding"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// InitOptions captures the flag values for `wave init`.
type InitOptions struct {
	Force       bool
	Merge       bool
	All         bool
	Adapter     string
	Workspace   string
	OutputPath  string
	Yes         bool
	Reconfigure bool
}

// NewInitCmd constructs the `wave init` cobra command.
func NewInitCmd() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Wave project",
		Long: `Create a new Wave project structure with default configuration.
Creates a wave.yaml manifest and .agents/personas/ directory with example prompts.

By default, only release-ready pipelines are included. Use --all to include
all embedded pipelines (useful for Wave contributors and developers).

Use --merge to add default configuration to an existing wave.yaml while
preserving your custom settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite existing files without prompting")
	cmd.Flags().BoolVar(&opts.Merge, "merge", false, "Merge defaults into existing configuration")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Include all pipelines regardless of release status")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "claude", "Default adapter to use")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", ".agents/workspaces", "Workspace directory path")
	cmd.Flags().StringVar(&opts.OutputPath, "manifest-path", "wave.yaml", "Output path for wave.yaml")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Answer yes to all confirmation prompts")
	cmd.Flags().BoolVar(&opts.Reconfigure, "reconfigure", false, "Re-run onboarding with current settings as defaults")

	return cmd
}

// --- Orchestration ---

func runInit(cmd *cobra.Command, opts InitOptions) error {
	if opts.Reconfigure {
		return runReconfigure(cmd, opts)
	}

	if err := onboarding.EnsureGitRepo(cmd.ErrOrStderr()); err != nil {
		return err
	}

	absOutputPath, err := filepath.Abs(opts.OutputPath)
	if err != nil {
		absOutputPath = opts.OutputPath
	}

	existingFile, err := os.Stat(opts.OutputPath)
	fileExists := err == nil

	if fileExists {
		if opts.Force && !opts.Merge {
			if !opts.Yes {
				confirmed, err := confirmForceOverwrite(cmd, absOutputPath)
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}
				if !confirmed {
					return fmt.Errorf("aborted: force overwrite cancelled (use --merge to preserve custom settings)")
				}
			}
			if existingFile.Mode().Perm()&0200 == 0 {
				return fmt.Errorf("cannot overwrite %s: file is read-only", absOutputPath)
			}
		} else {
			return runMerge(cmd, opts, absOutputPath)
		}
	}

	return runService(cmd, opts)
}

// runService delegates the cold-start scaffold to the onboarding Service so
// the CLI driver shares one code path with the webui driver. The interactive
// path will gain an InteractiveService in issue 1.2; until then both
// `--yes` and the default-interactive case route through BaselineService.
func runService(cmd *cobra.Command, opts InitOptions) error {
	svc := onboarding.NewBaselineService(cmd.ErrOrStderr())
	sess, err := svc.StartSession(cmd.Context(), ".", onboarding.StartOptions{
		Adapter:    opts.Adapter,
		Workspace:  opts.Workspace,
		OutputPath: opts.OutputPath,
		All:        opts.All,
	})
	if err != nil {
		return fmt.Errorf("onboarding: %w", err)
	}

	if sess.Assets != nil {
		onboarding.PrintInitSuccess(cmd.OutOrStdout(), opts.OutputPath, sess.Assets)
	}
	onboarding.SuggestFirstRun(cmd.OutOrStdout(), sess.Flavour)
	return nil
}

func runMerge(cmd *cobra.Command, opts InitOptions, absOutputPath string) error {
	existingData, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read existing manifest %s: %w", absOutputPath, err)
	}

	var existingManifest map[string]interface{}
	if err := yaml.Unmarshal(existingData, &existingManifest); err != nil {
		return fmt.Errorf("failed to parse existing manifest %s: %w", absOutputPath, err)
	}

	cwd, _ := os.Getwd()
	flavour := onboarding.DetectFlavour(cwd)
	project := onboarding.FlavourToProjectMap(flavour)
	assets, err := onboarding.LoadAssets(cmd.ErrOrStderr(), onboarding.AssetOptions{All: opts.All})
	if err != nil {
		return err
	}

	defaultManifest := onboarding.BuildDefaultManifest(opts.Adapter, opts.Workspace, project, assets.PersonaConfigs)

	summary := onboarding.ComputeChangeSummary(assets, existingManifest, defaultManifest)

	if summary.AlreadyUpToDate {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  Already up to date — no changes needed.\n\n")
		return nil
	}

	onboarding.DisplayChangeSummary(cmd.ErrOrStderr(), summary)

	confirmed, err := confirmMerge(cmd, opts)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted: merge cancelled by user")
	}

	if err := onboarding.ApplyChanges(summary, opts.OutputPath); err != nil {
		return err
	}

	onboarding.PrintMergeSuccess(cmd.OutOrStdout(), opts.OutputPath)
	return nil
}

// runReconfigure clears the onboarding sentinels and re-runs the baseline
// service so the project picks up any new defaults. The legacy wizard prompt
// flow is intentionally absent until issue 1.2 lands InteractiveService.
func runReconfigure(cmd *cobra.Command, opts InitOptions) error {
	if _, err := os.Stat(opts.OutputPath); err != nil {
		return fmt.Errorf("cannot reconfigure: %s not found\nRun 'wave init' first", opts.OutputPath)
	}

	_ = onboarding.ClearOnboarding(".agents")
	_ = onboarding.ClearSentinel(".")

	return runService(cmd, opts)
}

// --- Interactive prompts (stay in cmd: they touch cobra streams) ---

func confirmForceOverwrite(cmd *cobra.Command, path string) (bool, error) {
	if cmd.InOrStdin() == nil {
		return false, nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\n  WARNING: --force will overwrite %s\n", path)
	fmt.Fprintf(cmd.ErrOrStderr(), "  This will REPLACE all custom settings including:\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Custom personas and adapter configurations\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Project metadata (name, description)\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "\n  Consider using 'wave init --merge' to preserve custom settings.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Proceed with force overwrite? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

func confirmMerge(cmd *cobra.Command, opts InitOptions) (bool, error) {
	if opts.Yes || opts.Force {
		return true, nil
	}

	if !onboarding.IsInteractive() {
		return false, fmt.Errorf("non-interactive terminal detected: use --yes or --force to proceed without confirmation")
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "  Apply these changes? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}
