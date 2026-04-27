package state

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// StepState represents the state of a pipeline step.
type StepState string

const (
	StatePending        StepState = "pending"
	StateRunning        StepState = "running"
	StateCompleted      StepState = "completed"
	StateCompletedEmpty StepState = "completed_empty" // Step completed but produced no meaningful changes (zero diff in worktree)
	StateFailed         StepState = "failed"
	StateRetrying       StepState = "retrying"
	StateSkipped        StepState = "skipped"
	StateReworking      StepState = "reworking"
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
	VisitCount    int
}

// StateStore is the aggregate persistence surface combining every
// domain-scoped store. New consumers should depend on the smallest narrow
// interface that satisfies their call sites (RunStore, EventStore,
// OntologyStore, WebhookStore, ChatStore). The aggregate is retained for
// constructors and root-level orchestrators that span multiple domains.
type StateStore interface {
	RunStore
	EventStore
	OntologyStore
	WebhookStore
	ChatStore

	Close() error

	// Run tracking (ops commands)
	CreateRun(pipelineName string, input string) (string, error)
	CreateRunWithLimit(pipelineName string, input string, maxConcurrent int) (string, error)
	UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
	UpdateRunBranch(runID string, branch string) error
	GetRun(runID string) (*RunRecord, error)
	GetRunningRuns() ([]RunRecord, error)
	ListRuns(opts ListRunsOptions) ([]RunRecord, error)
	DeleteRun(runID string) error
	GetMostRecentRunID() (string, error)
	RunExists(runID string) (bool, error)
	GetRunStatus(runID string) (string, error)
	ListPipelineNamesByStatus(status string) ([]string, error)
	BackfillRunTokens() (int64, error)

	// Event logging
	LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64, model string, configuredModel string, adapter string) error
	GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error)
	GetEventAggregateStats(runID string) (*EventAggregateStats, error)

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

	// Process tracking (detached subprocess execution)
	UpdateRunPID(runID string, pid int) error

	// Step attempt tracking (retry/recovery)
	RecordStepAttempt(record *StepAttemptRecord) error
	GetStepAttempts(runID string, stepID string) ([]StepAttemptRecord, error)

	// Chat session tracking (bidirectional chat)
	SaveChatSession(session *ChatSession) error
	GetChatSession(sessionID string) (*ChatSession, error)
	ListChatSessions(runID string) ([]ChatSession, error)

	// Ontology usage tracking (decision lineage)
	RecordOntologyUsage(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error
	GetOntologyStats(contextName string) (*OntologyStats, error)
	GetOntologyStatsAll() ([]OntologyStats, error)

	// Checkpoint tracking (fork/rewind)
	SaveCheckpoint(record *CheckpointRecord) error
	GetCheckpoint(runID, stepID string) (*CheckpointRecord, error)
	GetCheckpoints(runID string) ([]CheckpointRecord, error)
	DeleteCheckpointsAfterStep(runID string, stepIndex int) error

	// Fork lineage
	CreateRunWithFork(pipelineName, input, forkedFromRunID string) (string, error)

	// Parent-child run linkage
	SetParentRun(childRunID, parentRunID, stepID string) error
	GetChildRuns(parentRunID string) ([]RunRecord, error)

	// Retrospective tracking
	SaveRetrospective(record *RetrospectiveRecord) error
	GetRetrospective(runID string) (*RetrospectiveRecord, error)
	ListRetrospectives(opts ListRetrosOptions) ([]RetrospectiveRecord, error)
	DeleteRetrospective(runID string) error
	UpdateRetrospectiveSmoothness(runID string, smoothness string) error
	UpdateRetrospectiveStatus(runID string, status string) error

	// Decision log (append-only structured decision tracking)
	RecordDecision(record *DecisionRecord) error
	GetDecisions(runID string) ([]*DecisionRecord, error)
	GetDecisionsByStep(runID, stepID string) ([]*DecisionRecord, error)
	GetDecisionsFiltered(runID string, opts DecisionQueryOptions) ([]*DecisionRecord, error)

	// Audit log (cross-run event queries)
	GetAuditEvents(states []string, limit, offset int) ([]LogRecord, error)

	// Webhook management
	CreateWebhook(webhook *Webhook) (int64, error)
	ListWebhooks() ([]*Webhook, error)
	GetWebhook(id int64) (*Webhook, error)
	UpdateWebhook(webhook *Webhook) error
	DeleteWebhook(id int64) error
	RecordWebhookDelivery(delivery *WebhookDelivery) error
	GetWebhookDeliveries(webhookID int64, limit int) ([]*WebhookDelivery, error)

	// Pipeline outcome persistence (survives worktree cleanup)
	RecordOutcome(runID, stepID, outcomeType, label, value string) error
	GetOutcomes(runID string) ([]OutcomeRecord, error)
	GetOutcomesByValue(outcomeType, value string) ([]OutcomeRecord, error)

	// Orchestration decision tracking (task classification feedback loop)
	RecordOrchestrationDecision(record *OrchestrationDecision) error
	UpdateOrchestrationOutcome(runID string, outcome string, tokensUsed int, durationMs int64) error
	GetOrchestrationStats(pipelineName string) (*OrchestrationStats, error)
	ListOrchestrationDecisionSummary(limit int) ([]OrchestrationDecisionSummary, error)
}

// ListRetrosOptions specifies filters for listing retrospectives.
type ListRetrosOptions struct {
	PipelineName string
	SinceUnix    int64
	Limit        int
}

