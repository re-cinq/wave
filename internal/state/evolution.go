package state

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PipelineEvalRecord captures the per-run signal aggregation that drives the
// evolution loop: judge score, contract pass/fail, retry count, failure class,
// human override flag, duration, and cost. Inserted at run completion (Phase 3
// of the onboarding-as-session epic).
type PipelineEvalRecord struct {
	PipelineName  string
	RunID         string
	JudgeScore    *float64
	ContractPass  *bool
	RetryCount    *int
	FailureClass  string
	HumanOverride *bool
	DurationMs    *int64
	CostDollars   *float64
	RecordedAt    time.Time
}

// PipelineVersionRecord pins a sha256+yaml_path tuple per (pipeline_name,
// version). Exactly one row per pipeline is active at a time; activation
// flips happen in CreatePipelineVersion (active=true) or ActivateVersion.
type PipelineVersionRecord struct {
	PipelineName string
	Version      int
	SHA256       string
	YAMLPath     string
	Active       bool
	CreatedAt    time.Time
}

// EvolutionProposalStatus enumerates the lifecycle states of an evolution
// proposal awaiting human review.
type EvolutionProposalStatus string

const (
	ProposalProposed   EvolutionProposalStatus = "proposed"
	ProposalApproved   EvolutionProposalStatus = "approved"
	ProposalRejected   EvolutionProposalStatus = "rejected"
	ProposalSuperseded EvolutionProposalStatus = "superseded"
)

// EvolutionProposalRecord is one row from evolution_proposal. The diff lives
// on disk at DiffPath; the SignalSummary is opaque JSON the producer chose.
type EvolutionProposalRecord struct {
	ID             int64
	PipelineName   string
	VersionBefore  int
	VersionAfter   int
	DiffPath       string
	Reason         string
	SignalSummary  string
	Status         EvolutionProposalStatus
	ProposedAt     time.Time
	DecidedAt      *time.Time
	DecidedBy      string
}

// EvolutionStore is the domain-scoped persistence surface for the evolution
// loop. Consumers wiring evolution-only logic (e.g. internal/evolution) should
// depend on this interface rather than the aggregate StateStore.
type EvolutionStore interface {
	RecordEval(rec PipelineEvalRecord) error
	GetEvalsForPipeline(pipelineName string, limit int) ([]PipelineEvalRecord, error)

	CreatePipelineVersion(rec PipelineVersionRecord) error
	ActivateVersion(pipelineName string, version int) error
	GetActiveVersion(pipelineName string) (*PipelineVersionRecord, error)
	ListPipelineVersions(pipelineName string) ([]PipelineVersionRecord, error)

	CreateProposal(rec EvolutionProposalRecord) (int64, error)
	DecideProposal(id int64, status EvolutionProposalStatus, decidedBy string) error
	GetProposal(id int64) (*EvolutionProposalRecord, error)
	ListProposalsByStatus(status EvolutionProposalStatus, limit int) ([]EvolutionProposalRecord, error)
	LastProposalAt(pipelineName string) (time.Time, bool, error)

	// ApproveProposalAndActivate atomically marks a proposal approved AND inserts
	// the new pipeline_version row (with active=true, deactivating priors). Both
	// effects commit together or roll back together — the caller cannot end up
	// in a half-state where the proposal is approved but no version row exists.
	ApproveProposalAndActivate(proposalID int64, decidedBy string, version PipelineVersionRecord) error
}

