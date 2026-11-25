package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// InitOptions captures the flag values for `wave init`.
type InitOptions struct {
	Force        bool
	Merge        bool
	All          bool
	Adapter      string
	Model        string
	Workspace    string
	OutputPath   string
	Yes          bool
	Reconfigure  bool
	TestCommand  string
	LintCommand  string
	BuildCommand string

	// displayPath is the manifest path as the user gave it, for messages;
	// OutputPath is relative to the project root init works in.
	displayPath string
	// explicitDir is true when the manifest path names another directory
	// (e.g. -m cfg/wave.yaml): the user chose the project root, so init
	// does not warn that it is below the git repository root.
	explicitDir bool
}

// NewInitCmd constructs the `wave init` cobra command.
func NewInitCmd() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Wave project",
		Long: `Create a new Wave project: a wave.yaml manifest, the .agents/ tree with
personas, pipelines, contracts and prompts, an AGENTS.md with your project
commands (plus a CLAUDE.md that imports it for the claude adapter), and
.gitignore entries for Wave's runtime state.

In an interactive terminal init asks for the adapter (claude or codex, the
first one found on PATH is the default), the model, and any test/lint/build
command it could not detect. With --yes, or when stdin is not a terminal,
every question takes its default. Flags always win over questions.

The chosen model is pinned for every complexity tier (cheapest, balanced,
strongest). Defaults: claude -> sonnet, codex -> gpt-6.2-sol.

By default, only release-ready pipelines are included. Use --all to include
all embedded pipelines (useful for Wave contributors and developers).

When wave.yaml already exists, init merges missing defaults into it and keeps
your settings. Use --reconfigure to change adapter, model or project commands
of an existing project (a backup is written to <manifest>.bak), or
--force to regenerate everything from scratch.`,
		Example: `  wave init
  wave init --yes --adapter codex
  wave init --adapter claude --model sonnet --test-command "make test"
  wave init --reconfigure --adapter codex --model gpt-6.2-sol`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runInit(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Regenerate wave.yaml and default files from scratch (asks for confirmation unless --yes)")
	cmd.Flags().BoolVar(&opts.Merge, "merge", false, "Merge defaults into existing configuration")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Include all pipelines regardless of release status")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "", "Adapter to use: claude or codex (default: first one found on PATH, else claude)")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Model for the adapter, pinned for every tier (default: sonnet for claude, gpt-6.2-sol for codex)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Workspace root written as runtime.workspace_root (default: unset, a per-project directory under the user cache dir, outside the checkout)")
	cmd.Flags().StringVar(&opts.OutputPath, "manifest-path", "wave.yaml", "Output path for wave.yaml (the global --manifest/-m is honoured too)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Answer every question with its default and confirm all prompts")
	cmd.Flags().BoolVar(&opts.Reconfigure, "reconfigure", false, "Change adapter, model or project commands of an existing project, keeping customisations")
	cmd.Flags().StringVar(&opts.TestCommand, "test-command", "", "Set project.test_command")
	cmd.Flags().StringVar(&opts.LintCommand, "lint-command", "", "Set project.lint_command")
	cmd.Flags().StringVar(&opts.BuildCommand, "build-command", "", "Set project.build_command")

	return cmd
}

// --- Orchestration ---

