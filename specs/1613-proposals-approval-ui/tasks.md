# Work Items

## Phase 1: Setup
- [ ] Item 1.1: Confirm `parseTemplates` glob in `internal/webui/embed.go` includes `templates/proposals/*.html`; extend if needed.
- [ ] Item 1.2: Verify pipeline-evolve emits the after-yaml file (probe path convention `<DiffPath>.after.yaml`); document the resolved convention in plan if different.

## Phase 2: Core Implementation
- [ ] Item 2.1: Implement `internal/webui/handlers_proposals.go` (list, detail, approve, reject, activation helper). [P]
- [ ] Item 2.2: Implement `cmd/wave/commands/proposals.go` (cobra parent + 4 subcommands + activation helper shared via small internal func). [P]
- [ ] Item 2.3: Author `internal/webui/templates/proposals/list.html` and `detail.html` (Tailwind classes consistent with other templates; inline fetch for approve/reject with `X-CSRF-Token`). [P]
- [ ] Item 2.4: Register routes in `internal/webui/routes.go` and `NewProposalsCmd()` in `cmd/wave/main.go`.
- [ ] Item 2.5: Add nav link to `/proposals` in `internal/webui/templates/layout.html` if global nav present.

## Phase 3: Testing
- [ ] Item 3.1: Write `internal/webui/handlers_proposals_test.go` (list filter, detail + diff render, approve flips active, reject leaves untouched, CSRF rejection, 404 on missing id). [P]
- [ ] Item 3.2: Write `cmd/wave/commands/proposals_test.go` (CLI list/show/approve/reject; approve activation idempotency; reject leaves versions). [P]
- [ ] Item 3.3: Acceptance-gate integration test: synthetic proposal → CLI approve → `GetActiveVersion` returns new row → loader resolves new yaml_path.

## Phase 4: Polish
- [ ] Item 4.1: Run `go test ./... -race` and `golangci-lint run ./...`; fix issues.
- [ ] Item 4.2: Manual smoke: start server, seed a proposal via temp script, click approve in browser, observe `pipeline_version.active` flip via `sqlite3 .agents/state.db`.
- [ ] Item 4.3: Open PR linking #1613, mention Phase 3.4 of Epic #1565.
