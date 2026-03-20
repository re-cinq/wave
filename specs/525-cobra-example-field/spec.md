# fix(cli): move embedded examples to cobra Example field

**Issue**: [#525](https://github.com/re-cinq/wave/issues/525)
**Author**: nextlevelshit
**Labels**: none
**Complexity**: simple

## Description

8 commands currently have examples embedded in their Long description text instead of using cobra's dedicated Example field.

The Example field provides better CLI documentation formatting and consistency. Migrate command examples from Long descriptions to the Example field in each command's cobra.Command struct.

Commands that need Example field migration:
- run
- pause
- resume
- cancel
- list
- logs
- doctor
- bench

This improves `wave <command> -h` output and makes examples more discoverable.

## Acceptance Criteria

- [ ] All commands listed above have examples in the `Example` field of their `cobra.Command` struct
- [ ] The `Long` field no longer contains example-style `wave <cmd> ...` lines
- [ ] `wave <command> -h` output shows examples in the dedicated "Examples:" section
- [ ] All existing tests pass

## Codebase Analysis

After inspecting each command file:

| Command | File | Current State | Action Needed |
|---------|------|---------------|---------------|
| run | `cmd/wave/commands/run.go` | Already has `Example` field. `Long` has 3 argument-pattern lines that look like examples | Move argument pattern lines from `Long` to `Example` (merge with existing) |
| pause | N/A | **Does not exist** in the codebase | No action — skip |
| resume | `cmd/wave/commands/resume.go` | Examples embedded in `Long` (lines 53-57). No `Example` field | Extract examples from `Long` into new `Example` field |
| cancel | `cmd/wave/commands/cancel.go` | Examples embedded in `Long` (lines 54-58). No `Example` field | Extract examples from `Long` into new `Example` field |
| list | `cmd/wave/commands/list.go` | `Long` contains argument/flag documentation, not examples. No `Example` field | Add `Example` field with usage examples |
| logs | `cmd/wave/commands/logs.go` | Examples embedded in `Long` (lines 63-72). No `Example` field | Extract examples from `Long` into new `Example` field |
| doctor | `cmd/wave/commands/doctor.go` | Already has `Example` field (lines 42-47). `Long` is clean | Already done — verify only |
| bench | `cmd/wave/commands/bench.go` | Parent cmd has subcommand list in `Long`. Subcommands (`run`, `report`, `list`, `compare`) already have `Example` fields | Add `Example` to parent bench command; subcommands already correct |