// WaitForConcurrencySlot polls GetRunningRuns until fewer than maxWorkers
// pipelines are running. Returns nil when a slot is available, or ctx.Err()
// if the context is cancelled. This is the single concurrency gate used by
// CLI --detach, WebUI, and TUI launch paths.
func WaitForConcurrencySlot(ctx context.Context, store StateStore, maxWorkers int, onWait func(running, max int)) error {
	for {
		running, err := store.GetRunningRuns()
		if err != nil {
			return nil // can't check, proceed optimistically
		}
		if len(running) < maxWorkers {
			return nil
		}
		if onWait != nil {
			onWait(len(running), maxWorkers)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

type stateStore struct {
	db    *sql.DB
	clock func() time.Time
}

func (s *stateStore) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
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
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for concurrent access
	// Enable WAL mode for better concurrent read/write performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Set busy timeout to 5 seconds to handle lock contention
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign key enforcement (disabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Load migration configuration from environment
	migrationConfig := LoadMigrationConfigFromEnv()

	if err := migrationConfig.Validate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("invalid migration configuration: %w", err)
	}

	if err := initializeWithMigrations(db, migrationConfig); err != nil {
		db.Close()
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
	now := s.now().Unix()

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
	          ON CONFLICT(step_id, pipeline_id) DO UPDATE SET
	              state = excluded.state,
	              retry_count = CASE WHEN excluded.state = 'retrying' THEN retry_count + 1 ELSE retry_count END,
	              started_at = COALESCE(started_at, excluded.started_at),
	              completed_at = excluded.completed_at,
	              error_message = excluded.error_message`

	var startedAt, completedAt *int64
	if state == StateRunning || state == StateRetrying {
		startedAt = &now
	}
	if state == StateCompleted || state == StateCompletedEmpty || state == StateFailed {
		completedAt = &now
	}

	_, execErr := s.db.Exec(query, stepID, pipelineID, string(state), startedAt, completedAt, errMsg)
	if execErr != nil {
		return fmt.Errorf("failed to save step state: %w", execErr)
	}

	return nil
}

// SaveStepVisitCount updates the visit count for a step in graph-mode pipelines.
func (s *stateStore) SaveStepVisitCount(pipelineID string, stepID string, count int) error {
	query := `UPDATE step_state SET visit_count = ? WHERE step_id = ? AND pipeline_id = ?`
	result, err := s.db.Exec(query, count, stepID, pipelineID)
	if err != nil {
		return fmt.Errorf("failed to save step visit count: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		// Step state doesn't exist yet — insert it with the visit count
		insertQuery := `INSERT INTO step_state (step_id, pipeline_id, state, retry_count, visit_count)
		                VALUES (?, ?, 'pending', 0, ?)`
		_, err := s.db.Exec(insertQuery, stepID, pipelineID, count)
		if err != nil {
			return fmt.Errorf("failed to insert step visit count: %w", err)
		}
	}
	return nil
}

// GetStepVisitCount retrieves the visit count for a step in graph-mode pipelines.
func (s *stateStore) GetStepVisitCount(pipelineID string, stepID string) (int, error) {
	query := `SELECT visit_count FROM step_state WHERE step_id = ? AND pipeline_id = ?`
	var count int
	err := s.db.QueryRow(query, stepID, pipelineID).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get step visit count: %w", err)
	}
	return count, nil
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
	query := `SELECT step_id, pipeline_id, state, retry_count, started_at, completed_at, workspace_path, error_message, visit_count
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
			&record.VisitCount,
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
	return s.CreateRunWithLimit(pipelineName, input, 0)
}

// CreateRunWithLimit creates a new run, atomically enforcing a concurrency limit.
// If maxConcurrent > 0, the INSERT is rejected when the limit is reached.
// Returns ErrConcurrencyLimit when the limit is hit.
func (s *stateStore) CreateRunWithLimit(pipelineName string, input string, maxConcurrent int) (string, error) {
	now := s.now()
	randBytes := make([]byte, 2)
	if _, err := rand.Read(randBytes); err != nil {
		randBytes = []byte{byte(now.Nanosecond() >> 8), byte(now.Nanosecond())}
	}
	suffix := hex.EncodeToString(randBytes)
	runID := fmt.Sprintf("%s-%s-%s", pipelineName, now.Format("20060102-150405"), suffix)

	if maxConcurrent > 0 {
		// Atomic check-and-insert within a transaction
		tx, err := s.db.Begin()
		if err != nil {
			return "", fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		var count int
		err = tx.QueryRow(`SELECT COUNT(*) FROM pipeline_run WHERE status IN ('running', 'pending') AND started_at > unixepoch() - 300`).Scan(&count)
		if err != nil {
			return "", fmt.Errorf("failed to count running runs: %w", err)
		}
		if count >= maxConcurrent {
			return "", ErrConcurrencyLimit
		}

		_, err = tx.Exec(`INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at)
		                   VALUES (?, ?, 'pending', ?, ?)`, runID, pipelineName, input, now.Unix())
		if err != nil {
			return "", fmt.Errorf("failed to create run: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("failed to commit run: %w", err)
		}
		return runID, nil
	}

	// No limit — simple insert
	_, err := s.db.Exec(`INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at)
	                      VALUES (?, ?, 'pending', ?, ?)`, runID, pipelineName, input, now.Unix())
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}
	return runID, nil
}

// ErrConcurrencyLimit is returned when max_concurrent_workers is reached.
var ErrConcurrencyLimit = fmt.Errorf("concurrency limit reached")

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

// UpdateRunBranch updates the branch_name for a pipeline run.
func (s *stateStore) UpdateRunBranch(runID string, branch string) error {
	query := `UPDATE pipeline_run SET branch_name = ? WHERE run_id = ?`
	result, err := s.db.Exec(query, branch, runID)
	if err != nil {
		return fmt.Errorf("failed to update run branch: %w", err)
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

// UpdateRunPID sets the OS process ID for a detached pipeline run.
func (s *stateStore) UpdateRunPID(runID string, pid int) error {
	query := `UPDATE pipeline_run SET pid = ? WHERE run_id = ?`
	_, err := s.db.Exec(query, pid, runID)
	if err != nil {
		return fmt.Errorf("failed to update run PID: %w", err)
	}
	return nil
}

// GetRun retrieves a single run record by ID.
func (s *stateStore) GetRun(runID string) (*RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id
	          FROM pipeline_run
	          WHERE run_id = ?`

	var record RunRecord
	var startedAt int64
	var completedAt, cancelledAt sql.NullInt64
	var input, currentStep, errorMessage, tagsJSON, branchName sql.NullString
	var pid sql.NullInt64
	var parentRunID, parentStepID, forkedFromRunID sql.NullString

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
		&branchName,
		&pid,
		&parentRunID,
		&parentStepID,
		&forkedFromRunID,
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
	if branchName.Valid {
		record.BranchName = branchName.String
	}
	if pid.Valid {
		record.PID = int(pid.Int64)
	}
	if parentRunID.Valid {
		record.ParentRunID = parentRunID.String
	}
	if parentStepID.Valid {
		record.ParentStepID = parentStepID.String
	}
	if forkedFromRunID.Valid {
		record.ForkedFromRunID = forkedFromRunID.String
	}

	return &record, nil
}

// GetRunningRuns returns all runs with status 'running'.
func (s *stateStore) GetRunningRuns() ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id
	          FROM pipeline_run
	          WHERE (status = 'running' OR (status = 'pending' AND started_at > unixepoch() - 300))
	          ORDER BY started_at DESC`

	return s.queryRuns(query)
}

// ListRuns returns runs matching the specified options.
func (s *stateStore) ListRuns(opts ListRunsOptions) ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id
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
		cutoff := s.now().Add(-opts.OlderThan).Unix()
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

	if opts.SinceUnix > 0 {
		query += " AND started_at >= ?"
		args = append(args, opts.SinceUnix)
	}

	// Cursor-based pagination: return runs before the cursor position
	if opts.BeforeUnix > 0 {
		if opts.BeforeRunID != "" {
			query += " AND (started_at < ? OR (started_at = ? AND run_id < ?))"
			args = append(args, opts.BeforeUnix, opts.BeforeUnix, opts.BeforeRunID)
		} else {
			query += " AND started_at < ?"
			args = append(args, opts.BeforeUnix)
		}
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	return s.queryRunsWithArgs(query, args...)
}

// GetMostRecentRunID returns the run_id with the most recent started_at.
// Returns ("", nil) when no runs exist so callers can switch on empty string
// without depending on database/sql sentinel errors.
func (s *stateStore) GetMostRecentRunID() (string, error) {
	var runID string
	err := s.db.QueryRow(
		`SELECT run_id FROM pipeline_run ORDER BY started_at DESC, run_id DESC LIMIT 1`,
	).Scan(&runID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query most recent run: %w", err)
	}
	return runID, nil
}

// RunExists reports whether a run with the given ID exists.
func (s *stateStore) RunExists(runID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM pipeline_run WHERE run_id = ?`, runID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check run existence: %w", err)
	}
	return count > 0, nil
}

// GetRunStatus returns the status of a run.
// Returns ("", nil) when the run does not exist.
func (s *stateStore) GetRunStatus(runID string) (string, error) {
	var status string
	err := s.db.QueryRow(
		`SELECT status FROM pipeline_run WHERE run_id = ?`, runID,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query run status: %w", err)
	}
	return status, nil
}

// ListPipelineNamesByStatus returns distinct pipeline names whose status matches
// the given status (case-insensitive). Falls back to pipeline_state if pipeline_run
// query fails (legacy schema compatibility).
func (s *stateStore) ListPipelineNamesByStatus(status string) ([]string, error) {
	names, err := s.listDistinctPipelineNames(
		`SELECT DISTINCT pipeline_name FROM pipeline_run WHERE LOWER(status) = LOWER(?)`,
		status,
	)
	if err == nil {
		return names, nil
	}
	// Fallback for legacy/partial schemas
	return s.listDistinctPipelineNames(
		`SELECT DISTINCT pipeline_name FROM pipeline_state WHERE LOWER(status) = LOWER(?)`,
		status,
	)
}

func (s *stateStore) listDistinctPipelineNames(query, status string) ([]string, error) {
	rows, err := s.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query pipeline names by status: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pipeline names: %w", err)
	}
	return names, nil
}

// BackfillRunTokens updates pipeline_run.total_tokens from event_log for
// finalized runs that still have 0 tokens. Idempotent — re-running yields 0
// affected rows once all runs have been backfilled.
func (s *stateStore) BackfillRunTokens() (int64, error) {
	result, err := s.db.Exec(`
		UPDATE pipeline_run SET total_tokens = (
			SELECT COALESCE(SUM(el.tokens_used), 0)
			FROM event_log el
			WHERE el.run_id = pipeline_run.run_id AND el.tokens_used > 0
		)
		WHERE total_tokens = 0
		AND status IN ('completed', 'failed', 'cancelled')
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to backfill run tokens: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n, nil
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
func (s *stateStore) LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64, model string, configuredModel string, adapter string) error {
	now := s.now().Unix()

	query := `INSERT INTO event_log (run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms, model, configured_model, adapter)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, now, stepID, state, persona, message, tokens, durationMs, model, configuredModel, adapter)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return nil
}

// GetEvents retrieves events for a run with optional filtering.
//
// Ordering rules:
//   - TailLimit > 0: query runs in DESC order with LIMIT, results are reversed
//     before return so callers always see ASC order. Other ordering flags are
//     ignored in this mode.
//   - OrderDesc: timestamp DESC, id DESC.
//   - Default: timestamp ASC.
func (s *stateStore) GetEvents(runID string, opts EventQueryOptions) ([]LogRecord, error) {
	query := `SELECT id, run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms, model, configured_model, adapter
	          FROM event_log
	          WHERE run_id = ?`
	args := []any{runID}

	if opts.AfterID > 0 {
		query += " AND id > ?"
		args = append(args, opts.AfterID)
	}
	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.ErrorsOnly {
		query += " AND state = 'failed'"
	}
	if opts.SinceUnix > 0 {
		query += " AND timestamp >= ?"
		args = append(args, opts.SinceUnix)
	}

	tailMode := opts.TailLimit > 0
	switch {
	case tailMode:
		query += " ORDER BY timestamp DESC, id DESC LIMIT ?"
		args = append(args, opts.TailLimit)
	case opts.OrderDesc:
		query += " ORDER BY timestamp DESC, id DESC"
	default:
		query += " ORDER BY timestamp ASC, id ASC"
	}

	if !tailMode && opts.Limit > 0 {
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
		var stepID, persona, message, model, configuredModel, adapter sql.NullString
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
			&model,
			&configuredModel,
			&adapter,
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
		if model.Valid {
			record.Model = model.String
		}
		if configuredModel.Valid {
			record.ConfiguredModel = configuredModel.String
		}
		if adapter.Valid {
			record.Adapter = adapter.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	// Tail mode queries DESC for SQL-side LIMIT correctness; flip to ASC for callers.
	if tailMode && len(records) > 1 {
		for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
			records[i], records[j] = records[j], records[i]
		}
	}

	return records, nil
}

// GetEventAggregateStats returns aggregate metrics over event_log entries in
// terminal states (completed, failed) for the given run.
func (s *stateStore) GetEventAggregateStats(runID string) (*EventAggregateStats, error) {
	var stats EventAggregateStats
	var avg, minD, maxD sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT
		    COUNT(*),
		    COALESCE(SUM(COALESCE(tokens_used, 0)), 0),
		    AVG(COALESCE(duration_ms, 0)),
		    MIN(COALESCE(duration_ms, 0)),
		    MAX(COALESCE(duration_ms, 0))
		FROM event_log
		WHERE run_id = ? AND state IN ('completed', 'failed')
	`, runID).Scan(&stats.TotalEvents, &stats.TotalTokens, &avg, &minD, &maxD)
	if err != nil {
		return nil, fmt.Errorf("failed to query event aggregate stats: %w", err)
	}
	if avg.Valid {
		stats.AvgDurationMs = avg.Float64
	}
	if minD.Valid {
		stats.MinDurationMs = minD.Float64
	}
	if maxD.Valid {
		stats.MaxDurationMs = maxD.Float64
	}
	return &stats, nil
}

// GetAuditEvents retrieves events across all runs, filtered by state types,
// ordered by timestamp descending. Used by the admin audit log viewer.
func (s *stateStore) GetAuditEvents(states []string, limit, offset int) ([]LogRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT e.id, e.run_id, e.timestamp, e.step_id, e.state, e.persona, e.message, e.tokens_used, e.duration_ms
	          FROM event_log e`

	var args []any
	if len(states) > 0 {
		placeholders := make([]string, len(states))
		for i, st := range states {
			placeholders[i] = "?"
			args = append(args, st)
		}
		query += " WHERE e.state IN (" + strings.Join(placeholders, ",") + ")"
	}

	query += " ORDER BY e.timestamp DESC, e.id DESC"
	query += " LIMIT ?"
	args = append(args, limit)
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
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
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
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
		return nil, fmt.Errorf("error iterating audit events: %w", err)
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

	// Force cancel: directly mark the run and all its running steps as cancelled.
	// This handles orphaned runs whose process is no longer running.
	if force {
		if err := s.UpdateRunStatus(runID, "cancelled", "", 0); err != nil {
			return fmt.Errorf("failed to force-cancel run: %w", err)
		}
		// Also cancel all running/pending steps
		_, _ = s.db.Exec(`UPDATE step_state SET state = 'cancelled', completed_at = ? WHERE pipeline_id = ? AND state IN ('running', 'pending', 'started')`, now, runID)
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
		var input, currentStep, errorMessage, tagsJSON, branchName sql.NullString
		var pid sql.NullInt64
		var parentRunID, parentStepID, forkedFromRunID sql.NullString

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
			&branchName,
			&pid,
			&parentRunID,
			&parentStepID,
			&forkedFromRunID,
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
		if branchName.Valid {
			record.BranchName = branchName.String
		}
		if pid.Valid {
			record.PID = int(pid.Int64)
		}
		if parentRunID.Valid {
			record.ParentRunID = parentRunID.String
		}
		if parentStepID.Valid {
			record.ParentStepID = parentStepID.String
		}
		if forkedFromRunID.Valid {
			record.ForkedFromRunID = forkedFromRunID.String
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

// RecordStepAttempt inserts a step attempt record into the step_attempt table.
func (s *stateStore) RecordStepAttempt(record *StepAttemptRecord) error {
	var completedAt *int64
	if record.CompletedAt != nil {
		t := record.CompletedAt.Unix()
		completedAt = &t
	}
	_, err := s.db.Exec(
		`INSERT INTO step_attempt (run_id, step_id, attempt, state, error_message, failure_class, stdout_tail, tokens_used, duration_ms, started_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.StepID, record.Attempt, record.State, record.ErrorMessage, record.FailureClass, record.StdoutTail, record.TokensUsed, record.DurationMs, record.StartedAt.Unix(), completedAt,
	)
	return err
}

// GetStepAttempts retrieves all attempt records for a step, ordered by attempt number.
func (s *stateStore) GetStepAttempts(runID string, stepID string) ([]StepAttemptRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, attempt, state, error_message, failure_class, stdout_tail, tokens_used, duration_ms, started_at, completed_at FROM step_attempt WHERE run_id = ? AND step_id = ? ORDER BY attempt ASC`,
		runID, stepID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []StepAttemptRecord
	for rows.Next() {
		var r StepAttemptRecord
		var startedAt int64
		var completedAtNull *int64
		err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &r.Attempt, &r.State, &r.ErrorMessage, &r.FailureClass, &r.StdoutTail, &r.TokensUsed, &r.DurationMs, &startedAt, &completedAtNull)
		if err != nil {
			return nil, err
		}
		r.StartedAt = time.Unix(startedAt, 0)
		if completedAtNull != nil {
			t := time.Unix(*completedAtNull, 0)
			r.CompletedAt = &t
		}
		records = append(records, r)
	}
	return records, nil
}

// SaveChatSession persists a chat session record. If a session with the same ID
// already exists, it updates last_resumed_at.
func (s *stateStore) SaveChatSession(session *ChatSession) error {
	var lastResumedAt *int64
	if session.LastResumedAt != nil {
		t := session.LastResumedAt.Unix()
		lastResumedAt = &t
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_session (session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET last_resumed_at = excluded.last_resumed_at`,
		session.SessionID, session.RunID, session.StepFilter, session.WorkspacePath, session.Model, session.CreatedAt.Unix(), lastResumedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save chat session: %w", err)
	}
	return nil
}

// GetChatSession retrieves a chat session by its session ID.
func (s *stateStore) GetChatSession(sessionID string) (*ChatSession, error) {
	row := s.db.QueryRow(
		`SELECT session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at FROM chat_session WHERE session_id = ?`,
		sessionID,
	)

	var cs ChatSession
	var createdAt int64
	var lastResumedAt *int64
	err := row.Scan(&cs.SessionID, &cs.RunID, &cs.StepFilter, &cs.WorkspacePath, &cs.Model, &createdAt, &lastResumedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat session %s: %w", sessionID, err)
	}
	cs.CreatedAt = time.Unix(createdAt, 0)
	if lastResumedAt != nil {
		t := time.Unix(*lastResumedAt, 0)
		cs.LastResumedAt = &t
	}
	return &cs, nil
}

// ListChatSessions returns all chat sessions for a pipeline run, ordered by creation time descending.
func (s *stateStore) ListChatSessions(runID string) ([]ChatSession, error) {
	rows, err := s.db.Query(
		`SELECT session_id, run_id, step_filter, workspace_path, model, created_at, last_resumed_at FROM chat_session WHERE run_id = ? ORDER BY created_at DESC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list chat sessions for run %s: %w", runID, err)
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var cs ChatSession
		var createdAt int64
		var lastResumedAt *int64
		err := rows.Scan(&cs.SessionID, &cs.RunID, &cs.StepFilter, &cs.WorkspacePath, &cs.Model, &createdAt, &lastResumedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat session: %w", err)
		}
		cs.CreatedAt = time.Unix(createdAt, 0)
		if lastResumedAt != nil {
			t := time.Unix(*lastResumedAt, 0)
			cs.LastResumedAt = &t
		}
		sessions = append(sessions, cs)
	}
	return sessions, nil
}

// RecordOntologyUsage inserts an ontology usage record for decision lineage tracking.
func (s *stateStore) RecordOntologyUsage(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error {
	var cp *int
	if contractPassed != nil {
		v := 0
		if *contractPassed {
			v = 1
		}
		cp = &v
	}
	_, err := s.db.Exec(
		`INSERT INTO ontology_usage (run_id, step_id, context_name, invariant_count, step_status, contract_passed) VALUES (?, ?, ?, ?, ?, ?)`,
		runID, stepID, contextName, invariantCount, status, cp,
	)
	if err != nil {
		return fmt.Errorf("failed to record ontology usage: %w", err)
	}
	return nil
}

// GetOntologyStats returns aggregated statistics for a single ontology context.
func (s *stateStore) GetOntologyStats(contextName string) (*OntologyStats, error) {
	row := s.db.QueryRow(
		`SELECT context_name,
		        COUNT(*) as total_runs,
		        SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) as successes,
		        SUM(CASE WHEN step_status = 'failed' THEN 1 ELSE 0 END) as failures,
		        ROUND(100.0 * SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) / COUNT(*), 1) as success_rate,
		        MAX(created_at) as last_used
		 FROM ontology_usage
		 WHERE context_name = ?
		 GROUP BY context_name`,
		contextName,
	)

	var stats OntologyStats
	var lastUsed int64
	err := row.Scan(&stats.ContextName, &stats.TotalRuns, &stats.Successes, &stats.Failures, &stats.SuccessRate, &lastUsed)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return &OntologyStats{ContextName: contextName}, nil
		}
		return nil, fmt.Errorf("failed to get ontology stats for %s: %w", contextName, err)
	}
	stats.LastUsed = time.Unix(lastUsed, 0)
	return &stats, nil
}

// GetOntologyStatsAll returns aggregated statistics for all ontology contexts, sorted by total_runs DESC.
func (s *stateStore) GetOntologyStatsAll() ([]OntologyStats, error) {
	rows, err := s.db.Query(
		`SELECT context_name,
		        COUNT(*) as total_runs,
		        SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) as successes,
		        SUM(CASE WHEN step_status = 'failed' THEN 1 ELSE 0 END) as failures,
		        ROUND(100.0 * SUM(CASE WHEN step_status = 'success' THEN 1 ELSE 0 END) / COUNT(*), 1) as success_rate,
		        MAX(created_at) as last_used
		 FROM ontology_usage
		 GROUP BY context_name
		 ORDER BY total_runs DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all ontology stats: %w", err)
	}
	defer rows.Close()

	var allStats []OntologyStats
	for rows.Next() {
		var stats OntologyStats
		var lastUsed int64
		err := rows.Scan(&stats.ContextName, &stats.TotalRuns, &stats.Successes, &stats.Failures, &stats.SuccessRate, &lastUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ontology stats: %w", err)
		}
		stats.LastUsed = time.Unix(lastUsed, 0)
		allStats = append(allStats, stats)
	}
	return allStats, nil
}

// --- Checkpoint tracking (fork/rewind) ---

func (s *stateStore) SaveCheckpoint(record *CheckpointRecord) error {
	now := time.Now().Unix()

	// Upsert: replace existing checkpoint for same run+step
	query := `INSERT INTO checkpoint (run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(run_id, step_id) DO UPDATE SET
	              step_index = excluded.step_index,
	              workspace_path = excluded.workspace_path,
	              workspace_commit_sha = excluded.workspace_commit_sha,
	              artifact_snapshot = excluded.artifact_snapshot,
	              created_at = excluded.created_at`

	_, err := s.db.Exec(query, record.RunID, record.StepID, record.StepIndex, record.WorkspacePath, record.WorkspaceCommitSHA, record.ArtifactSnapshot, now)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

func (s *stateStore) GetCheckpoint(runID, stepID string) (*CheckpointRecord, error) {
	query := `SELECT id, run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at
	          FROM checkpoint
	          WHERE run_id = ? AND step_id = ?`

	var record CheckpointRecord
	var createdAt int64
	var sha sql.NullString

	err := s.db.QueryRow(query, runID, stepID).Scan(
		&record.ID, &record.RunID, &record.StepID, &record.StepIndex,
		&record.WorkspacePath, &sha, &record.ArtifactSnapshot, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("checkpoint not found for run %s step %s", runID, stepID)
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if sha.Valid {
		record.WorkspaceCommitSHA = sha.String
	}
	record.CreatedAt = time.Unix(createdAt, 0)
	return &record, nil
}

func (s *stateStore) GetCheckpoints(runID string) ([]CheckpointRecord, error) {
	query := `SELECT id, run_id, step_id, step_index, workspace_path, workspace_commit_sha, artifact_snapshot, created_at
	          FROM checkpoint
	          WHERE run_id = ?
	          ORDER BY step_index ASC`

	rows, err := s.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer rows.Close()

	var records []CheckpointRecord
	for rows.Next() {
		var record CheckpointRecord
		var createdAt int64
		var sha sql.NullString

		err := rows.Scan(
			&record.ID, &record.RunID, &record.StepID, &record.StepIndex,
			&record.WorkspacePath, &sha, &record.ArtifactSnapshot, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}

		if sha.Valid {
			record.WorkspaceCommitSHA = sha.String
		}
		record.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoints: %w", err)
	}
	return records, nil
}

func (s *stateStore) DeleteCheckpointsAfterStep(runID string, stepIndex int) error {
	query := `DELETE FROM checkpoint WHERE run_id = ? AND step_index > ?`
	_, err := s.db.Exec(query, runID, stepIndex)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoints after step index %d: %w", stepIndex, err)
	}
	return nil
}

func (s *stateStore) CreateRunWithFork(pipelineName, input, forkedFromRunID string) (string, error) {
	now := time.Now()
	randBytes := make([]byte, 2)
	if _, err := rand.Read(randBytes); err != nil {
		randBytes = []byte{byte(now.Nanosecond() >> 8), byte(now.Nanosecond())}
	}
	suffix := hex.EncodeToString(randBytes)
	runID := fmt.Sprintf("%s-%s-%s", pipelineName, now.Format("20060102-150405"), suffix)

	query := `INSERT INTO pipeline_run (run_id, pipeline_name, status, input, started_at, forked_from_run_id)
	          VALUES (?, ?, 'pending', ?, ?, ?)`

	_, err := s.db.Exec(query, runID, pipelineName, input, now.Unix(), forkedFromRunID)
	if err != nil {
		return "", fmt.Errorf("failed to create forked run: %w", err)
	}
	return runID, nil
}

// =============================================================================
// Parent-Child Run Linkage
// =============================================================================

// SetParentRun sets the parent run ID and step ID on a child run record.
func (s *stateStore) SetParentRun(childRunID, parentRunID, stepID string) error {
	query := `UPDATE pipeline_run SET parent_run_id = ?, parent_step_id = ? WHERE run_id = ?`

	result, err := s.db.Exec(query, parentRunID, stepID, childRunID)
	if err != nil {
		return fmt.Errorf("failed to set parent run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", childRunID)
	}

	return nil
}

// GetChildRuns returns all runs that are children of the specified parent run,
// ordered by started_at.
func (s *stateStore) GetChildRuns(parentRunID string) ([]RunRecord, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, cancelled_at, error_message, tags_json, branch_name, pid,
	                 parent_run_id, parent_step_id, forked_from_run_id
	          FROM pipeline_run
	          WHERE parent_run_id = ?
	          ORDER BY started_at ASC`

	return s.queryRunsWithArgs(query, parentRunID)
}

// SaveRetrospective saves a retrospective index record.
func (s *stateStore) SaveRetrospective(record *RetrospectiveRecord) error {
	// Check if exists first
	var exists int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM retrospective WHERE run_id = ?", record.RunID).Scan(&exists)
	if exists > 0 {
		_, err := s.db.Exec(`
			UPDATE retrospective SET smoothness = ?, status = ?, file_path = ?
			WHERE run_id = ?
		`, record.Smoothness, record.Status, record.FilePath, record.RunID)
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO retrospective (run_id, pipeline_name, smoothness, status, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, record.RunID, record.PipelineName, record.Smoothness, record.Status, record.FilePath, record.CreatedAt.Unix())
	return err
}

// GetRetrospective retrieves a retrospective record by run ID.
func (s *stateStore) GetRetrospective(runID string) (*RetrospectiveRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, run_id, pipeline_name, smoothness, status, file_path, created_at
		FROM retrospective WHERE run_id = ?
	`, runID)

	var r RetrospectiveRecord
	var createdAt int64
	err := row.Scan(&r.ID, &r.RunID, &r.PipelineName, &r.Smoothness, &r.Status, &r.FilePath, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("retrospective not found for run %s: %w", runID, err)
	}
	r.CreatedAt = time.Unix(createdAt, 0)
	return &r, nil
}

// ListRetrospectives returns retrospectives matching the given filters.
func (s *stateStore) ListRetrospectives(opts ListRetrosOptions) ([]RetrospectiveRecord, error) {
	query := "SELECT id, run_id, pipeline_name, smoothness, status, file_path, created_at FROM retrospective WHERE 1=1"
	var args []interface{}

	if opts.PipelineName != "" {
		query += " AND pipeline_name = ?"
		args = append(args, opts.PipelineName)
	}
	if opts.SinceUnix > 0 {
		query += " AND created_at >= ?"
		args = append(args, opts.SinceUnix)
	}
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list retrospectives: %w", err)
	}
	defer rows.Close()

	var records []RetrospectiveRecord
	for rows.Next() {
		var r RetrospectiveRecord
		var createdAt int64
		if err := rows.Scan(&r.ID, &r.RunID, &r.PipelineName, &r.Smoothness, &r.Status, &r.FilePath, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan retrospective: %w", err)
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, r)
	}
	return records, nil
}

