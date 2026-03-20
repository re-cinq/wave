# docs: add doc.go files to 10 core internal packages

**Issue**: [#530](https://github.com/re-cinq/wave/issues/530)
**Author**: nextlevelshit
**Labels**: none
**Complexity**: simple

## Description

25 of 28 internal packages have no doc.go file. Only continuous, defaults, and display have package-level documentation. Critical packages like pipeline, adapter, contract, audit, state, workspace, and manifest are undocumented.

## Target Packages

Add doc.go files to:
1. `internal/pipeline/`
2. `internal/adapter/`
3. `internal/contract/`
4. `internal/manifest/`
5. `internal/workspace/`
6. `internal/state/`
7. `internal/event/`
8. `internal/audit/`
9. `internal/security/`
10. `internal/relay/`

## Acceptance Criteria

- Each of the 10 listed packages has a `doc.go` file
- Each doc.go follows the existing convention (`internal/continuous/doc.go`): package-level comment followed by `package <name>`
- Package comments accurately describe the package's purpose and key responsibilities
- `go doc ./internal/<pkg>` produces meaningful output for all 10 packages
- All existing tests continue to pass
