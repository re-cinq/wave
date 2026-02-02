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
	Close() error
}

type stateStore struct {
	db *sql.DB
}

func NewStateStore(dbPath string) (StateStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
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
		var errMsg sql.NullString

		err := rows.Scan(
			&record.StepID,
			&record.PipelineID,
			&record.State,
			&record.RetryCount,
			&startedAt,
			&completedAt,
			&record.WorkspacePath,
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

func (s *stateStore) Close() error {
	return s.db.Close()
}
