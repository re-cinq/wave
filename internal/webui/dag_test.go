//go:build webui

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

	// Verify nodes are in different layers (different X positions)
	xPositions := make(map[int]bool)
	for _, n := range layout.Nodes {
		xPositions[n.X] = true
	}
	if len(xPositions) != 3 {
		t.Errorf("expected 3 different X positions for linear chain, got %d", len(xPositions))
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

	// Width should accommodate at least 3 layers
	if layout.Width < 3*layerGapX {
		t.Errorf("expected width >= %d for 3-layer graph, got %d", 3*layerGapX, layout.Width)
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
