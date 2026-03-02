package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderDAGPreview_Empty(t *testing.T) {
	result := RenderDAGPreview(nil, 80)
	assert.Equal(t, "", result)

	result = RenderDAGPreview([]Proposal{}, 80)
	assert.Equal(t, "", result)
}

func TestRenderDAGPreview_SingleNode(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "build", Reason: "compile the project"},
	}

	result := RenderDAGPreview(proposals, 80)
	assert.Contains(t, result, "[ ")
	assert.Contains(t, result, "build")
	assert.Contains(t, result, " ]")

	// A single node should have no connector characters.
	assert.NotContains(t, result, "│")
	assert.NotContains(t, result, "╱")
	assert.NotContains(t, result, "╲")
}

func TestRenderDAGPreview_LinearChain(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "A"},
		{Pipeline: "B", Dependencies: []string{"A"}},
		{Pipeline: "C", Dependencies: []string{"B"}},
	}

	result := RenderDAGPreview(proposals, 80)

	assert.Contains(t, result, "[ ")
	assert.Contains(t, result, "A")
	assert.Contains(t, result, "B")
	assert.Contains(t, result, "C")
	assert.Contains(t, result, " ]")

	// Should contain vertical connectors between layers.
	assert.Contains(t, result, "│")

	// A must appear before B, and B before C in the rendered output.
	// Strip ANSI for reliable index comparison.
	plain := stripANSI(result)
	idxA := strings.Index(plain, "A")
	idxB := strings.Index(plain, "B")
	idxC := strings.Index(plain, "C")
	assert.Greater(t, idxB, idxA, "B should appear after A")
	assert.Greater(t, idxC, idxB, "C should appear after B")
}

func TestRenderDAGPreview_Diamond(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "A"},
		{Pipeline: "B", Dependencies: []string{"A"}},
		{Pipeline: "C", Dependencies: []string{"A"}},
		{Pipeline: "D", Dependencies: []string{"B", "C"}},
	}

	result := RenderDAGPreview(proposals, 80)

	// All four pipelines must appear.
	for _, name := range []string{"A", "B", "C", "D"} {
		assert.Contains(t, result, name, "should contain pipeline %s", name)
	}

	// Fan-out from A to B,C should produce fan-out connectors.
	assert.True(t,
		strings.Contains(result, "╱") || strings.Contains(result, "╲"),
		"diamond should contain fan-out or fan-in connectors",
	)

	// A should appear first, D should appear last.
	plain := stripANSI(result)
	idxA := strings.Index(plain, "A")
	idxD := strings.LastIndex(plain, "D")
	assert.Greater(t, idxD, idxA, "D should appear after A")
}

func TestRenderDAGPreview_ParallelGroup(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "A"},
		{Pipeline: "B", Dependencies: []string{"A"}, ParallelGroup: "group1"},
		{Pipeline: "C", Dependencies: []string{"A"}, ParallelGroup: "group1"},
	}

	result := RenderDAGPreview(proposals, 80)

	assert.Contains(t, result, "parallel", "parallel group should be labeled")
	assert.Contains(t, result, "B")
	assert.Contains(t, result, "C")
}

func TestRenderDAGPreview_SmallTermWidth(t *testing.T) {
	proposals := []Proposal{
		{Pipeline: "test"},
	}

	// termWidth below minimum (20) should be clamped up.
	result := RenderDAGPreview(proposals, 10)
	assert.NotEmpty(t, result, "small termWidth should still produce output")
	assert.Contains(t, result, "test")
}

func TestRenderDAGPreview_LabelTruncation(t *testing.T) {
	longName := strings.Repeat("a", 55) // 55 chars, well over what 30-width can show
	proposals := []Proposal{
		{Pipeline: longName},
	}

	result := RenderDAGPreview(proposals, 30)
	assert.Contains(t, result, "…", "long label should be truncated with ellipsis")
}

