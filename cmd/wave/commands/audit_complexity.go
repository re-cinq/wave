package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/complexity"
	"github.com/spf13/cobra"
)

// Exit codes for `wave audit complexity`.
const (
	auditExitOK      = 0
	auditExitBreach  = 1
	auditExitIOError = 2
)

// NewAuditCmd creates the `audit` parent command with deterministic, in-tree
// audit subcommands. Unlike the LLM-driven audit pipelines (audit-security,
// audit-architecture, etc.), commands under this group run pure code-analysis
// and gate via exit codes and structured output.
func NewAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Deterministic in-tree audit subcommands",
		Long: `Code-analysis subcommands that complement the LLM-driven audit pipelines.

Subcommands:
  complexity    Score Go functions by cyclomatic and cognitive complexity`,
	}
	cmd.AddCommand(newAuditComplexityCmd())
	return cmd
}

func newAuditComplexityCmd() *cobra.Command {
	var (
		maxCyclomatic  int
		maxCognitive   int
		warnCyclomatic int
		warnCognitive  int
		outputPath     string
		excludes       []string
		format         string
		includeTests   bool
	)

	cmd := &cobra.Command{
		Use:   "complexity [paths...]",
		Short: "Score Go functions by cyclomatic and cognitive complexity",
		Long: `Walk the given paths (default: current directory), parse Go source files,
and score each function for cyclomatic and cognitive complexity. Functions
exceeding the configured thresholds emit findings to the output file.

Output format conforms to the shared-findings schema so the result is
consumable by aggregate/iterate audit pipelines.

Exit codes:
  0  all functions pass thresholds
  1  one or more functions exceed a fail threshold
  2  IO or parse error`,
		Example: "  wave audit complexity internal/pipeline\n" +
			"  wave audit complexity --max-cyclomatic 20 --output findings.json ./...",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := normalizeAuditPaths(args)
			opts := complexity.Options{
				MaxCyclomatic:  maxCyclomatic,
				MaxCognitive:   maxCognitive,
				WarnCyclomatic: warnCyclomatic,
				WarnCognitive:  warnCognitive,
				IncludeTests:   includeTests,
				Excludes:       excludes,
			}
			report, err := complexity.Analyze(paths, opts)
			if err != nil {
				return cliExitErr(auditExitIOError, fmt.Errorf("analyze: %w", err))
			}
			doc := complexity.ToSharedFindings(report, opts)
			switch strings.ToLower(format) {
			case "summary":
				if err := writeSummary(cmd.OutOrStdout(), report, doc); err != nil {
					return cliExitErr(auditExitIOError, err)
				}
			default:
				if err := writeFindings(outputPath, doc); err != nil {
					return cliExitErr(auditExitIOError, fmt.Errorf("write findings: %w", err))
				}
			}
			if doc.HasBreach() {
				printBreaches(cmd.ErrOrStderr(), doc)
				return cliExitErr(auditExitBreach, errors.New("complexity threshold breach"))
			}
			fmt.Fprintln(cmd.ErrOrStderr(), doc.Summary)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().IntVar(&maxCyclomatic, "max-cyclomatic", 15, "fail threshold for cyclomatic complexity")
	cmd.Flags().IntVar(&maxCognitive, "max-cognitive", 15, "fail threshold for cognitive complexity")
	cmd.Flags().IntVar(&warnCyclomatic, "warn-cyclomatic", 10, "warn threshold for cyclomatic complexity")
	cmd.Flags().IntVar(&warnCognitive, "warn-cognitive", 10, "warn threshold for cognitive complexity")
	cmd.Flags().StringVarP(&outputPath, "output", "o", ".agents/output/findings.json", "path to write findings JSON")
	cmd.Flags().StringSliceVar(&excludes, "exclude", nil, "substring patterns to skip (repeatable)")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json (write to --output) or summary (stdout)")
	cmd.Flags().BoolVar(&includeTests, "include-tests", false, "also score _test.go files")

	return cmd
}

// normalizeAuditPaths defaults to current directory when no args given,
// stripping the Go-style `./...` suffix.
func normalizeAuditPaths(args []string) []string {
	if len(args) == 0 {
		return []string{"."}
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSuffix(a, "/...")
		if a == "" {
			a = "."
		}
		out = append(out, a)
	}
	return out
}

func writeFindings(path string, doc complexity.FindingsDocument) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func writeSummary(w io.Writer, report complexity.Report, doc complexity.FindingsDocument) error {
	if _, err := fmt.Fprintf(w, "scanned %d file(s), %d function(s)\n", report.FileCount, len(report.Scores)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, doc.Summary); err != nil {
		return err
	}
	for _, f := range doc.Findings {
		if _, err := fmt.Fprintf(w, "  [%s] %s:%d %s — %s\n", f.Severity, f.File, f.Line, f.Item, f.Description); err != nil {
			return err
		}
	}
	return nil
}

func printBreaches(w io.Writer, doc complexity.FindingsDocument) {
	fmt.Fprintln(w, doc.Summary)
	for _, f := range doc.Findings {
		if f.Severity != "high" {
			continue
		}
		fmt.Fprintf(w, "BREACH %s:%d %s — %s\n", f.File, f.Line, f.Item, f.Description)
	}
}

// cliExitError carries a non-zero exit code out of RunE so main can read it.
type cliExitError struct {
	code int
	err  error
}

func (e *cliExitError) Error() string { return e.err.Error() }
func (e *cliExitError) Unwrap() error { return e.err }
func (e *cliExitError) ExitCode() int { return e.code }

func cliExitErr(code int, err error) error {
	return &cliExitError{code: code, err: err}
}

// ExitCodeFor returns the exit code carried by err, or 1 if none.
// Defined here so main.go can honor command-specific exit codes (e.g.,
// 1 for breach vs 2 for IO error in `wave audit complexity`).
func ExitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	var ec interface{ ExitCode() int }
	if errors.As(err, &ec) {
		if c := ec.ExitCode(); c > 0 {
			return c
		}
	}
	return 1
}