// DeleteRetrospective removes a retrospective record by run ID.
func (s *stateStore) DeleteRetrospective(runID string) error {
	_, err := s.db.Exec("DELETE FROM retrospective WHERE run_id = ?", runID)
	return err
}

// UpdateRetrospectiveSmoothness updates the smoothness rating for a retrospective.
func (s *stateStore) UpdateRetrospectiveSmoothness(runID string, smoothness string) error {
	_, err := s.db.Exec("UPDATE retrospective SET smoothness = ? WHERE run_id = ?", smoothness, runID)
	return err
}

// UpdateRetrospectiveStatus updates the status for a retrospective.
func (s *stateStore) UpdateRetrospectiveStatus(runID string, status string) error {
	_, err := s.db.Exec("UPDATE retrospective SET status = ? WHERE run_id = ?", status, runID)
	return err
}

// RecordDecision appends a decision record to the decision log.
func (s *stateStore) RecordDecision(record *DecisionRecord) error {
	ts := record.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	contextJSON := record.Context
	if contextJSON == "" {
		contextJSON = "{}"
	}
	result, err := s.db.Exec(
		`INSERT INTO decision_log (run_id, step_id, timestamp, category, decision, rationale, context_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.StepID, ts.Unix(), record.Category, record.Decision, record.Rationale, contextJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to record decision: %w", err)
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

// GetDecisions returns all decision records for a run, ordered by timestamp.
func (s *stateStore) GetDecisions(runID string) ([]*DecisionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
		FROM decision_log WHERE run_id = ? ORDER BY timestamp ASC, id ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// GetDecisionsByStep returns decision records for a specific run and step.
func (s *stateStore) GetDecisionsByStep(runID, stepID string) ([]*DecisionRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
		FROM decision_log WHERE run_id = ? AND step_id = ? ORDER BY timestamp ASC, id ASC`,
		runID, stepID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions by step: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// GetDecisionsFiltered returns decision records for a run filtered by step
// and/or category. Empty filter values match all entries on that field.
func (s *stateStore) GetDecisionsFiltered(runID string, opts DecisionQueryOptions) ([]*DecisionRecord, error) {
	query := `SELECT id, run_id, step_id, timestamp, category, decision, rationale, context_json
	          FROM decision_log WHERE run_id = ?`
	args := []any{runID}

	if opts.StepID != "" {
		query += " AND step_id = ?"
		args = append(args, opts.StepID)
	}
	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}
	query += " ORDER BY timestamp ASC, id ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()
	return scanDecisionRecords(rows)
}

// scanDecisionRecords scans rows into DecisionRecord slices.
func scanDecisionRecords(rows *sql.Rows) ([]*DecisionRecord, error) {
	var records []*DecisionRecord
	for rows.Next() {
		var r DecisionRecord
		var ts int64
		err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &ts, &r.Category, &r.Decision, &r.Rationale, &r.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decision record: %w", err)
		}
		r.Timestamp = time.Unix(ts, 0)
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating decision records: %w", err)
	}
	return records, nil
}

