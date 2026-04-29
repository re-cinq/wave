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
	// StateRejected marks a terminal "design rejection" — a contract with
	// on_failure: rejected fired because the persona output deliberately
	// signalled the work is non-actionable (e.g. issue already implemented).
	// It is not a runtime failure; UIs render it distinctly from
	// StateFailed. See internal/event for the canonical definition.
	StateRejected StepState = event.StateRejected
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

// StateStore is the aggregate persistence surface — the union of every
// domain-scoped sub-interface (RunStore, EventStore, OntologyStore,
// WebhookStore, ChatStore) plus Close.
//
// DEPRECATED for new code: consumers should depend on the smallest narrow
// interface that satisfies their call sites. StateStore is retained as the
// composed type returned by NewStateStore so root-level constructors can
// dispatch domain handles, and so legacy multi-domain consumers continue to
// compile until they can be narrowed.
type StateStore interface {
	RunStore
	EventStore
	OntologyStore
	WebhookStore
	ChatStore

	Close() error
}

// runningRunsLister is the minimal store surface WaitForConcurrencySlot needs.
type runningRunsLister interface {
	GetRunningRuns() ([]RunRecord, error)
}

// WaitForConcurrencySlot polls GetRunningRuns until fewer than maxWorkers
// pipelines are running. Returns nil when a slot is available, or ctx.Err()
// if the context is cancelled. This is the single concurrency gate used by
// CLI --detach, WebUI, and TUI launch paths.
func WaitForConcurrencySlot(ctx context.Context, store runningRunsLister, maxWorkers int, onWait func(running, max int)) error {
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

// UnderlyingDB returns the *sql.DB handle backing the given StateStore, or
// nil if the store is not the canonical sqlite-backed implementation (e.g. a
// test mock). This is the seam used by adjacent packages — internal/metrics
// in particular — that need to share the same connection pool without
// importing internal/state's private types.
func UnderlyingDB(s StateStore) *sql.DB {
	if ss, ok := s.(*stateStore); ok {
		return ss.db
	}
	return nil
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
