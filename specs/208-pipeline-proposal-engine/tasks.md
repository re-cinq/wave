# Tasks

## Phase 1: Foundation Types & Schema

- [X] Task 1.1: Create `internal/pipeline/proposal/types.go` with core types: `ForgeType` enum (GitHub, GitLab, Gitea, Bitbucket, Unknown), `HealthArtifact` struct (provisional schema matching #207 expectations), `ProposalItem` struct (pipeline name, rationale, priority, parallel group, dependency edges), `Proposal` struct (items, forge type, timestamp, metadata)
- [X] Task 1.2: Create `.wave/contracts/pipeline-proposal.schema.json` — JSON Schema Draft 07 defining the proposal output artifact structure with required fields: `forge_type`, `proposals` array, `timestamp`, and optional `health_summary`
- [X] Task 1.3: Create `internal/pipeline/proposal/types_test.go` — test JSON marshaling/unmarshaling of types, validation of ForgeType constants, ProposalItem required fields

## Phase 2: Pipeline Catalog Discovery

- [X] Task 2.1: Create `internal/pipeline/proposal/catalog.go` — `Catalog` type that discovers pipelines from one or more directories, parses YAML metadata (name, description, category, requires, input source), deduplicates by name, returns `[]CatalogEntry` sorted by name
- [X] Task 2.2: Create `internal/pipeline/proposal/filter.go` — `ForgeFilter` that classifies pipelines by name prefix (`gh-*`, `gl-*`, `gt-*`, `bb-*`), filters catalog entries to only include forge-matching and forge-agnostic pipelines, exports `DetectForgeFromPrefix(name string) ForgeType` and `FilterByForge(entries []CatalogEntry, forge ForgeType) []CatalogEntry`
- [X] Task 2.3: Create `internal/pipeline/proposal/catalog_test.go` — test catalog discovery with temp directories, test dedup logic, test empty directory [P]
- [X] Task 2.4: Create `internal/pipeline/proposal/filter_test.go` — test forge prefix detection for all four forge types plus agnostic, test filtering with mixed catalog [P]

## Phase 3: Scoring & Proposal Generation

- [X] Task 3.1: Create `internal/pipeline/proposal/scoring.go` — `Scorer` interface with `Score(entry CatalogEntry, health HealthArtifact) float64`, `DefaultScorer` implementation that maps health signals (test failures → `test-gen`, dead code → `dead-code`, doc issues → `doc-audit/doc-fix`, security → `security-scan`) to pipeline relevance scores
- [X] Task 3.2: Create `internal/pipeline/proposal/engine.go` — `Engine` type with `NewEngine(catalog *Catalog, opts ...EngineOption)`, `Propose(health HealthArtifact, forgeType ForgeType) (*Proposal, error)` method that: loads catalog, filters by forge, scores each pipeline, constructs dependency edges, assigns parallel groups, sorts by priority, handles edge cases (empty result, all completed)
- [X] Task 3.3: Create `internal/pipeline/proposal/scoring_test.go` — test default scorer with various health signals, test zero scores, test custom scorer via interface [P]
- [X] Task 3.4: Create `internal/pipeline/proposal/engine_test.go` — test full proposal generation with mock catalog and health artifact, test empty catalog, test forge filtering integration, test parallel group assignment, test dependency edge construction [P]

## Phase 4: Testing & Validation

- [X] Task 4.1: Write integration test that loads real `.wave/pipelines/` catalog, feeds a synthetic health artifact, and validates the proposal JSON output against `.wave/contracts/pipeline-proposal.schema.json`
- [X] Task 4.2: Run `go test -race ./internal/pipeline/proposal/...` — ensure all tests pass with race detector
- [X] Task 4.3: Run `go vet ./internal/pipeline/proposal/...` — ensure no vet warnings

## Phase 5: Polish & Documentation

- [X] Task 5.1: Add doc comments to all exported types and functions in the proposal package
- [X] Task 5.2: Verify proposal schema is referenced correctly and matches the Go types
- [X] Task 5.3: Final validation: `go test -race ./...` to ensure no regressions across the entire codebase
