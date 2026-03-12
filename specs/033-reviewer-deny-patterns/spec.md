# Expand Reviewer Persona Deny Patterns to Block Destructive Actions

> Issue: [re-cinq/wave#33](https://github.com/re-cinq/wave/issues/33)
> Labels: enhancement, priority: medium, security
> Author: nextlevelshit
> State: OPEN

## Summary

Expand the reviewer persona's deny patterns to block destructive actions (file deletion, git push/commit) and source code writes for additional languages beyond Go and TypeScript.

## Context

During PR #32 review, it was noted that the original spec for the reviewer persona had more comprehensive deny patterns than what was implemented. The reviewer persona is intended for **quality review and validation only** — it should never modify source code or perform destructive operations.

## Current Deny Patterns (wave.yaml)

```yaml
deny:
  - Write(*.go)
  - Write(*.ts)
  - Edit(*)
```

## Proposed Deny Patterns

```yaml
deny:
  - Write(*.go)
  - Write(*.ts)
  - Write(*.py)
  - Write(*.rs)
  - Edit(*)
  - Bash(rm *)
  - Bash(git push*)
  - Bash(git commit*)
```

### Rationale for Each Addition

| Pattern | Reason |
|---------|--------|
| `Write(*.py)` | Block Python source modifications — reviewer should not edit code |
| `Write(*.rs)` | Block Rust source modifications — same principle |
| `Bash(rm *)` | Prevent file deletions — reviewer should never delete files |
| `Bash(git push*)` | Prevent pushes to remote — reviewer has no business pushing |
| `Bash(git commit*)` | Prevent commits — reviewer should only read and report |

### Configurability

These expanded patterns should be the **default** deny set for the reviewer persona. Projects can override via their own `wave.yaml` if they need a different set. No additional configuration mechanism is needed — Wave's existing manifest override system handles this.

## Implementation Notes

- Update the reviewer persona definition in `wave.yaml`
- Update the embedded default reviewer persona in `internal/defaults/personas/reviewer.yaml`
- The deny pattern matching logic in `internal/adapter/permissions.go` already supports these glob patterns
- No runtime changes needed — this is a configuration-only change plus test additions

## Acceptance Criteria

- [ ] Update reviewer persona deny patterns in `wave.yaml` with the expanded set
- [ ] Update embedded default reviewer persona in `internal/defaults/personas/reviewer.yaml`
- [ ] Verify deny pattern enforcement via existing security validation tests
- [ ] Add test cases for the new deny patterns (rm, git push, git commit, Write(*.py), Write(*.rs))
- [ ] Update reviewer persona documentation if needed
