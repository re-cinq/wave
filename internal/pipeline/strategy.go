package pipeline

import "context"

// StrategyExecutor is the unified interface for step-level fan-out / dispatch
// strategies. Matrix expansion, iterate / aggregate / branch / loop / gate
// composition primitives, and bare sub-pipeline launches all implement this
// interface.
//
// A StrategyExecutor takes a step that has already been routed through the
// scheduler and runs whatever expansion or sub-execution the step's shape
// requires. Implementations own context propagation to children, worker-pool
// management (when applicable), and result aggregation back into the parent
// PipelineExecution.
//
// The interface is intentionally narrow. Authoring a new strategy is a single
// file: implement Execute and add a clause to the registry below.
type StrategyExecutor interface {
	Execute(ctx context.Context, execution *PipelineExecution, step *Step) error
}

// strategyKind classifies a registry entry so callers that have already
// filtered to composition steps (nested loop sub-steps, the standalone
// CompositionExecutor) can skip non-composition shapes.
type strategyKind int

const (
	// strategyKindStepLevel applies at the executeStep dispatch boundary
	// only — concurrency and matrix expansion.
	strategyKindStepLevel strategyKind = iota
	// strategyKindComposition applies inside executeCompositionStep too —
	// gate, iterate, aggregate, branch, loop, sub-pipeline.
	strategyKindComposition
)

// strategyEntry pairs a match predicate with the strategy that handles it.
// Order in the registry determines dispatch precedence: the first matching
// entry wins. This mirrors the original dispatch order in executeStep and
// executeCompositionStep so behaviour is preserved exactly.
type strategyEntry struct {
	kind  strategyKind
	match func(step *Step) bool
	build func(e *DefaultPipelineExecutor) StrategyExecutor
}

// strategyRegistry is the ordered list of step-strategy dispatch rules.
// Adding a new primitive is one entry here plus a small struct implementing
// StrategyExecutor.
var strategyRegistry = []strategyEntry{
	{
		kind:  strategyKindStepLevel,
		match: func(step *Step) bool { return step.Concurrency > 1 },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return concurrencyStrategy{e: e} },
	},
	{
		kind:  strategyKindStepLevel,
		match: func(step *Step) bool { return step.Strategy != nil && step.Strategy.Type == "matrix" },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return matrixStrategy{e: e} },
	},
	{
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.Gate != nil },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return gateStrategy{e: e} },
	},
	{
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.Iterate != nil },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return iterateStrategy{e: e} },
	},
	{
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.Aggregate != nil },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return aggregateStrategy{e: e} },
	},
	{
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.Branch != nil },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return branchStrategy{e: e} },
	},
	{
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.Loop != nil },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return loopStrategy{e: e} },
	},
	{
		// Bare sub-pipeline launch must come last among composition shapes
		// because Iterate / Branch / Loop steps may also set SubPipeline.
		kind:  strategyKindComposition,
		match: func(step *Step) bool { return step.SubPipeline != "" },
		build: func(e *DefaultPipelineExecutor) StrategyExecutor { return subPipelineStrategy{e: e} },
	},
}

// selectStrategy returns the StrategyExecutor that should handle the given
// step, or nil when no strategy applies (the step is a regular persona /
// command step and should run through the standard adapter pipeline).
func selectStrategy(e *DefaultPipelineExecutor, step *Step) StrategyExecutor {
	for _, entry := range strategyRegistry {
		if entry.match(step) {
			return entry.build(e)
		}
	}
	return nil
}

// selectCompositionStrategy returns the StrategyExecutor for a composition
// step, skipping the step-level entries (concurrency, matrix). Used by
// callers that have already routed into the composition dispatch boundary
// (executor_composition.go and the standalone CompositionExecutor).
func selectCompositionStrategy(e *DefaultPipelineExecutor, step *Step) StrategyExecutor {
	for _, entry := range strategyRegistry {
		if entry.kind != strategyKindComposition {
			continue
		}
		if entry.match(step) {
			return entry.build(e)
		}
	}
	return nil
}

// concurrencyStrategy dispatches to executeConcurrentStep.
type concurrencyStrategy struct{ e *DefaultPipelineExecutor }

func (s concurrencyStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeConcurrentStep(ctx, execution, step)
}

// matrixStrategy dispatches to executeMatrixStep.
type matrixStrategy struct{ e *DefaultPipelineExecutor }

func (s matrixStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeMatrixStep(ctx, execution, step)
}

// iterateStrategy dispatches to executeIterateInDAG.
type iterateStrategy struct{ e *DefaultPipelineExecutor }

func (s iterateStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeIterateInDAG(ctx, execution, step)
}

// aggregateStrategy dispatches to executeAggregateInDAG.
type aggregateStrategy struct{ e *DefaultPipelineExecutor }

func (s aggregateStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeAggregateInDAG(ctx, execution, step)
}

// branchStrategy dispatches to executeBranchInDAG.
type branchStrategy struct{ e *DefaultPipelineExecutor }

func (s branchStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeBranchInDAG(ctx, execution, step)
}

// loopStrategy dispatches to executeLoopInDAG.
type loopStrategy struct{ e *DefaultPipelineExecutor }

func (s loopStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeLoopInDAG(ctx, execution, step)
}

// gateStrategy dispatches to executeGateInDAG.
type gateStrategy struct{ e *DefaultPipelineExecutor }

func (s gateStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return s.e.executeGateInDAG(ctx, execution, step)
}

// subPipelineStrategy launches a bare sub-pipeline step.
type subPipelineStrategy struct{ e *DefaultPipelineExecutor }

func (s subPipelineStrategy) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	input := s.e.resolveSubPipelineInput(execution, step)
	return s.e.runNamedSubPipeline(ctx, execution, step, step.SubPipeline, input, compositionLaunchInfo{kind: "sub_pipeline_child"})
}
