package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
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

	// Performance metrics (spec 018)
	RecordPerformanceMetric(metric *PerformanceMetricRecord) error
	GetPerformanceMetrics(runID string, stepID string) ([]PerformanceMetricRecord, error)
	GetStepPerformanceStats(pipelineName string, stepID string, since time.Time) (*StepPerformanceStats, error)
	GetRecentPerformanceHistory(opts PerformanceQueryOptions) ([]PerformanceMetricRecord, error)
	CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error)

	// Progress tracking (spec 018 - Enhanced Progress Visualization)
	SaveProgressSnapshot(runID string, stepID string, progress int, action string, etaMs int64, validationPhase string, compactionStats string) error
	GetProgressSnapshots(runID string, stepID string, limit int) ([]ProgressSnapshotRecord, error)
	UpdateStepProgress(runID string, stepID string, persona string, state string, progress int, action string, message string, etaMs int64, tokens int) error
	GetStepProgress(stepID string) (*StepProgressRecord, error)
	GetAllStepProgress(runID string) ([]StepProgressRecord, error)
	UpdatePipelineProgress(runID string, totalSteps int, completedSteps int, currentStepIndex int, overallProgress int, etaMs int64) error
	GetPipelineProgress(runID string) (*PipelineProgressRecord, error)
	SaveArtifactMetadata(artifactID int64, runID string, stepID string, previewText string, mimeType string, encoding string, metadataJSON string) error
	GetArtifactMetadata(artifactID int64) (*ArtifactMetadataRecord, error)

	// Tags support
	SetRunTags(runID string, tags []string) error
	GetRunTags(runID string) ([]string, error)
	AddRunTag(runID string, tag string) error
	RemoveRunTag(runID string, tag string) error
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

	// Load migration configuration from environment
	migrationConfig := LoadMigrationConfigFromEnv()

	if err := migrationConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid migration configuration: %w", err)
	}

	if !migrationConfig.ShouldUseMigrations() {
		return nil, fmt.Errorf("legacy schema initialization has been removed; migrations are now the only supported method — remove the WAVE_MIGRATION_ENABLED=false setting")
	}

	if err := initializeWithMigrations(db, migrationConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize with migrations: %w", err)
	}

	return &stateStore{db: db}, nil
}

