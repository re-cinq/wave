package state

import "time"

// RunStore is the domain-scoped persistence surface for pipeline + step
// lifecycle: runs, cancellation, tags, parent/child linkage, checkpoints,
// retros, decisions, outcomes, orchestration decisions, performance metrics,
// progress snapshots, step attempts, and visit counts.
//
// Consumers that only touch run/step lifecycle data should depend on this
// interface rather than the aggregate StateStore.
type RunStore interface {
	// Pipeline state
	SavePipelineState(id string, status string, input string) error
	GetPipelineState(id string) (*PipelineStateRecord, error)
	ListRecentPipelines(limit int) ([]PipelineStateRecord, error)

	// Step state
	SaveStepState(pipelineID string, stepID string, state StepState, err string) error
	GetStepStates(pipelineID string) ([]StepStateRecord, error)
	SaveStepVisitCount(pipelineID string, stepID string, count int) error
	GetStepVisitCount(pipelineID string, stepID string) (int, error)
	RecordStepAttempt(record *StepAttemptRecord) error
	GetStepAttempts(runID string, stepID string) ([]StepAttemptRecord, error)

	// Run tracking
	CreateRun(pipelineName string, input string) (string, error)
	CreateRunWithLimit(pipelineName string, input string, maxConcurrent int) (string, error)
	CreateRunWithFork(pipelineName, input, forkedFromRunID string) (string, error)
	UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
	UpdateRunBranch(runID string, branch string) error
	UpdateRunPID(runID string, pid int) error
	UpdateRunHeartbeat(runID string) error
	ReapOrphans(staleAfter time.Duration) (int, error)
	GetRun(runID string) (*RunRecord, error)
	GetRunningRuns() ([]RunRecord, error)
	ListRuns(opts ListRunsOptions) ([]RunRecord, error)
	DeleteRun(runID string) error
	GetMostRecentRunID() (string, error)
	RunExists(runID string) (bool, error)
	GetRunStatus(runID string) (string, error)
	ListPipelineNamesByStatus(status string) ([]string, error)
	BackfillRunTokens() (int64, error)

	// Cancellation
	RequestCancellation(runID string, force bool) error
	CheckCancellation(runID string) (*CancellationRecord, error)
	ClearCancellation(runID string) error

	// Tags
	SetRunTags(runID string, tags []string) error
	GetRunTags(runID string) ([]string, error)
	AddRunTag(runID string, tag string) error
	RemoveRunTag(runID string, tag string) error

	// Parent/child run linkage
	SetParentRun(childRunID, parentRunID, stepID string) error
	SetRunComposition(childRunID, runKind, subPipelineRef, iterateMode string, iterateIndex, iterateTotal *int) error
	GetSubtreeTokens(rootRunID string) (int64, error)
	GetChildRuns(parentRunID string) ([]RunRecord, error)

	// Checkpoints (fork/rewind)
	SaveCheckpoint(record *CheckpointRecord) error
	GetCheckpoint(runID, stepID string) (*CheckpointRecord, error)
	GetCheckpoints(runID string) ([]CheckpointRecord, error)
	DeleteCheckpointsAfterStep(runID string, stepIndex int) error

	// Retrospectives
	SaveRetrospective(record *RetrospectiveRecord) error
	GetRetrospective(runID string) (*RetrospectiveRecord, error)
	ListRetrospectives(opts ListRetrosOptions) ([]RetrospectiveRecord, error)
	DeleteRetrospective(runID string) error
	UpdateRetrospectiveSmoothness(runID string, smoothness string) error
	UpdateRetrospectiveStatus(runID string, status string) error

	// Decision log
	RecordDecision(record *DecisionRecord) error
	GetDecisions(runID string) ([]*DecisionRecord, error)
	GetDecisionsByStep(runID, stepID string) ([]*DecisionRecord, error)
	GetDecisionsFiltered(runID string, opts DecisionQueryOptions) ([]*DecisionRecord, error)

	// Outcomes
	RecordOutcome(runID, stepID, outcomeType, label, value, description string, metadata map[string]any) error
	GetOutcomes(runID string) ([]OutcomeRecord, error)
	GetOutcomesByValue(outcomeType, value string) ([]OutcomeRecord, error)

	// Orchestration decisions
	RecordOrchestrationDecision(record *OrchestrationDecision) error
	UpdateOrchestrationOutcome(runID string, outcome string, tokensUsed int, durationMs int64) error
	GetOrchestrationStats(pipelineName string) (*OrchestrationStats, error)
	ListOrchestrationDecisionSummary(limit int) ([]OrchestrationDecisionSummary, error)

	// Performance metrics
	RecordPerformanceMetric(metric *PerformanceMetricRecord) error
	GetPerformanceMetrics(runID string, stepID string) ([]PerformanceMetricRecord, error)
	GetStepPerformanceStats(pipelineName string, stepID string, since time.Time) (*StepPerformanceStats, error)
	GetRecentPerformanceHistory(opts PerformanceQueryOptions) ([]PerformanceMetricRecord, error)
	CleanupOldPerformanceMetrics(olderThan time.Duration) (int, error)

	// Progress
	SaveProgressSnapshot(runID string, stepID string, progress int, action string, etaMs int64, validationPhase string, compactionStats string) error
	GetProgressSnapshots(runID string, stepID string, limit int) ([]ProgressSnapshotRecord, error)
	UpdateStepProgress(runID string, stepID string, persona string, state string, progress int, action string, message string, etaMs int64, tokens int) error
	GetStepProgress(stepID string) (*StepProgressRecord, error)
	GetAllStepProgress(runID string) ([]StepProgressRecord, error)
	UpdatePipelineProgress(runID string, totalSteps int, completedSteps int, currentStepIndex int, overallProgress int, etaMs int64) error
	GetPipelineProgress(runID string) (*PipelineProgressRecord, error)
}
