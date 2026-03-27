# Tasks

## Phase 1: Schema Extensions

- [X] Task 1.1: Add graph types to `internal/pipeline/types.go` -- `EdgeConfig` struct with `Target` and `Condition` fields; add `Edges []EdgeConfig`, `MaxVisits int`, `Script string`, `Type string` fields to `Step`; add `MaxStepVisits int` to `Pipeline`
- [X] Task 1.2: Add `VisitCount int` field to `StepStateRecord` in `internal/state/store.go`; add `SaveStepVisitCount(pipelineID, stepID string, count int) error` and `GetStepVisitCount(pipelineID, stepID string) (int, error)` to `StateStore` interface
- [X] Task 1.3: Implement visit count persistence in `internal/state/sqlite.go` -- add `visit_count` column to `step_states` table migration, implement new interface methods

## Phase 2: Core Implementation

- [X] Task 2.1: Create `internal/pipeline/condition.go` -- `ConditionExpr` type with `Namespace`, `Key`, `Value` fields; `ParseCondition(expr string) (ConditionExpr, error)` parser for `outcome=X` and `context.K=V` forms; `EvaluateCondition(expr ConditionExpr, ctx *StepContext) bool` evaluator [P]
- [X] Task 2.2: Create `internal/pipeline/graph.go` -- `GraphWalker` struct with visit tracking, edge following, and step dispatch; `Walk(ctx, pipeline, execution)` method that replaces topological sort loop for graph-mode pipelines; `evaluateEdges(step, result)` to pick next step from edge list [P]
- [X] Task 2.3: Add `IsGraphPipeline(p *Pipeline) bool` detection to `internal/pipeline/dag.go`; add `ValidateGraph(p *Pipeline) error` that validates edge targets exist, max_visits > 0, conditional steps have edges, and at least one terminal path exists [P]
- [X] Task 2.4: Add `executeCommandStep(ctx, execution, step) error` to `internal/pipeline/executor.go` -- runs `step.Script` via `os/exec`, captures stdout/stderr, stores as step artifact, sets outcome based on exit code [P]

## Phase 3: Integration

- [X] Task 3.1: Modify `Execute()` in `internal/pipeline/executor.go` to detect graph-mode pipelines via `IsGraphPipeline()` and dispatch to `executeGraphPipeline()` which creates a `GraphWalker` and runs it
- [X] Task 3.2: Wire command step execution into the main `executeStep()` flow -- when `step.Exec.Type == "command"` or `step.Script != ""`, call `executeCommandStep()` instead of adapter execution
- [X] Task 3.3: Update `internal/pipeline/dryrun.go` validation to accept `type: conditional` steps, validate `edges` field structure, validate `max_visits` range, validate `script` is non-empty for command steps

## Phase 4: Safety Mechanisms

- [X] Task 4.1: Implement circuit breaker in `GraphWalker` -- track last 3 error messages per step, normalize (strip timestamps/line numbers), terminate if all 3 identical
- [X] Task 4.2: Implement graph-level `max_step_visits` enforcement -- total visits across all steps cannot exceed pipeline limit (default 50)

## Phase 5: Testing

- [X] Task 5.1: Write unit tests for condition parser in `internal/pipeline/condition_test.go` -- `outcome=success`, `outcome=failure`, `context.key=value`, empty (unconditional), malformed input [P]
- [X] Task 5.2: Write unit tests for graph walker in `internal/pipeline/graph_test.go` -- linear graph (no loops), diamond convergence, simple loop with max_visits, loop with conditional exit, circuit breaker trigger, max_step_visits enforcement [P]
- [X] Task 5.3: Write unit tests for command step execution -- stdout capture, stderr capture, non-zero exit code, timeout [P]
- [X] Task 5.4: Write integration test for backward compatibility -- existing DAG pipeline detected as DAG mode, executed via topological sort unchanged [P]
- [X] Task 5.5: Write integration test for end-to-end implement-test-fix cycle using mock adapter [P]
- [X] Task 5.6: Write unit tests for visit count state persistence -- save, retrieve, resume with existing counts [P]

## Phase 6: Validation

- [X] Task 6.1: Run `go test ./...` and `go test -race ./...` to verify all existing tests pass
- [ ] Task 6.2: Run `golangci-lint run ./...` to verify no lint issues
- [ ] Task 6.3: Verify existing pipeline YAML files load without validation errors by running dryrun against sample pipelines
