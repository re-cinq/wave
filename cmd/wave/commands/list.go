package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/listing"
	"github.com/spf13/cobra"
)

// Re-exported listing types for backward compatibility with tests inside this
// package and any in-tree consumers. New code should import internal/listing
// directly.
type (
	PipelineInfo  = listing.PipelineInfo
	PersonaInfo   = listing.PersonaInfo
	AdapterInfo   = listing.AdapterInfo
	RunInfo       = listing.RunInfo
	ContractInfo  = listing.ContractInfo
	ContractUsage = listing.ContractUsage
	SkillInfo     = listing.SkillInfo
	ListOutput    = listing.Output
)

// ListOptions holds flags shared across `wave list` invocations.
type ListOptions struct {
	Manifest string
	Format   string
}

// ListRunsOptions holds options for the `wave list runs` subcommand.
type ListRunsOptions struct {
	Limit    int
	Pipeline string
	Status   string
	Format   string
}

// Flags specific to `list runs` accumulated on the parent command for
// backward compatibility — they are surfaced as `--limit`, `--run-pipeline`,
// `--run-status` and only consulted when the filter is "runs".
var (
	listRunsLimit    int
	listRunsPipeline string
	listRunsStatus   string
)

// printLogo is intentionally a no-op. The ASCII banner added noise without
// value and broke machine readability (clig.dev). Kept as a function to avoid
// churn at call sites.
func printLogo() {}

// formatDuration is a thin alias preserved so existing in-package tests keep
// compiling. New code should call listing.FormatDuration.
func formatDuration(d time.Duration) string { return listing.FormatDuration(d) }