// initializeWithMigrations initializes the database using the migration system
func initializeWithMigrations(db *sql.DB, config *MigrationConfig) error {
	// Initialize migration system
	migrationManager := NewMigrationManager(db)

	// Create migration tracking table
	if err := migrationManager.InitializeMigrationTable(); err != nil {
		return fmt.Errorf("failed to initialize migration table: %w", err)
	}

	// Only auto-migrate if configured to do so
	if !config.ShouldAutoMigrate() {
		return nil
	}

	// Check if this is a fresh database or needs migration
	currentVersion, err := migrationManager.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	allMigrations := GetAllMigrations()

	// Filter migrations based on max version config
	if config.MaxMigrationVersion > 0 {
		var filteredMigrations []Migration
		for _, migration := range allMigrations {
			if migration.Version <= config.MaxMigrationVersion {
				filteredMigrations = append(filteredMigrations, migration)
			}
		}
		allMigrations = filteredMigrations
	}

	if currentVersion == 0 {
		// Fresh database - check if it has existing tables from old schema system
		var tableCount int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT IN ('schema_migrations')").Scan(&tableCount)
		if err != nil {
			return fmt.Errorf("failed to check existing tables: %w", err)
		}

		if tableCount > 0 {
			// Existing database without migration tracking - mark all migrations as applied
			fmt.Printf("Detected existing database without migration tracking, marking schema up to version %d as applied\n", len(allMigrations))
			for _, migration := range allMigrations {
				checksum := calculateChecksum(migration.Up)
				now := time.Now().Unix()

				_, err := db.Exec(
					"INSERT INTO schema_migrations (version, description, applied_at, checksum) VALUES (?, ?, ?, ?)",
					migration.Version, migration.Description, now, checksum,
				)
				if err != nil {
					return fmt.Errorf("failed to mark migration %d as applied: %w", migration.Version, err)
				}
			}
		} else {
			// Fresh database - apply all migrations
			maxVersion := config.GetMaxVersion()
			if err := migrationManager.MigrateUp(allMigrations, maxVersion); err != nil {
				return fmt.Errorf("failed to apply initial migrations: %w", err)
			}
		}
	} else {
		// Apply any pending migrations up to the max version
		maxVersion := config.GetMaxVersion()
		if err := migrationManager.MigrateUp(allMigrations, maxVersion); err != nil {
			return fmt.Errorf("failed to apply pending migrations: %w", err)
		}
	}

	return nil
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
	// Use crypto/rand for collision-resistant suffix
	randBytes := make([]byte, 2)
	if _, err := rand.Read(randBytes); err != nil {
		// Fallback to nanoseconds if crypto/rand fails
		randBytes = []byte{byte(now.Nanosecond() >> 8), byte(now.Nanosecond())}
	}
	suffix := hex.EncodeToString(randBytes)
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
	                 started_at, completed_at, cancelled_at, error_message, tags_json
	          FROM pipeline_run
	          WHERE run_id = ?`

	var record RunRecord
	var startedAt int64
	var completedAt, cancelledAt sql.NullInt64
	var input, currentStep, errorMessage, tagsJSON sql.NullString

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
		&tagsJSON,
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
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &record.Tags); err != nil {
			// If JSON parsing fails, treat as empty tags
			record.Tags = []string{}
		}
	}

	return &record, nil
}

// GetRunningRuns returns all runs with status 'running'.
func (s *stateStore) GetRunningRuns() ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json
	          FROM pipeline_run
	          WHERE status = 'running'
	          ORDER BY started_at DESC`

	return s.queryRuns(query)
}

// ListRuns returns runs matching the specified options.
func (s *stateStore) ListRuns(opts ListRunsOptions) ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json
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
	// Filter by tags - run must have at least one of the specified tags
	if len(opts.Tags) > 0 {
		// Use SQLite's json_each to search within tags_json array
		query += " AND ("
		for i, tag := range opts.Tags {
			if i > 0 {
				query += " OR "
			}
			query += "EXISTS (SELECT 1 FROM json_each(tags_json) WHERE json_each.value = ?)"
			args = append(args, tag)
		}
		query += ")"
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
		var input, currentStep, errorMessage, tagsJSON sql.NullString

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
			&tagsJSON,
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
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &record.Tags); err != nil {
				// If JSON parsing fails, treat as empty tags
				record.Tags = []string{}
			}
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating runs: %w", err)
	}

	return records, nil
}

