# Tasks

## Phase 1: Data Structures & Manifest Parsing
- [X] Task 1.1: Add `SkillInfo` JSON output struct to `list.go` with fields: `Name`, `Check`, `Install`, `Installed` (bool), `UsedBy` ([]string)
- [X] Task 1.2: Add `Skills` field to `manifestData2` struct in `list.go` to parse the `skills` YAML map
- [X] Task 1.3: Add `Skills []SkillInfo` field to `ListOutput` struct for JSON output

## Phase 2: Core Implementation
- [X] Task 2.1: Implement `collectSkills()` function that reads skill configs from manifest, runs each skill's check command via `sh -c`, and returns `[]SkillInfo` [P]
- [X] Task 2.2: Implement `collectSkillPipelineUsage()` helper that scans `.wave/pipelines/*.yaml` for `requires.skills` arrays and returns a `map[string][]string` (skill name → pipeline names) [P]
- [X] Task 2.3: Implement `listSkillsTable()` function following the `listAdaptersTable` pattern — show status icon (✓/✗), skill name, check command, install command, and pipeline usage

## Phase 3: Integration into List Command
- [X] Task 3.1: Add `"skills"` to `ValidArgs` in `NewListCmd()` and update command help text (Use and Long)
- [X] Task 3.2: Add `showSkills` logic to `runList()` — include skills in the table output section with proper spacing
- [X] Task 3.3: Add skills to JSON output branch in `runList()` — populate `output.Skills` when `showSkills` is true

## Phase 4: Testing
- [X] Task 4.1: Add `TestListCmd_Skills_TableFormat` — verifies skills header and skill names appear in output [P]
- [X] Task 4.2: Add `TestListCmd_Skills_ShowsStatus` — uses `true`/`false` as check commands to verify installed/missing status [P]
- [X] Task 4.3: Add `TestListCmd_Skills_ShowsPipelineUsage` — creates pipeline with `requires.skills` and verifies cross-reference [P]
- [X] Task 4.4: Add `TestListCmd_Skills_NoSkillsDefined` — verifies "(none defined)" message [P]
- [X] Task 4.5: Add `TestListCmd_Skills_JSONFormat` — verifies valid JSON with `skills` field [P]
- [X] Task 4.6: Add `TestListCmd_Skills_SortedAlphabetically` — verifies alphabetical ordering [P]
- [X] Task 4.7: Update `TestListCmd_FilterOptions` table-driven test to include `skills` filter case
- [X] Task 4.8: Update `TestListCmd_All_ShowsEverything` to assert `Skills` header appears

## Phase 5: Validation
- [X] Task 5.1: Run `go test ./cmd/wave/commands/...` and verify all tests pass
- [X] Task 5.2: Run `go vet ./...` and verify no issues
- [X] Task 5.3: Run `go build ./...` and verify successful compilation