func runInit(cmd *cobra.Command, opts InitOptions) error {
	opts.OutputPath = resolveInitManifestPath(cmd, opts.OutputPath)
	opts.displayPath = opts.OutputPath

	// Wave resolves every path in a manifest (personas, pipelines,
	// workspaces) relative to the manifest's directory, so a manifest in
	// another directory makes that directory the project root.
	if dir := filepath.Dir(opts.OutputPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
		prev, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("failed to enter %s: %w", dir, err)
		}
		defer func() { _ = os.Chdir(prev) }()
		opts.OutputPath = filepath.Base(opts.OutputPath)
		opts.explicitDir = true
	}

	if opts.Adapter != "" {
		if err := onboarding.ValidateInitAdapter(opts.Adapter); err != nil {
			return NewCLIError(CodeInvalidArgs, err.Error(), "Use --adapter claude or --adapter codex")
		}
	}

	if opts.Reconfigure {
		return runReconfigure(cmd, opts)
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
					return err
				}
				if !confirmed {
					return fmt.Errorf("aborted: force overwrite cancelled (use --merge or --reconfigure to preserve custom settings)")
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

// resolveInitManifestPath honours the global --manifest/-m flag when
// --manifest-path was not given explicitly.
func resolveInitManifestPath(cmd *cobra.Command, outputPath string) string {
	if f := cmd.Flags().Lookup("manifest-path"); f != nil && f.Changed {
		return outputPath
	}
	if f := cmd.Flags().Lookup("manifest"); f != nil && f.Changed && f.Value.String() != "" {
		return f.Value.String()
	}
	return outputPath
}

// runService runs the cold-start scaffold: it settles git, adapter, model and
// project commands (asking in an interactive terminal), then delegates to the
// onboarding Service so the CLI shares one code path with other drivers.
func runService(cmd *cobra.Command, opts InitOptions) error {
	stderr := cmd.ErrOrStderr()
	interactive := !opts.Yes && onboarding.IsInteractive()
	p := newPrompter(cmd)

	if !onboarding.InsideGitRepo() && interactive {
		ok, err := p.confirm("  No git repository here. Wave runs every step in a git worktree.\n  Initialize a git repository in this directory? [Y/n]: ", true)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("aborted: wave needs a git repository (run 'git init' first)")
		}
	}
	if err := onboarding.EnsureGitRepo(stderr); err != nil {
		return err
	}
	if !opts.explicitDir {
		warnIfSubdirectory(stderr)
	}

	adapter, model, err := chooseAdapterAndModel(p, interactive, opts.Adapter, opts.Model, "", "")
	if err != nil {
		return err
	}
	warnIfAdapterMissing(stderr, adapter)

	commands := onboarding.ProjectCommands{Test: opts.TestCommand, Lint: opts.LintCommand, Build: opts.BuildCommand}
	if interactive {
		cwd, _ := os.Getwd()
		detected := onboarding.ApplyProjectCommands(onboarding.FlavourToProjectMap(onboarding.DetectFlavour(cwd)), commands)
		if commands, err = askMissingCommands(p, detected, commands); err != nil {
			return err
		}
	}

	svc := onboarding.NewBaselineService(stderr)
	sess, err := svc.StartSession(cmd.Context(), ".", onboarding.StartOptions{
		Adapter:    adapter,
		Model:      model,
		Commands:   commands,
		Workspace:  opts.Workspace,
		OutputPath: opts.OutputPath,
		All:        opts.All,
	})
	if err != nil {
		return fmt.Errorf("onboarding: %w", err)
	}

	if sess.Result != nil {
		onboarding.PrintInitSuccess(cmd.OutOrStdout(), opts.displayPath, opts.Workspace, sess.Result)
	}
	onboarding.SuggestFirstRun(cmd.OutOrStdout(), sess.Flavour)
	return nil
}

// chooseAdapterAndModel resolves the adapter and model from flags, the
// current configuration and (interactively) the user.
func chooseAdapterAndModel(p *prompter, interactive bool, flagAdapter, flagModel, curAdapter, curModel string) (string, string, error) {
	adapter := flagAdapter
	if adapter == "" {
		if onboarding.ValidateInitAdapter(curAdapter) == nil {
			adapter = curAdapter
		} else {
			adapter = onboarding.DefaultInitAdapter(nil)
		}
		if interactive {
			installed := onboarding.DetectInstalledAdapters(nil)
			note := "none of them found on PATH"
			if len(installed) > 0 {
				note = "found on PATH: " + strings.Join(installed, ", ")
			}
			answer, err := p.ask(fmt.Sprintf("  Adapter [claude/codex] (%s) [%s]: ", note, adapter))
			if err != nil {
				return "", "", err
			}
			if answer != "" {
				if err := onboarding.ValidateInitAdapter(answer); err != nil {
					return "", "", err
				}
				adapter = answer
			}
		}
	}

	model := flagModel
	if model == "" {
		model = onboarding.DefaultModelFor(adapter)
		if adapter == curAdapter && curModel != "" {
			model = curModel
		}
		if interactive {
			answer, err := p.ask(fmt.Sprintf("  Model for %s (used for every tier) [%s]: ", adapter, model))
			if err != nil {
				return "", "", err
			}
			if answer != "" {
				model = answer
			}
		}
	}
	return adapter, model, nil
}

