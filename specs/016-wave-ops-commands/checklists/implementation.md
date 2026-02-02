# Implementation Checklist: Wave Ops Commands

**Feature**: 016-wave-ops-commands
**Created**: 2026-02-02

---

## Pre-Implementation Checklist

- [ ] Dependencies verified (015-wave-cli-implementation complete)
- [ ] Design reviewed with team
- [ ] Test plan approved
- [ ] SQLite state schema supports required queries
- [ ] Workspace isolation requirements understood

---

## Command: `wave status`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Subcommand structure defined (status, status <run-id>)
- [ ] Root command shows current/recent pipelines

### Flags
- [ ] `--all` - Show all recent pipelines
- [ ] `--format <type>` - Output format (table, json)
- [ ] `--manifest <path>` - Path to manifest file

### Implementation
- [ ] Query running pipelines from state DB
- [ ] Query recent completed pipelines
- [ ] Display pipeline name, status, current step, elapsed time
- [ ] Display token usage statistics
- [ ] Support run-id lookup for detailed view

### Help & Documentation
- [ ] Help text written with examples
- [ ] Error messages include actionable suggestions

### Testing
- [ ] Unit tests for status formatting
- [ ] Unit tests for state queries
- [ ] Integration test: start pipeline, verify status shows running
- [ ] Integration test: completed pipeline shows final status
- [ ] Test JSON output format

### Error Handling
- [ ] Handle no pipelines found gracefully
- [ ] Handle invalid run-id
- [ ] Handle state DB connection errors

---

## Command: `wave logs`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Default shows most recent pipeline logs
- [ ] Support run-id argument for specific pipeline

### Flags
- [ ] `--step <name>` - Filter by step name
- [ ] `--persona <name>` - Filter by persona
- [ ] `--errors` - Show only errors and failures
- [ ] `--follow` - Stream logs in real-time
- [ ] `--format <type>` - Output format (text, json)
- [ ] `--manifest <path>` - Path to manifest file

### Implementation
- [ ] Read logs from workspace/state
- [ ] Parse and format adapter responses
- [ ] Display contract validation results
- [ ] Implement step filtering
- [ ] Implement persona filtering
- [ ] Implement error-only filtering
- [ ] Implement real-time streaming (--follow)

### Help & Documentation
- [ ] Help text written with examples
- [ ] Error messages include actionable suggestions

### Testing
- [ ] Unit tests for log parsing/formatting
- [ ] Unit tests for filter logic
- [ ] Integration test: retrieve logs from completed pipeline
- [ ] Integration test: --step filter works correctly
- [ ] Integration test: --errors shows only failures
- [ ] Integration test: --follow streams output
- [ ] Test <500ms streaming latency

### Error Handling
- [ ] Handle no logs found
- [ ] Handle invalid step/persona names
- [ ] Handle corrupted log files

---

## Command: `wave clean`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Confirmation prompt before destructive operations
- [ ] Summary of what was/would be cleaned

### Flags
- [ ] `--pipeline <name>` - Clean specific pipeline workspace
- [ ] `--all` - Clean all workspaces and state
- [ ] `--force` - Skip confirmation
- [ ] `--keep-last <n>` - Keep N most recent workspaces
- [ ] `--dry-run` - Show what would be deleted
- [ ] `--manifest <path>` - Path to manifest file

### Implementation
- [ ] Enumerate workspace directories
- [ ] Calculate disk usage per workspace
- [ ] Implement --keep-last retention logic
- [ ] Implement --pipeline filtering
- [ ] Implement atomic cleanup (no partial deletions)
- [ ] Display cleanup summary (count, space freed)

### Help & Documentation
- [ ] Help text written with examples
- [ ] Error messages include actionable suggestions

### Testing
- [ ] Unit tests for retention logic
- [ ] Unit tests for workspace enumeration
- [ ] Integration test: --all removes everything after confirm
- [ ] Integration test: --keep-last 2 retains correct workspaces
- [ ] Integration test: --pipeline filters correctly
- [ ] Integration test: --dry-run shows but doesn't delete
- [ ] Test handles 1000+ workspaces efficiently

### Error Handling
- [ ] Handle permission errors
- [ ] Handle workspaces in use by running pipelines
- [ ] Handle missing workspace directories

---

## Command: `wave list`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Subcommands: pipelines, personas, adapters
- [ ] Default (no subcommand) shows help

