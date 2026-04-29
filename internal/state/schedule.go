package state

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ScheduleRecord is one cron-driven pipeline schedule. InputRef is opaque JSON
// (a literal input, a worksource query selector, or empty for runs that take
// no input). NextFireAt is what the scheduler tick loop reads to decide whose
// turn it is; UpdateScheduleNextFire bumps it after a fire.
type ScheduleRecord struct {
	ID           int64
	PipelineName string
	CronExpr     string
	InputRef     string
	Active       bool
	NextFireAt   *time.Time
	LastRunID    string
	CreatedAt    time.Time
}

// ScheduleStore is the domain-scoped persistence surface for cron-driven
// pipeline runs. Phase 0 PRE-6 introduces the in-process scheduler that
// consumes this table.
type ScheduleStore interface {
	CreateSchedule(rec ScheduleRecord) (int64, error)
	UpdateScheduleNextFire(id int64, nextFireAt time.Time, lastRunID string) error
	DeactivateSchedule(id int64) error
	GetSchedule(id int64) (*ScheduleRecord, error)
	ListSchedules() ([]ScheduleRecord, error)
	ListDueSchedules(now time.Time) ([]ScheduleRecord, error)
}

func (s *stateStore) CreateSchedule(rec ScheduleRecord) (int64, error) {
	if rec.PipelineName == "" || rec.CronExpr == "" {
		return 0, errors.New("CreateSchedule: pipeline_name and cron_expr required")
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	var nextFire any
	if rec.NextFireAt != nil {
		nextFire = rec.NextFireAt.Unix()
	}
	res, err := s.db.Exec(
		`INSERT INTO schedule
			(pipeline_name, cron_expr, input_ref, active, next_fire_at, last_run_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rec.PipelineName, rec.CronExpr, nullEmptyString(rec.InputRef), rec.Active,
		nextFire, nullEmptyString(rec.LastRunID), rec.CreatedAt.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateScheduleNextFire records that the schedule fired (LastRunID) and
// advances NextFireAt to the next computed tick.
func (s *stateStore) UpdateScheduleNextFire(id int64, nextFireAt time.Time, lastRunID string) error {
	res, err := s.db.Exec(
		`UPDATE schedule SET next_fire_at = ?, last_run_id = ? WHERE id = ?`,
		nextFireAt.Unix(), nullEmptyString(lastRunID), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("UpdateScheduleNextFire: id %d not found", id)
	}
	return nil
}

func (s *stateStore) DeactivateSchedule(id int64) error {
	res, err := s.db.Exec(`UPDATE schedule SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("DeactivateSchedule: id %d not found", id)
	}
	return nil
}

func (s *stateStore) GetSchedule(id int64) (*ScheduleRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, pipeline_name, cron_expr, input_ref, active, next_fire_at, last_run_id, created_at
		 FROM schedule WHERE id = ?`,
		id,
	)
	rec, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rec, err
}

// ListSchedules returns every schedule row, newest first.
func (s *stateStore) ListSchedules() ([]ScheduleRecord, error) {
	return s.querySchedules(
		`SELECT id, pipeline_name, cron_expr, input_ref, active, next_fire_at, last_run_id, created_at
		 FROM schedule ORDER BY created_at DESC`,
	)
}

// ListDueSchedules returns active schedules whose next_fire_at is at or
// before `now`, oldest-due first. The scheduler iterates these to fire runs.
func (s *stateStore) ListDueSchedules(now time.Time) ([]ScheduleRecord, error) {
	return s.querySchedules(
		`SELECT id, pipeline_name, cron_expr, input_ref, active, next_fire_at, last_run_id, created_at
		 FROM schedule WHERE active = 1 AND next_fire_at IS NOT NULL AND next_fire_at <= ?
		 ORDER BY next_fire_at ASC`,
		now.Unix(),
	)
}

func (s *stateStore) querySchedules(q string, args ...any) ([]ScheduleRecord, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScheduleRecord
	for rows.Next() {
		rec, err := scanScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanScheduleRow(row rowScanner) (*ScheduleRecord, error) {
	var (
		r          ScheduleRecord
		inputRef   sql.NullString
		nextFire   sql.NullInt64
		lastRunID  sql.NullString
		createdAt  int64
	)
	if err := row.Scan(&r.ID, &r.PipelineName, &r.CronExpr, &inputRef, &r.Active,
		&nextFire, &lastRunID, &createdAt); err != nil {
		return nil, err
	}
	if inputRef.Valid {
		r.InputRef = inputRef.String
	}
	if nextFire.Valid {
		t := time.Unix(nextFire.Int64, 0)
		r.NextFireAt = &t
	}
	if lastRunID.Valid {
		r.LastRunID = lastRunID.String
	}
	r.CreatedAt = time.Unix(createdAt, 0)
	return &r, nil
}

func scanSchedule(row *sql.Row) (*ScheduleRecord, error) {
	return scanScheduleRow(row)
}