// RecordEval inserts a row into pipeline_eval. The (pipeline_name, run_id)
// composite key prevents double-recording when the executor's terminal hook
// fires more than once.
func (s *stateStore) RecordEval(rec PipelineEvalRecord) error {
	if rec.PipelineName == "" || rec.RunID == "" {
		return errors.New("RecordEval: pipeline_name and run_id are required")
	}
	if rec.RecordedAt.IsZero() {
		rec.RecordedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO pipeline_eval
			(pipeline_name, run_id, judge_score, contract_pass, retry_count,
			 failure_class, human_override, duration_ms, cost_dollars, recorded_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.PipelineName, rec.RunID,
		nullableFloat(rec.JudgeScore),
		nullableBool(rec.ContractPass),
		nullableInt(rec.RetryCount),
		nullEmptyString(rec.FailureClass),
		nullableBool(rec.HumanOverride),
		nullableInt64(rec.DurationMs),
		nullableFloat(rec.CostDollars),
		rec.RecordedAt.Unix(),
	)
	return err
}

// GetEvalsForPipeline returns the most recent eval rows for a pipeline,
// newest first. Limit ≤ 0 returns all rows.
func (s *stateStore) GetEvalsForPipeline(pipelineName string, limit int) ([]PipelineEvalRecord, error) {
	q := `SELECT pipeline_name, run_id, judge_score, contract_pass, retry_count,
		failure_class, human_override, duration_ms, cost_dollars, recorded_at
		FROM pipeline_eval WHERE pipeline_name = ? ORDER BY recorded_at DESC`
	args := []any{pipelineName}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PipelineEvalRecord
	for rows.Next() {
		var (
			r            PipelineEvalRecord
			judgeScore   sql.NullFloat64
			contractPass sql.NullBool
			retryCount   sql.NullInt64
			failureClass sql.NullString
			humanOver    sql.NullBool
			durationMs   sql.NullInt64
			costDollars  sql.NullFloat64
			recordedAt   int64
		)
		if err := rows.Scan(&r.PipelineName, &r.RunID, &judgeScore, &contractPass, &retryCount,
			&failureClass, &humanOver, &durationMs, &costDollars, &recordedAt); err != nil {
			return nil, err
		}
		if judgeScore.Valid {
			v := judgeScore.Float64
			r.JudgeScore = &v
		}
		if contractPass.Valid {
			v := contractPass.Bool
			r.ContractPass = &v
		}
		if retryCount.Valid {
			v := int(retryCount.Int64)
			r.RetryCount = &v
		}
		if failureClass.Valid {
			r.FailureClass = failureClass.String
		}
		if humanOver.Valid {
			v := humanOver.Bool
			r.HumanOverride = &v
		}
		if durationMs.Valid {
			v := durationMs.Int64
			r.DurationMs = &v
		}
		if costDollars.Valid {
			v := costDollars.Float64
			r.CostDollars = &v
		}
		r.RecordedAt = time.Unix(recordedAt, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreatePipelineVersion inserts a new pipeline_version row. When rec.Active is
// true, all other rows for the same pipeline_name are deactivated atomically.
func (s *stateStore) CreatePipelineVersion(rec PipelineVersionRecord) error {
	if rec.PipelineName == "" || rec.SHA256 == "" || rec.YAMLPath == "" {
		return errors.New("CreatePipelineVersion: pipeline_name, sha256, yaml_path required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertPipelineVersionTx(tx, &rec); err != nil {
		return err
	}
	return tx.Commit()
}

// insertPipelineVersionTx writes a pipeline_version row inside the supplied
// transaction. When rec.Active is true, all other rows for the same
// pipeline_name are deactivated first. Shared by CreatePipelineVersion and
// ApproveProposalAndActivate so the schema-touching SQL lives in one place.
// Mutates rec.CreatedAt when zero.
func insertPipelineVersionTx(tx *sql.Tx, rec *PipelineVersionRecord) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	if rec.Active {
		if _, err := tx.Exec(
			`UPDATE pipeline_version SET active = 0 WHERE pipeline_name = ?`,
			rec.PipelineName,
		); err != nil {
			return fmt.Errorf("deactivate prior versions: %w", err)
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO pipeline_version (pipeline_name, version, sha256, yaml_path, active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		rec.PipelineName, rec.Version, rec.SHA256, rec.YAMLPath, rec.Active, rec.CreatedAt.Unix(),
	); err != nil {
		return err
	}
	return nil
}

// ActivateVersion flips the active flag to the requested version, deactivating
// all other versions of the same pipeline atomically.
func (s *stateStore) ActivateVersion(pipelineName string, version int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`UPDATE pipeline_version SET active = 0 WHERE pipeline_name = ?`,
		pipelineName,
	); err != nil {
		return err
	}
	res, err := tx.Exec(
		`UPDATE pipeline_version SET active = 1 WHERE pipeline_name = ? AND version = ?`,
		pipelineName, version,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ActivateVersion: pipeline %q version %d not found", pipelineName, version)
	}
	return tx.Commit()
}

// GetActiveVersion returns the row with active=1 for the pipeline, or nil if
// no version is active.
func (s *stateStore) GetActiveVersion(pipelineName string) (*PipelineVersionRecord, error) {
	row := s.db.QueryRow(
		`SELECT pipeline_name, version, sha256, yaml_path, active, created_at
		 FROM pipeline_version WHERE pipeline_name = ? AND active = 1`,
		pipelineName,
	)
	rec, err := scanPipelineVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rec, err
}

// ListPipelineVersions returns every version row for a pipeline, newest first.
func (s *stateStore) ListPipelineVersions(pipelineName string) ([]PipelineVersionRecord, error) {
	rows, err := s.db.Query(
		`SELECT pipeline_name, version, sha256, yaml_path, active, created_at
		 FROM pipeline_version WHERE pipeline_name = ? ORDER BY version DESC`,
		pipelineName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PipelineVersionRecord
	for rows.Next() {
		var r PipelineVersionRecord
		var createdAt int64
		if err := rows.Scan(&r.PipelineName, &r.Version, &r.SHA256, &r.YAMLPath, &r.Active, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanPipelineVersion(row *sql.Row) (*PipelineVersionRecord, error) {
	var r PipelineVersionRecord
	var createdAt int64
	if err := row.Scan(&r.PipelineName, &r.Version, &r.SHA256, &r.YAMLPath, &r.Active, &createdAt); err != nil {
		return nil, err
	}
	r.CreatedAt = time.Unix(createdAt, 0)
	return &r, nil
}

// CreateProposal inserts an evolution_proposal row in 'proposed' state and
// returns the autoincrement id. ProposedAt defaults to time.Now() when zero.
func (s *stateStore) CreateProposal(rec EvolutionProposalRecord) (int64, error) {
	if rec.PipelineName == "" || rec.DiffPath == "" || rec.Reason == "" {
		return 0, errors.New("CreateProposal: pipeline_name, diff_path, reason required")
	}
	if rec.Status == "" {
		rec.Status = ProposalProposed
	}
	if rec.ProposedAt.IsZero() {
		rec.ProposedAt = time.Now()
	}
	res, err := s.db.Exec(
		`INSERT INTO evolution_proposal
			(pipeline_name, version_before, version_after, diff_path, reason,
			 signal_summary, status, proposed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.PipelineName, rec.VersionBefore, rec.VersionAfter, rec.DiffPath,
		rec.Reason, rec.SignalSummary, string(rec.Status), rec.ProposedAt.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ApproveProposalAndActivate wraps the proposal-decide and version-create
// effects in a single transaction so the cross-table coupling never lands in
// a half-state. On any error the tx rolls back, leaving the proposal in
// `proposed` status and no new version row.
func (s *stateStore) ApproveProposalAndActivate(proposalID int64, decidedBy string, rec PipelineVersionRecord) error {
	if rec.PipelineName == "" || rec.SHA256 == "" || rec.YAMLPath == "" {
		return errors.New("ApproveProposalAndActivate: pipeline_name, sha256, yaml_path required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertPipelineVersionTx(tx, &rec); err != nil {
		return err
	}
	res, err := tx.Exec(
		`UPDATE evolution_proposal
		 SET status = ?, decided_at = ?, decided_by = ?
		 WHERE id = ? AND status = 'proposed'`,
		string(ProposalApproved), time.Now().Unix(), decidedBy, proposalID,
	)
	if err != nil {
		return fmt.Errorf("decide proposal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ApproveProposalAndActivate: proposal %d not in 'proposed' state", proposalID)
	}
	return tx.Commit()
}

// DecideProposal updates a proposal to a terminal status and records who
// decided. Returns an error if the proposal is already terminal.
func (s *stateStore) DecideProposal(id int64, status EvolutionProposalStatus, decidedBy string) error {
	if status == ProposalProposed {
		return errors.New("DecideProposal: cannot decide back to 'proposed'")
	}
	res, err := s.db.Exec(
		`UPDATE evolution_proposal
		 SET status = ?, decided_at = ?, decided_by = ?
		 WHERE id = ? AND status = 'proposed'`,
		string(status), time.Now().Unix(), decidedBy, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("DecideProposal: proposal %d not in 'proposed' state", id)
	}
	return nil
}

// GetProposal returns one proposal by id, or nil if not found.
func (s *stateStore) GetProposal(id int64) (*EvolutionProposalRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, pipeline_name, version_before, version_after, diff_path,
			reason, signal_summary, status, proposed_at, decided_at, decided_by
		 FROM evolution_proposal WHERE id = ?`,
		id,
	)
	rec, err := scanProposal(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rec, err
}

// ListProposalsByStatus returns proposals matching status, newest first.
// Limit ≤ 0 returns all.
func (s *stateStore) ListProposalsByStatus(status EvolutionProposalStatus, limit int) ([]EvolutionProposalRecord, error) {
	q := `SELECT id, pipeline_name, version_before, version_after, diff_path,
		reason, signal_summary, status, proposed_at, decided_at, decided_by
		FROM evolution_proposal WHERE status = ? ORDER BY proposed_at DESC`
	args := []any{string(status)}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EvolutionProposalRecord
	for rows.Next() {
		var (
			r          EvolutionProposalRecord
			decidedAt  sql.NullInt64
			decidedBy  sql.NullString
			proposedAt int64
			statusStr  string
		)
		if err := rows.Scan(&r.ID, &r.PipelineName, &r.VersionBefore, &r.VersionAfter,
			&r.DiffPath, &r.Reason, &r.SignalSummary, &statusStr, &proposedAt,
			&decidedAt, &decidedBy); err != nil {
			return nil, err
		}
		r.Status = EvolutionProposalStatus(statusStr)
		r.ProposedAt = time.Unix(proposedAt, 0)
		if decidedAt.Valid {
			t := time.Unix(decidedAt.Int64, 0)
			r.DecidedAt = &t
		}
		if decidedBy.Valid {
			r.DecidedBy = decidedBy.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LastProposalAt returns the proposed_at timestamp of the most recent
// evolution_proposal row for pipelineName, regardless of status. The bool
// reports whether any row exists; on (false, nil) the time return is zero.
// Phase 3.3 trigger heuristics use this to anchor "since last evolution"
// windows.
func (s *stateStore) LastProposalAt(pipelineName string) (time.Time, bool, error) {
	var ts int64
	row := s.db.QueryRow(
		`SELECT proposed_at FROM evolution_proposal
		 WHERE pipeline_name = ? ORDER BY proposed_at DESC LIMIT 1`,
		pipelineName,
	)
	if err := row.Scan(&ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return time.Unix(ts, 0), true, nil
}

func scanProposal(row *sql.Row) (*EvolutionProposalRecord, error) {
	var (
		r          EvolutionProposalRecord
		decidedAt  sql.NullInt64
		decidedBy  sql.NullString
		proposedAt int64
		statusStr  string
	)
	if err := row.Scan(&r.ID, &r.PipelineName, &r.VersionBefore, &r.VersionAfter,
		&r.DiffPath, &r.Reason, &r.SignalSummary, &statusStr, &proposedAt,
		&decidedAt, &decidedBy); err != nil {
		return nil, err
	}
	r.Status = EvolutionProposalStatus(statusStr)
	r.ProposedAt = time.Unix(proposedAt, 0)
	if decidedAt.Valid {
		t := time.Unix(decidedAt.Int64, 0)
		r.DecidedAt = &t
	}
	if decidedBy.Valid {
		r.DecidedBy = decidedBy.String
	}
	return &r, nil
}

// nullable* helpers convert *T into a sql-friendly value (nil → NULL).
func nullableFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullableBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullableInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullEmptyString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
