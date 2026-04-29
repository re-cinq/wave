package worksource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/recinq/wave/internal/state"
)

func TestTriggerRoundTrip(t *testing.T) {
	cases := []struct {
		dashed     Trigger
		underscore state.WorksourceTrigger
	}{
		{TriggerOnDemand, state.TriggerOnDemand},
		{TriggerOnLabel, state.TriggerOnLabel},
		{TriggerOnOpen, state.TriggerOnOpen},
		{TriggerScheduled, state.TriggerScheduled},
	}
	for _, c := range cases {
		t.Run(string(c.dashed), func(t *testing.T) {
			got, ok := triggerToState(c.dashed)
			require.True(t, ok)
			assert.Equal(t, c.underscore, got)

			back, ok := triggerFromState(c.underscore)
			require.True(t, ok)
			assert.Equal(t, c.dashed, back)
		})
	}
}

func TestTriggerToState_Unknown(t *testing.T) {
	_, ok := triggerToState("garbage")
	assert.False(t, ok)
}

func TestTriggerFromState_Unknown(t *testing.T) {
	_, ok := triggerFromState(state.WorksourceTrigger("garbage"))
	assert.False(t, ok)
}

func TestMarshalSelector_Empty(t *testing.T) {
	got, err := marshalSelector(BindingSpec{})
	require.NoError(t, err)
	assert.Equal(t, "{}", got)
}

func TestMarshalSelector_Pinned(t *testing.T) {
	got, err := marshalSelector(BindingSpec{
		LabelFilter: []string{"bug", "ready-for-impl"},
		State:       "open",
		Kinds:       []string{"issue"},
	})
	require.NoError(t, err)
	// Field order is fixed by Go struct field order — pin the wire form.
	assert.Equal(t, `{"labels":["bug","ready-for-impl"],"state":"open","kinds":["issue"]}`, got)
}

func TestMarshalSelector_PartialFields(t *testing.T) {
	got, err := marshalSelector(BindingSpec{State: "closed"})
	require.NoError(t, err)
	assert.Equal(t, `{"state":"closed"}`, got)
}

func TestUnmarshalSelector_RoundTrip(t *testing.T) {
	in := `{"labels":["bug"],"state":"open","kinds":["issue"]}`
	p, err := unmarshalSelector(in)
	require.NoError(t, err)
	assert.Equal(t, []string{"bug"}, p.Labels)
	assert.Equal(t, "open", p.State)
	assert.Equal(t, []string{"issue"}, p.Kinds)
}

func TestUnmarshalSelector_EmptyForms(t *testing.T) {
	for _, raw := range []string{"", "{}"} {
		p, err := unmarshalSelector(raw)
		require.NoError(t, err)
		assert.Empty(t, p.Labels)
		assert.Empty(t, p.State)
		assert.Empty(t, p.Kinds)
	}
}

func TestUnmarshalSelector_Bad(t *testing.T) {
	_, err := unmarshalSelector("not json")
	assert.Error(t, err)
}

func TestFromStoreRecord_Round(t *testing.T) {
	rec, err := fromStoreRecord(state.WorksourceBindingRecord{
		ID:           42,
		Forge:        "github",
		Repo:         "owner/repo",
		Selector:     `{"labels":["bug"],"state":"open"}`,
		PipelineName: "impl-issue",
		Trigger:      state.TriggerOnLabel,
		Active:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, BindingID(42), rec.ID)
	assert.Equal(t, TriggerOnLabel, rec.Trigger)
	assert.Equal(t, []string{"bug"}, rec.LabelFilter)
	assert.Equal(t, "open", rec.State)
}

func TestFromStoreRecord_BadTrigger(t *testing.T) {
	_, err := fromStoreRecord(state.WorksourceBindingRecord{
		ID: 1, Forge: "github", Repo: "x/y", Trigger: state.WorksourceTrigger("???"),
	})
	assert.Error(t, err)
}

func TestFromStoreRecord_BadSelector(t *testing.T) {
	_, err := fromStoreRecord(state.WorksourceBindingRecord{
		ID: 1, Forge: "github", Repo: "x/y", Trigger: state.TriggerOnDemand,
		Selector: "not json",
	})
	assert.Error(t, err)
}