// RecordPerformanceMetric records a performance metric for a step.
func (s *stateStore) RecordPerformanceMetric(metric *PerformanceMetricRecord) error {
	startedAt := metric.StartedAt.Unix()
	var completedAt *int64
	if metric.CompletedAt != nil {
		ca := metric.CompletedAt.Unix()
		completedAt = &ca
	}

	query := `INSERT INTO performance_metric (
	              run_id, step_id, pipeline_name, persona, started_at, completed_at,
	              duration_ms, tokens_used, files_modified, artifacts_generated,
	              memory_bytes, success, error_message
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.Exec(
		query,
		metric.RunID,
		metric.StepID,
		metric.PipelineName,
		metric.Persona,
		startedAt,
		completedAt,
		metric.DurationMs,
		metric.TokensUsed,
		metric.FilesModified,
		metric.ArtifactsGenerated,
		metric.MemoryBytes,
		metric.Success,
		metric.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to record performance metric: %w", err)
	}

	// Set the ID on the metric
	if id, err := result.LastInsertId(); err == nil {
		metric.ID = id
	}

	return nil
}

// GetPerformanceMetrics retrieves performance metrics for a run, optionally filtered by step.
func (s *stateStore) GetPerformanceMetrics(runID string, stepID string) ([]PerformanceMetricRecord, error) {
	query := `SELECT id, run_id, step_id, pipeline_name, persona, started_at, completed_at,
	                 duration_ms, tokens_used, files_modified, artifacts_generated,
	                 memory_bytes, success, error_message
	          FROM performance_metric
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY started_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance metrics: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetricRecord
	for rows.Next() {
		var metric PerformanceMetricRecord
		var startedAt int64
		var completedAt sql.NullInt64
		var persona, errorMessage sql.NullString
		var tokensUsed, filesModified, artifactsGenerated sql.NullInt64
		var memoryBytes, durationMs sql.NullInt64

		err := rows.Scan(
			&metric.ID,
			&metric.RunID,
			&metric.StepID,
			&metric.PipelineName,
			&persona,
			&startedAt,
			&completedAt,
			&durationMs,
			&tokensUsed,
			&filesModified,
			&artifactsGenerated,
			&memoryBytes,
			&metric.Success,
			&errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
		}

		metric.StartedAt = time.Unix(startedAt, 0)
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			metric.CompletedAt = &t
		}
		if persona.Valid {
			metric.Persona = persona.String
		}
		if durationMs.Valid {
			metric.DurationMs = durationMs.Int64
		}
		if tokensUsed.Valid {
			metric.TokensUsed = int(tokensUsed.Int64)
		}
		if filesModified.Valid {
			metric.FilesModified = int(filesModified.Int64)
		}
		if artifactsGenerated.Valid {
			metric.ArtifactsGenerated = int(artifactsGenerated.Int64)
		}
		if memoryBytes.Valid {
			metric.MemoryBytes = memoryBytes.Int64
		}
		if errorMessage.Valid {
			metric.ErrorMessage = errorMessage.String
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance metrics: %w", err)
	}

	return metrics, nil
}

// GetStepPerformanceStats retrieves aggregated performance statistics for a step.
func (s *stateStore) GetStepPerformanceStats(pipelineName string, stepID string, since time.Time) (*StepPerformanceStats, error) {
	query := `SELECT
	              COUNT(*) as total_runs,
	              SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_runs,
	              SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_runs,
	              AVG(duration_ms) as avg_duration,
	              MIN(duration_ms) as min_duration,
	              MAX(duration_ms) as max_duration,
	              AVG(tokens_used) as avg_tokens,
	              SUM(tokens_used) as total_tokens,
	              AVG(files_modified) as avg_files,
	              AVG(artifacts_generated) as avg_artifacts,
	              MAX(started_at) as last_run,
	              persona
	          FROM performance_metric
	          WHERE pipeline_name = ? AND step_id = ? AND started_at >= ?
	          GROUP BY step_id, persona`

	var stats StepPerformanceStats
	var lastRun int64
	var avgDuration, avgTokens, avgFiles, avgArtifacts sql.NullFloat64
	var minDuration, maxDuration, totalTokens sql.NullInt64
	var persona sql.NullString

	err := s.db.QueryRow(query, pipelineName, stepID, since.Unix()).Scan(
		&stats.TotalRuns,
		&stats.SuccessfulRuns,
		&stats.FailedRuns,
		&avgDuration,
		&minDuration,
		&maxDuration,
		&avgTokens,
		&totalTokens,
		&avgFiles,
		&avgArtifacts,
		&lastRun,
		&persona,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// No metrics found - return empty stats
			return &StepPerformanceStats{
				StepID: stepID,
			}, nil
		}
		return nil, fmt.Errorf("failed to get step performance stats: %w", err)
	}

	stats.StepID = stepID
	if persona.Valid {
		stats.Persona = persona.String
	}
	if avgDuration.Valid {
		stats.AvgDurationMs = int64(avgDuration.Float64)
	}
	if minDuration.Valid {
		stats.MinDurationMs = minDuration.Int64
	}
	if maxDuration.Valid {
		stats.MaxDurationMs = maxDuration.Int64
	}
	if avgTokens.Valid {
		stats.AvgTokensUsed = int(avgTokens.Float64)
	}
	if totalTokens.Valid {
		stats.TotalTokensUsed = int(totalTokens.Int64)
	}
	if avgFiles.Valid {
		stats.AvgFilesModified = int(avgFiles.Float64)
	}
	if avgArtifacts.Valid {
		stats.AvgArtifacts = int(avgArtifacts.Float64)
	}
	stats.LastRunAt = time.Unix(lastRun, 0)

	// Calculate token burn rate (tokens per second)
	if stats.AvgDurationMs > 0 && stats.AvgTokensUsed > 0 {
		stats.TokenBurnRate = float64(stats.AvgTokensUsed) / (float64(stats.AvgDurationMs) / 1000.0)
	}

	return &stats, nil
}

