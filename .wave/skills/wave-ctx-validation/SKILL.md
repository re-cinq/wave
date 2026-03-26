---
name: wave-ctx-validation
description: Domain context for Wave's output validation and contract enforcement bounded context
---

# Validation Context

Contract validation at step handovers, preflight dependency checks, adaptive retry strategies, and input schema validation.

## Invariants

- Hard contract failures (`must_pass: true`) block step completion and prevent the step from being marked successful
- Soft contract failures (`must_pass: false`) log warnings but allow the step to proceed
- JSON Schema validation uses draft-07 via `github.com/santhosh-tekuri/jsonschema/v6`
- Test suite contracts default their working directory to `project_root` (not the workspace) because tests almost always need actual project context (go.mod, package.json, etc.)
- Unresolved template variables in contract commands (e.g. `{{ project.test_command }}`) are detected and reported as non-retryable errors rather than silently executing broken commands
- Contract validation MUST NOT pass silently on failure -- every failure is either an error or a warning, never swallowed
- Preflight checks run before any step executes; missing tools and skills produce typed errors (`ToolError`, `SkillError`) that carry the list of missing items

## Key Decisions

- Six contract types with a single `ContractValidator` interface: `json_schema`, `typescript_interface`, `test_suite`, `markdown_spec`, `format`, `non_empty_file`
- `ValidateWithAdaptiveRetry` classifies failures into types (schema_mismatch, missing_content, format_error, quality_gate, structure, unknown) and generates targeted repair prompts for the persona's retry
- JSON cleaning is attempted before schema validation: strips comments (single-line, multi-line, hash), removes trailing commas, and normalizes whitespace -- all changes are tracked for diagnostics
- Wrapper detection identifies when LLM output is wrapped in markdown code fences or conversational preamble and extracts the actual JSON payload
- `contract_test_command` in `wave.yaml` provides a separate command for contract validation (e.g. `go test ./...` without `-race`) to avoid the race detector's overhead and output noise in contract checks
- `resolveContractDir` resolves the `dir` field: `"project_root"` walks up to the git root, empty defaults to workspace for JSON/format contracts but `project_root` for test suites, and absolute paths are used as-is
- The `FailureClassifier` assigns a confidence score (0.0-1.0) to each classification for use in retry budget decisions

## Domain Vocabulary

| Term | Meaning |
|------|---------|
| Contract | A validation rule applied to step output at the handover boundary; defined in `handover.contract` |
| ContractValidator | Interface with `Validate(cfg, workspacePath) error` -- one implementation per contract type |
| ValidationError | Structured error with contract type, message, details slice, retryable flag, and attempt tracking |
| JSON Schema | Draft-07 schema file (`.wave/contracts/<name>.schema.json`) validated against step output |
| Test suite | A shell command executed in a working directory; exit code 0 = pass, non-zero = fail |
| Markdown spec | Validates markdown output against structural requirements (headings, sections) |
| Format validator | Checks file format (JSON parseable, YAML parseable, non-empty) |
| Adaptive retry | Classifies failures, generates repair prompts, and manages retry budgets with backoff |
| FailureType | Classification enum: `schema_mismatch`, `missing_content`, `format_error`, `quality_gate`, `structure`, `unknown` |
| RetryStrategy | Interface for retry policy: `ShouldRetry()`, `GetRetryDelay()`, `GenerateRepairPrompt()` |
| Preflight | Pre-execution validation that required tools are on PATH and required skills are provisioned |
| Input validator | Schema-based validation of artifacts injected into a step (`SchemaPath` on `ArtifactRef`) |
| Wrapper detection | Heuristic extraction of JSON from LLM output that may be wrapped in markdown fences or prose |

## Neighboring Contexts

- **Execution** (`internal/pipeline/`) -- the executor calls `contract.Validate()` at step handovers and uses `preflight.Checker` before the first step; retry/rework policies in `RetryConfig` and `HandoverConfig` feed back into the executor's step loop
- **Configuration** (`internal/manifest/`) -- `project.contract_test_command` and `project.test_command` provide the commands used by test suite contracts; fallback logic lives in `ProjectVars()`
- **Security** (`internal/security/`) -- schema file paths are validated by `PathValidator` before loading

## Key Files

- `internal/contract/contract.go` -- `ContractValidator` interface, `Validate()`, `ValidateWithRetries()`, `ValidateWithAdaptiveRetry()`, `ValidationError` type
- `internal/contract/jsonschema.go` -- `jsonSchemaValidator`, JSON cleaning, draft-07 validation
- `internal/contract/testsuite.go` -- `testSuiteValidator`, command parsing, working directory resolution, unresolved template detection
- `internal/contract/format_validator.go` -- `FormatValidator` for JSON/YAML parseability checks
- `internal/contract/markdownspec.go` -- `markdownSpecValidator` for structural markdown validation
- `internal/contract/non_empty_file.go` -- `nonEmptyFileValidator` for file existence and non-emptiness
- `internal/contract/typescript.go` -- `typeScriptValidator` for TypeScript interface validation
- `internal/contract/retry_strategy.go` -- `AdaptiveRetryStrategy`, `FailureClassifier`, `ClassifiedFailure`, repair prompt generation
- `internal/contract/json_cleaner.go` -- JSON preprocessing (comment removal, trailing comma cleanup)
- `internal/contract/json_recovery.go` -- recovery heuristics for malformed JSON output
- `internal/contract/wrapper_detection.go` -- extraction of JSON payloads from markdown-wrapped LLM output
- `internal/contract/input_validator.go` -- artifact input schema validation
- `internal/contract/validation_error_formatter.go` -- human-readable error formatting for contract failures
- `internal/preflight/preflight.go` -- `Checker`, `SkillError`, `ToolError`, tool-on-PATH and skill availability validation
