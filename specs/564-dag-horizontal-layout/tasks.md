# Tasks

## Phase 1: Backend — Axis Swap in Layout Engine

- [X] Task 1.1: Rename constants in `dag.go` — `layerGapY` → `layerGapX` (180), `nodeGapX` → `nodeGapY` (80); update all references
- [X] Task 1.2: Rewrite `ComputeDAGLayout()` node positioning — layers map to X axis (columns), nodes within layer to Y axis (rows); center nodes vertically within their layer
- [X] Task 1.3: Rewrite edge bezier computation — edges go right-center of source → left-center of target; control points use horizontal midpoint
- [X] Task 1.4: Update `Width` / `Height` calculation — Width based on number of layers × layerGapX; Height based on max nodes in any layer × nodeGapY

## Phase 2: Frontend — SVG Template and Interactions

- [X] Task 2.1: Update `dag_svg.html` — add status icon `<circle>` to each node `<g>`; adjust `<text>` positions for status icon offset [P]
- [X] Task 2.2: Update `dag.js` — replace scroll-to-card on click with toggle for inline detail overlay; detail panel shows status, persona, duration, tokens; includes "Go to step" link [P]
- [X] Task 2.3: Update `sse.js` `updateDAGNodeStatus()` — also update status icon circle class; refresh detail overlay content if visible [P]
- [X] Task 2.4: Update `style.css` — DAG container `overflow-x: auto` for wide graphs; detail overlay positioning and styles; status icon colors; responsive breakpoint adjustments [P]

## Phase 3: Testing

- [X] Task 3.1: Update `dag_test.go` — `TestComputeDAGLayout_LinearChain` checks X positions (not Y); `TestComputeDAGLayout_Diamond` checks Width (not Height)
- [X] Task 3.2: Add `TestComputeDAGLayout_FanOut` — 1 source → 3 parallel targets at same X, different Y
- [X] Task 3.3: Add `TestComputeDAGLayout_EdgeDirectionLTR` — verify all edges have FromX < ToX
- [X] Task 3.4: Run `go test ./internal/webui/...` to verify all tests pass

## Phase 4: Polish

- [X] Task 4.1: Verify SVG arrowhead marker orientation works for horizontal edges
- [X] Task 4.2: Test responsive layout — mobile view stacks sidebar above main content
- [X] Task 4.3: Run `go test ./...` full suite to catch any regressions
