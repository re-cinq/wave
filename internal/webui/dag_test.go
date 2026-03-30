package webui

import (
	"testing"
)

func TestComputeDAGLayout_Empty(t *testing.T) {
	result := ComputeDAGLayout(nil)
	if result != nil {
		t.Error("expected nil for empty steps")
	}
}

func TestComputeDAGLayout_SingleNode(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "step1", Persona: "worker", Status: "completed"},
	}

	layout := ComputeDAGLayout(steps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if len(layout.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(layout.Nodes))
	}
	if len(layout.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(layout.Edges))
	}
}

func TestComputeDAGLayout_LinearChain(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "step1", Persona: "spec", Status: "completed"},
		{ID: "step2", Persona: "impl", Status: "running", Dependencies: []string{"step1"}},
		{ID: "step3", Persona: "test", Status: "pending", Dependencies: []string{"step2"}},
	}

	layout := ComputeDAGLayout(steps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if len(layout.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(layout.Nodes))
	}
	if len(layout.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(layout.Edges))
	}

	// Verify nodes are in different layers (different X positions in left-to-right layout)
	xPositions := make(map[int]bool)
	for _, n := range layout.Nodes {
		xPositions[n.X] = true
	}
	if len(xPositions) != 3 {
		t.Errorf("expected 3 different X positions for linear chain, got %d", len(xPositions))
	}

	// All nodes in a linear chain have 1 node per layer, so Y should be the same
	yPositions := make(map[int]bool)
	for _, n := range layout.Nodes {
		yPositions[n.Y] = true
	}
	if len(yPositions) != 1 {
		t.Errorf("expected all nodes to share same Y position in linear chain, got %d unique", len(yPositions))
	}
}

func TestComputeDAGLayout_Diamond(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "start", Persona: "nav", Status: "completed"},
		{ID: "left", Persona: "dev1", Status: "completed", Dependencies: []string{"start"}},
		{ID: "right", Persona: "dev2", Status: "completed", Dependencies: []string{"start"}},
		{ID: "end", Persona: "review", Status: "pending", Dependencies: []string{"left", "right"}},
	}

	layout := ComputeDAGLayout(steps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if len(layout.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(layout.Nodes))
	}
	if len(layout.Edges) != 4 {
		t.Errorf("expected 4 edges, got %d", len(layout.Edges))
	}

	// Width should accommodate at least 3 layers (left-to-right layout)
	if layout.Width < 3*layerGapX {
		t.Errorf("expected width >= %d for 3-layer graph, got %d", 3*layerGapX, layout.Width)
	}
}

func TestComputeDAGLayout_FanOut(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "source", Persona: "nav", Status: "completed"},
		{ID: "a", Persona: "dev", Status: "pending", Dependencies: []string{"source"}},
		{ID: "b", Persona: "dev", Status: "pending", Dependencies: []string{"source"}},
		{ID: "c", Persona: "dev", Status: "pending", Dependencies: []string{"source"}},
	}

	layout := ComputeDAGLayout(steps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if len(layout.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(layout.Nodes))
	}

	// Find target nodes (a, b, c)
	nodeMap := make(map[string]DAGLayoutNode)
	for _, n := range layout.Nodes {
		nodeMap[n.ID] = n
	}

	// All 3 targets should share the same X position (same layer)
	if nodeMap["a"].X != nodeMap["b"].X || nodeMap["b"].X != nodeMap["c"].X {
		t.Errorf("fan-out targets should share X position: a=%d, b=%d, c=%d",
			nodeMap["a"].X, nodeMap["b"].X, nodeMap["c"].X)
	}

	// All 3 targets should have different Y positions
	ySet := map[int]bool{nodeMap["a"].Y: true, nodeMap["b"].Y: true, nodeMap["c"].Y: true}
	if len(ySet) != 3 {
		t.Errorf("fan-out targets should have different Y positions, got %d unique",
			len(ySet))
	}

	// Source should be to the left of targets
	if nodeMap["source"].X >= nodeMap["a"].X {
		t.Errorf("source X (%d) should be less than target X (%d)",
			nodeMap["source"].X, nodeMap["a"].X)
	}
}

