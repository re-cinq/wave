package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/recinq/wave/internal/proposals"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

const proposalsDBPath = ".agents/state.db"

// NewProposalsCmd creates the `wave proposals` parent command.
func NewProposalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proposals",
		Short: "Manage evolution proposals",
		Long: `List, inspect, approve, or reject evolution proposals.

Proposals come from the pipeline-evolve meta-pipeline. Approving creates a
new pipeline_version row with active=true, atomically deactivating priors.
The next 'wave run <pipeline>' picks up the new yaml.`,
	}

	cmd.AddCommand(newProposalsListCmd())
	cmd.AddCommand(newProposalsShowCmd())
	cmd.AddCommand(newProposalsApproveCmd())
	cmd.AddCommand(newProposalsRejectCmd())
	cmd.AddCommand(newProposalsRollbackCmd())

	return cmd
}

func newProposalsListCmd() *cobra.Command {
	var statusFlag, format string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List proposals (default: status=proposed)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			format = ResolveFormat(cmd, format)
			return runProposalsList(statusFlag, format)
		},
	}
	cmd.Flags().StringVar(&statusFlag, "status", "proposed", "Filter by status: proposed, approved, rejected, superseded")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	return cmd
}

func newProposalsShowCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a single proposal with reason + signal summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format = ResolveFormat(cmd, format)
			return runProposalsShow(args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	return cmd
}

func newProposalsApproveCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a proposal and activate the new pipeline version",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runProposalsApprove(args[0], reason)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Optional approval note (recorded in decided_by)")
	return cmd
}

func newProposalsRejectCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runProposalsReject(args[0], reason)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Rejection reason (recorded in decided_by)")
	return cmd
}

func newProposalsRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <pipeline_name>",
		Short: "Activate the previous pipeline version (auto-rollback)",
		Long: `Find the active pipeline_version and flip activation to the
prior version (highest version_id below the active one). Useful when an
approved evolution misbehaves in production. Fails when no prior version
exists.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runProposalsRollback(strings.TrimSpace(args[0]))
		},
	}
	return cmd
}

func runProposalsRollback(pipelineName string) error {
	if pipelineName == "" {
		return NewCLIError(CodeInvalidArgs, "pipeline name required", "")
	}
	store, err := openProposalStore()
	if err != nil {
		return err
	}
	defer store.Close()

	prior, current, err := proposals.PriorVersion(store, pipelineName)
	if err != nil {
		switch {
		case errors.Is(err, proposals.ErrNoActiveVersion):
			return NewCLIError(CodeValidationFailed, err.Error(),
				"No active pipeline_version row found; nothing to roll back")
		case errors.Is(err, proposals.ErrNoPriorVersion):
			return NewCLIError(CodeValidationFailed, err.Error(),
				"Active version is the first one; cannot roll back further")
		default:
			return NewCLIError(CodeInternalError, err.Error(), "").WithCause(err)
		}
	}
	if err := store.ActivateVersion(pipelineName, prior.Version); err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("activate v%d: %s", prior.Version, err), "").WithCause(err)
	}
	fmt.Printf("Rolled back %s: v%d -> v%d (%s)\n",
		pipelineName, current.Version, prior.Version, prior.YAMLPath)
	return nil
}

func openProposalStore() (state.StateStore, error) {
	if _, err := os.Stat(proposalsDBPath); os.IsNotExist(err) {
		return nil, NewCLIError(CodeStateDBError,
			fmt.Sprintf("state database not found: %s", proposalsDBPath),
			"Run 'wave run' once to create the database, or check your working directory")
	}
	store, err := state.NewStateStore(proposalsDBPath)
	if err != nil {
		return nil, NewCLIError(CodeStateDBError,
			fmt.Sprintf("failed to open state database: %s", err),
			"Check .agents/state.db file permissions").WithCause(err)
	}
	return store, nil
}

func parseProposalsID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("invalid proposal id: %q", raw),
			"Pass a positive integer id (see 'wave proposals list')")
	}
	return id, nil
}

func runProposalsList(statusFlag, format string) error {
	store, err := openProposalStore()
	if err != nil {
		return err
	}
	defer store.Close()

	st := state.EvolutionProposalStatus(strings.TrimSpace(statusFlag))
	if st == "" {
		st = state.ProposalProposed
	}
	recs, err := store.ListProposalsByStatus(st, 0)
	if err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("list proposals: %s", err), "").WithCause(err)
	}

	if format == "json" {
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, proposalToMap(r))
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if len(recs) == 0 {
		fmt.Printf("No proposals with status %s\n", st)
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPIPELINE\tVERSIONS\tSTATUS\tPROPOSED\tREASON")
	for _, r := range recs {
		fmt.Fprintf(tw, "%d\t%s\tv%d->v%d\t%s\t%s\t%s\n",
			r.ID, r.PipelineName, r.VersionBefore, r.VersionAfter,
			r.Status, r.ProposedAt.Format("2006-01-02 15:04"), truncateProposalsField(r.Reason, 60))
	}
	return tw.Flush()
}

func runProposalsShow(idStr, format string) error {
	id, err := parseProposalsID(idStr)
	if err != nil {
		return err
	}
	store, err := openProposalStore()
	if err != nil {
		return err
	}
	defer store.Close()

	rec, err := store.GetProposal(id)
	if err != nil {
		return NewCLIError(CodeInternalError, err.Error(), "").WithCause(err)
	}
	if rec == nil {
		return NewCLIError(CodeRunNotFound,
			fmt.Sprintf("proposal %d not found", id),
			"Run 'wave proposals list' to see available proposals")
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(proposalToMap(*rec))
	}

	fmt.Printf("Proposal #%d\n", rec.ID)
	fmt.Printf("  Pipeline:   %s\n", rec.PipelineName)
	fmt.Printf("  Versions:   v%d -> v%d\n", rec.VersionBefore, rec.VersionAfter)
	fmt.Printf("  Status:     %s\n", rec.Status)
	fmt.Printf("  Proposed:   %s\n", rec.ProposedAt.Format("2006-01-02 15:04:05"))
	if rec.DecidedAt != nil {
		fmt.Printf("  Decided:    %s by %s\n", rec.DecidedAt.Format("2006-01-02 15:04:05"), rec.DecidedBy)
	}
	fmt.Printf("  Diff:       %s\n", rec.DiffPath)
	fmt.Printf("\nReason:\n  %s\n", rec.Reason)
	if rec.SignalSummary != "" {
		fmt.Printf("\nSignal summary:\n  %s\n", rec.SignalSummary)
	}
	return nil
}

func runProposalsApprove(idStr, reason string) error {
	id, err := parseProposalsID(idStr)
	if err != nil {
		return err
	}
	store, err := openProposalStore()
	if err != nil {
		return err
	}
	defer store.Close()

	rec, err := store.GetProposal(id)
	if err != nil {
		return NewCLIError(CodeInternalError, err.Error(), "").WithCause(err)
	}
	if rec == nil {
		return NewCLIError(CodeRunNotFound,
			fmt.Sprintf("proposal %d not found", id), "")
	}
	decidedBy := buildDecidedBy(reason)

	res, err := proposals.Approve(store, rec, decidedBy)
	if err != nil {
		return mapApproveError(err)
	}
	fmt.Printf("Approved proposal #%d. Activated %s v%d (%s)\n",
		rec.ID, rec.PipelineName, res.NewVersion, res.YAMLPath)
	return nil
}

func runProposalsReject(idStr, reason string) error {
	id, err := parseProposalsID(idStr)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return NewCLIError(CodeInvalidArgs,
			"--reason is required to reject a proposal",
			"Provide a short rationale: wave proposals reject <id> --reason \"…\"")
	}
	store, err := openProposalStore()
	if err != nil {
		return err
	}
	defer store.Close()

	decidedBy := buildDecidedBy(reason)
	if err := store.DecideProposal(id, state.ProposalRejected, decidedBy); err != nil {
		return NewCLIError(CodeValidationFailed,
			fmt.Sprintf("reject failed: %s", err),
			"The proposal may already be decided or removed").WithCause(err)
	}
	fmt.Printf("Rejected proposal #%d (reason: %s)\n", id, reason)
	return nil
}

// buildDecidedBy concatenates user identity with the optional reason so the
// audit row preserves both. The schema only has decided_by; the reason is
// folded into the same column to avoid a schema migration for a single field.
func buildDecidedBy(reason string) string {
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		user = "cli"
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		return user + ": " + reason
	}
	return user
}

func mapApproveError(err error) error {
	switch {
	case errors.Is(err, proposals.ErrAlreadyDecided):
		return NewCLIError(CodeValidationFailed, err.Error(),
			"Proposal already decided; check 'wave proposals show <id>'").WithCause(err)
	case errors.Is(err, proposals.ErrVersionConflict):
		// Exit code 2 surfaces via the CLIError code mapping in main.
		return NewCLIError(CodeValidationFailed, err.Error(),
			"Another approval likely landed first; re-list and retry").WithCause(err)
	case errors.Is(err, proposals.ErrAfterYAMLMissing):
		return NewCLIError(CodeValidationFailed, err.Error(),
			"pipeline-evolve must emit <DiffPath>.after.yaml alongside the diff").WithCause(err)
	default:
		return NewCLIError(CodeInternalError, err.Error(), "").WithCause(err)
	}
}

func proposalToMap(r state.EvolutionProposalRecord) map[string]any {
	out := map[string]any{
		"id":             r.ID,
		"pipeline_name":  r.PipelineName,
		"version_before": r.VersionBefore,
		"version_after":  r.VersionAfter,
		"diff_path":      r.DiffPath,
		"reason":         r.Reason,
		"signal_summary": r.SignalSummary,
		"status":         string(r.Status),
		"proposed_at":    r.ProposedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DecidedAt != nil {
		out["decided_at"] = r.DecidedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if r.DecidedBy != "" {
		out["decided_by"] = r.DecidedBy
	}
	return out
}

func truncateProposalsField(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
