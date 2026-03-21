# Tasks

## Phase 1: Core Guide

- [X] Task 1.1: Create `docs/guides/adapter-development.md` with document structure and introduction explaining when to build a first-class adapter vs using ProcessGroupRunner
- [X] Task 1.2: Write the AdapterRunner interface section — document `Run()` method, `AdapterRunConfig` fields, `AdapterResult` structure, and a complete skeleton adapter implementation
- [X] Task 1.3: Write the streaming events section — document `StreamEvent` struct, `OnStreamEvent` callback pattern, event types, and how to parse NDJSON output into events
- [X] Task 1.4: Write the sandbox integration section — document Docker sandbox wrapping via `sandbox.NewSandbox()`, curated environment via `BuildCuratedEnvironment()`, domain filtering, and `settings.json` generation
- [X] Task 1.5: Write the testing patterns section — document `MockAdapter` with functional options, `configCapturingAdapter` pattern, integration test examples, and table-driven test structure

## Phase 2: Integration

- [X] Task 2.1: Write the registration section — document adding a case to `ResolveAdapter()` in `opencode.go` and manifest adapter configuration
- [X] Task 2.2: Write the workspace setup section — document CLAUDE.md assembly layers, `settings.json` generation, and skill command copying
- [X] Task 2.3: Add cross-references to existing docs (`concepts/adapters.md`, `reference/adapters.md`, `examples/custom-adapter.md`)

## Phase 3: Validation

- [X] Task 3.1: Verify all referenced file paths exist in the codebase
- [X] Task 3.2: Verify Go code examples are syntactically valid
- [X] Task 3.3: Verify markdown renders correctly (headings, code blocks, tables)