// GetRecentPerformanceHistory retrieves recent performance metrics with optional filters.
func (s *stateStore) GetRecentPerformanceHistory(opts PerformanceQueryOptions) ([]PerformanceMetricRecord, error) {
	query := `SELECT id, run_id, step_id, pipeline_name, persona, started_at, completed_at,
	                 duration_ms, tokens_used, files_modified, artifacts_generated,
	                 memory_bytes, success, error_message
	          FROM performance_metric
	          WHERE 1=1`
	args := []any{}

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.Persona != "" {
		query += " AND persona = ?"
		args = append(args, opts.Persona)
	}
	if !opts.Since.IsZero() {
		query += " AND started_at >= ?"
		args = append(args, opts.Since.Unix())
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance history: %w", err)
	}
	defer rows.Close()

	var metrics []PerformanceMetricRecord
	for rows.Next() {
		var metric PerformanceMetricRecord
		var startedAt int64
		var completedAt sql.NullInt64
		var persona, errorMessage sql.NullString
		var tokensUsed, filesModified, artifactsGenerated sql.NullInt64
		var memoryBytes, durationMs sql.NullInt64

		err := rows.Scan(
			&metric.ID,
			&metric.RunID,
			&metric.StepID,
			&metric.PipelineName,
			&persona,
			&startedAt,
			&completedAt,
			&durationMs,
			&tokensUsed,
			&filesModified,
			&artifactsGenerated,
			&memoryBytes,
			&metric.Success,
			&errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performance metric: %w", err)
		}

		metric.StartedAt = time.Unix(startedAt, 0)
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			metric.CompletedAt = &t
		}
		if persona.Valid {
			metric.Persona = persona.String
		}
		if durationMs.Valid {
			metric.DurationMs = durationMs.Int64
		}
		if tokensUsed.Valid {
			metric.TokensUsed = int(tokensUsed.Int64)
		}
		if filesModified.Valid {
			metric.FilesModified = int(filesModified.Int64)
		}
		if artifactsGenerated.Valid {
			metric.ArtifactsGenerated = int(artifactsGenerated.Int64)
		}
		if memoryBytes.Valid {
			metric.MemoryBytes = memoryBytes.Int64
		}
		if errorMessage.Valid {
			metric.ErrorMessage = errorMessage.String
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance history: %w", err)
	}

	return metrics, nil
}

// CleanupOldPerformanceMetrics removes performance metrics older than the specified duration.
// Returns the number of metrics deleted.
func (s *stateStore) CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Unix()

	query := `DELETE FROM performance_metric WHERE started_at < ?`

	result, err := s.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old performance metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// =============================================================================
// Progress Tracking Methods (spec 018 - Enhanced Progress Visualization)
// =============================================================================

