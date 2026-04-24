# Tasks

## Phase 1: Setup
- [X] Task 1.1: Check repository root for existing `math.go` and sibling `.go` files to determine target package
- [X] Task 1.2: Confirm branch `004-add-utility` active

## Phase 2: Core Implementation
- [X] Task 2.1: Create or update `math.go` with package declaration matching repo
- [X] Task 2.2: Add `// add returns the sum of a and b.` godoc + `func add(a, b int) int { return a + b }`

## Phase 3: Testing
- [X] Task 3.1: Run `go build ./...` — verify clean
- [X] Task 3.2: Run `go vet ./...` — verify clean
- (No unit/integration tests — explicitly out of scope per issue)

## Phase 4: Polish
- [X] Task 4.1: Stage `math.go` (and `specs/004-add-utility/` planning docs)
- [X] Task 4.2: Commit with conventional message, open PR referencing issue #4
