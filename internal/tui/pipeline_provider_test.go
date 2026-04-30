package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// baseStateStore — no-op implementation of state.StateStore.
// Embed this in mocks so only the needed methods must be overridden.
// ---------------------------------------------------------------------------

type baseStateStore struct{}

func (b baseStateStore) SavePipelineState(string, string, string) error { return nil }
func (b baseStateStore) SaveStepState(string, string, state.StepState, string) error {
	return nil
}
func (b baseStateStore) GetPipelineState(string) (*state.PipelineStateRecord, error) {
	return nil, nil
}
func (b baseStateStore) GetStepStates(string) ([]state.StepStateRecord, error) { return nil, nil }
func (b baseStateStore) ListRecentPipelines(int) ([]state.PipelineStateRecord, error) {
	return nil, nil
}
func (b baseStateStore) Close() error                                           { return nil }
func (b baseStateStore) CreateRun(string, string) (string, error)               { return "", nil }
func (b baseStateStore) CreateRunWithLimit(string, string, int) (string, error) { return "", nil }
func (b baseStateStore) UpdateRunStatus(string, string, string, int) error      { return nil }
func (b baseStateStore) UpdateRunBranch(string, string) error                   { return nil }
func (b baseStateStore) GetRun(string) (*state.RunRecord, error)                { return nil, nil }
func (b baseStateStore) GetRunningRuns() ([]state.RunRecord, error)             { return nil, nil }
func (b baseStateStore) ListRuns(state.ListRunsOptions) ([]state.RunRecord, error) {
	return nil, nil
}
func (b baseStateStore) DeleteRun(string) error { return nil }
func (b baseStateStore) LogEvent(string, string, string, string, string, int, int64, string, string, string) error {
	return nil
}
func (b baseStateStore) GetEvents(string, state.EventQueryOptions) ([]state.LogRecord, error) {
	return nil, nil
}
func (b baseStateStore) RegisterArtifact(string, string, string, string, string, int64) error {
	return nil
}
func (b baseStateStore) GetArtifacts(string, string) ([]state.ArtifactRecord, error) {
	return nil, nil
}
func (b baseStateStore) RequestCancellation(string, bool) error { return nil }
func (b baseStateStore) CheckCancellation(string) (*state.CancellationRecord, error) {
	return nil, nil
}
func (b baseStateStore) ClearCancellation(string) error { return nil }
func (b baseStateStore) SaveProgressSnapshot(string, string, int, string, int64, string, string) error {
	return nil
}
func (b baseStateStore) GetProgressSnapshots(string, string, int) ([]state.ProgressSnapshotRecord, error) {
	return nil, nil
}
func (b baseStateStore) UpdateStepProgress(string, string, string, string, int, string, string, int64, int) error {
	return nil
}
func (b baseStateStore) GetStepProgress(string) (*state.StepProgressRecord, error) {
	return nil, nil
}
func (b baseStateStore) GetAllStepProgress(string) ([]state.StepProgressRecord, error) {
	return nil, nil
}
func (b baseStateStore) UpdatePipelineProgress(string, int, int, int, int, int64) error {
	return nil
}
func (b baseStateStore) GetPipelineProgress(string) (*state.PipelineProgressRecord, error) {
	return nil, nil
}
func (b baseStateStore) SaveArtifactMetadata(int64, string, string, string, string, string, string) error {
	return nil
}
func (b baseStateStore) GetArtifactMetadata(int64) (*state.ArtifactMetadataRecord, error) {
	return nil, nil
}
func (b baseStateStore) SetRunTags(string, []string) error   { return nil }
func (b baseStateStore) GetRunTags(string) ([]string, error) { return nil, nil }
func (b baseStateStore) AddRunTag(string, string) error      { return nil }
func (b baseStateStore) RemoveRunTag(string, string) error   { return nil }
func (b baseStateStore) UpdateRunPID(string, int) error      { return nil }
func (b baseStateStore) UpdateRunHeartbeat(string) error                { return nil }
func (b baseStateStore) ReapOrphans(time.Duration) (int, error)         { return 0, nil }
func (b baseStateStore) RecordStepAttempt(*state.StepAttemptRecord) error {
	return nil
}
func (b baseStateStore) GetStepAttempts(string, string) ([]state.StepAttemptRecord, error) {
	return nil, nil
}
func (b baseStateStore) SaveChatSession(*state.ChatSession) error { return nil }
func (b baseStateStore) GetChatSession(string) (*state.ChatSession, error) {
	return nil, errors.New("not found")
}
func (b baseStateStore) ListChatSessions(string) ([]state.ChatSession, error) { return nil, nil }
func (b baseStateStore) SaveStepVisitCount(string, string, int) error        { return nil }
func (b baseStateStore) GetStepVisitCount(string, string) (int, error)       { return 0, nil }
func (b baseStateStore) SaveCheckpoint(*state.CheckpointRecord) error        { return nil }
func (b baseStateStore) GetCheckpoint(string, string) (*state.CheckpointRecord, error) {
	return nil, nil
}
func (b baseStateStore) DeleteCheckpointsAfterStep(string, int) error            { return nil }
func (b baseStateStore) GetCheckpoints(string) ([]state.CheckpointRecord, error) { return nil, nil }
func (b baseStateStore) CreateRunWithFork(string, string, string) (string, error) {
	return "", nil
}
func (b baseStateStore) SetParentRun(string, string, string) error { return nil }
func (b baseStateStore) SetRunComposition(string, string, string, string, *int, *int) error {
	return nil
}
func (b baseStateStore) GetSubtreeTokens(string) (int64, error) { return 0, nil }
func (b baseStateStore) GetChildRuns(string) ([]state.RunRecord, error)     { return nil, nil }
func (b baseStateStore) RecordDecision(*state.DecisionRecord) error           { return nil }
func (b baseStateStore) GetDecisions(string) ([]*state.DecisionRecord, error) { return nil, nil }
func (b baseStateStore) GetDecisionsByStep(string, string) ([]*state.DecisionRecord, error) {
	return nil, nil
}
func (b baseStateStore) GetDecisionsFiltered(string, state.DecisionQueryOptions) ([]*state.DecisionRecord, error) {
	return nil, nil
}
func (b baseStateStore) GetMostRecentRunID() (string, error)              { return "", nil }
func (b baseStateStore) RunExists(string) (bool, error)                   { return false, nil }
func (b baseStateStore) GetRunStatus(string) (string, error)              { return "", nil }
func (b baseStateStore) ListPipelineNamesByStatus(string) ([]string, error) {
	return nil, nil
}
func (b baseStateStore) BackfillRunTokens() (int64, error) { return 0, nil }
func (b baseStateStore) GetEventAggregateStats(string) (*state.EventAggregateStats, error) {
	return &state.EventAggregateStats{}, nil
}
func (b baseStateStore) GetAuditEvents([]string, int, int) ([]state.LogRecord, error) {
	return nil, nil
}
func (b baseStateStore) CreateWebhook(*state.Webhook) (int64, error)        { return 0, nil }
func (b baseStateStore) ListWebhooks() ([]*state.Webhook, error)            { return nil, nil }
func (b baseStateStore) GetWebhook(int64) (*state.Webhook, error)           { return nil, nil }
func (b baseStateStore) UpdateWebhook(*state.Webhook) error                 { return nil }
func (b baseStateStore) DeleteWebhook(int64) error                          { return nil }
func (b baseStateStore) RecordWebhookDelivery(*state.WebhookDelivery) error { return nil }
func (b baseStateStore) GetWebhookDeliveries(int64, int) ([]*state.WebhookDelivery, error) {
	return nil, nil
}
func (b baseStateStore) RecordOutcome(string, string, string, string, string, string, map[string]any) error {
	return nil
}
func (b baseStateStore) GetOutcomes(string) ([]state.OutcomeRecord, error) { return nil, nil }
func (b baseStateStore) GetOutcomesByValue(string, string) ([]state.OutcomeRecord, error) {
	return nil, nil
}
func (b baseStateStore) RecordOrchestrationDecision(*state.OrchestrationDecision) error {
	return nil
}
func (b baseStateStore) UpdateOrchestrationOutcome(string, string, int, int64) error { return nil }
func (b baseStateStore) GetOrchestrationStats(string) (*state.OrchestrationStats, error) {
	return nil, nil
}
func (b baseStateStore) ListOrchestrationDecisionSummary(int) ([]state.OrchestrationDecisionSummary, error) {
	return nil, nil
}

