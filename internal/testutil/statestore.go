package testutil

import (
	"errors"
	"sync"
	"time"

	"github.com/recinq/wave/internal/state"
)

// MockStateStore implements state.StateStore with configurable behavior.
// All methods default to no-op (returning zero values/nil errors).
// Use functional options to override specific methods.
type MockStateStore struct {
	mu sync.RWMutex

	// Overridable method implementations
	savePipelineState             func(id, status, input string) error
	savePipelineStateLocked       bool // if true, savePipelineState manages its own locking
	saveStepState                 func(pipelineID, stepID string, st state.StepState, errMsg string) error
	getPipelineState              func(id string) (*state.PipelineStateRecord, error)
	getStepStates                 func(pipelineID string) ([]state.StepStateRecord, error)
	listRecentPipelines           func(limit int) ([]state.PipelineStateRecord, error)
	close                         func() error
	createRun                     func(pipelineName, input string) (string, error)
	updateRunStatus               func(runID, status, currentStep string, tokens int) error
	updateRunBranch               func(runID, branch string) error
	getRun                        func(runID string) (*state.RunRecord, error)
	getRunningRuns                func() ([]state.RunRecord, error)
	listRuns                      func(opts state.ListRunsOptions) ([]state.RunRecord, error)
	deleteRun                     func(runID string) error
	logEvent                      func(runID, stepID, st, persona, message string, tokens int, durationMs int64) error
	getEvents                     func(runID string, opts state.EventQueryOptions) ([]state.LogRecord, error)
	registerArtifact              func(runID, stepID, name, path, artifactType string, sizeBytes int64) error
	getArtifacts                  func(runID, stepID string) ([]state.ArtifactRecord, error)
	requestCancellation           func(runID string, force bool) error
	checkCancellation             func(runID string) (*state.CancellationRecord, error)
	clearCancellation             func(runID string) error
	recordPerformanceMetric       func(metric *state.PerformanceMetricRecord) error
	getPerformanceMetrics         func(runID, stepID string) ([]state.PerformanceMetricRecord, error)
	getStepPerformanceStats       func(pipelineName, stepID string, since time.Time) (*state.StepPerformanceStats, error)
	getRecentPerformanceHistory   func(opts state.PerformanceQueryOptions) ([]state.PerformanceMetricRecord, error)
	cleanupOldPerformanceMetrics  func(olderThan time.Duration) (int, error)
	saveProgressSnapshot          func(runID, stepID string, progress int, action string, etaMs int64, validationPhase, compactionStats string) error
	getProgressSnapshots          func(runID, stepID string, limit int) ([]state.ProgressSnapshotRecord, error)
	updateStepProgress            func(runID, stepID, persona, st string, progress int, action, message string, etaMs int64, tokens int) error
	getStepProgress               func(stepID string) (*state.StepProgressRecord, error)
	getAllStepProgress             func(runID string) ([]state.StepProgressRecord, error)
	updatePipelineProgress        func(runID string, totalSteps, completedSteps, currentStepIndex, overallProgress int, etaMs int64) error
	getPipelineProgress           func(runID string) (*state.PipelineProgressRecord, error)
	saveArtifactMetadata          func(artifactID int64, runID, stepID, previewText, mimeType, encoding, metadataJSON string) error
	getArtifactMetadata           func(artifactID int64) (*state.ArtifactMetadataRecord, error)
	setRunTags                    func(runID string, tags []string) error
	getRunTags                    func(runID string) ([]string, error)
	addRunTag                     func(runID, tag string) error
	removeRunTag                  func(runID, tag string) error
	updateRunPID                  func(runID string, pid int) error
	recordStepAttempt             func(record *state.StepAttemptRecord) error
	getStepAttempts               func(runID, stepID string) ([]state.StepAttemptRecord, error)
	saveChatSession               func(session *state.ChatSession) error
	getChatSession                func(sessionID string) (*state.ChatSession, error)
	listChatSessions              func(runID string) ([]state.ChatSession, error)
	recordOntologyUsage           func(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error
	getOntologyStats              func(contextName string) (*state.OntologyStats, error)
	getOntologyStatsAll           func() ([]state.OntologyStats, error)

	// Internal storage for default implementations
	pipelineStates map[string]*state.PipelineStateRecord
	stepStates     map[string][]state.StepStateRecord
}

// MockStateStoreOption configures a MockStateStore.
type MockStateStoreOption func(*MockStateStore)

// NewMockStateStore creates a new MockStateStore with default no-op behavior.
// The default SavePipelineState, GetPipelineState, SaveStepState, and GetStepStates
// use in-memory maps (matching the original executor_test.go behavior).
func NewMockStateStore(opts ...MockStateStoreOption) *MockStateStore {
	m := &MockStateStore{
		pipelineStates: make(map[string]*state.PipelineStateRecord),
		stepStates:     make(map[string][]state.StepStateRecord),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *MockStateStore) SavePipelineState(id, status, input string) error {
	if m.savePipelineState != nil {
		return m.savePipelineState(id, status, input)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.pipelineStates[id] = &state.PipelineStateRecord{
		PipelineID: id,
		Name:       id,
		Status:     status,
		Input:      input,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return nil
}

func (m *MockStateStore) GetPipelineState(id string) (*state.PipelineStateRecord, error) {
	if m.getPipelineState != nil {
		return m.getPipelineState(id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, exists := m.pipelineStates[id]
	if !exists {
		return nil, errors.New("pipeline state not found")
	}
	return record, nil
}

func (m *MockStateStore) SaveStepState(pipelineID, stepID string, stepState state.StepState, errMsg string) error {
	if m.saveStepState != nil {
		return m.saveStepState(pipelineID, stepID, stepState, errMsg)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stepRecord := state.StepStateRecord{
		StepID:     stepID,
		PipelineID: pipelineID,
		State:      stepState,
	}
	m.stepStates[pipelineID] = append(m.stepStates[pipelineID], stepRecord)
	return nil
}

func (m *MockStateStore) GetStepStates(pipelineID string) ([]state.StepStateRecord, error) {
	if m.getStepStates != nil {
		return m.getStepStates(pipelineID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stepStates[pipelineID], nil
}

func (m *MockStateStore) ListRecentPipelines(limit int) ([]state.PipelineStateRecord, error) {
	if m.listRecentPipelines != nil {
		return m.listRecentPipelines(limit)
	}
	return nil, nil
}

func (m *MockStateStore) Close() error {
	if m.close != nil {
		return m.close()
	}
	return nil
}

func (m *MockStateStore) CreateRun(pipelineName, input string) (string, error) {
	if m.createRun != nil {
		return m.createRun(pipelineName, input)
	}
	return "", nil
}

func (m *MockStateStore) UpdateRunStatus(runID, status, currentStep string, tokens int) error {
	if m.updateRunStatus != nil {
		return m.updateRunStatus(runID, status, currentStep, tokens)
	}
	return nil
}

func (m *MockStateStore) UpdateRunBranch(runID, branch string) error {
	if m.updateRunBranch != nil {
		return m.updateRunBranch(runID, branch)
	}
	return nil
}

func (m *MockStateStore) GetRun(runID string) (*state.RunRecord, error) {
	if m.getRun != nil {
		return m.getRun(runID)
	}
	return nil, nil
}

func (m *MockStateStore) GetRunningRuns() ([]state.RunRecord, error) {
	if m.getRunningRuns != nil {
		return m.getRunningRuns()
	}
	return nil, nil
}

func (m *MockStateStore) ListRuns(opts state.ListRunsOptions) ([]state.RunRecord, error) {
	if m.listRuns != nil {
		return m.listRuns(opts)
	}
	return nil, nil
}

func (m *MockStateStore) DeleteRun(runID string) error {
	if m.deleteRun != nil {
		return m.deleteRun(runID)
	}
	return nil
}

func (m *MockStateStore) LogEvent(runID, stepID, st, persona, message string, tokens int, durationMs int64) error {
	if m.logEvent != nil {
		return m.logEvent(runID, stepID, st, persona, message, tokens, durationMs)
	}
	return nil
}

func (m *MockStateStore) GetEvents(runID string, opts state.EventQueryOptions) ([]state.LogRecord, error) {
	if m.getEvents != nil {
		return m.getEvents(runID, opts)
	}
	return nil, nil
}

func (m *MockStateStore) RegisterArtifact(runID, stepID, name, path, artifactType string, sizeBytes int64) error {
	if m.registerArtifact != nil {
		return m.registerArtifact(runID, stepID, name, path, artifactType, sizeBytes)
	}
	return nil
}

func (m *MockStateStore) GetArtifacts(runID, stepID string) ([]state.ArtifactRecord, error) {
	if m.getArtifacts != nil {
		return m.getArtifacts(runID, stepID)
	}
	return nil, nil
}

func (m *MockStateStore) RequestCancellation(runID string, force bool) error {
	if m.requestCancellation != nil {
		return m.requestCancellation(runID, force)
	}
	return nil
}

func (m *MockStateStore) CheckCancellation(runID string) (*state.CancellationRecord, error) {
	if m.checkCancellation != nil {
		return m.checkCancellation(runID)
	}
	return nil, nil
}

func (m *MockStateStore) ClearCancellation(runID string) error {
	if m.clearCancellation != nil {
		return m.clearCancellation(runID)
	}
	return nil
}

func (m *MockStateStore) RecordPerformanceMetric(metric *state.PerformanceMetricRecord) error {
	if m.recordPerformanceMetric != nil {
		return m.recordPerformanceMetric(metric)
	}
	return nil
}

func (m *MockStateStore) GetPerformanceMetrics(runID, stepID string) ([]state.PerformanceMetricRecord, error) {
	if m.getPerformanceMetrics != nil {
		return m.getPerformanceMetrics(runID, stepID)
	}
	return nil, nil
}

func (m *MockStateStore) GetStepPerformanceStats(pipelineName, stepID string, since time.Time) (*state.StepPerformanceStats, error) {
	if m.getStepPerformanceStats != nil {
		return m.getStepPerformanceStats(pipelineName, stepID, since)
	}
	return nil, nil
}

func (m *MockStateStore) GetRecentPerformanceHistory(opts state.PerformanceQueryOptions) ([]state.PerformanceMetricRecord, error) {
	if m.getRecentPerformanceHistory != nil {
		return m.getRecentPerformanceHistory(opts)
	}
	return nil, nil
}

func (m *MockStateStore) CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error) {
	if m.cleanupOldPerformanceMetrics != nil {
		return m.cleanupOldPerformanceMetrics(olderThan)
	}
	return 0, nil
}

func (m *MockStateStore) SaveProgressSnapshot(runID, stepID string, progress int, action string, etaMs int64, validationPhase, compactionStats string) error {
	if m.saveProgressSnapshot != nil {
		return m.saveProgressSnapshot(runID, stepID, progress, action, etaMs, validationPhase, compactionStats)
	}
	return nil
}

func (m *MockStateStore) GetProgressSnapshots(runID, stepID string, limit int) ([]state.ProgressSnapshotRecord, error) {
	if m.getProgressSnapshots != nil {
		return m.getProgressSnapshots(runID, stepID, limit)
	}
	return nil, nil
}

func (m *MockStateStore) UpdateStepProgress(runID, stepID, persona, st string, progress int, action, message string, etaMs int64, tokens int) error {
	if m.updateStepProgress != nil {
		return m.updateStepProgress(runID, stepID, persona, st, progress, action, message, etaMs, tokens)
	}
	return nil
}

func (m *MockStateStore) GetStepProgress(stepID string) (*state.StepProgressRecord, error) {
	if m.getStepProgress != nil {
		return m.getStepProgress(stepID)
	}
	return nil, nil
}

func (m *MockStateStore) GetAllStepProgress(runID string) ([]state.StepProgressRecord, error) {
	if m.getAllStepProgress != nil {
		return m.getAllStepProgress(runID)
	}
	return nil, nil
}

func (m *MockStateStore) UpdatePipelineProgress(runID string, totalSteps, completedSteps, currentStepIndex, overallProgress int, etaMs int64) error {
	if m.updatePipelineProgress != nil {
		return m.updatePipelineProgress(runID, totalSteps, completedSteps, currentStepIndex, overallProgress, etaMs)
	}
	return nil
}

func (m *MockStateStore) GetPipelineProgress(runID string) (*state.PipelineProgressRecord, error) {
	if m.getPipelineProgress != nil {
		return m.getPipelineProgress(runID)
	}
	return nil, nil
}

func (m *MockStateStore) SaveArtifactMetadata(artifactID int64, runID, stepID, previewText, mimeType, encoding, metadataJSON string) error {
	if m.saveArtifactMetadata != nil {
		return m.saveArtifactMetadata(artifactID, runID, stepID, previewText, mimeType, encoding, metadataJSON)
	}
	return nil
}

func (m *MockStateStore) GetArtifactMetadata(artifactID int64) (*state.ArtifactMetadataRecord, error) {
	if m.getArtifactMetadata != nil {
		return m.getArtifactMetadata(artifactID)
	}
	return nil, nil
}

func (m *MockStateStore) SetRunTags(runID string, tags []string) error {
	if m.setRunTags != nil {
		return m.setRunTags(runID, tags)
	}
	return nil
}

func (m *MockStateStore) GetRunTags(runID string) ([]string, error) {
	if m.getRunTags != nil {
		return m.getRunTags(runID)
	}
	return nil, nil
}

func (m *MockStateStore) AddRunTag(runID, tag string) error {
	if m.addRunTag != nil {
		return m.addRunTag(runID, tag)
	}
	return nil
}

func (m *MockStateStore) RemoveRunTag(runID, tag string) error {
	if m.removeRunTag != nil {
		return m.removeRunTag(runID, tag)
	}
	return nil
}

func (m *MockStateStore) UpdateRunPID(runID string, pid int) error {
	if m.updateRunPID != nil {
		return m.updateRunPID(runID, pid)
	}
	return nil
}

func (m *MockStateStore) RecordStepAttempt(record *state.StepAttemptRecord) error {
	if m.recordStepAttempt != nil {
		return m.recordStepAttempt(record)
	}
	return nil
}

func (m *MockStateStore) GetStepAttempts(runID, stepID string) ([]state.StepAttemptRecord, error) {
	if m.getStepAttempts != nil {
		return m.getStepAttempts(runID, stepID)
	}
	return nil, nil
}

func (m *MockStateStore) SaveChatSession(session *state.ChatSession) error {
	if m.saveChatSession != nil {
		return m.saveChatSession(session)
	}
	return nil
}

func (m *MockStateStore) GetChatSession(sessionID string) (*state.ChatSession, error) {
	if m.getChatSession != nil {
		return m.getChatSession(sessionID)
	}
	return nil, errors.New("not found")
}

func (m *MockStateStore) ListChatSessions(runID string) ([]state.ChatSession, error) {
	if m.listChatSessions != nil {
		return m.listChatSessions(runID)
	}
	return nil, nil
}

func (m *MockStateStore) RecordOntologyUsage(runID, stepID, contextName string, invariantCount int, status string, contractPassed *bool) error {
	if m.recordOntologyUsage != nil {
		return m.recordOntologyUsage(runID, stepID, contextName, invariantCount, status, contractPassed)
	}
	return nil
}

func (m *MockStateStore) GetOntologyStats(contextName string) (*state.OntologyStats, error) {
	if m.getOntologyStats != nil {
		return m.getOntologyStats(contextName)
	}
	return &state.OntologyStats{ContextName: contextName}, nil
}

func (m *MockStateStore) GetOntologyStatsAll() ([]state.OntologyStats, error) {
	if m.getOntologyStatsAll != nil {
		return m.getOntologyStatsAll()
	}
	return nil, nil
}

// Functional options for overriding specific methods.

func WithSavePipelineState(fn func(id, status, input string) error) MockStateStoreOption {
	return func(m *MockStateStore) { m.savePipelineState = fn }
}

func WithGetPipelineState(fn func(id string) (*state.PipelineStateRecord, error)) MockStateStoreOption {
	return func(m *MockStateStore) { m.getPipelineState = fn }
}

func WithSaveStepState(fn func(pipelineID, stepID string, st state.StepState, errMsg string) error) MockStateStoreOption {
	return func(m *MockStateStore) { m.saveStepState = fn }
}

func WithGetStepStates(fn func(pipelineID string) ([]state.StepStateRecord, error)) MockStateStoreOption {
	return func(m *MockStateStore) { m.getStepStates = fn }
}

func WithRecordStepAttempt(fn func(record *state.StepAttemptRecord) error) MockStateStoreOption {
	return func(m *MockStateStore) { m.recordStepAttempt = fn }
}

func WithGetStepAttempts(fn func(runID, stepID string) ([]state.StepAttemptRecord, error)) MockStateStoreOption {
	return func(m *MockStateStore) { m.getStepAttempts = fn }
}

func WithCreateRun(fn func(pipelineName, input string) (string, error)) MockStateStoreOption {
	return func(m *MockStateStore) { m.createRun = fn }
}

func WithUpdateRunStatus(fn func(runID, status, currentStep string, tokens int) error) MockStateStoreOption {
	return func(m *MockStateStore) { m.updateRunStatus = fn }
}

func WithLogEvent(fn func(runID, stepID, st, persona, message string, tokens int, durationMs int64) error) MockStateStoreOption {
	return func(m *MockStateStore) { m.logEvent = fn }
}