func TestComputeDAGLayout_EdgeDirectionLTR(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "start", Persona: "nav", Status: "completed"},
		{ID: "left", Persona: "dev1", Status: "completed", Dependencies: []string{"start"}},
		{ID: "right", Persona: "dev2", Status: "completed", Dependencies: []string{"start"}},
		{ID: "end", Persona: "review", Status: "pending", Dependencies: []string{"left", "right"}},
	}

	layout := ComputeDAGLayout(steps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	for _, edge := range layout.Edges {
		if edge.FromX >= edge.ToX {
			t.Errorf("edge %s→%s has FromX=%d >= ToX=%d, expected left-to-right flow",
				edge.From, edge.To, edge.FromX, edge.ToX)
		}
	}
}

func TestStripExcludedDeps(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "a", Dependencies: []string{"rework"}},
		{ID: "b", Dependencies: []string{"a", "rework"}},
		{ID: "c"},
	}
	excluded := map[string]bool{"rework": true}
	stripExcludedDeps(steps, excluded)

	if len(steps[0].Dependencies) != 0 {
		t.Errorf("expected 0 deps for step a, got %v", steps[0].Dependencies)
	}
	if len(steps[1].Dependencies) != 1 || steps[1].Dependencies[0] != "a" {
		t.Errorf("expected [a] deps for step b, got %v", steps[1].Dependencies)
	}
	if steps[2].Dependencies != nil {
		t.Errorf("expected nil deps for step c, got %v", steps[2].Dependencies)
	}
}

func TestComputeDAGLayout_ReworkOnlyExcluded(t *testing.T) {
	// Simulate the filtering that handlers do: rework-only steps are excluded
	// before passing to ComputeDAGLayout, and dangling deps are stripped.
	allSteps := []DAGStepInput{
		{ID: "fetch", Persona: "nav", Status: "completed"},
		{ID: "implement", Persona: "dev", Status: "running", Dependencies: []string{"fetch"}},
		{ID: "fix-implement", Persona: "dev", Status: "pending"}, // rework_only — no deps
		{ID: "create-pr", Persona: "nav", Status: "pending", Dependencies: []string{"implement"}},
	}

	// Filter out rework-only step (as handlers would)
	excluded := map[string]bool{"fix-implement": true}
	var dagSteps []DAGStepInput
	for _, s := range allSteps {
		if !excluded[s.ID] {
			dagSteps = append(dagSteps, s)
		}
	}
	stripExcludedDeps(dagSteps, excluded)

	layout := ComputeDAGLayout(dagSteps)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if len(layout.Nodes) != 3 {
		t.Errorf("expected 3 nodes (rework step excluded), got %d", len(layout.Nodes))
	}

	// Verify fix-implement is not in the layout
	for _, n := range layout.Nodes {
		if n.ID == "fix-implement" {
			t.Error("rework-only step fix-implement should not appear in DAG layout")
		}
	}

	// Verify the remaining nodes form a valid 3-layer chain
	nodeMap := make(map[string]DAGLayoutNode)
	for _, n := range layout.Nodes {
		nodeMap[n.ID] = n
	}
	if nodeMap["fetch"].X >= nodeMap["implement"].X {
		t.Error("fetch should be left of implement")
	}
	if nodeMap["implement"].X >= nodeMap["create-pr"].X {
		t.Error("implement should be left of create-pr")
	}
}

func TestAssignLayers(t *testing.T) {
	steps := []DAGStepInput{
		{ID: "a"},
		{ID: "b", Dependencies: []string{"a"}},
		{ID: "c", Dependencies: []string{"a"}},
		{ID: "d", Dependencies: []string{"b", "c"}},
	}

	adj := map[string][]string{
		"a": {"b", "c"},
		"b": {"d"},
		"c": {"d"},
	}
	inDeg := map[string]int{
		"a": 0,
		"b": 1,
		"c": 1,
		"d": 2,
	}

	layers := assignLayers(steps, adj, inDeg)
	if len(layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(layers))
	}

	// First layer should contain "a"
	if len(layers[0]) != 1 || layers[0][0] != "a" {
		t.Errorf("expected first layer to contain only 'a', got %v", layers[0])
	}

	// Second layer should contain "b" and "c"
	if len(layers[1]) != 2 {
		t.Errorf("expected second layer to have 2 nodes, got %d", len(layers[1]))
	}

	// Third layer should contain "d"
	if len(layers[2]) != 1 || layers[2][0] != "d" {
		t.Errorf("expected third layer to contain only 'd', got %v", layers[2])
	}
}
