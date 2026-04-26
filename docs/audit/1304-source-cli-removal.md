# Audit #1304 — `internal/skill/source_cli.go` removal

**Status**: resolved (intentional removal, no production code change required)

## Finding

The wave-audit pipeline flagged PR [#1080](https://github.com/re-cinq/wave/pull/1080)
as `partial` because `internal/skill/source_cli.go` — introduced by that PR —
is absent at HEAD. Issue: [#1304](https://github.com/re-cinq/wave/issues/1304).

## Investigation

`internal/skill/source_cli.go` was deliberately deleted in commit
[`6e0fc562`](https://github.com/re-cinq/wave/commit/6e0fc562) —
*refactor(skills): drop tessl/lockfile/publish/classify, rebuild CLI minimal*.

That commit was part of the [#1113](https://github.com/re-cinq/wave/issues/1113)
skills overhaul, which trimmed the skills installer down to a file-only path.
Alongside `source_cli.go`, the following adapters and subsystems were dropped:

- tessl / bmad / openspec / speckit / github / url source adapters
- the skill classifier
- the lockfile subsystem
- the publish subsystem

`NewDefaultRouter` was reduced accordingly. The `wave skills` command surface
was collapsed from 8 subcommands to 4 (`list`, `check`, `add`, `doctor`).

## Resolution

The audit finding is accurate but represents intentional follow-up work landed
after PR #1080. No file restoration or code change is needed. This document
serves as the audit trail so future audits can skip this finding.

## References

- PR #1080 (introduction): refactor: rename StepError→StepExecutionError, deduplicate skill Install methods
- Commit `6e0fc562` (removal): refactor(skills): drop tessl/lockfile/publish/classify, rebuild CLI minimal
- Issue #1113: skills overhaul
- Issue #1304: this audit