// SaveProgressSnapshot records a point-in-time progress snapshot.
func (s *stateStore) SaveProgressSnapshot(runID string, stepID string, progress int, action string, etaMs int64, validationPhase string, compactionStats string) error {
	now := time.Now().Unix()

	query := `INSERT INTO progress_snapshot (
	              run_id, step_id, timestamp, progress, current_action,
	              estimated_time_ms, validation_phase, compaction_stats
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, stepID, now, progress, action, etaMs, validationPhase, compactionStats)
	if err != nil {
		return fmt.Errorf("failed to save progress snapshot: %w", err)
	}

	return nil
}

// GetProgressSnapshots retrieves progress snapshots for a run/step.
func (s *stateStore) GetProgressSnapshots(runID string, stepID string, limit int) ([]ProgressSnapshotRecord, error) {
	query := `SELECT id, run_id, step_id, timestamp, progress, current_action,
	                 estimated_time_ms, validation_phase, compaction_stats
	          FROM progress_snapshot
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress snapshots: %w", err)
	}
	defer rows.Close()

	var records []ProgressSnapshotRecord
	for rows.Next() {
		var record ProgressSnapshotRecord
		var timestamp int64
		var currentAction, validationPhase, compactionStats sql.NullString
		var estimatedTimeMs sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&record.StepID,
			&timestamp,
			&record.Progress,
			&currentAction,
			&estimatedTimeMs,
			&validationPhase,
			&compactionStats,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress snapshot: %w", err)
		}

		record.Timestamp = time.Unix(timestamp, 0)
		if currentAction.Valid {
			record.CurrentAction = currentAction.String
		}
		if estimatedTimeMs.Valid {
			record.EstimatedTimeMs = estimatedTimeMs.Int64
		}
		if validationPhase.Valid {
			record.ValidationPhase = validationPhase.String
		}
		if compactionStats.Valid {
			record.CompactionStats = compactionStats.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating progress snapshots: %w", err)
	}

	return records, nil
}

// UpdateStepProgress updates or creates a step progress record.
func (s *stateStore) UpdateStepProgress(runID string, stepID string, persona string, state string, progress int, action string, message string, etaMs int64, tokens int) error {
	now := time.Now().Unix()

	query := `INSERT INTO step_progress (
	              step_id, run_id, persona, state, progress, current_action,
	              message, started_at, updated_at, estimated_completion_ms, tokens_used
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(step_id) DO UPDATE SET
	              persona = excluded.persona,
	              state = excluded.state,
	              progress = excluded.progress,
	              current_action = excluded.current_action,
	              message = excluded.message,
	              updated_at = excluded.updated_at,
	              estimated_completion_ms = excluded.estimated_completion_ms,
	              tokens_used = excluded.tokens_used`

	_, err := s.db.Exec(query, stepID, runID, persona, state, progress, action, message, now, now, etaMs, tokens)
	if err != nil {
		return fmt.Errorf("failed to update step progress: %w", err)
	}

	return nil
}

// GetStepProgress retrieves the current progress for a specific step.
func (s *stateStore) GetStepProgress(stepID string) (*StepProgressRecord, error) {
	query := `SELECT step_id, run_id, persona, state, progress, current_action,
	                 message, started_at, updated_at, estimated_completion_ms, tokens_used
	          FROM step_progress
	          WHERE step_id = ?`

	var record StepProgressRecord
	var persona, currentAction, message sql.NullString
	var startedAt, updatedAt int64
	var estimatedCompletionMs sql.NullInt64

	err := s.db.QueryRow(query, stepID).Scan(
		&record.StepID,
		&record.RunID,
		&persona,
		&record.State,
		&record.Progress,
		&currentAction,
		&message,
		&startedAt,
		&updatedAt,
		&estimatedCompletionMs,
		&record.TokensUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("step progress not found: %s", stepID)
		}
		return nil, fmt.Errorf("failed to get step progress: %w", err)
	}

	if persona.Valid {
		record.Persona = persona.String
	}
	if currentAction.Valid {
		record.CurrentAction = currentAction.String
	}
	if message.Valid {
		record.Message = message.String
	}
	if startedAt > 0 {
		t := time.Unix(startedAt, 0)
		record.StartedAt = &t
	}
	record.UpdatedAt = time.Unix(updatedAt, 0)
	if estimatedCompletionMs.Valid {
		record.EstimatedCompletionMs = estimatedCompletionMs.Int64
	}

	return &record, nil
}

