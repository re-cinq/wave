# Implementation Plan: Expand Reviewer Deny Patterns

## 1. Objective

Add `Write(*.py)`, `Write(*.rs)`, `Bash(rm *)`, `Bash(git push*)`, and `Bash(git commit*)` to the reviewer persona's deny patterns in both `wave.yaml` and the embedded default, then add test coverage for each new pattern.

## 2. Approach

This is a configuration-only change with test additions. The deny pattern matching infrastructure (`internal/adapter/permissions.go`) already supports all the glob patterns needed — `Write(*.py)` uses `filepath.Match`, and `Bash(rm *)` / `Bash(git push*)` / `Bash(git commit*)` use `matchStringGlob` with space-aware prefix matching.

The change touches three layers:
1. **Project manifest** (`wave.yaml`) — the reviewer persona's live deny list
2. **Embedded defaults** (`internal/defaults/personas/reviewer.yaml`) — the built-in default deny list
3. **Tests** — expand `internal/manifest/permissions_test.go` to cover each new pattern

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `wave.yaml` | modify | Add 5 new deny patterns to reviewer persona (lines 233-236) |
| `internal/defaults/personas/reviewer.yaml` | modify | Add `Write(*.py)`, `Write(*.rs)`, `Bash(rm *)` to embedded default deny list |
| `internal/manifest/permissions_test.go` | modify | Add test cases for new deny patterns in reviewer persona |

## 4. Architecture Decisions

### No runtime code changes needed
The `PermissionChecker.CheckPermission` in `internal/adapter/permissions.go` already:
- Iterates deny patterns first (deny-takes-precedence)
- Calls `matchToolPattern` → `parseToolPattern` + `matchGlob`
- `matchGlob` dispatches to `matchStringGlob` for patterns with spaces (Bash commands)
- `filepath.Match` handles `*.py`, `*.rs` file extension patterns

### Consistency between wave.yaml and defaults
The `wave.yaml` reviewer persona (line 219) and the embedded default (`internal/defaults/personas/reviewer.yaml`) both need updating. The `wave.yaml` is the project-specific configuration; the embedded default is what new projects get.

### Embedded default vs wave.yaml differences
The embedded default (`reviewer.yaml`) currently has a broader deny set (includes `Bash(git push*)`, `Bash(git commit*)`) while the project `wave.yaml` has a narrower set. This change aligns `wave.yaml` with the intended default and adds the missing language patterns to both.

### Test fixture alignment
The test helper `createTestManifestWithPersonas` in `internal/manifest/permissions_test.go` hardcodes persona configurations. It must be updated to include the new deny patterns so the test fixture matches the real configuration.

## 5. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Existing tests break due to deny count assertions | Low | The test `TestPersonaPermission_ReviewerCannotWriteSourceFiles` checks for `Write(*.go)` and `Write(*.ts)` specifically — no count assertions that would break |
| `Bash(rm *)` too broad — blocks `rm` in non-destructive contexts | Low | The reviewer has no business running `rm` at all. Projects can override if needed |
| Pattern matching edge cases | Very Low | The `matchStringGlob` function handles prefix matching (`rm *` → `strings.HasPrefix(text, "rm ")`) which correctly matches `rm foo.txt`, `rm -rf /`, etc. |

## 6. Testing Strategy

### Unit Tests (in `internal/manifest/permissions_test.go`)

1. **Extend `TestPersonaPermission_ReviewerCannotWriteSourceFiles`** — add assertions for `Write(*.py)` and `Write(*.rs)` deny patterns
2. **Add `TestPersonaPermission_ReviewerCannotRunDestructiveCommands`** — new test function covering:
   - `Bash(rm foo.txt)` → denied by `Bash(rm *)`
   - `Bash(rm -rf /tmp)` → denied by `Bash(rm *)`
   - `Bash(git push origin main)` → denied by `Bash(git push*)`
   - `Bash(git commit -m "msg")` → denied by `Bash(git commit*)`
   - `Bash(go test ./...)` → still allowed (not blocked by new patterns)
   - `Bash(git log --oneline)` → still allowed
3. **Update test fixture** — add new deny patterns to `createTestManifestWithPersonas` reviewer entry
4. **Update `TestPersonaPermission_ArtifactCreationScenarios`** — add scenarios for `.py` and `.rs` files
5. **Extend `TestPersonaPermission_DenyPatternTakesPrecedence`** — add cases for `rm` and `git push/commit` patterns

### Integration test
The existing `TestLoadWaveYAML_PersonaPermissions` test loads the real `wave.yaml` and validates reviewer deny patterns. It currently checks for `Write(*.go)` — extend it to also verify the new patterns are present.

### Validation
Run `go test ./internal/manifest/ -v -run TestPersona` to verify all permission tests pass.
Run `go test ./...` for full suite.
