package state

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var (
	//go:embed schema.sql
	schemaFS embed.FS
)

// StepState represents the state of a pipeline step.
type StepState string

const (
	StatePending   StepState = "pending"
	StateRunning   StepState = "running"
	StateCompleted StepState = "completed"
	StateFailed    StepState = "failed"
	StateRetrying  StepState = "retrying"
)

// PipelineStateRecord holds persisted pipeline state.
type PipelineStateRecord struct {
	PipelineID string
	Name       string
	Status     string
	Input      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// StepStateRecord holds persisted step state.
type StepStateRecord struct {
	StepID        string
	PipelineID    string
	State         StepState
	RetryCount    int
	StartedAt     *time.Time
	CompletedAt   *time.Time
	WorkspacePath string
	ErrorMessage  string
}

// StateStore persists and retrieves pipeline execution state.
type StateStore interface {
	SavePipelineState(id string, status string, input string) error
	SaveStepState(pipelineID string, stepID string, state StepState, err string) error
	GetPipelineState(id string) (*PipelineStateRecord, error)
	GetStepStates(pipelineID string) ([]StepStateRecord, error)
	ListRecentPipelines(limit int) ([]PipelineStateRecord, error)
	Close() error

	// Run tracking (ops commands)
	CreateRun(pipelineName string, input string) (string, error)
	UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
	GetRun(runID string) (*RunRecord, error)
	GetRunningRuns() ([]RunRecord, error)
	ListRuns(opts ListRunsOptions) ([]RunRecord, error)
	DeleteRun(runID string) error

	// Event logging
	LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64) error
	GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error)

	// Artifact tracking
	RegisterArtifact(runID string, stepID string, name string, path string, artifactType string, sizeBytes int64) error
	GetArtifacts(runID string, stepID string) ([]ArtifactRecord, error)

	// Cancellation
	RequestCancellation(runID string, force bool) error
	CheckCancellation(runID string) (*CancellationRecord, error)
	ClearCancellation(runID string) error
}

type stateStore struct {
	db *sql.DB
}