// GetAllStepProgress retrieves progress for all steps in a run.
func (s *stateStore) GetAllStepProgress(runID string) ([]StepProgressRecord, error) {
	query := `SELECT step_id, run_id, persona, state, progress, current_action,
	                 message, started_at, updated_at, estimated_completion_ms, tokens_used
	          FROM step_progress
	          WHERE run_id = ?
	          ORDER BY updated_at ASC`

	rows, err := s.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query step progress: %w", err)
	}
	defer rows.Close()

	var records []StepProgressRecord
	for rows.Next() {
		var record StepProgressRecord
		var persona, currentAction, message sql.NullString
		var startedAt, updatedAt int64
		var estimatedCompletionMs sql.NullInt64

		err := rows.Scan(
			&record.StepID,
			&record.RunID,
			&persona,
			&record.State,
			&record.Progress,
			&currentAction,
			&message,
			&startedAt,
			&updatedAt,
			&estimatedCompletionMs,
			&record.TokensUsed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step progress: %w", err)
		}

		if persona.Valid {
			record.Persona = persona.String
		}
		if currentAction.Valid {
			record.CurrentAction = currentAction.String
		}
		if message.Valid {
			record.Message = message.String
		}
		if startedAt > 0 {
			t := time.Unix(startedAt, 0)
			record.StartedAt = &t
		}
		record.UpdatedAt = time.Unix(updatedAt, 0)
		if estimatedCompletionMs.Valid {
			record.EstimatedCompletionMs = estimatedCompletionMs.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step progress: %w", err)
	}

	return records, nil
}

