# Tasks

## Phase 1: Messages and State
- [X] Task 1.1: Add `SuggestLaunchedMsg` to `suggest_messages.go` — carries the launched pipeline name
- [X] Task 1.2: Add `launched` map field to `SuggestListModel` in `suggest_list.go` to track launched proposal names

## Phase 2: Core Implementation
- [X] Task 2.1: Phase-contextual rendering in `SuggestDetailModel.View()` — use `guidedPhase` to render different footer hints per phase [P]
- [X] Task 2.2: Wire `SuggestModifyMsg` in `content.go` to open `ConfigureFormMsg` with pre-populated pipeline name and input instead of redirecting to `SuggestLaunchMsg` [P]
- [X] Task 2.3: Emit `SuggestLaunchedMsg` from `SuggestLaunchMsg` handler in `content.go` after successful launch transition
- [X] Task 2.4: Handle `SuggestLaunchedMsg` in `SuggestListModel.Update()` to mark proposal as launched in the `launched` map
- [X] Task 2.5: Render launched badge (`✓`) in `SuggestListModel.View()` for proposals that have been launched
- [X] Task 2.6: Add fleet auto-refresh — emit pipeline list refresh command with short delay after `SuggestLaunchMsg` transitions to fleet in `content.go`

## Phase 3: Testing
- [X] Task 3.1: Add tests for phase-contextual hints in `suggest_detail_test.go` — verify different footer text for `GuidedPhaseProposals` vs `GuidedPhaseFleet` [P]
- [X] Task 3.2: Add tests for launched badge in `suggest_list_test.go` — verify `✓` marker appears for launched proposals [P]
- [X] Task 3.3: Add tests for `SuggestModifyMsg` flow in `content_test.go` — verify it emits `ConfigureFormMsg` [P]
- [X] Task 3.4: Add test for `SuggestLaunchedMsg` tracking in `suggest_list_test.go` — verify `launched` map updates on message [P]

## Phase 4: Polish
- [X] Task 4.1: Verify all tests pass with `go test ./internal/tui/...`
- [X] Task 4.2: Run `go vet ./internal/tui/...` and fix any issues