func NewStateStore(dbPath string) (StateStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for SQLite
	// SQLite performs best with limited connections due to its locking model
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for concurrent access
	// Enable WAL mode for better concurrent read/write performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Set busy timeout to 5 seconds to handle lock contention
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign key enforcement (disabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &stateStore{db: db}, nil
}

func (s *stateStore) SavePipelineState(id string, status string, input string) error {
	now := time.Now().Unix()

	query := `INSERT INTO pipeline_state (pipeline_id, pipeline_name, status, input, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)
	          ON CONFLICT(pipeline_id) DO UPDATE SET
	              status = excluded.status,
	              input = excluded.input,
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, id, id, status, input, now, now)
	if err != nil {
		return fmt.Errorf("failed to save pipeline state: %w", err)
	}

	return nil
}

func (s *stateStore) SaveStepState(pipelineID string, stepID string, state StepState, errMsg string) error {
	now := time.Now().Unix()

	query := `INSERT INTO step_state (step_id, pipeline_id, state, retry_count, started_at, completed_at, workspace_path, error_message)
	          VALUES (?, ?, ?, 0, ?, ?, NULL, ?)
	          ON CONFLICT(step_id) DO UPDATE SET
	              state = excluded.state,
	              retry_count = CASE WHEN excluded.state = 'retrying' THEN retry_count + 1 ELSE retry_count END,
	              started_at = COALESCE(started_at, excluded.started_at),
	              completed_at = excluded.completed_at,
	              error_message = excluded.error_message`

	var startedAt, completedAt *int64
	if state == StateRunning || state == StateRetrying {
		startedAt = &now
	}
	if state == StateCompleted || state == StateFailed {
		completedAt = &now
	}

	_, execErr := s.db.Exec(query, stepID, pipelineID, string(state), startedAt, completedAt, errMsg)
	if execErr != nil {
		return fmt.Errorf("failed to save step state: %w", execErr)
	}

	return nil
}

func (s *stateStore) GetPipelineState(id string) (*PipelineStateRecord, error) {
	query := `SELECT pipeline_id, pipeline_name, status, input, created_at, updated_at
	          FROM pipeline_state
	          WHERE pipeline_id = ?`

	var record PipelineStateRecord
	var createdAt, updatedAt int64

	err := s.db.QueryRow(query, id).Scan(
		&record.PipelineID,
		&record.Name,
		&record.Status,
		&record.Input,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pipeline state not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get pipeline state: %w", err)
	}

	record.CreatedAt = time.Unix(createdAt, 0)
	record.UpdatedAt = time.Unix(updatedAt, 0)

	return &record, nil
}

func (s *stateStore) GetStepStates(pipelineID string) ([]StepStateRecord, error) {
	query := `SELECT step_id, pipeline_id, state, retry_count, started_at, completed_at, workspace_path, error_message
	          FROM step_state
	          WHERE pipeline_id = ?
	          ORDER BY step_id`

	rows, err := s.db.Query(query, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to query step states: %w", err)
	}
	defer rows.Close()

	var records []StepStateRecord
	for rows.Next() {
		var record StepStateRecord
		var startedAt, completedAt sql.NullInt64
		var workspacePath, errMsg sql.NullString

		err := rows.Scan(
			&record.StepID,
			&record.PipelineID,
			&record.State,
			&record.RetryCount,
			&startedAt,
			&completedAt,
			&workspacePath,
			&errMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step state: %w", err)
		}

		if startedAt.Valid {
			t := time.Unix(startedAt.Int64, 0)
			record.StartedAt = &t
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			record.CompletedAt = &t
		}
		if workspacePath.Valid {
			record.WorkspacePath = workspacePath.String
		}
		if errMsg.Valid {
			record.ErrorMessage = errMsg.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step states: %w", err)
	}

	return records, nil
}

func (s *stateStore) ListRecentPipelines(limit int) ([]PipelineStateRecord, error) {
	query := `SELECT pipeline_id, pipeline_name, status, input, created_at, updated_at
	          FROM pipeline_state
	          ORDER BY updated_at DESC
	          LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent pipelines: %w", err)
	}
	defer rows.Close()

	var records []PipelineStateRecord
	for rows.Next() {
		var record PipelineStateRecord
		var createdAt, updatedAt int64

		err := rows.Scan(
			&record.PipelineID,
			&record.Name,
			&record.Status,
			&record.Input,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pipeline record: %w", err)
		}

		record.CreatedAt = time.Unix(createdAt, 0)
		record.UpdatedAt = time.Unix(updatedAt, 0)
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pipeline records: %w", err)
	}

	return records, nil
}

func (s *stateStore) Close() error {
	return s.db.Close()
}

// CreateRun creates a new pipeline run record and returns the generated run ID.
// Run ID format: {pipeline_name}-{timestamp}-{random} e.g., debug-20260202-143022-a1b2
func (s *stateStore) CreateRun(pipelineName string, input string) (string, error) {
	now := time.Now()
	// Include nanoseconds truncated to 4 chars to avoid collisions when multiple runs are created in the same second
	suffix := fmt.Sprintf("%04d", now.Nanosecond()/100000)
	runID := fmt.Sprintf("%s-%s-%s", pipelineName, now.Format("20060102-150405"), suffix)

	query := `INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at)
	          VALUES (?, ?, 'pending', ?, ?)`

	_, err := s.db.Exec(query, runID, pipelineName, input, now.Unix())
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	return runID, nil
}