// UpdatePipelineProgress updates pipeline-level progress aggregation.
func (s *stateStore) UpdatePipelineProgress(runID string, totalSteps int, completedSteps int, currentStepIndex int, overallProgress int, etaMs int64) error {
	now := time.Now().Unix()

	query := `INSERT INTO pipeline_progress (
	              run_id, total_steps, completed_steps, current_step_index,
	              overall_progress, estimated_completion_ms, updated_at
	          ) VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(run_id) DO UPDATE SET
	              total_steps = excluded.total_steps,
	              completed_steps = excluded.completed_steps,
	              current_step_index = excluded.current_step_index,
	              overall_progress = excluded.overall_progress,
	              estimated_completion_ms = excluded.estimated_completion_ms,
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, runID, totalSteps, completedSteps, currentStepIndex, overallProgress, etaMs, now)
	if err != nil {
		return fmt.Errorf("failed to update pipeline progress: %w", err)
	}

	return nil
}

// GetPipelineProgress retrieves pipeline-level progress.
func (s *stateStore) GetPipelineProgress(runID string) (*PipelineProgressRecord, error) {
	query := `SELECT run_id, total_steps, completed_steps, current_step_index,
	                 overall_progress, estimated_completion_ms, updated_at
	          FROM pipeline_progress
	          WHERE run_id = ?`

	var record PipelineProgressRecord
	var updatedAt int64
	var estimatedCompletionMs sql.NullInt64

	err := s.db.QueryRow(query, runID).Scan(
		&record.RunID,
		&record.TotalSteps,
		&record.CompletedSteps,
		&record.CurrentStepIndex,
		&record.OverallProgress,
		&estimatedCompletionMs,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pipeline progress not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get pipeline progress: %w", err)
	}

	record.UpdatedAt = time.Unix(updatedAt, 0)
	if estimatedCompletionMs.Valid {
		record.EstimatedCompletionMs = estimatedCompletionMs.Int64
	}

	return &record, nil
}

// SaveArtifactMetadata saves extended metadata for an artifact.
func (s *stateStore) SaveArtifactMetadata(artifactID int64, runID string, stepID string, previewText string, mimeType string, encoding string, metadataJSON string) error {
	now := time.Now().Unix()

	query := `INSERT INTO artifact_metadata (
	              artifact_id, run_id, step_id, preview_text, mime_type,
	              encoding, metadata_json, indexed_at
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(artifact_id) DO UPDATE SET
	              preview_text = excluded.preview_text,
	              mime_type = excluded.mime_type,
	              encoding = excluded.encoding,
	              metadata_json = excluded.metadata_json,
	              indexed_at = excluded.indexed_at`

	_, err := s.db.Exec(query, artifactID, runID, stepID, previewText, mimeType, encoding, metadataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to save artifact metadata: %w", err)
	}

	return nil
}

// GetArtifactMetadata retrieves extended metadata for an artifact.
func (s *stateStore) GetArtifactMetadata(artifactID int64) (*ArtifactMetadataRecord, error) {
	query := `SELECT artifact_id, run_id, step_id, preview_text, mime_type,
	                 encoding, metadata_json, indexed_at
	          FROM artifact_metadata
	          WHERE artifact_id = ?`

	var record ArtifactMetadataRecord
	var indexedAt int64
	var previewText, mimeType, encoding, metadataJSON sql.NullString

	err := s.db.QueryRow(query, artifactID).Scan(
		&record.ArtifactID,
		&record.RunID,
		&record.StepID,
		&previewText,
		&mimeType,
		&encoding,
		&metadataJSON,
		&indexedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact metadata not found: %d", artifactID)
		}
		return nil, fmt.Errorf("failed to get artifact metadata: %w", err)
	}

	if previewText.Valid {
		record.PreviewText = previewText.String
	}
	if mimeType.Valid {
		record.MimeType = mimeType.String
	}
	if encoding.Valid {
		record.Encoding = encoding.String
	}
	if metadataJSON.Valid {
		record.MetadataJSON = metadataJSON.String
	}
	record.IndexedAt = time.Unix(indexedAt, 0)

	return &record, nil
}

// =============================================================================
// Tags Support Methods
// =============================================================================

// SetRunTags sets the tags for a pipeline run, replacing any existing tags.
func (s *stateStore) SetRunTags(runID string, tags []string) error {
	// Ensure tags is not nil for JSON encoding
	if tags == nil {
		tags = []string{}
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `UPDATE pipeline_run SET tags_json = ? WHERE run_id = ?`

	result, err := s.db.Exec(query, string(tagsJSON), runID)
	if err != nil {
		return fmt.Errorf("failed to set run tags: %w", err)
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

// GetRunTags retrieves the tags for a pipeline run.
func (s *stateStore) GetRunTags(runID string) ([]string, error) {
	query := `SELECT tags_json FROM pipeline_run WHERE run_id = ?`

	var tagsJSON sql.NullString
	err := s.db.QueryRow(query, runID).Scan(&tagsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run tags: %w", err)
	}

	if !tagsJSON.Valid || tagsJSON.String == "" {
		return []string{}, nil
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
		return []string{}, nil
	}

	return tags, nil
}

// AddRunTag adds a tag to a pipeline run if it doesn't already exist.
func (s *stateStore) AddRunTag(runID string, tag string) error {
	// Get current tags
	tags, err := s.GetRunTags(runID)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, existingTag := range tags {
		if existingTag == tag {
			return nil // Tag already exists, nothing to do
		}
	}

	// Add the new tag
	tags = append(tags, tag)

	return s.SetRunTags(runID, tags)
}

// RemoveRunTag removes a tag from a pipeline run.
func (s *stateStore) RemoveRunTag(runID string, tag string) error {
	// Get current tags
	tags, err := s.GetRunTags(runID)
	if err != nil {
		return err
	}

	// Filter out the tag to remove
	newTags := make([]string, 0, len(tags))
	for _, existingTag := range tags {
		if existingTag != tag {
			newTags = append(newTags, existingTag)
		}
	}

	return s.SetRunTags(runID, newTags)
}
