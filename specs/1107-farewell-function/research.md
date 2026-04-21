# Phase 0 Research: Farewell Function

All spec clarifications (2026-04-21) are already resolved. Research below
captures the small design decisions that remain.

## Decision 1: Package location — `internal/farewell`

- **Decision**: New package `internal/farewell` with `Farewell(name string) string`
  and `WriteFarewell(w io.Writer, name string, suppress bool)` helpers.
- **Rationale**: Wave already uses `internal/humanize` for small UX helpers;
  a sibling `internal/farewell` keeps concerns separated and greppable.
  Exposing via `internal/` is sufficient — embedders importing Wave use
  internal packages already.
- **Alternatives**:
  - Drop into `internal/humanize` — rejected: humanize is about number/time
    formatting, mixing unrelated text would muddy the package.
  - Top-level `pkg/farewell` — rejected: Wave does not use a `pkg/` layout.

## Decision 2: Suppression source — reuse existing quiet/TTY logic

- **Decision**: Reuse `cmd/wave/commands/output.go` quiet/TTY detection
  (already resolves `--quiet`, `--json`, and non-TTY into a `quiet` format).
  No new flag. TTY check via `term.IsTerminal` (already an indirect dep).
- **Rationale**: Spec FR-011 requires reuse; this is the existing seam.
- **Alternatives**: New `--no-farewell` flag — rejected per FR-011.

## Decision 3: Name source — `$USER` only

- **Decision**: Read `os.Getenv("USER")`; if empty, render the generic form.
  No lookup of `os/user.Current()` fallback (keeps it dependency-free and
  avoids cgo path on some platforms).
- **Rationale**: Matches spec clarification; deterministic; avoids edge cases
  around container UIDs without passwd entries.
- **Alternatives**: `os/user.Current()` — rejected: can require cgo and fail
  in minimal containers; `$USER` is already the universal signal.

## Decision 4: Fixed string template

- **Decision**: Generic default: `"Farewell — see you next wave."`
  Name form: `"Farewell, <name> — see you next wave."` (name verbatim; no
  trimming beyond `strings.TrimSpace`).
- **Rationale**: Spec FR-009 mandates single fixed English string; satisfies
  SC-003 determinism.
- **Alternatives**: Randomised pool — rejected per clarification.

## Decision 5: Write target — stdout, no newline duplication

- **Decision**: `WriteFarewell` writes `msg + "\n"` to the passed `io.Writer`
  (normally `os.Stdout`). Returns the write error (ignored by callers since
  farewell is non-essential).
- **Rationale**: FR-006 (stdout, does not alter exit code). FR-007 handled by
  caller: only invoke on successful command paths.

## Decision 6: Failure-path integration

- **Decision**: Callers only invoke the farewell on the success branch of the
  command handler (after `cmd.Execute()` returns `nil`). No wrapping of error
  paths, no signal handler changes (Ctrl+C branch already skips success-only
  post-run).
- **Rationale**: Satisfies FR-007 and edge-case "Ctrl+C MAY be skipped" with
  zero extra logic.

## Decision 7: TUI integration

- **Decision**: TUI teardown (post-`tea.Program.Run`) prints the farewell via
  the same `WriteFarewell` helper, with `suppress = !isTTY || quietFlag`.
- **Rationale**: Shared helper guarantees identical wording per FR-008.

## Open Questions

None. All spec `[NEEDS CLARIFICATION]` markers were resolved in the clarify
step on 2026-04-21.