// askMissingCommands asks for every project command that is neither detected
// nor given by flag. Empty answers leave the command unset.
func askMissingCommands(p *prompter, project map[string]interface{}, c onboarding.ProjectCommands) (onboarding.ProjectCommands, error) {
	missing := onboarding.MissingProjectCommands(project)
	for _, k := range missing {
		answer, err := p.ask(fmt.Sprintf("  %s command (not detected; leave empty to skip): ", strings.ToUpper(k[:1])+k[1:]))
		if err != nil {
			return c, err
		}
		switch k {
		case "test":
			c.Test = answer
		case "lint":
			c.Lint = answer
		case "build":
			c.Build = answer
		}
	}
	return c, nil
}

// warnIfSubdirectory warns when init runs below the root of a git work tree.
func warnIfSubdirectory(w io.Writer) {
	top := onboarding.GitTopLevel()
	if top == "" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	if real, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = real
	}
	if real, err := filepath.EvalSymlinks(top); err == nil {
		top = real
	}
	if cwd == top {
		return
	}
	fmt.Fprintf(w, "  Warning: %s is a subdirectory of the git repository at %s.\n", cwd, top)
	fmt.Fprintf(w, "  Wave is normally initialised at the repository root; project detection only looks at this directory.\n")
}

// warnIfAdapterMissing tells the user when the chosen adapter binary is not
// installed, so the first `wave run` does not fail mid-step. It only points
// at other adapters that are actually installed.
func warnIfAdapterMissing(w io.Writer, adapter string) {
	if _, err := lookPath(adapter); err == nil {
		return
	}
	var alts []string
	for _, a := range onboarding.SupportedInitAdapters() {
		if a == adapter {
			continue
		}
		if _, err := lookPath(a); err == nil {
			alts = append(alts, "--adapter "+a)
		}
	}
	if len(alts) == 0 {
		fmt.Fprintf(w, "  Warning: adapter binary %q not found on PATH. Install it before 'wave run'.\n", adapter)
		return
	}
	fmt.Fprintf(w, "  Warning: adapter binary %q not found on PATH. Install it before 'wave run', or re-run init with %s.\n",
		adapter, strings.Join(alts, " or "))
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

	// Merge only adds what is missing: keep the adapter the project already
	// uses so re-running init never adds adapters the user removed.
	adapter := opts.Adapter
	if adapter == "" {
		adapter = onboarding.AdapterOfManifest(existingManifest)
	}
	if adapter == "" {
		adapter = onboarding.DefaultInitAdapter(nil)
	}

	cwd, _ := os.Getwd()
	flavour := onboarding.DetectFlavour(cwd)
	project := onboarding.FlavourToProjectMap(flavour)
	project = onboarding.ApplyProjectCommands(project, onboarding.ProjectCommands{Test: opts.TestCommand, Lint: opts.LintCommand, Build: opts.BuildCommand})
	assets, err := onboarding.LoadAssets(cmd.ErrOrStderr(), onboarding.AssetOptions{All: opts.All})
	if err != nil {
		return err
	}
	forgeInfo, _ := forge.DetectFromGitRemotes()
	onboarding.ScopeAssetsToForge(assets, forgeInfo.Type, opts.All)

	defaultManifest := onboarding.BuildManifest(onboarding.ManifestOptions{
		Adapter:        adapter,
		Model:          opts.Model,
		Workspace:      opts.Workspace,
		Project:        project,
		PersonaConfigs: assets.PersonaConfigs,
	})

	summary := onboarding.ComputeChangeSummary(assets, existingManifest, defaultManifest)

	if summary.AlreadyUpToDate {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  Already up to date: no default files or settings are missing.\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "  To change adapter, model or project commands run 'wave init --reconfigure';\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "  to regenerate everything from scratch run 'wave init --force'.\n\n")
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

	onboarding.PrintMergeSuccess(cmd.OutOrStdout(), opts.displayPath)
	return nil
}

// runReconfigure changes adapter, model or project commands of an existing
// project. Existing settings are the defaults for every question; all other
// customisations (metadata, personas, edited prompt files) are preserved, and
// the previous manifest is saved to <manifest>.bak.
func runReconfigure(cmd *cobra.Command, opts InitOptions) error {
	data, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("cannot reconfigure: %s not found\nRun 'wave init' first", opts.OutputPath)
	}
	var existing map[string]interface{}
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("cannot reconfigure: failed to parse %s: %w", opts.OutputPath, err)
	}
	if existing == nil {
		existing = map[string]interface{}{}
	}

	interactive := !opts.Yes && onboarding.IsInteractive()
	p := newPrompter(cmd)

	curAdapter := onboarding.AdapterOfManifest(existing)
	curModel := onboarding.ModelOfManifest(existing, curAdapter)
	adapter, model, err := chooseAdapterAndModel(p, interactive, opts.Adapter, opts.Model, curAdapter, curModel)
	if err != nil {
		return err
	}
	warnIfAdapterMissing(cmd.ErrOrStderr(), adapter)

	cwd, _ := os.Getwd()
	project := onboarding.FlavourToProjectMap(onboarding.DetectFlavour(cwd))
	commands := onboarding.ProjectCommands{Test: opts.TestCommand, Lint: opts.LintCommand, Build: opts.BuildCommand}
	if interactive {
		current, _ := existing["project"].(map[string]interface{})
		merged := onboarding.ApplyProjectCommands(onboarding.MergeManifests(project, current), commands)
		if commands, err = askMissingCommands(p, merged, commands); err != nil {
			return err
		}
	}

	assets, err := onboarding.LoadAssets(cmd.ErrOrStderr(), onboarding.AssetOptions{All: opts.All})
	if err != nil {
		return err
	}
	forgeInfo, _ := forge.DetectFromGitRemotes()
	onboarding.ScopeAssetsToForge(assets, forgeInfo.Type, opts.All)
	defaults := onboarding.BuildManifest(onboarding.ManifestOptions{
		Adapter:        adapter,
		Model:          model,
		Workspace:      opts.Workspace,
		Project:        project,
		PersonaConfigs: assets.PersonaConfigs,
	})

	summary := onboarding.ComputeChangeSummary(assets, existing, defaults)
	summary.MergedManifest = onboarding.ApplyReconfigure(summary.MergedManifest, curAdapter, adapter, model, commands)

	backup := opts.OutputPath + ".bak"
	if err := os.WriteFile(backup, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup %s: %w", backup, err)
	}
	if err := onboarding.ApplyChanges(summary, opts.OutputPath); err != nil {
		return err
	}
	_ = onboarding.ClearOnboarding(".agents")
	if err := onboarding.MarkDoneAt("."); err != nil {
		return err
	}

	onboarding.PrintReconfigureSuccess(cmd.OutOrStdout(), opts.displayPath, opts.displayPath+".bak", adapter, model)
	return nil
}

