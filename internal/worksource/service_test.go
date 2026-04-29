package worksource

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/recinq/wave/internal/state"
)

func newTestService(t *testing.T) (Service, func()) {
	t.Helper()
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)
	cleanup := func() { _ = store.Close() }
	return NewService(store), cleanup
}

func validSpec() BindingSpec {
	return BindingSpec{
		Forge:        "github",
		RepoPattern:  "re-cinq/wave",
		PipelineName: "impl-issue",
		Trigger:      TriggerOnLabel,
		LabelFilter:  []string{"ready-for-impl"},
		State:        "open",
		Kinds:        []string{"issue"},
	}
}

func TestService_CreateGetRoundTrip(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	id, err := svc.CreateBinding(ctx, validSpec())
	require.NoError(t, err)
	require.NotZero(t, id)

	got, err := svc.GetBinding(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "github", got.Forge)
	assert.Equal(t, "re-cinq/wave", got.RepoPattern)
	assert.Equal(t, TriggerOnLabel, got.Trigger)
	assert.Equal(t, []string{"ready-for-impl"}, got.LabelFilter)
	assert.Equal(t, "open", got.State)
	assert.Equal(t, []string{"issue"}, got.Kinds)
	assert.True(t, got.Active)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestService_CreateDefaultsActive(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	spec := validSpec()
	spec.Active = false // ignored — Create defaults to active=true.
	id, err := svc.CreateBinding(ctx, spec)
	require.NoError(t, err)

	got, err := svc.GetBinding(ctx, id)
	require.NoError(t, err)
	assert.True(t, got.Active)
}

func TestService_ListByForge(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	specs := []BindingSpec{
		validSpec(),
		{Forge: "gitea", RepoPattern: "x/y", PipelineName: "p", Trigger: TriggerOnDemand},
		{Forge: "github", RepoPattern: "other/repo", PipelineName: "p", Trigger: TriggerOnDemand},
	}
	for _, s := range specs {
		_, err := svc.CreateBinding(ctx, s)
		require.NoError(t, err)
	}

	github, err := svc.ListBindings(ctx, BindingFilter{Forge: "github"})
	require.NoError(t, err)
	assert.Len(t, github, 2)

	repoFilter, err := svc.ListBindings(ctx, BindingFilter{Forge: "github", Repo: "re-cinq/wave"})
	require.NoError(t, err)
	assert.Len(t, repoFilter, 1)

	all, err := svc.ListBindings(ctx, BindingFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestService_UpdateMutatesFields(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	id, err := svc.CreateBinding(ctx, validSpec())
	require.NoError(t, err)

	original, err := svc.GetBinding(ctx, id)
	require.NoError(t, err)

	updated := validSpec()
	updated.PipelineName = "impl-issue-v2"
	updated.LabelFilter = []string{"bug"}
	updated.Trigger = TriggerOnDemand
	require.NoError(t, svc.UpdateBinding(ctx, id, updated))

	got, err := svc.GetBinding(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "impl-issue-v2", got.PipelineName)
	assert.Equal(t, TriggerOnDemand, got.Trigger)
	assert.Equal(t, []string{"bug"}, got.LabelFilter)
	// CreatedAt is preserved across update.
	assert.Equal(t, original.CreatedAt.Unix(), got.CreatedAt.Unix())
	assert.True(t, got.Active)
}

func TestService_DeleteSoftDeactivates(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	id, err := svc.CreateBinding(ctx, validSpec())
	require.NoError(t, err)

	require.NoError(t, svc.DeleteBinding(ctx, id))

	// Still exists; just inactive.
	got, err := svc.GetBinding(ctx, id)
	require.NoError(t, err)
	assert.False(t, got.Active)

	// MatchBindings (which uses ListActiveBindings) excludes it.
	matches, err := svc.MatchBindings(ctx, WorkItemRef{
		Forge: "github", Repo: "re-cinq/wave", Kind: "issue",
		Labels: []string{"ready-for-impl"}, State: "open",
	})
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestService_MatchBindings(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	// One matches by exact repo + label + state.
	idMatch, err := svc.CreateBinding(ctx, BindingSpec{
		Forge: "github", RepoPattern: "re-cinq/wave",
		PipelineName: "impl-issue", Trigger: TriggerOnLabel,
		LabelFilter: []string{"ready-for-impl"}, State: "open",
		Kinds: []string{"issue"},
	})
	require.NoError(t, err)

	// One matches by glob.
	idGlob, err := svc.CreateBinding(ctx, BindingSpec{
		Forge: "github", RepoPattern: "re-cinq/*",
		PipelineName: "scope", Trigger: TriggerOnOpen,
	})
	require.NoError(t, err)

	// One does not match (different forge).
	_, err = svc.CreateBinding(ctx, BindingSpec{
		Forge: "gitea", RepoPattern: "re-cinq/wave",
		PipelineName: "p", Trigger: TriggerOnDemand,
	})
	require.NoError(t, err)

	// One does not match (label filter mismatch).
	_, err = svc.CreateBinding(ctx, BindingSpec{
		Forge: "github", RepoPattern: "re-cinq/wave",
		PipelineName: "p", Trigger: TriggerOnLabel,
		LabelFilter: []string{"won't-fix"},
	})
	require.NoError(t, err)

	got, err := svc.MatchBindings(ctx, WorkItemRef{
		Forge: "github", Repo: "re-cinq/wave", Kind: "issue",
		ID: "1591", Labels: []string{"ready-for-impl", "enhancement"},
		State: "open",
	})
	require.NoError(t, err)
	require.Len(t, got, 2)

	gotIDs := map[BindingID]bool{got[0].ID: true, got[1].ID: true}
	assert.True(t, gotIDs[idMatch])
	assert.True(t, gotIDs[idGlob])
}

func TestService_InvalidSpecRejection(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	cases := []struct {
		name string
		spec BindingSpec
	}{
		{"empty forge", BindingSpec{RepoPattern: "a/b", PipelineName: "p", Trigger: TriggerOnDemand}},
		{"empty pipeline", BindingSpec{Forge: "github", RepoPattern: "a/b", Trigger: TriggerOnDemand}},
		{"unknown trigger", BindingSpec{Forge: "github", RepoPattern: "a/b", PipelineName: "p", Trigger: "bogus"}},
		{"empty repo", BindingSpec{Forge: "github", PipelineName: "p", Trigger: TriggerOnDemand}},
		{"malformed glob", BindingSpec{Forge: "github", RepoPattern: "[", PipelineName: "p", Trigger: TriggerOnDemand}},
		{"double-star glob", BindingSpec{Forge: "github", RepoPattern: "**/repo", PipelineName: "p", Trigger: TriggerOnDemand}},
		{"empty label", BindingSpec{Forge: "github", RepoPattern: "a/b", PipelineName: "p", Trigger: TriggerOnDemand, LabelFilter: []string{""}}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.CreateBinding(ctx, c.spec)
			assert.Error(t, err)

			// Update should also reject.
			err = svc.UpdateBinding(ctx, BindingID(1), c.spec)
			assert.Error(t, err)
		})
	}
}

func TestService_MissingID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	_, err := svc.GetBinding(ctx, 9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))

	err = svc.UpdateBinding(ctx, 9999, validSpec())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))

	err = svc.DeleteBinding(ctx, 9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestService_CtxCancelled(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.CreateBinding(ctx, validSpec())
	assert.ErrorIs(t, err, context.Canceled)

	_, err = svc.GetBinding(ctx, 1)
	assert.ErrorIs(t, err, context.Canceled)

	_, err = svc.ListBindings(ctx, BindingFilter{})
	assert.ErrorIs(t, err, context.Canceled)

	err = svc.UpdateBinding(ctx, 1, validSpec())
	assert.ErrorIs(t, err, context.Canceled)

	err = svc.DeleteBinding(ctx, 1)
	assert.ErrorIs(t, err, context.Canceled)

	_, err = svc.MatchBindings(ctx, WorkItemRef{})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestService_ZeroID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	err := svc.UpdateBinding(ctx, 0, validSpec())
	assert.Error(t, err)

	err = svc.DeleteBinding(ctx, 0)
	assert.Error(t, err)
}