// UpdateRunStatus updates the status, current step, and token count for a run.
// Sets completed_at if status is completed, failed, or cancelled.
func (s *stateStore) UpdateRunStatus(runID string, status string, currentStep string, tokens int) error {
	now := time.Now().Unix()

	var completedAt *int64
	var cancelledAt *int64
	if status == "completed" || status == "failed" {
		completedAt = &now
	}
	if status == "cancelled" {
		cancelledAt = &now
		completedAt = &now
	}

	query := `UPDATE pipeline_run
	          SET status = ?, current_step = ?, total_tokens = ?, completed_at = ?, cancelled_at = ?
	          WHERE run_id = ?`

	result, err := s.db.Exec(query, status, currentStep, tokens, completedAt, cancelledAt, runID)
	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	return nil
}

// GetRun retrieves a single run record by ID.
func (s *stateStore) GetRun(runID string) (*RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message
	          FROM pipeline_run
	          WHERE run_id = ?`

	var record RunRecord
	var startedAt int64
	var completedAt, cancelledAt sql.NullInt64
	var input, currentStep, errorMessage sql.NullString

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&record.PipelineName,
		&record.Status,
		&input,
		&currentStep,
		&record.TotalTokens,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&errorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	record.StartedAt = time.Unix(startedAt, 0)
	if input.Valid {
		record.Input = input.String
	}
	if currentStep.Valid {
		record.CurrentStep = currentStep.String
	}
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		record.CompletedAt = &t
	}
	if cancelledAt.Valid {
		t := time.Unix(cancelledAt.Int64, 0)
		record.CancelledAt = &t
	}
	if errorMessage.Valid {
		record.ErrorMessage = errorMessage.String
	}

	return &record, nil
}

// GetRunningRuns returns all runs with status 'running'.
func (s *stateStore) GetRunningRuns() ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message
	          FROM pipeline_run
	          WHERE status = 'running'
	          ORDER BY started_at DESC`

	return s.queryRuns(query)
}

// ListRuns returns runs matching the specified options.
func (s *stateStore) ListRuns(opts ListRunsOptions) ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message
	          FROM pipeline_run
	          WHERE 1=1`
	args := []any{}

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.OlderThan > 0 {
		cutoff := time.Now().Add(-opts.OlderThan).Unix()
		query += " AND started_at < ?"
		args = append(args, cutoff)
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	return s.queryRunsWithArgs(query, args...)
}

// DeleteRun removes a run and its associated events, artifacts, and cancellation records.
func (s *stateStore) DeleteRun(runID string) error {
	// Due to foreign key ON DELETE CASCADE, deleting from pipeline_run
	// will automatically delete related event_log, artifact, and cancellation records.
	query := `DELETE FROM pipeline_run WHERE run_id = ?`

	result, err := s.db.Exec(query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	return nil
}

// LogEvent records an event in the event_log table.
func (s *stateStore) LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64) error {
	now := time.Now().Unix()

	query := `INSERT INTO event_log (run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, now, stepID, state, persona, message, tokens, durationMs)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return nil
}