// CreateWebhook inserts a new webhook and returns its ID.
func (s *stateStore) CreateWebhook(webhook *Webhook) (int64, error) {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal webhook events: %w", err)
	}
	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal webhook headers: %w", err)
	}

	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO webhooks (name, url, events, matcher, headers, secret, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		webhook.Name, webhook.URL, string(eventsJSON), webhook.Matcher,
		string(headersJSON), webhook.Secret, webhook.Active, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get webhook ID: %w", err)
	}

	webhook.ID = id
	webhook.CreatedAt = now
	webhook.UpdatedAt = now
	return id, nil
}

// ListWebhooks returns all registered webhooks.
func (s *stateStore) ListWebhooks() ([]*Webhook, error) {
	rows, err := s.db.Query(
		`SELECT id, name, url, events, matcher, headers, secret, active, created_at, updated_at
		FROM webhooks ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	return scanWebhookRows(rows)
}

// GetWebhook retrieves a webhook by ID.
func (s *stateStore) GetWebhook(id int64) (*Webhook, error) {
	row := s.db.QueryRow(
		`SELECT id, name, url, events, matcher, headers, secret, active, created_at, updated_at
		FROM webhooks WHERE id = ?`,
		id,
	)

	w, err := scanWebhookRow(row)
	if err != nil {
		return nil, fmt.Errorf("webhook not found (id=%d): %w", id, err)
	}
	return w, nil
}

// UpdateWebhook updates an existing webhook.
func (s *stateStore) UpdateWebhook(webhook *Webhook) error {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook events: %w", err)
	}
	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook headers: %w", err)
	}

	now := time.Now()
	result, err := s.db.Exec(
		`UPDATE webhooks SET name = ?, url = ?, events = ?, matcher = ?, headers = ?, secret = ?, active = ?, updated_at = ?
		WHERE id = ?`,
		webhook.Name, webhook.URL, string(eventsJSON), webhook.Matcher,
		string(headersJSON), webhook.Secret, webhook.Active, now, webhook.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("webhook not found (id=%d)", webhook.ID)
	}

	webhook.UpdatedAt = now
	return nil
}

// DeleteWebhook removes a webhook by ID. Associated deliveries are cascade-deleted.
func (s *stateStore) DeleteWebhook(id int64) error {
	result, err := s.db.Exec("DELETE FROM webhooks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("webhook not found (id=%d)", id)
	}
	return nil
}

// RecordWebhookDelivery records a webhook delivery attempt.
func (s *stateStore) RecordWebhookDelivery(delivery *WebhookDelivery) error {
	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO webhook_deliveries (webhook_id, run_id, event, status_code, response_time_ms, error, delivered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		delivery.WebhookID, delivery.RunID, delivery.Event,
		delivery.StatusCode, delivery.ResponseTimeMs, delivery.Error, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record webhook delivery: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get delivery ID: %w", err)
	}

	delivery.ID = id
	delivery.DeliveredAt = now
	return nil
}

// GetWebhookDeliveries retrieves delivery records for a webhook, most recent first.
func (s *stateStore) GetWebhookDeliveries(webhookID int64, limit int) ([]*WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, webhook_id, run_id, event, status_code, response_time_ms, error, delivered_at
		FROM webhook_deliveries WHERE webhook_id = ?
		ORDER BY delivered_at DESC LIMIT ?`,
		webhookID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		var deliveredAt time.Time
		var errStr sql.NullString
		var statusCode sql.NullInt64
		var responseTimeMs sql.NullInt64

		err := rows.Scan(&d.ID, &d.WebhookID, &d.RunID, &d.Event,
			&statusCode, &responseTimeMs, &errStr, &deliveredAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook delivery: %w", err)
		}

		d.DeliveredAt = deliveredAt
		if statusCode.Valid {
			d.StatusCode = int(statusCode.Int64)
		}
		if responseTimeMs.Valid {
			d.ResponseTimeMs = responseTimeMs.Int64
		}
		if errStr.Valid {
			d.Error = errStr.String
		}
		deliveries = append(deliveries, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook deliveries: %w", err)
	}
	return deliveries, nil
}

// RecordOutcome persists a pipeline outcome (PR URL, issue URL, etc.) in the state DB.
// This survives worktree cleanup, unlike artifact files.
func (s *stateStore) RecordOutcome(runID, stepID, outcomeType, label, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO pipeline_outcome (run_id, step_id, type, label, value, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		runID, stepID, outcomeType, label, value, time.Now().Unix(),
	)
	return err
}

// GetOutcomes returns all outcomes for a run.
func (s *stateStore) GetOutcomes(runID string) ([]OutcomeRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, run_id, step_id, type, label, value, created_at FROM pipeline_outcome WHERE run_id = ? ORDER BY created_at",
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOutcomeRows(rows)
}

// GetOutcomesByValue finds runs that produced a specific outcome value (e.g., a PR URL).
func (s *stateStore) GetOutcomesByValue(outcomeType, value string) ([]OutcomeRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, run_id, step_id, type, label, value, created_at FROM pipeline_outcome WHERE type = ? AND value LIKE ? ORDER BY created_at DESC",
		outcomeType, "%"+value+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOutcomeRows(rows)
}

func scanOutcomeRows(rows *sql.Rows) ([]OutcomeRecord, error) {
	var records []OutcomeRecord
	for rows.Next() {
		var r OutcomeRecord
		var createdAt int64
		if err := rows.Scan(&r.ID, &r.RunID, &r.StepID, &r.Type, &r.Label, &r.Value, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		records = append(records, r)
	}
	return records, rows.Err()
}

// scanWebhookRow scans a single webhook row.
func scanWebhookRow(row *sql.Row) (*Webhook, error) {
	var w Webhook
	var eventsJSON, headersJSON string
	var active int

	err := row.Scan(&w.ID, &w.Name, &w.URL, &eventsJSON, &w.Matcher,
		&headersJSON, &w.Secret, &active, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}

	w.Active = active != 0

	if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook events: %w", err)
	}
	if w.Events == nil {
		w.Events = []string{}
	}

	if err := json.Unmarshal([]byte(headersJSON), &w.Headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook headers: %w", err)
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}

	return &w, nil
}

// scanWebhookRows scans multiple webhook rows.
func scanWebhookRows(rows *sql.Rows) ([]*Webhook, error) {
	var webhooks []*Webhook
	for rows.Next() {
		var w Webhook
		var eventsJSON, headersJSON string
		var active int

		err := rows.Scan(&w.ID, &w.Name, &w.URL, &eventsJSON, &w.Matcher,
			&headersJSON, &w.Secret, &active, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook row: %w", err)
		}

		w.Active = active != 0

		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook events: %w", err)
		}
		if w.Events == nil {
			w.Events = []string{}
		}

		if err := json.Unmarshal([]byte(headersJSON), &w.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook headers: %w", err)
		}
		if w.Headers == nil {
			w.Headers = map[string]string{}
		}

		webhooks = append(webhooks, &w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook rows: %w", err)
	}
	return webhooks, nil
}

// RecordOrchestrationDecision inserts a new orchestration decision record.
func (s *stateStore) RecordOrchestrationDecision(record *OrchestrationDecision) error {
	_, err := s.db.Exec(
		`INSERT INTO orchestration_decision (run_id, input_text, domain, complexity, pipeline_name, model_tier, reason, outcome, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.InputText, record.Domain, record.Complexity,
		record.PipelineName, record.ModelTier, record.Reason, "pending",
		time.Now().Unix(),
	)
	return err
}

// UpdateOrchestrationOutcome updates the outcome of an orchestration decision after pipeline completion.
func (s *stateStore) UpdateOrchestrationOutcome(runID string, outcome string, tokensUsed int, durationMs int64) error {
	_, err := s.db.Exec(
		`UPDATE orchestration_decision SET outcome = ?, tokens_used = ?, duration_ms = ?, completed_at = ? WHERE run_id = ?`,
		outcome, tokensUsed, durationMs, time.Now().Unix(), runID,
	)
	return err
}

// ListOrchestrationDecisionSummary returns aggregated decision stats grouped by domain, complexity, pipeline.
func (s *stateStore) ListOrchestrationDecisionSummary(limit int) ([]OrchestrationDecisionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT domain, complexity, pipeline_name,
		        COUNT(*) as total,
		        SUM(CASE WHEN outcome = 'completed' THEN 1 ELSE 0 END) as completed,
		        SUM(CASE WHEN outcome = 'failed' THEN 1 ELSE 0 END) as failed,
		        COALESCE(AVG(CASE WHEN tokens_used > 0 THEN tokens_used END), 0) as avg_tokens,
		        COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) as avg_duration
		 FROM orchestration_decision
		 GROUP BY domain, complexity, pipeline_name
		 ORDER BY total DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []OrchestrationDecisionSummary
	for rows.Next() {
		var s OrchestrationDecisionSummary
		if err := rows.Scan(&s.Domain, &s.Complexity, &s.PipelineName,
			&s.Total, &s.Completed, &s.Failed, &s.AvgTokens, &s.AvgDurationMs); err != nil {
			return nil, err
		}
		if s.Total > 0 {
			s.SuccessRate = float64(s.Completed) / float64(s.Total) * 100
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetOrchestrationStats returns aggregate stats for a pipeline name.
func (s *stateStore) GetOrchestrationStats(pipelineName string) (*OrchestrationStats, error) {
	row := s.db.QueryRow(
		`SELECT pipeline_name,
		        COUNT(*) as total,
		        SUM(CASE WHEN outcome = 'completed' THEN 1 ELSE 0 END) as completed,
		        SUM(CASE WHEN outcome = 'failed' THEN 1 ELSE 0 END) as failed,
		        SUM(CASE WHEN outcome = 'cancelled' THEN 1 ELSE 0 END) as cancelled,
		        COALESCE(AVG(CASE WHEN tokens_used > 0 THEN tokens_used END), 0) as avg_tokens,
		        COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) as avg_duration
		 FROM orchestration_decision
		 WHERE pipeline_name = ?
		 GROUP BY pipeline_name`,
		pipelineName,
	)

	var stats OrchestrationStats
	err := row.Scan(&stats.PipelineName, &stats.TotalRuns, &stats.Completed, &stats.Failed, &stats.Cancelled, &stats.AvgTokens, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