// NewListCmd returns the root `wave list` cobra command.
func NewListCmd() *cobra.Command {
	var opts ListOptions

	cmd := &cobra.Command{
		Use:   "list [adapters|runs|pipelines|personas|contracts|skills|compositions]",
		Short: "List Wave configuration and resources",
		Long: `List Wave configuration, resources, and execution history.

Arguments:
  adapters       List configured LLM adapters with availability status
  runs           List recent pipeline executions
  pipelines      List available pipelines with step flows
  personas       List configured personas with permissions
  contracts      List contract schemas and their usage in pipelines
  skills         List declared skills with installation status
  compositions   List composition pipelines with sub-pipeline and step type details

With no arguments, lists all categories.

For 'list runs', additional flags are available:
  --limit N           Maximum number of runs to show (default 10)
  --run-pipeline P    Filter to specific pipeline
  --run-status S      Filter by status (running, completed, failed, cancelled)`,
		Example: `  wave list pipelines
  wave list runs
  wave list runs --limit 5 --run-status failed
  wave list personas --format json
  wave list contracts
  wave list skills`,
		ValidArgs: []string{"adapters", "runs", "pipelines", "personas", "contracts", "skills", "compositions"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			opts.Format = ResolveFormat(cmd, opts.Format)
			if err := runList(opts, filter); err != nil {
				var cliErr *CLIError
				if !errors.As(err, &cliErr) {
					return NewCLIError(CodeInternalError, err.Error(), "").WithCause(err)
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")

	cmd.Flags().IntVar(&listRunsLimit, "limit", 10, "Maximum number of runs to show (for 'list runs')")
	cmd.Flags().StringVar(&listRunsPipeline, "run-pipeline", "", "Filter to specific pipeline (for 'list runs')")
	cmd.Flags().StringVar(&listRunsStatus, "run-status", "", "Filter by status (for 'list runs')")

	return cmd
}

func runList(opts ListOptions, filter string) error {
	showAll := filter == ""
	showAdapters := showAll || filter == "adapters"
	showRuns := showAll || filter == "runs"
	showPipelines := showAll || filter == "pipelines"
	showPersonas := showAll || filter == "personas"
	showContracts := showAll || filter == "contracts"
	showSkills := showAll || filter == "skills"

	if filter == "runs" {
		return runListRuns(ListRunsOptions{
			Limit:    listRunsLimit,
			Pipeline: listRunsPipeline,
			Status:   listRunsStatus,
			Format:   opts.Format,
		})
	}

	if filter == "compositions" {
		return runListCompositions(listing.DefaultPipelineDir, opts.Format)
	}

	if opts.Format == "json" {
		return emitJSONList(opts, showAdapters, showRuns, showPipelines, showPersonas, showContracts, showSkills)
	}

	printLogo()

	manifest, manifestErr := listing.LoadManifest(opts.Manifest)
	if manifestErr != nil && (showPersonas || showAdapters) {
		fmt.Printf("(manifest not found: %s)\n", opts.Manifest)
		return nil
	}

	if showAdapters {
		renderAdaptersTable(listing.ListAdapters(manifest.Adapters))
		if showAll {
			fmt.Println()
		}
	}

	if showRuns {
		runs, err := listing.ListRuns(listing.RunsOptions{
			Limit:    listRunsLimit,
			Pipeline: listRunsPipeline,
			Status:   listRunsStatus,
		})
		if err == nil {
			renderRunsTable(runs)
			if showAll {
				fmt.Println()
			}
		}
	}

	if showPipelines {
		pipelines, err := listing.ListPipelines()
		if err != nil {
			return err
		}
		renderPipelinesTable(pipelines)
		if showAll {
			fmt.Println()
		}
	}

	if showPersonas {
		renderPersonasTable(listing.ListPersonas(manifest.Personas))
		if showAll {
			fmt.Println()
		}
	}

	if showContracts {
		contracts, err := listing.ListContracts()
		if err == nil {
			renderContractsTable(contracts)
		}
		if showAll {
			fmt.Println()
		}
	}

	if showSkills {
		renderSkillsTable(listing.ListSkills(listing.CollectSkillsFromPipelines()))
	}

	return nil
}

func emitJSONList(opts ListOptions, showAdapters, showRuns, showPipelines, showPersonas, showContracts, showSkills bool) error {
	output := ListOutput{}

	manifest, manifestErr := listing.LoadManifest(opts.Manifest)
	if manifestErr == nil {
		if showAdapters {
			output.Adapters = listing.ListAdapters(manifest.Adapters)
		}
		if showPersonas {
			output.Personas = listing.ListPersonas(manifest.Personas)
		}
		if showSkills {
			output.Skills = listing.ListSkills(listing.CollectSkillsFromPipelines())
		}
	}

	if showRuns {
		runs, err := listing.ListRuns(listing.RunsOptions{
			Limit:    listRunsLimit,
			Pipeline: listRunsPipeline,
			Status:   listRunsStatus,
		})
		if err == nil {
			output.Runs = runs
		}
	}

	if showPipelines {
		pipelines, err := listing.ListPipelines()
		if err != nil {
			return err
		}
		output.Pipelines = pipelines
	}

	if showContracts {
		contracts, err := listing.ListContracts()
		if err == nil {
			output.Contracts = contracts
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// runListRuns executes the `wave list runs` subcommand.
func runListRuns(opts ListRunsOptions) error {
	runs, err := listing.ListRuns(listing.RunsOptions{
		Limit:    opts.Limit,
		Pipeline: opts.Pipeline,
		Status:   opts.Status,
	})
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		jsonBytes, err := json.MarshalIndent(ListOutput{Runs: runs}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	printLogo()
	renderRunsTable(runs)
	return nil
}

// runListCompositions handles the `wave list compositions` subcommand.
func runListCompositions(pipelinesDir, format string) error {
	compositions, err := listing.ListCompositions(pipelinesDir)
	if err != nil {
		return err
	}

	if format == "json" {
		// Compositions emit a bare array (historic CLI shape) rather than the
		// aggregated ListOutput envelope.
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if compositions == nil {
			compositions = []listing.CompositionInfo{}
		}
		return enc.Encode(compositions)
	}

	printLogo()
	renderCompositionsTable(compositions)
	return nil
}

// sectionSeparator prints a dim horizontal rule sized to the terminal.
func sectionSeparator(f *display.Formatter) {
	sepWidth := display.GetTerminalWidth()
	if sepWidth < 40 {
		sepWidth = 40
	}
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", sepWidth)))
}

// sectionHeader prints "<title>\n<separator>" using the configured formatter.
func sectionHeader(f *display.Formatter, title string) {
	fmt.Println()
	fmt.Printf("%s\n", f.Colorize(title, "\033[1;37m"))
	sectionSeparator(f)
}