// GetEvents retrieves events for a run with optional filtering.
func (s *stateStore) GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error) {
	query := `SELECT id, run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms
	          FROM event_log
	          WHERE run_id = ?`
	args := []any{runID}

	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.ErrorsOnly {
		query += " AND state = 'failed'"
	}

	query += " ORDER BY timestamp ASC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var records []LogRecord
	for rows.Next() {
		var record LogRecord
		var timestamp int64
		var stepID, persona, message sql.NullString
		var tokensUsed, durationMs sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&timestamp,
			&stepID,
			&record.State,
			&persona,
			&message,
			&tokensUsed,
			&durationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		record.Timestamp = time.Unix(timestamp, 0)
		if stepID.Valid {
			record.StepID = stepID.String
		}
		if persona.Valid {
			record.Persona = persona.String
		}
		if message.Valid {
			record.Message = message.String
		}
		if tokensUsed.Valid {
			record.TokensUsed = int(tokensUsed.Int64)
		}
		if durationMs.Valid {
			record.DurationMs = durationMs.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return records, nil
}

// RegisterArtifact records an artifact in the artifact table.
func (s *stateStore) RegisterArtifact(runID string, stepID string, name string, path string, artifactType string, sizeBytes int64) error {
	now := time.Now().Unix()

	query := `INSERT INTO artifact (run_id, step_id, name, path, type, size_bytes, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, stepID, name, path, artifactType, sizeBytes, now)
	if err != nil {
		return fmt.Errorf("failed to register artifact: %w", err)
	}

	return nil
}

// GetArtifacts retrieves artifacts for a run, optionally filtered by step ID.
func (s *stateStore) GetArtifacts(runID string, stepID string) ([]ArtifactRecord, error) {
	query := `SELECT id, run_id, step_id, name, path, type, size_bytes, created_at
	          FROM artifact
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var records []ArtifactRecord
	for rows.Next() {
		var record ArtifactRecord
		var createdAt int64
		var artifactType sql.NullString
		var sizeBytes sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&record.StepID,
			&record.Name,
			&record.Path,
			&artifactType,
			&sizeBytes,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}

		record.CreatedAt = time.Unix(createdAt, 0)
		if artifactType.Valid {
			record.Type = artifactType.String
		}
		if sizeBytes.Valid {
			record.SizeBytes = sizeBytes.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return records, nil
}

// RequestCancellation sets a cancellation flag for a run.
func (s *stateStore) RequestCancellation(runID string, force bool) error {
	now := time.Now().Unix()

	query := `INSERT INTO cancellation (run_id, requested_at, force)
	          VALUES (?, ?, ?)
	          ON CONFLICT(run_id) DO UPDATE SET
	              requested_at = excluded.requested_at,
	              force = excluded.force`

	_, err := s.db.Exec(query, runID, now, force)
	if err != nil {
		return fmt.Errorf("failed to request cancellation: %w", err)
	}

	return nil
}

// CheckCancellation checks if a cancellation has been requested for a run.
// Returns nil if no cancellation is pending.
func (s *stateStore) CheckCancellation(runID string) (*CancellationRecord, error) {
	query := `SELECT run_id, requested_at, force
	          FROM cancellation
	          WHERE run_id = ?`

	var record CancellationRecord
	var requestedAt int64

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&requestedAt,
		&record.Force,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check cancellation: %w", err)
	}

	record.RequestedAt = time.Unix(requestedAt, 0)

	return &record, nil
}

// ClearCancellation removes the cancellation flag for a run.
func (s *stateStore) ClearCancellation(runID string) error {
	query := `DELETE FROM cancellation WHERE run_id = ?`

	_, err := s.db.Exec(query, runID)
	if err != nil {
		return fmt.Errorf("failed to clear cancellation: %w", err)
	}

	return nil
}

// Helper methods

func (s *stateStore) queryRuns(query string) ([]RunRecord, error) {
	return s.queryRunsWithArgs(query)
}

func (s *stateStore) queryRunsWithArgs(query string, args ...any) ([]RunRecord, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	var records []RunRecord
	for rows.Next() {
		var record RunRecord
		var startedAt int64
		var completedAt, cancelledAt sql.NullInt64
		var input, currentStep, errorMessage sql.NullString

		err := rows.Scan(
			&record.RunID,
			&record.PipelineName,
			&record.Status,
			&input,
			&currentStep,
			&record.TotalTokens,
			&startedAt,
			&completedAt,
			&cancelledAt,
			&errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}

		record.StartedAt = time.Unix(startedAt, 0)
		if input.Valid {
			record.Input = input.String
		}
		if currentStep.Valid {
			record.CurrentStep = currentStep.String
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			record.CompletedAt = &t
		}
		if cancelledAt.Valid {
			t := time.Unix(cancelledAt.Int64, 0)
			record.CancelledAt = &t
		}
		if errorMessage.Valid {
			record.ErrorMessage = errorMessage.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating runs: %w", err)
	}

	return records, nil
}
