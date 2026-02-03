# Test Generation

Analyze coverage gaps and generate comprehensive tests. Wave's test-gen pipeline identifies untested code, generates test cases, and verifies they compile and run.

## Quick Start

```bash
wave run --pipeline test-gen --input "generate tests for the pipeline package"
```

Expected output:

```
[10:00:01] started   analyze-coverage   (navigator)              Starting step
[10:00:32] completed analyze-coverage   (navigator)   31s   2.9k Analysis complete
[10:00:33] started   generate-tests     (craftsman)              Starting step
[10:01:45] completed generate-tests     (craftsman)   72s   8.5k Generation complete
[10:01:46] started   verify-coverage    (auditor)                Starting step
[10:02:10] completed verify-coverage    (auditor)     24s   1.8k Verification complete

Pipeline test-gen completed in 129s
Artifacts: output/generated-tests.md
```

## Complete Pipeline

This is the full `test-gen` pipeline from `.wave/pipelines/test-gen.yaml`:

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
        path: output/coverage-analysis.json
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
        path: output/generated-tests.md
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
        path: output/coverage-verification.md
        type: markdown
```

## Example Output

The pipeline produces `output/generated-tests.md`:

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

```go
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
```

### contract_test.go

```go
func TestJSONSchemaValidator_Validate(t *testing.T) {
    tests := []struct {
        name    string
        schema  string
        input   string
        wantErr bool
    }{
        {
            name:    "valid object",
            schema:  `{"type":"object","required":["name"]}`,
            input:   `{"name":"test"}`,
            wantErr: false,
        },
        {
            name:    "missing required field",
            schema:  `{"type":"object","required":["name"]}`,
            input:   `{}`,
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
```
```

## Customization

### Target specific coverage

```bash
wave run --pipeline test-gen --input "increase coverage to 80% for internal/contract"
```

### Generate benchmark tests

```bash
wave run --pipeline test-gen --input "add benchmarks for hot paths in the executor"
```

### Focus on error paths

```bash
wave run --pipeline test-gen --input "test error handling in api/handlers"
```

### Generate integration tests

Create a specialized pipeline for integration tests:

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
        path: output/integrations.json
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
        path: output/integration-tests.md
        type: markdown
```

## Contract Validation

The test-gen pipeline includes a contract that runs the generated tests:

```yaml
handover:
  contract:
    type: test_suite
    command: "go test ./... -v"
    must_pass: false      # Allow generation even if some tests fail
    on_failure: retry     # Retry to fix compilation errors
    max_retries: 3
```

This ensures generated tests at least compile. Set `must_pass: true` to require all tests pass.

## Next Steps

- [Code Review](/use-cases/code-review) - Review generated tests in PRs
- [Documentation](/use-cases/docs-generation) - Document test patterns
- [Concepts: Contracts](/guide/contracts) - Learn about test suite validation
