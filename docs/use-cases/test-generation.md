---
title: Test Generation
description: Analyze coverage gaps and generate comprehensive tests with edge case handling
---

# Test Generation

<div class="use-case-meta">
  <span class="complexity-badge intermediate">Intermediate</span>
  <span class="category-badge">Testing</span>
</div>

Analyze coverage gaps and generate comprehensive tests. Wave's test-gen pipeline identifies untested code, generates test cases, and verifies they compile and run.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Go project with existing test infrastructure
- Understanding of [gh-pr-review](/use-cases/gh-pr-review) pipeline (recommended)
- Familiarity with table-driven tests and mocking patterns

## Quick Start

```bash
wave run test-gen "generate tests for the pipeline package"
```

With `-o text`:

```
[10:00:01] → analyze-coverage (navigator)
[10:00:01]   analyze-coverage: Executing agent
[10:00:32] ✓ analyze-coverage completed (31.0s, 2.9k tokens)
[10:00:33] → generate-tests (craftsman)
[10:01:45] ✓ generate-tests completed (72.0s, 8.5k tokens)
[10:01:46] → verify-coverage (auditor)
[10:02:10] ✓ verify-coverage completed (24.0s, 1.8k tokens)

  ✓ Pipeline 'test-gen' completed successfully (2m 9s)
```

## Complete Pipeline

This is the full `test-gen` pipeline from `.wave/pipelines/test-gen.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: test-gen
  description: "Generate comprehensive test coverage"

input:
  source: cli

steps:
  - id: analyze-coverage
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze test coverage for: {{ input }}

        1. Run coverage analysis (go test -cover)
        2. Identify uncovered functions and branches
        3. Find edge cases not tested
        4. Map dependencies that need mocking

        Output as JSON:
        {
          "current_coverage": "X%",
          "uncovered_functions": [],
          "uncovered_branches": [],
          "edge_cases": [],
          "mock_requirements": []
        }
    output_artifacts:
      - name: coverage
        path: .wave/output/coverage-analysis.json
        type: json

  - id: generate-tests
    persona: craftsman
    dependencies: [analyze-coverage]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze-coverage
          artifact: coverage
          as: gaps
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Generate tests to improve coverage for: {{ input }}

        Requirements:
        1. Write table-driven tests where appropriate
        2. Cover happy path, error cases, and edge cases
        3. Use descriptive test names (TestFunction_Condition_Expected)
        4. Add mocks for external dependencies
        5. Include benchmarks for performance-critical code

        Follow existing test patterns in the codebase.
    handover:
      contract:
        type: test_suite
        command: "go test ./... -v"
        must_pass: false
        on_failure: retry
        max_retries: 3
    output_artifacts:
      - name: tests
        path: .wave/output/generated-tests.md
        type: markdown

  - id: verify-coverage
    persona: auditor
    dependencies: [generate-tests]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Verify the generated tests:

        1. Run coverage again - did it improve?
        2. Are tests meaningful (not just line coverage)?
        3. Do tests actually catch bugs?
        4. Are mocks appropriate and minimal?
        5. Is test code maintainable?

        Output: coverage delta and quality assessment
    output_artifacts:
      - name: verification
        path: .wave/output/coverage-verification.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces three artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `coverage` | `.wave/output/coverage-analysis.json` | Coverage analysis with gaps identified |
| `tests` | `.wave/output/generated-tests.md` | Generated test code and explanations |
| `verification` | `.wave/output/coverage-verification.md` | Coverage improvement verification |

### Example Output

The pipeline produces `.wave/output/generated-tests.md`:

```markdown
# Generated Tests for pipeline Package

## Coverage Improvement

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Line Coverage | 62% | 78% | +16% |
| Branch Coverage | 45% | 68% | +23% |
| Functions Covered | 18/25 | 23/25 | +5 |

## New Test Files

### executor_test.go

` ` `go
func TestExecutor_Run_Success(t *testing.T) {
    tests := []struct {
        name     string
        pipeline Pipeline
        input    string
        want     *Result
        wantErr  bool
    }{
        {
            name:     "single step pipeline",
            pipeline: singleStepPipeline(),
            input:    "test input",
            want:     &Result{Steps: 1, Status: "completed"},
            wantErr:  false,
        },
        {
            name:     "multi step with dependencies",
            pipeline: multiStepPipeline(),
            input:    "test input",
            want:     &Result{Steps: 3, Status: "completed"},
            wantErr:  false,
        },
        {
            name:     "empty input",
            pipeline: singleStepPipeline(),
            input:    "",
            want:     nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            e := NewExecutor(DefaultConfig())
            got, err := e.Run(context.Background(), tt.pipeline, tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Run() = %v, want %v", got, tt.want)
            }
        })
    }
}

func TestExecutor_Run_Timeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()

    e := NewExecutor(DefaultConfig())
    _, err := e.Run(ctx, slowPipeline(), "input")

    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected deadline exceeded, got %v", err)
    }
}
` ` `

### contract_test.go

` ` `go
func TestJSONSchemaValidator_Validate(t *testing.T) {
    tests := []struct {
        name    string
        schema  string
        input   string
        wantErr bool
    }{
        {
            name:    "valid object",
            schema:  ` + "`" + `{"type":"object","required":["name"]}` + "`" + `,
            input:   ` + "`" + `{"name":"test"}` + "`" + `,
            wantErr: false,
        },
        {
            name:    "missing required field",
            schema:  ` + "`" + `{"type":"object","required":["name"]}` + "`" + `,
            input:   ` + "`" + `{}` + "`" + `,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            v := NewJSONSchemaValidator(tt.schema)
            err := v.Validate([]byte(tt.input))
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
` ` `
```

## Customization

### Target specific coverage

```bash
wave run test-gen "increase coverage to 80% for internal/contract"
```

### Generate benchmark tests

```bash
wave run test-gen "add benchmarks for hot paths in the executor"
```

### Focus on error paths

```bash
wave run test-gen "test error handling in api/handlers"
```

### Generate integration tests

Create a specialized pipeline for integration tests:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: integration-tests
  description: "Generate integration tests"

steps:
  - id: map-integrations
    persona: navigator
    exec:
      source: |
        Map all integration points: {{ input }}
        Identify: database calls, API endpoints, external services
        Output test scenarios for each integration.
    output_artifacts:
      - name: integrations
        path: .wave/output/integrations.json
        type: json

  - id: generate
    persona: craftsman
    dependencies: [map-integrations]
    memory:
      inject_artifacts:
        - step: map-integrations
          artifact: integrations
          as: scenarios
    exec:
      source: |
        Generate integration tests with:
        - Test containers for databases
        - HTTP mocks for external APIs
        - Cleanup between tests
    handover:
      contract:
        type: test_suite
        command: "go test -tags=integration ./..."
        must_pass: false
        max_retries: 2
    output_artifacts:
      - name: tests
        path: .wave/output/integration-tests.md
        type: markdown
```

</div>

## Contract Validation

The test-gen pipeline includes a contract that runs the generated tests:

<div v-pre>

```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./... -v"
    must_pass: false      # Allow generation even if some tests fail
    on_failure: retry     # Retry to fix compilation errors
    max_retries: 3
```

</div>

This ensures generated tests at least compile. Set `must_pass: true` to require all tests pass.

## Related Use Cases

- [Code Review](/use-cases/gh-pr-review) - Review generated tests in PRs
- [Documentation Generation](/future/use-cases/documentation-generation) - Document test patterns
- [Refactoring](/use-cases/refactoring) - Generate tests before refactoring

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Learn about test suite validation
- [Concepts: Personas](/concepts/personas) - Understanding the craftsman persona

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>