// --- Interactive prompts (stay in cmd: they touch cobra streams) ---

// prompter reads answers from the command's stdin through one buffered
// reader, so consecutive questions do not lose buffered input.
type prompter struct {
	cmd    *cobra.Command
	reader *bufio.Reader
}

func newPrompter(cmd *cobra.Command) *prompter {
	return &prompter{cmd: cmd}
}

func (p *prompter) ask(question string) (string, error) {
	if p.reader == nil {
		p.reader = bufio.NewReader(p.cmd.InOrStdin())
	}
	fmt.Fprint(p.cmd.ErrOrStderr(), question)
	line, err := p.reader.ReadString('\n')
	switch {
	case err == nil:
	case errors.Is(err, io.EOF):
		// No more input: a partial last line is the answer, otherwise the
		// default applies.
	default:
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *prompter) confirm(question string, def bool) (bool, error) {
	answer, err := p.ask(question)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(answer) {
	case "":
		return def, nil
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func confirmForceOverwrite(cmd *cobra.Command, path string) (bool, error) {
	if cmd.InOrStdin() == nil {
		return false, nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\n  WARNING: --force will overwrite %s\n", path)
	fmt.Fprintf(cmd.ErrOrStderr(), "  This will REPLACE all custom settings including:\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Custom personas and adapter configurations\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Project metadata (name, description)\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "\n  Consider 'wave init --merge' or 'wave init --reconfigure' to preserve custom settings.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Proceed with force overwrite? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil && (response == "" || !errors.Is(err, io.EOF)) {
		if errors.Is(err, io.EOF) {
			return false, fmt.Errorf("--force needs confirmation but stdin has no input: re-run with --force --yes to overwrite %s, or use --reconfigure to keep your settings", path)
		}
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

func confirmMerge(cmd *cobra.Command, opts InitOptions) (bool, error) {
	if opts.Yes || opts.Force {
		return true, nil
	}

	if !onboarding.IsInteractive() {
		return false, fmt.Errorf("non-interactive terminal: re-run with --yes to apply the changes above (your existing settings and files are kept)")
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
