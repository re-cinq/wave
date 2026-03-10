package tui

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helper to build a CompatibilityResult with specific flow matches
// ---------------------------------------------------------------------------

// buildCompatibilityResult creates a CompatibilityResult from a sequence of
// two pipelines for testing renderArtifactFlow.
func buildCompatibilityResult(
	sourceName, targetName string,
	outputs []pipeline.ArtifactDef,
	inputs []pipeline.ArtifactRef,
) CompatibilityResult {
	var seq Sequence
	seq.Add(sourceName, testPipeline(sourceName, outputs, nil))
	seq.Add(targetName, testPipeline(targetName, nil, inputs))
	return ValidateSequence(seq)
}

// ===========================================================================
// ComposeDetailModel tests
// ===========================================================================

func TestComposeDetailModel(t *testing.T) {
	t.Run("render with compatible flows shows green indicator", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		)
		assert.Equal(t, CompatibilityValid, result.Status)

		// Render at compact width (< 120)
		output := renderArtifactFlow(result, 80)

		assert.Contains(t, output, "✓", "compatible flow should show green checkmark")
		assert.Contains(t, output, "(compatible)", "compatible flow should show '(compatible)' label")
	})

	t.Run("render with missing required shows red indicator", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}}, // not matching "report"
		)
		assert.Equal(t, CompatibilityError, result.Status)

		output := renderArtifactFlow(result, 80)

		assert.Contains(t, output, "✗", "missing required flow should show red cross")
		assert.Contains(t, output, "(missing", "missing required flow should show '(missing' label")
	})

	t.Run("render with optional mismatch shows yellow indicator", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
			[]pipeline.ArtifactRef{{Artifact: "hints", As: "hints_info", Optional: true}},
		)
		assert.Equal(t, CompatibilityWarning, result.Status)

		output := renderArtifactFlow(result, 80)

		assert.Contains(t, output, "⚠", "optional mismatch should show yellow warning")
		assert.Contains(t, output, "(optional", "optional mismatch should show '(optional' label")
	})

	t.Run("render degrades to text-only below 120", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		)

		compact := renderArtifactFlow(result, 80)
		full := renderArtifactFlow(result, 120)

		// At width < 120 (compact), no box-drawing characters should appear
		assert.False(t, strings.Contains(compact, "┌"),
			"compact mode (width 80) should not contain box-drawing characters")
		assert.False(t, strings.Contains(compact, "└"),
			"compact mode (width 80) should not contain box-drawing characters")

		// At width >= 120 (full), box-drawing characters should appear
		assert.True(t, strings.Contains(full, "┌"),
			"full mode (width 120) should contain box-drawing characters")
		assert.True(t, strings.Contains(full, "└"),
			"full mode (width 120) should contain box-drawing characters")
	})

	t.Run("empty validation renders placeholder", func(t *testing.T) {
		m := NewComposeDetailModel()
		m.SetSize(80, 20)

		view := m.View()
		assert.Contains(t, view, "Add pipelines to see artifact flow",
			"empty validation should render placeholder text")
	})

	t.Run("SetSize updates viewport dimensions", func(t *testing.T) {
		m := NewComposeDetailModel()

		m.SetSize(100, 50)

		assert.Equal(t, 100, m.width)
		assert.Equal(t, 50, m.height)
		assert.Equal(t, 100, m.viewport.Width)
		assert.Equal(t, 50, m.viewport.Height)
	})
}
