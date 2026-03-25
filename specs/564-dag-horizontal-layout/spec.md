# feat(webui): DAG visualization redesign — horizontal left-to-right layout

**Issue**: [#564](https://github.com/re-cinq/wave/issues/564)
**Parent**: Extracted from #550 — Feature 4
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit

## Problem

The pipeline DAG is rendered top-to-bottom, while GitHub Actions and GitLab CI use left-to-right horizontal layouts. The current DAG also lacks click-to-expand step details.

## Changes Required

### Backend (`internal/webui/dag.go`)
- Swap `ComputeDAGLayout()` from vertical (layerGapY, nodeGapX) to horizontal (layerGapX, nodeGapY)
- Adjust SVG viewBox dimensions for horizontal flow
- Recalculate bezier curve edge paths for left-to-right direction

### Frontend (`dag_svg.html` + `sse.js`)
- Update SVG template for horizontal node placement
- Add click-to-expand inline step details (currently clicking scrolls to step card)
- Step nodes show: status icon, step ID, persona, duration
- Edges show dependency flow direction
- Status updates via SSE remain working (`updateDAGNodeStatus()`)

### Edge Cases
- Linear pipelines (single column -> single row)
- Fan-out topologies (one step -> many parallel)
- Fan-in topologies (many steps -> single join)
- Mixed fan-out/fan-in (diamond patterns)

## Acceptance Criteria

- [ ] DAG renders left-to-right
- [ ] Step nodes show status icon, ID, persona, duration
- [ ] Edges show dependency flow with directional arrows
- [ ] Click on a node expands step details inline (or scrolls to card)
- [ ] DAG renders correctly for linear, fan-out, fan-in, and diamond topologies
- [ ] Live status updates work during running pipelines
