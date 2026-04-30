package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordEval_BasicInsert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	score := 0.85
	pass := true
	retries := 1
	dur := int64(12500)
	cost := 0.42

	rec := PipelineEvalRecord{
		PipelineName: "impl-issue",
		RunID:        "run-1",
		JudgeScore:   &score,
		ContractPass: &pass,
		RetryCount:   &retries,
		FailureClass: "",
		DurationMs:   &dur,
		CostDollars:  &cost,
	}
	require.NoError(t, store.RecordEval(rec))

	got, err := store.GetEvalsForPipeline("impl-issue", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "run-1", got[0].RunID)
	assert.NotNil(t, got[0].JudgeScore)
	assert.InDelta(t, 0.85, *got[0].JudgeScore, 1e-9)
	assert.NotNil(t, got[0].ContractPass)
	assert.True(t, *got[0].ContractPass)
}

func TestGetEvalsForPipeline_OrderingAndLimit(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	for i, runID := range []string{"r1", "r2", "r3"} {
		rec := PipelineEvalRecord{
			PipelineName: "p",
			RunID:        runID,
			RecordedAt:   now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, store.RecordEval(rec))
	}

	all, err := store.GetEvalsForPipeline("p", 0)
	require.NoError(t, err)
	require.Len(t, all, 3)
	// Newest first
	assert.Equal(t, "r3", all[0].RunID)
	assert.Equal(t, "r1", all[2].RunID)

	limited, err := store.GetEvalsForPipeline("p", 2)
	require.NoError(t, err)
	require.Len(t, limited, 2)
}

func TestPipelineVersion_CreateAndActivate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	v1 := PipelineVersionRecord{PipelineName: "p", Version: 1, SHA256: "abc", YAMLPath: ".agents/p.v1.yaml", Active: true}
	require.NoError(t, store.CreatePipelineVersion(v1))

	v2 := PipelineVersionRecord{PipelineName: "p", Version: 2, SHA256: "def", YAMLPath: ".agents/p.v2.yaml", Active: true}
	require.NoError(t, store.CreatePipelineVersion(v2))

	// Creating v2 with active=true should have deactivated v1.
	active, err := store.GetActiveVersion("p")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, 2, active.Version)

	// Roll back to v1.
	require.NoError(t, store.ActivateVersion("p", 1))
	active, err = store.GetActiveVersion("p")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, 1, active.Version)
}

func TestActivateVersion_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	err := store.ActivateVersion("ghost", 99)
	assert.Error(t, err)
}

func TestProposal_CreateDecideList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	id, err := store.CreateProposal(EvolutionProposalRecord{
		PipelineName:   "p",
		VersionBefore:  1,
		VersionAfter:   2,
		DiffPath:       ".agents/proposals/1.diff",
		Reason:         "judge_score below 0.80 over 10 runs",
		SignalSummary:  `{"avg_judge":0.72}`,
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	listed, err := store.ListProposalsByStatus(ProposalProposed, 10)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, id, listed[0].ID)

	require.NoError(t, store.DecideProposal(id, ProposalApproved, "nextlevelshit"))

	got, err := store.GetProposal(id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, ProposalApproved, got.Status)
	require.NotNil(t, got.DecidedAt)
	assert.Equal(t, "nextlevelshit", got.DecidedBy)

	// Decide twice → second call must error (already terminal).
	err = store.DecideProposal(id, ProposalRejected, "x")
	assert.Error(t, err)
}

func TestLastProposalAt_EmptyStore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ts, ok, err := store.LastProposalAt("ghost")
	require.NoError(t, err)
	assert.False(t, ok, "no rows → ok=false")
	assert.True(t, ts.IsZero(), "no rows → zero time")
}

func TestLastProposalAt_SingleProposal(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	want := time.Unix(time.Now().Unix(), 0) // truncate to second precision (column is int64 seconds)
	_, err := store.CreateProposal(EvolutionProposalRecord{
		PipelineName:  "p",
		VersionBefore: 1,
		VersionAfter:  2,
		DiffPath:      "x",
		Reason:        "y",
		ProposedAt:    want,
	})
	require.NoError(t, err)
	got, ok, err := store.LastProposalAt("p")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, want.Unix(), got.Unix())
}

func TestLastProposalAt_ReturnsMostRecent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	older := time.Unix(1_700_000_000, 0)
	newer := time.Unix(1_700_000_500, 0)
	for _, ts := range []time.Time{older, newer} {
		_, err := store.CreateProposal(EvolutionProposalRecord{
			PipelineName:  "p",
			VersionBefore: 1,
			VersionAfter:  2,
			DiffPath:      "x",
			Reason:        "y",
			ProposedAt:    ts,
		})
		require.NoError(t, err)
	}
	got, ok, err := store.LastProposalAt("p")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, newer.Unix(), got.Unix())
}

func TestProposal_DecideRejectsProposedTransition(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	id, err := store.CreateProposal(EvolutionProposalRecord{
		PipelineName: "p", VersionBefore: 1, VersionAfter: 2,
		DiffPath: "x", Reason: "y",
	})
	require.NoError(t, err)
	err = store.DecideProposal(id, ProposalProposed, "x")
	assert.Error(t, err, "decide → proposed must be rejected")
}
