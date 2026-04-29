package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorksourceBinding_CreateGetUpdateDeactivate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	id, err := store.CreateBinding(WorksourceBindingRecord{
		Forge:        "github",
		Repo:         "re-cinq/wave",
		Selector:     `{"labels":["bug"],"state":"open"}`,
		PipelineName: "impl-issue",
		Trigger:      TriggerOnLabel,
		Active:       true,
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	got, err := store.GetBinding(id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "github", got.Forge)
	assert.Equal(t, TriggerOnLabel, got.Trigger)
	assert.True(t, got.Active)

	got.PipelineName = "impl-issue-v2"
	require.NoError(t, store.UpdateBinding(*got))

	got2, err := store.GetBinding(id)
	require.NoError(t, err)
	assert.Equal(t, "impl-issue-v2", got2.PipelineName)

	require.NoError(t, store.DeactivateBinding(id))
	got3, err := store.GetBinding(id)
	require.NoError(t, err)
	assert.False(t, got3.Active)
}

func TestWorksourceBinding_ListByForgeRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	for _, b := range []WorksourceBindingRecord{
		{Forge: "github", Repo: "a/b", Selector: "{}", PipelineName: "p1", Trigger: TriggerOnDemand, Active: true},
		{Forge: "gitea", Repo: "x/y", Selector: "{}", PipelineName: "p2", Trigger: TriggerOnDemand, Active: true},
		{Forge: "github", Repo: "a/b", Selector: "{}", PipelineName: "p3", Trigger: TriggerOnDemand, Active: false},
	} {
		_, err := store.CreateBinding(b)
		require.NoError(t, err)
	}

	github, err := store.ListBindings("github", "")
	require.NoError(t, err)
	assert.Len(t, github, 2)

	repoAB, err := store.ListBindings("github", "a/b")
	require.NoError(t, err)
	assert.Len(t, repoAB, 2)

	all, err := store.ListBindings("", "")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	active, err := store.ListActiveBindings()
	require.NoError(t, err)
	assert.Len(t, active, 2)
}

func TestWorksourceBinding_UpdateMissing(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	err := store.UpdateBinding(WorksourceBindingRecord{
		ID: 999, Forge: "x", Repo: "y", Selector: "{}",
		PipelineName: "p", Trigger: TriggerOnDemand, Active: true,
	})
	assert.Error(t, err)
}
