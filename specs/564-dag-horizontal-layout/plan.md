# Implementation Plan — DAG Horizontal Layout

## 1. Objective

Redesign the pipeline DAG visualization from a top-to-bottom vertical layout to a left-to-right horizontal layout (matching GitHub Actions / GitLab CI conventions), and add click-to-expand inline step details on DAG nodes.

## 2. Approach

The change is primarily a coordinate axis swap in the Go layout engine plus corresponding SVG template and CSS updates. The topological sort / layer assignment algorithm (`assignLayers`) is direction-agnostic and stays unchanged. The node positioning and edge bezier computation in `ComputeDAGLayout()` need axis swapping: layers map to X (columns) instead of Y (rows), and nodes within a layer map to Y instead of X.

### Key strategy:
1. **Backend axis swap** — Rename constants for clarity, swap X/Y in node positioning and edge computation
2. **SVG template update** — Add a status icon per node, adjust viewBox for wider-than-tall layouts
3. **Click-to-expand** — Add inline detail panel that toggles on click (replaces scroll-to-card for DAG nodes)
4. **CSS adjustments** — DAG container flows horizontally, responsive breakpoints updated
5. **Test updates** — Update assertions from Y-axis layer separation to X-axis

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/dag.go` | modify | Swap axis in `ComputeDAGLayout()`: layers on X, nodes on Y; update edge bezier for L→R; rename constants |
| `internal/webui/dag_test.go` | modify | Update assertions: linear chain checks X positions (not Y); diamond checks Width (not Height) |
| `internal/webui/templates/partials/dag_svg.html` | modify | Add status icon to node `<g>`, adjust text positions, update arrowhead orientation |
| `internal/webui/static/dag.js` | modify | Add click-to-expand toggle logic (inline detail panel), keep scroll-to-card as secondary |
| `internal/webui/static/sse.js` | modify | Update `updateDAGNodeStatus()` to also update status icon in expanded detail |
| `internal/webui/static/style.css` | modify | DAG container horizontal scrolling, expanded detail panel styles, responsive adjustments |

## 4. Architecture Decisions

### 4.1 Axis swap constants

Current constants:
```go
layerGapY  = 80   // vertical gap between layers
nodeGapX   = 170  // horizontal gap between same-layer nodes
```

New constants:
```go
layerGapX  = 180  // horizontal gap between layers (columns)
nodeGapY   = 80   // vertical gap between nodes in same layer (rows)
```

`layerGapX` is wider than the old `layerGapY` because horizontal flow needs more room for readable labels plus edge paths. `nodeGapY` matches the old `layerGapY` since vertical stacking of same-layer nodes needs similar spacing.

### 4.2 Edge bezier curves

Current (top-to-bottom): edges go from bottom-center of source to top-center of target, with vertical bezier control points.

New (left-to-right): edges go from right-center of source to left-center of target, with horizontal bezier control points:
```
fromX = node.X + nodeWidth       (right edge)
fromY = node.Y + nodeHeight/2    (vertical center)
toX   = target.X                 (left edge)
toY   = target.Y + nodeHeight/2  (vertical center)
midX  = (fromX + toX) / 2
CX1   = midX, CY1 = fromY       (control point near source)
CX2   = midX, CY2 = toY         (control point near target)
```

### 4.3 Click-to-expand

Rather than replacing the current scroll-to-card behavior entirely, add a toggle mechanism:
- Click toggles an inline detail panel below/beside the node in the SVG
- The detail panel shows: status, persona, duration, token count
- A "Go to step" link in the detail panel scrolls to the step card
- This is implemented as a `<foreignObject>` in SVG or as a DOM overlay positioned relative to the SVG

Decision: Use a DOM overlay (positioned `<div>`) rather than SVG `<foreignObject>` because:
- Better text rendering and CSS styling
- Consistent with the existing tooltip approach
- `<foreignObject>` has browser quirks and doesn't support all CSS features

### 4.4 Status icons

Add a small status indicator (colored circle or icon) to each node in the SVG:
- Pending: hollow circle (outline only)
- Running: filled circle with pulse animation
- Completed: filled circle (green)
- Failed: filled circle (red)
- Cancelled: filled circle (grey)

Implemented as a `<circle>` element within each node `<g>` group, positioned at top-left corner.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| SVG viewBox too wide for sidebar | Medium | Low | Use `overflow-x: auto` on container; SVG has `width: 100%` with `preserveAspectRatio` |
| Bezier curves look wrong for long horizontal spans | Low | Medium | Use midpoint control points; tested with diamond and fan-out topologies |
| Click-to-expand overlay positioning breaks on scroll | Low | Medium | Position relative to DAG container, not page; recalculate on scroll |
| Test assertions brittle to constant changes | Low | Low | Test relationships (node A.X < node B.X) rather than absolute values |
| SSE updates don't refresh expanded detail panel | Medium | Low | `updateDAGNodeStatus()` also updates overlay if visible |

## 6. Testing Strategy

### Unit tests (`dag_test.go`)
- **TestComputeDAGLayout_SingleNode** — verify single node positioned at padding origin (no axis-specific assertion change needed)
- **TestComputeDAGLayout_LinearChain** — verify 3 nodes have 3 distinct X positions (was Y positions), same Y (was same X)
- **TestComputeDAGLayout_Diamond** — verify Width accommodates 3 layers (was Height); middle layer has 2 nodes at same X but different Y
- **TestAssignLayers** — unchanged (algorithm is direction-agnostic)
- **New: TestComputeDAGLayout_FanOut** — 1 source → 3 parallel targets, verify all targets at same X, different Y
- **New: TestComputeDAGLayout_EdgeDirectionLTR** — verify FromX < ToX for all edges (left-to-right flow)

### Manual testing
- Load run detail page with various pipeline shapes (linear, diamond, fan-out, fan-in)
- Verify click-to-expand shows detail panel and click again hides it
- Verify SSE updates animate running nodes and update detail panel
- Test responsive behavior at narrow viewports (mobile collapse)
