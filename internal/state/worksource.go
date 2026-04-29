package state

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// WorksourceTrigger enumerates how a worksource binding fires runs.
type WorksourceTrigger string

const (
	TriggerOnDemand  WorksourceTrigger = "on_demand"
	TriggerOnLabel   WorksourceTrigger = "on_label"
	TriggerOnOpen    WorksourceTrigger = "on_open"
	TriggerScheduled WorksourceTrigger = "scheduled"
)

// WorksourceBindingRecord maps a forge query (selector JSON) to a pipeline.
// Selector and Config are stored as opaque JSON so the bindings table stays
// schema-stable as the dispatch layer grows.
type WorksourceBindingRecord struct {
	ID           int64
	Forge        string
	Repo         string
	Selector     string // JSON: { labels:[], state:'open', ... }
	PipelineName string
	Trigger      WorksourceTrigger
	Config       string // JSON: cron, debounce, etc — may be empty
	Active       bool
	CreatedAt    time.Time
}

// WorksourceStore is the domain-scoped persistence surface for issue→pipeline
// dispatch bindings. Phase 2 of the onboarding-as-session epic populates it.
type WorksourceStore interface {
	CreateBinding(rec WorksourceBindingRecord) (int64, error)
	UpdateBinding(rec WorksourceBindingRecord) error
	DeactivateBinding(id int64) error
	GetBinding(id int64) (*WorksourceBindingRecord, error)
	ListBindings(forge, repo string) ([]WorksourceBindingRecord, error)
	ListActiveBindings() ([]WorksourceBindingRecord, error)
}

func (s *stateStore) CreateBinding(rec WorksourceBindingRecord) (int64, error) {
	if rec.Forge == "" || rec.Repo == "" || rec.PipelineName == "" || rec.Trigger == "" {
		return 0, errors.New("CreateBinding: forge, repo, pipeline_name, trigger required")
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	if rec.Selector == "" {
		rec.Selector = "{}"
	}
	res, err := s.db.Exec(
		`INSERT INTO worksource_binding
			(forge, repo, selector, pipeline_name, trigger, config, active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.Forge, rec.Repo, rec.Selector, rec.PipelineName, string(rec.Trigger),
		nullEmptyString(rec.Config), rec.Active, rec.CreatedAt.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateBinding rewrites all mutable fields of a binding by id. Returns an
// error if the row does not exist.
func (s *stateStore) UpdateBinding(rec WorksourceBindingRecord) error {
	if rec.ID == 0 {
		return errors.New("UpdateBinding: id required")
	}
	res, err := s.db.Exec(
		`UPDATE worksource_binding
		 SET forge = ?, repo = ?, selector = ?, pipeline_name = ?,
			 trigger = ?, config = ?, active = ?
		 WHERE id = ?`,
		rec.Forge, rec.Repo, rec.Selector, rec.PipelineName, string(rec.Trigger),
		nullEmptyString(rec.Config), rec.Active, rec.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("UpdateBinding: id %d not found", rec.ID)
	}
	return nil
}

// DeactivateBinding flips active to false. Bindings are not hard-deleted so
// run history retains the binding context.
func (s *stateStore) DeactivateBinding(id int64) error {
	res, err := s.db.Exec(`UPDATE worksource_binding SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("DeactivateBinding: id %d not found", id)
	}
	return nil
}

func (s *stateStore) GetBinding(id int64) (*WorksourceBindingRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, forge, repo, selector, pipeline_name, trigger, config, active, created_at
		 FROM worksource_binding WHERE id = ?`,
		id,
	)
	rec, err := scanBinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rec, err
}

// ListBindings returns bindings filtered by forge+repo. Empty filters match
// all rows. Newest first.
func (s *stateStore) ListBindings(forge, repo string) ([]WorksourceBindingRecord, error) {
	q := `SELECT id, forge, repo, selector, pipeline_name, trigger, config, active, created_at
		FROM worksource_binding WHERE 1 = 1`
	var args []any
	if forge != "" {
		q += " AND forge = ?"
		args = append(args, forge)
	}
	if repo != "" {
		q += " AND repo = ?"
		args = append(args, repo)
	}
	q += " ORDER BY created_at DESC"
	return s.queryBindings(q, args...)
}

// ListActiveBindings returns every active binding across forges. Used by the
// poller to know what to scan.
func (s *stateStore) ListActiveBindings() ([]WorksourceBindingRecord, error) {
	return s.queryBindings(
		`SELECT id, forge, repo, selector, pipeline_name, trigger, config, active, created_at
		 FROM worksource_binding WHERE active = 1 ORDER BY created_at DESC`,
	)
}

func (s *stateStore) queryBindings(q string, args ...any) ([]WorksourceBindingRecord, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorksourceBindingRecord
	for rows.Next() {
		var (
			r         WorksourceBindingRecord
			cfg       sql.NullString
			trigger   string
			createdAt int64
		)
		if err := rows.Scan(&r.ID, &r.Forge, &r.Repo, &r.Selector, &r.PipelineName,
			&trigger, &cfg, &r.Active, &createdAt); err != nil {
			return nil, err
		}
		r.Trigger = WorksourceTrigger(trigger)
		if cfg.Valid {
			r.Config = cfg.String
		}
		r.CreatedAt = time.Unix(createdAt, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanBinding(row *sql.Row) (*WorksourceBindingRecord, error) {
	var (
		r         WorksourceBindingRecord
		cfg       sql.NullString
		trigger   string
		createdAt int64
	)
	if err := row.Scan(&r.ID, &r.Forge, &r.Repo, &r.Selector, &r.PipelineName,
		&trigger, &cfg, &r.Active, &createdAt); err != nil {
		return nil, err
	}
	r.Trigger = WorksourceTrigger(trigger)
	if cfg.Valid {
		r.Config = cfg.String
	}
	r.CreatedAt = time.Unix(createdAt, 0)
	return &r, nil
}