### Flags
- [ ] `--format <type>` - Output format (table, json)
- [ ] `--manifest <path>` - Path to manifest file

### Implementation: `wave list pipelines`
- [ ] Parse manifest for pipeline definitions
- [ ] Display pipeline name and description
- [ ] Display step count per pipeline

### Implementation: `wave list personas`
- [ ] Parse manifest for persona definitions
- [ ] Display persona name
- [ ] Display allowed tools summary
- [ ] Display adapter assignment

### Implementation: `wave list adapters`
- [ ] Parse manifest for adapter definitions
- [ ] Display adapter name and type
- [ ] Display adapter status (if applicable)

### Help & Documentation
- [ ] Help text for main list command
- [ ] Help text for each subcommand
- [ ] Examples in help text

### Testing
- [ ] Unit tests for manifest parsing
- [ ] Unit tests for table/JSON formatting
- [ ] Integration test: list pipelines shows all defined
- [ ] Integration test: list personas shows all defined
- [ ] Integration test: list adapters shows all defined
- [ ] Integration test: --format json produces valid JSON

### Error Handling
- [ ] Handle missing/invalid manifest
- [ ] Handle empty lists gracefully

---

## Command: `wave cancel`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Default cancels most recent running pipeline
- [ ] Support run-id argument for specific pipeline

### Flags
- [ ] `--force` - Interrupt immediately without waiting
- [ ] `--manifest <path>` - Path to manifest file

### Implementation
- [ ] Identify running pipelines from state
- [ ] Implement graceful cancellation (complete current step)
- [ ] Implement forced cancellation (immediate interrupt)
- [ ] Update state to reflect cancellation
- [ ] Clean up resources on cancel

### Help & Documentation
- [ ] Help text written with examples
- [ ] Error messages include actionable suggestions

### Testing
- [ ] Unit tests for cancellation state transitions
- [ ] Integration test: cancel stops after current step
- [ ] Integration test: --force interrupts immediately
- [ ] Integration test: no running pipeline shows message
- [ ] Test cancellation is safe with concurrent execution

### Error Handling
- [ ] Handle no running pipelines
- [ ] Handle invalid run-id
- [ ] Handle already completed/cancelled pipelines

---

## Command: `wave artifacts`

### Structure
- [ ] Command registered in `cmd/wave/`
- [ ] Default lists artifacts from most recent pipeline
- [ ] Support run-id argument for specific pipeline

### Flags
- [ ] `--step <name>` - Filter by step name
- [ ] `--export <path>` - Export artifacts to directory
- [ ] `--format <type>` - Output format (table, json)
- [ ] `--manifest <path>` - Path to manifest file

### Implementation
- [ ] Enumerate artifacts from workspace
- [ ] Display step, artifact name, and path
- [ ] Display artifact size
- [ ] Implement step filtering
- [ ] Implement export to specified directory
- [ ] Preserve artifact structure on export

### Help & Documentation
- [ ] Help text written with examples
- [ ] Error messages include actionable suggestions

### Testing
- [ ] Unit tests for artifact enumeration
- [ ] Unit tests for export logic
- [ ] Integration test: list shows all artifacts
- [ ] Integration test: --step filters correctly
- [ ] Integration test: --export copies to directory
- [ ] Integration test: --format json produces valid JSON

### Error Handling
- [ ] Handle no artifacts found
- [ ] Handle invalid step names
- [ ] Handle export path errors (permissions, exists)
- [ ] Handle missing artifact files

---

## Post-Implementation Checklist

### Testing
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Tests pass with race detector (`go test -race ./...`)
- [ ] Performance benchmarks meet NFRs:
  - [ ] `wave status` returns within 100ms
  - [ ] `wave logs` streams with <500ms latency
  - [ ] `wave clean` handles 1000+ workspaces

### Documentation
- [ ] CLAUDE.md updated with new commands
- [ ] Help text reviewed for clarity
- [ ] Examples work as documented

### Code Quality
- [ ] `gofmt` formatting applied
- [ ] `go vet` passes
- [ ] No new linter warnings
- [ ] Code reviewed

### Integration
- [ ] Commands work with existing pipelines
- [ ] Commands respect workspace isolation
- [ ] Commands work in CI mode
- [ ] Concurrent execution is safe

### Open Questions Resolved
- [ ] Decision on log levels for `wave logs`
- [ ] Decision on automatic cleanup mode for `wave clean`
- [ ] Decision on cancellation mechanism (SIGTERM vs token)