// EvolutionStore stubs (epic #1565 PRE-5).
func (b baseStateStore) RecordEval(state.PipelineEvalRecord) error { return nil }
func (b baseStateStore) GetEvalsForPipeline(string, int) ([]state.PipelineEvalRecord, error) {
	return nil, nil
}
func (b baseStateStore) CreatePipelineVersion(state.PipelineVersionRecord) error { return nil }
func (b baseStateStore) ActivateVersion(string, int) error                       { return nil }
func (b baseStateStore) GetActiveVersion(string) (*state.PipelineVersionRecord, error) {
	return nil, nil
}
func (b baseStateStore) ListPipelineVersions(string) ([]state.PipelineVersionRecord, error) {
	return nil, nil
}
func (b baseStateStore) CreateProposal(state.EvolutionProposalRecord) (int64, error) { return 0, nil }
func (b baseStateStore) DecideProposal(int64, state.EvolutionProposalStatus, string) error {
	return nil
}
func (b baseStateStore) GetProposal(int64) (*state.EvolutionProposalRecord, error) { return nil, nil }
func (b baseStateStore) ListProposalsByStatus(state.EvolutionProposalStatus, int) ([]state.EvolutionProposalRecord, error) {
	return nil, nil
}
func (b baseStateStore) LastProposalAt(string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// WorksourceStore stubs (epic #1565 PRE-5).
func (b baseStateStore) CreateBinding(state.WorksourceBindingRecord) (int64, error) { return 0, nil }
func (b baseStateStore) UpdateBinding(state.WorksourceBindingRecord) error          { return nil }
func (b baseStateStore) DeactivateBinding(int64) error                              { return nil }
func (b baseStateStore) GetBinding(int64) (*state.WorksourceBindingRecord, error)   { return nil, nil }
func (b baseStateStore) ListBindings(string, string) ([]state.WorksourceBindingRecord, error) {
	return nil, nil
}
func (b baseStateStore) ListActiveBindings() ([]state.WorksourceBindingRecord, error) {
	return nil, nil
}

// ScheduleStore stubs (epic #1565 PRE-5).
func (b baseStateStore) CreateSchedule(state.ScheduleRecord) (int64, error) { return 0, nil }
func (b baseStateStore) UpdateScheduleNextFire(int64, time.Time, string) error {
	return nil
}
func (b baseStateStore) DeactivateSchedule(int64) error                  { return nil }
func (b baseStateStore) GetSchedule(int64) (*state.ScheduleRecord, error) { return nil, nil }
func (b baseStateStore) ListSchedules() ([]state.ScheduleRecord, error)   { return nil, nil }
func (b baseStateStore) ListDueSchedules(time.Time) ([]state.ScheduleRecord, error) {
	return nil, nil
}

// Compile-time check: baseStateStore must satisfy state.StateStore.
var _ state.StateStore = baseStateStore{}

// ---------------------------------------------------------------------------
// mockStateStore — overrides only GetRunningRuns and ListRuns.
// ---------------------------------------------------------------------------

type mockStateStore struct {
	baseStateStore
	runningRuns    []state.RunRecord
	runningRunsErr error
	listRuns       []state.RunRecord
	listRunsErr    error
	listRunsOpts   state.ListRunsOptions // captured for assertions
}

func (m *mockStateStore) GetRunningRuns() ([]state.RunRecord, error) {
	return m.runningRuns, m.runningRunsErr
}

func (m *mockStateStore) ListRuns(opts state.ListRunsOptions) ([]state.RunRecord, error) {
	m.listRunsOpts = opts
	return m.listRuns, m.listRunsErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func timeAt(hour, min int) time.Time {
	// Use a fixed recent date for tests that don't depend on time-based filtering.
	return time.Date(2026, 3, 6, hour, min, 0, 0, time.UTC)
}

// recentTimeAt returns a time from today with the given minute offset from now,
// guaranteeing the result is within the staleRunCutoff window.
func recentTimeAt(minutesAgo int) time.Time {
	return time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDefaultPipelineDataProvider_FetchRunningPipelines(t *testing.T) {
	startedOlder := recentTimeAt(30) // 30 minutes ago — within staleRunCutoff
	startedNewer := recentTimeAt(10) // 10 minutes ago

	mock := &mockStateStore{
		runningRuns: []state.RunRecord{
			{
				RunID:        "run-1",
				PipelineName: "speckit-flow",
				BranchName:   "feat/speckit",
				StartedAt:    startedOlder,
				Status:       "running",
			},
			{
				RunID:        "run-2",
				PipelineName: "wave-evolve",
				BranchName:   "feat/evolve",
				StartedAt:    startedNewer,
				Status:       "running",
			},
		},
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchRunningPipelines()
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Verify field mapping — newest first after sort
	assert.Equal(t, "run-2", got[0].RunID)
	assert.Equal(t, "wave-evolve", got[0].Name)
	assert.Equal(t, "feat/evolve", got[0].BranchName)

	assert.Equal(t, "run-1", got[1].RunID)
	assert.Equal(t, "speckit-flow", got[1].Name)
	assert.Equal(t, "feat/speckit", got[1].BranchName)
}

func TestDefaultPipelineDataProvider_FetchRunningPipelines_SortOrder(t *testing.T) {
	// Records arrive oldest-first; verify result is newest-first.
	mock := &mockStateStore{
		runningRuns: []state.RunRecord{
			{RunID: "old", StartedAt: recentTimeAt(30)},
			{RunID: "new", StartedAt: recentTimeAt(5)},
		},
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchRunningPipelines()
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "new", got[0].RunID)
	assert.Equal(t, "old", got[1].RunID)
}

func TestDefaultPipelineDataProvider_FetchRunningPipelines_FiltersStale(t *testing.T) {
	// Runs without a PID that are older than staleRunCutoff should be filtered out.
	mock := &mockStateStore{
		runningRuns: []state.RunRecord{
			{RunID: "fresh", StartedAt: recentTimeAt(5)},                                    // 5 min ago — kept
			{RunID: "stale", StartedAt: time.Now().Add(-(staleRunCutoff + 10*time.Minute))}, // well past cutoff — filtered
		},
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchRunningPipelines()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "fresh", got[0].RunID)
}

func TestDefaultPipelineDataProvider_FetchFinishedPipelines(t *testing.T) {
	completedAt := timeAt(10, 30)
	cancelledAt := timeAt(11, 15)
	mock := &mockStateStore{
		listRuns: []state.RunRecord{
			{RunID: "r1", PipelineName: "p1", Status: "running", StartedAt: timeAt(10, 0)},
			{RunID: "r2", PipelineName: "p2", Status: "completed", StartedAt: timeAt(9, 0), CompletedAt: &completedAt, BranchName: "b2"},
			{RunID: "r3", PipelineName: "p3", Status: "failed", StartedAt: timeAt(8, 0), CompletedAt: &completedAt, BranchName: "b3"},
			{RunID: "r4", PipelineName: "p4", Status: "cancelled", StartedAt: timeAt(7, 0), CancelledAt: &cancelledAt, BranchName: "b4"},
			{RunID: "r5", PipelineName: "p5", Status: "pending", StartedAt: timeAt(6, 0)},
		},
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchFinishedPipelines(10)
	require.NoError(t, err)

	// Only terminal statuses: completed, failed, cancelled
	require.Len(t, got, 3)

	assert.Equal(t, "r2", got[0].RunID)
	assert.Equal(t, "completed", got[0].Status)
	assert.Equal(t, "p2", got[0].Name)
	assert.Equal(t, "b2", got[0].BranchName)

	assert.Equal(t, "r3", got[1].RunID)
	assert.Equal(t, "failed", got[1].Status)

	assert.Equal(t, "r4", got[2].RunID)
	assert.Equal(t, "cancelled", got[2].Status)
}

func TestDefaultPipelineDataProvider_FetchFinishedPipelines_Duration(t *testing.T) {
	tests := []struct {
		name         string
		record       state.RunRecord
		wantDuration time.Duration
		wantComplAt  time.Time
	}{
		{
			name: "completed — Duration = CompletedAt - StartedAt",
			record: state.RunRecord{
				RunID:       "r1",
				Status:      "completed",
				StartedAt:   timeAt(10, 0),
				CompletedAt: timePtr(timeAt(10, 45)),
			},
			wantDuration: 45 * time.Minute,
			wantComplAt:  timeAt(10, 45),
		},
		{
			name: "cancelled — Duration = CancelledAt - StartedAt",
			record: state.RunRecord{
				RunID:       "r2",
				Status:      "cancelled",
				StartedAt:   timeAt(14, 0),
				CancelledAt: timePtr(timeAt(14, 20)),
			},
			wantDuration: 20 * time.Minute,
			wantComplAt:  timeAt(14, 20),
		},
		{
			name: "failed with CompletedAt",
			record: state.RunRecord{
				RunID:       "r3",
				Status:      "failed",
				StartedAt:   timeAt(9, 0),
				CompletedAt: timePtr(timeAt(9, 10)),
			},
			wantDuration: 10 * time.Minute,
			wantComplAt:  timeAt(9, 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockStateStore{
				listRuns: []state.RunRecord{tt.record},
			}
			provider := NewDefaultPipelineDataProvider(mock, "")
			got, err := provider.FetchFinishedPipelines(10)
			require.NoError(t, err)
			require.Len(t, got, 1)

			assert.Equal(t, tt.wantDuration, got[0].Duration)
			assert.Equal(t, tt.wantComplAt, got[0].CompletedAt)
		})
	}
}

func TestDefaultPipelineDataProvider_FetchFinishedPipelines_Limit(t *testing.T) {
	completedAt := timeAt(12, 0)
	records := make([]state.RunRecord, 10)
	for i := range records {
		records[i] = state.RunRecord{
			RunID:       "r" + string(rune('A'+i)),
			Status:      "completed",
			StartedAt:   timeAt(10, i),
			CompletedAt: &completedAt,
		}
	}

	mock := &mockStateStore{listRuns: records}
	provider := NewDefaultPipelineDataProvider(mock, "")

	got, err := provider.FetchFinishedPipelines(3)
	require.NoError(t, err)
	assert.Len(t, got, 3)

	// Verify ListRuns was called with limit*3
	assert.Equal(t, 9, mock.listRunsOpts.Limit)
}

func TestDefaultPipelineDataProvider_FetchAvailablePipelines(t *testing.T) {
	dir := t.TempDir()
	yaml := `kind: WavePipeline
metadata:
  name: test-pipeline
  description: A test pipeline
input:
  source: cli
steps:
  - id: step1
    persona: navigator
`
	err := os.WriteFile(filepath.Join(dir, "test-pipeline.yaml"), []byte(yaml), 0644)
	require.NoError(t, err)

	mock := &mockStateStore{}
	provider := NewDefaultPipelineDataProvider(mock, dir)

	got, err := provider.FetchAvailablePipelines()
	require.NoError(t, err)
	require.Len(t, got, 1)

	assert.Equal(t, "test-pipeline", got[0].Name)
	assert.Equal(t, "A test pipeline", got[0].Description)
	assert.Equal(t, 1, got[0].StepCount)
}

func TestDefaultPipelineDataProvider_FetchRunningPipelines_Error(t *testing.T) {
	mock := &mockStateStore{
		runningRunsErr: errors.New("database connection lost"),
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchRunningPipelines()

	assert.Nil(t, got)
	assert.EqualError(t, err, "database connection lost")
}

func TestDefaultPipelineDataProvider_FetchFinishedPipelines_Error(t *testing.T) {
	mock := &mockStateStore{
		listRunsErr: errors.New("query timeout"),
	}

	provider := NewDefaultPipelineDataProvider(mock, "")
	got, err := provider.FetchFinishedPipelines(5)

	assert.Nil(t, got)
	assert.EqualError(t, err, "query timeout")
}
