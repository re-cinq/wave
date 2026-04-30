package worksource

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/state"
)

// BindingID is the persistent identifier of a worksource binding row.
type BindingID int64

// Trigger is the externally-facing dashed form of state.WorksourceTrigger.
// Service callers use this enum so they need not import internal/state.
type Trigger string

// Trigger values use the dashed form documented in the issue. Internally
// they are converted to the underscored state.WorksourceTrigger before
// persistence.
const (
	TriggerOnDemand  Trigger = "on-demand"
	TriggerOnLabel   Trigger = "on-label"
	TriggerOnOpen    Trigger = "on-open"
	TriggerScheduled Trigger = "scheduled"
)

// BindingSpec is the writable shape of a worksource binding. It is what
// CreateBinding and UpdateBinding accept.
type BindingSpec struct {
	// Forge is the forge type — "github", "gitea", "codeberg", etc. Required.
	Forge string
	// RepoPattern is a path.Match glob (or exact string) matched against the
	// "owner/repo" coordinate of an incoming work-item. Required.
	RepoPattern string
	// PipelineName is the pipeline the binding fires. Required.
	PipelineName string
	// Trigger is the trigger mode. Required; must be one of the Trigger* constants.
	Trigger Trigger
	// LabelFilter is the any-of set of labels to match against work-item labels.
	// Empty means no label filter.
	LabelFilter []string
	// State filters by work-item state ("open"|"closed"|""). Empty means any.
	State string
	// Kinds filters by work-item kind ("issue"|"pull_request"|...). Empty
	// means any.
	Kinds []string
	// Config is opaque JSON for per-trigger configuration (e.g. cron, debounce).
	// Stored verbatim. May be empty.
	Config string
	// Active marks the binding as eligible for matching. Defaults to true on
	// CreateBinding when zero-value.
	Active bool
}

// BindingRecord is the read shape returned by GetBinding/ListBindings/MatchBindings.
// CreatedAt is set by the store.
type BindingRecord struct {
	ID           BindingID
	Forge        string
	RepoPattern  string
	PipelineName string
	Trigger      Trigger
	LabelFilter  []string
	State        string
	Kinds        []string
	Config       string
	Active       bool
	CreatedAt    time.Time
}

// BindingFilter is the optional filter applied by ListBindings. Empty fields
// match all rows.
type BindingFilter struct {
	Forge string
	Repo  string
}

// WorkItemRef is the in-memory mirror of the #2.1 work_item_ref schema. It is
// the input to MatchBindings.
type WorkItemRef struct {
	Forge  string
	Repo   string
	Kind   string
	ID     string
	Title  string
	URL    string
	Labels []string
	State  string
}

// selectorPayload is the JSON wire form persisted in
// state.WorksourceBindingRecord.Selector. The service is the only writer.
type selectorPayload struct {
	Labels []string `json:"labels,omitempty"`
	State  string   `json:"state,omitempty"`
	Kinds  []string `json:"kinds,omitempty"`
}

// triggerToState converts the dashed external Trigger to the underscored
// state.WorksourceTrigger used by the storage layer.
func triggerToState(t Trigger) (state.WorksourceTrigger, bool) {
	switch t {
	case TriggerOnDemand:
		return state.TriggerOnDemand, true
	case TriggerOnLabel:
		return state.TriggerOnLabel, true
	case TriggerOnOpen:
		return state.TriggerOnOpen, true
	case TriggerScheduled:
		return state.TriggerScheduled, true
	}
	return "", false
}

// triggerFromState converts the underscored state.WorksourceTrigger back to
// the dashed external Trigger.
func triggerFromState(t state.WorksourceTrigger) (Trigger, bool) {
	switch t {
	case state.TriggerOnDemand:
		return TriggerOnDemand, true
	case state.TriggerOnLabel:
		return TriggerOnLabel, true
	case state.TriggerOnOpen:
		return TriggerOnOpen, true
	case state.TriggerScheduled:
		return TriggerScheduled, true
	}
	return "", false
}

// marshalSelector encodes the spec's selector fields into the JSON wire form.
// Returns "{}" for an empty selector so SQL never sees an empty string.
func marshalSelector(spec BindingSpec) (string, error) {
	p := selectorPayload{
		Labels: spec.LabelFilter,
		State:  spec.State,
		Kinds:  spec.Kinds,
	}
	if len(p.Labels) == 0 && p.State == "" && len(p.Kinds) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal selector: %w", err)
	}
	return string(b), nil
}

// unmarshalSelector decodes the stored JSON wire form back into the typed
// fields exposed on BindingRecord.
func unmarshalSelector(raw string) (selectorPayload, error) {
	var p selectorPayload
	if raw == "" || raw == "{}" {
		return p, nil
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return p, fmt.Errorf("unmarshal selector: %w", err)
	}
	return p, nil
}

// fromStoreRecord translates a state.WorksourceBindingRecord into the
// service-layer BindingRecord. Returns an error if the persisted trigger or
// selector JSON is unrecognisable.
func fromStoreRecord(rec state.WorksourceBindingRecord) (BindingRecord, error) {
	trig, ok := triggerFromState(rec.Trigger)
	if !ok {
		return BindingRecord{}, fmt.Errorf("unknown trigger %q on binding %d", rec.Trigger, rec.ID)
	}
	sel, err := unmarshalSelector(rec.Selector)
	if err != nil {
		return BindingRecord{}, fmt.Errorf("binding %d: %w", rec.ID, err)
	}
	return BindingRecord{
		ID:           BindingID(rec.ID),
		Forge:        rec.Forge,
		RepoPattern:  rec.Repo,
		PipelineName: rec.PipelineName,
		Trigger:      trig,
		LabelFilter:  sel.Labels,
		State:        sel.State,
		Kinds:        sel.Kinds,
		Config:       rec.Config,
		Active:       rec.Active,
		CreatedAt:    rec.CreatedAt,
	}, nil
}