func TestAssignProposalLayers(t *testing.T) {
	t.Run("linear chain produces 3 layers", func(t *testing.T) {
		proposals := []Proposal{
			{Pipeline: "A"},
			{Pipeline: "B", Dependencies: []string{"A"}},
			{Pipeline: "C", Dependencies: []string{"B"}},
		}

		proposalMap := make(map[string]Proposal, len(proposals))
		for _, p := range proposals {
			proposalMap[p.Pipeline] = p
		}

		adj := make(map[string][]string)
		inDeg := make(map[string]int, len(proposals))
		for _, p := range proposals {
			if _, ok := inDeg[p.Pipeline]; !ok {
				inDeg[p.Pipeline] = 0
			}
			for _, dep := range p.Dependencies {
				if _, exists := proposalMap[dep]; exists {
					adj[dep] = append(adj[dep], p.Pipeline)
					inDeg[p.Pipeline]++
				}
			}
		}

		layers := assignProposalLayers(proposals, adj, inDeg)
		assert.Len(t, layers, 3)
		assert.Equal(t, []string{"A"}, layers[0])
		assert.Equal(t, []string{"B"}, layers[1])
		assert.Equal(t, []string{"C"}, layers[2])
	})

	t.Run("diamond produces 3 layers with B and C in same layer", func(t *testing.T) {
		proposals := []Proposal{
			{Pipeline: "A"},
			{Pipeline: "B", Dependencies: []string{"A"}},
			{Pipeline: "C", Dependencies: []string{"A"}},
			{Pipeline: "D", Dependencies: []string{"B", "C"}},
		}

		proposalMap := make(map[string]Proposal, len(proposals))
		for _, p := range proposals {
			proposalMap[p.Pipeline] = p
		}

		adj := make(map[string][]string)
		inDeg := make(map[string]int, len(proposals))
		for _, p := range proposals {
			if _, ok := inDeg[p.Pipeline]; !ok {
				inDeg[p.Pipeline] = 0
			}
			for _, dep := range p.Dependencies {
				if _, exists := proposalMap[dep]; exists {
					adj[dep] = append(adj[dep], p.Pipeline)
					inDeg[p.Pipeline]++
				}
			}
		}

		layers := assignProposalLayers(proposals, adj, inDeg)
		assert.Len(t, layers, 3)
		assert.Equal(t, []string{"A"}, layers[0])
		assert.Equal(t, []string{"B", "C"}, layers[1])
		assert.Equal(t, []string{"D"}, layers[2])
	})

	t.Run("single node produces 1 layer", func(t *testing.T) {
		proposals := []Proposal{
			{Pipeline: "X"},
		}

		adj := make(map[string][]string)
		inDeg := map[string]int{"X": 0}

		layers := assignProposalLayers(proposals, adj, inDeg)
		assert.Len(t, layers, 1)
		assert.Equal(t, []string{"X"}, layers[0])
	})
}

func TestCenterLine(t *testing.T) {
	t.Run("short string gets padded", func(t *testing.T) {
		result := centerLine("hi", 40)
		// "hi" is 2 chars wide; padding should be (40-2)/2 = 19 spaces.
		assert.True(t, len(result) > len("hi"), "centered string should be longer than input")
		assert.True(t, strings.HasPrefix(result, " "), "centered string should start with spaces")
		assert.Contains(t, result, "hi")
	})

	t.Run("string wider than termWidth returned as-is", func(t *testing.T) {
		wide := strings.Repeat("x", 50)
		result := centerLine(wide, 30)
		assert.Equal(t, wide, result)
	})

	t.Run("exact width returned as-is", func(t *testing.T) {
		exact := strings.Repeat("x", 40)
		result := centerLine(exact, 40)
		assert.Equal(t, exact, result)
	})
}

// stripANSI removes ANSI escape sequences from a string for reliable text
// comparison. It handles CSI sequences (\x1b[...m) and OSC sequences.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			// CSI sequences end with a letter in the range 0x40-0x7E.
			if s[i] >= 0x40 && s[i] <= 0x7E {
				inEsc = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
