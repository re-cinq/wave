package state

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"

	_ "modernc.org/sqlite"
)

// StepState represents the state of a pipeline step. It aliases event.StepState
// — internal/event is the canonical owner of the lifecycle vocabulary, and
// internal/state imports it (reversing the historical event -> state import).
type StepState = event.StepState

// Step lifecycle constants. These re-export the canonical untyped string
// constants from internal/event so they are usable both as plain strings and
// as StepState values (typed alias of event.StepState).
const (
	StatePending        StepState = event.StatePending
	StateRunning        StepState = event.StateRunning
	StateCompleted      StepState = event.StateCompleted
	StateCompletedEmpty StepState = event.StateCompletedEmpty
	StateFailed         StepState = event.StateFailed
	StateRetrying       StepState = event.StateRetrying
	StateSkipped        StepState = event.StateSkipped
	StateReworking      StepState = event.StateReworking
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
	// Liveness tracking — running processes call this periodically so the
	// reconciler can flag zombies whose owning process died without
	// updating the DB.
	UpdateRunHeartbeat(runID string) error
	ReapOrphans(staleAfter time.Duration) (int, error)

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
	SetRunComposition(childRunID, runKind, subPipelineRef, iterateMode string, iterateIndex, iterateTotal *int) error
	GetChildRuns(parentRunID string) ([]RunRecord, error)
	GetSubtreeTokens(rootRunID string) (int64, error)

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
