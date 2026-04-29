# Work Items

## Phase 1: Setup
- [X] Item 1.1: Confirm validator library used by existing shared-schema tests (likely `xeipuuv/gojsonschema` or `santhosh-tekuri/jsonschema`); reuse it. Inspect any sibling `*_test.go` under `internal/contract/schemas/shared/` first; if none, pick the validator already imported elsewhere in `internal/contract/`. — Result: `github.com/santhosh-tekuri/jsonschema/v6` (already used by `internal/contract/jsonschema.go`).
- [X] Item 1.2: Verify `docs/adr/010-pipeline-io-protocol.md` structure to choose update spot (table vs. footer). — No registered-types table; bumped the implementation-status counts and added a "Schema additions" footer table.
- [X] Item 1.3: `grep -r "kind.*forge.*repo" docs/` to confirm the draft snippet at `onboarding-as-session-plan.md:392` isn't quoted elsewhere before replacing. — Confirmed only `docs/scope/onboarding-as-session-plan.md`.

## Phase 2: Core Implementation
- [X] Item 2.1: Write `internal/contract/schemas/shared/work_item_ref.json` (draft-07, $id `wave://shared/work_item_ref`, title `WorkItemRef`, properties + `allOf if/then` conditional, `additionalProperties: false`). [P]
- [X] Item 2.2: Add `"work_item_ref"` to `expected` slice in `registry_test.go` `TestRegistryContainsCanonicalTypes`; bump `TestNamesSorted` floor from 7 to 8. [P]

## Phase 3: Testing
- [X] Item 3.1: Add `internal/contract/schemas/shared/work_item_ref_test.go` with table-driven cases:
  - 6 positive fixtures — one per `source` enum value (github, gitea, gitlab, bitbucket, schedule, manual).
  - Negative: forge source missing `forge_host` → validation error.
  - Negative: unknown `source` value → rejected by enum.
  - Negative: extra property → rejected by `additionalProperties: false`.
  - Negative: `state: "wip"` → rejected.
  - Negative: `created_at: "yesterday"` → rejected by `format: date-time`.
- [X] Item 3.2: Run `go test ./internal/contract/schemas/shared/...` locally; confirm registry test still passes.
- [X] Item 3.3: Run `go test ./...` — full suite. Catch any consumer that referenced `Names()` length explicitly. [P]
- [X] Item 3.4: Run `golangci-lint run ./internal/contract/...`. [P]

## Phase 4: Polish
- [X] Item 4.1: Update `docs/adr/010-pipeline-io-protocol.md` — register the new shared type. [P]
- [X] Item 4.2: Edit `docs/scope/onboarding-as-session-plan.md`:
  - Tick the Phase 2.1 row at line 435.
  - Replace the inline draft snippet (lines ~392–404) with a one-liner: "See canonical schema at `internal/contract/schemas/shared/work_item_ref.json`." [P]
- [X] Item 4.3: Update `docs/adr/016-work-source-centric-webui-ia.md:286` D9 reference if shape diverges meaningfully from what that ADR assumed (current text just names the schema, so likely no change needed — verify). — Confirmed: line 286 only names the schema; no edit needed.
- [X] Item 4.4: Final `make test` + `golangci-lint`; commit message `feat(contract): add work_item_ref shared schema (#1590)`; open PR linking #1565 and #1590. PR description must call out: (a) path-convention reconciliation, (b) `.agents/contracts/` deliberately skipped with rationale, (c) D2 conditional-required pattern.
