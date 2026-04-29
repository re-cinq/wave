package worksource

import (
	"context"
	"errors"
	"fmt"

	"github.com/recinq/wave/internal/state"
)

// ErrNotFound is returned by Get/Update/Delete when the binding id does not
// exist.
var ErrNotFound = errors.New("worksource: binding not found")

// Service is the domain interface for managing worksource bindings.
//
// All methods take context.Context as the first argument. The current
// implementation only honours ctx.Err() at entry — the underlying
// state.WorksourceStore does not yet accept ctx. Once it does, the service
// signatures already match.
type Service interface {
	// CreateBinding validates the spec, persists the binding, and returns the
	// new id. Active defaults to true.
	CreateBinding(ctx context.Context, spec BindingSpec) (BindingID, error)

	// ListBindings returns all bindings matching filter (forge and/or repo
	// equality). Newest first. Includes inactive bindings.
	ListBindings(ctx context.Context, filter BindingFilter) ([]BindingRecord, error)

	// GetBinding returns the binding for id, or ErrNotFound.
	GetBinding(ctx context.Context, id BindingID) (BindingRecord, error)

	// UpdateBinding overwrites the mutable fields of binding id with spec.
	// CreatedAt is preserved from the existing row. Returns ErrNotFound if
	// the row is gone.
	UpdateBinding(ctx context.Context, id BindingID, spec BindingSpec) error

	// DeleteBinding is a soft-delete: the binding is marked inactive.
	// Run-history references survive. Returns ErrNotFound if the row is gone.
	DeleteBinding(ctx context.Context, id BindingID) error

	// MatchBindings returns every active binding whose forge/repo glob/labels/
	// state/kind admit the given WorkItemRef. The filter runs in-memory over
	// state.ListActiveBindings.
	MatchBindings(ctx context.Context, ref WorkItemRef) ([]BindingRecord, error)
}

// NewService returns a Service over the given store. The store is the only
// required dependency.
func NewService(store state.WorksourceStore) Service {
	return &service{store: store}
}

type service struct {
	store state.WorksourceStore
}

func (s *service) CreateBinding(ctx context.Context, spec BindingSpec) (BindingID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := validateSpec(spec); err != nil {
		return 0, err
	}
	selector, err := marshalSelector(spec)
	if err != nil {
		return 0, err
	}
	trig, _ := triggerToState(spec.Trigger) // already validated.
	id, err := s.store.CreateBinding(state.WorksourceBindingRecord{
		Forge:        spec.Forge,
		Repo:         spec.RepoPattern,
		Selector:     selector,
		PipelineName: spec.PipelineName,
		Trigger:      trig,
		Config:       spec.Config,
		Active:       true,
	})
	if err != nil {
		return 0, fmt.Errorf("worksource: create binding: %w", err)
	}
	return BindingID(id), nil
}

func (s *service) GetBinding(ctx context.Context, id BindingID) (BindingRecord, error) {
	if err := ctx.Err(); err != nil {
		return BindingRecord{}, err
	}
	rec, err := s.store.GetBinding(int64(id))
	if err != nil {
		return BindingRecord{}, fmt.Errorf("worksource: get binding %d: %w", id, err)
	}
	if rec == nil {
		return BindingRecord{}, fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	return fromStoreRecord(*rec)
}

func (s *service) ListBindings(ctx context.Context, filter BindingFilter) ([]BindingRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.store.ListBindings(filter.Forge, filter.Repo)
	if err != nil {
		return nil, fmt.Errorf("worksource: list bindings: %w", err)
	}
	out := make([]BindingRecord, 0, len(rows))
	for _, r := range rows {
		rec, err := fromStoreRecord(r)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *service) UpdateBinding(ctx context.Context, id BindingID, spec BindingSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("worksource: update binding: id required")
	}
	if err := validateSpec(spec); err != nil {
		return err
	}
	cur, err := s.store.GetBinding(int64(id))
	if err != nil {
		return fmt.Errorf("worksource: update binding %d: %w", id, err)
	}
	if cur == nil {
		return fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	selector, err := marshalSelector(spec)
	if err != nil {
		return err
	}
	trig, _ := triggerToState(spec.Trigger) // already validated.
	updated := state.WorksourceBindingRecord{
		ID:           int64(id),
		Forge:        spec.Forge,
		Repo:         spec.RepoPattern,
		Selector:     selector,
		PipelineName: spec.PipelineName,
		Trigger:      trig,
		Config:       spec.Config,
		Active:       cur.Active, // Update preserves active flag; Delete is the way to deactivate.
		CreatedAt:    cur.CreatedAt,
	}
	if err := s.store.UpdateBinding(updated); err != nil {
		return fmt.Errorf("worksource: update binding %d: %w", id, err)
	}
	return nil
}

func (s *service) DeleteBinding(ctx context.Context, id BindingID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("worksource: delete binding: id required")
	}
	cur, err := s.store.GetBinding(int64(id))
	if err != nil {
		return fmt.Errorf("worksource: delete binding %d: %w", id, err)
	}
	if cur == nil {
		return fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	if err := s.store.DeactivateBinding(int64(id)); err != nil {
		return fmt.Errorf("worksource: delete binding %d: %w", id, err)
	}
	return nil
}

func (s *service) MatchBindings(ctx context.Context, ref WorkItemRef) ([]BindingRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.store.ListActiveBindings()
	if err != nil {
		return nil, fmt.Errorf("worksource: list active bindings: %w", err)
	}
	out := make([]BindingRecord, 0, len(rows))
	for _, r := range rows {
		rec, err := fromStoreRecord(r)
		if err != nil {
			return nil, err
		}
		if matches(rec, ref) {
			out = append(out, rec)
		}
	}
	return out, nil
}
