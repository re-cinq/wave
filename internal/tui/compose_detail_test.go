package tui

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper to build a CompatibilityResult with specific flow matches
// ---------------------------------------------------------------------------

// buildCompatibilityResult creates a CompatibilityResult from a sequence of
// two pipelines for testing renderArtifactFlow.
func buildCompatibilityResult(
	sourceName, targetName string, //nolint:unparam // test helper
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

		// Render at compact width (< 120) in sequential mode
		output := renderArtifactFlow(result, 80, false, Sequence{}, nil)

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

		output := renderArtifactFlow(result, 80, false, Sequence{}, nil)

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

		output := renderArtifactFlow(result, 80, false, Sequence{}, nil)

		assert.Contains(t, output, "⚠", "optional mismatch should show yellow warning")
		assert.Contains(t, output, "(optional", "optional mismatch should show '(optional' label")
	})

	t.Run("render degrades to text-only below 120", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		)

		compact := renderArtifactFlow(result, 80, false, Sequence{}, nil)
		full := renderArtifactFlow(result, 120, false, Sequence{}, nil)

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

	t.Run("renderExecutionPlan single stage", func(t *testing.T) {
		var seq Sequence
		seq.Add("pipeline-a", nil)
		seq.Add("pipeline-b", nil)
		stages := [][]int{{0, 1}}

		output := renderExecutionPlan(seq, stages, 80)
		assert.Contains(t, output, "Stage 1 (parallel)")
		assert.Contains(t, output, "pipeline-a")
		assert.Contains(t, output, "pipeline-b")
		assert.Contains(t, output, "┌─")
		assert.Contains(t, output, "└─")
	})

	t.Run("renderExecutionPlan multi-stage", func(t *testing.T) {
		var seq Sequence
		seq.Add("pipeline-a", nil)
		seq.Add("pipeline-b", nil)
		seq.Add("pipeline-c", nil)
		stages := [][]int{{0, 1}, {2}}

		output := renderExecutionPlan(seq, stages, 80)
		assert.Contains(t, output, "Stage 1 (parallel)")
		assert.Contains(t, output, "Stage 2 (sequential)")
		assert.Contains(t, output, "pipeline-c")
		assert.Contains(t, output, "│", "should have connector between stages")
	})

	t.Run("ComposeSequenceChangedMsg with parallel updates detail with stage-aware validation", func(t *testing.T) {
		m := NewComposeDetailModel()
		m.SetSize(80, 20)

		// Two pipelines in parallel (same stage) — should produce no inter-flow
		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		msg := ComposeSequenceChangedMsg{
			Sequence:   seq,
			Validation: ValidateSequence(seq), // linear validation (would show flow)
			Parallel:   true,
			Stages:     [][]int{{0, 1}}, // both in same stage
		}

		m, _ = m.Update(msg)
		assert.True(t, m.parallel)
		assert.Equal(t, 1, len(m.stages))
		// Stage-aware validation: single stage = no cross-stage flows
		assert.Equal(t, 0, len(m.validation.Flows),
			"single parallel stage should produce no cross-stage flows")
	})

	t.Run("ComposeSequenceChangedMsg with parallel multi-stage uses stage-aware validation", func(t *testing.T) {
		m := NewComposeDetailModel()
		m.SetSize(80, 20)

		var seq Sequence
		seq.Add("producer", testPipeline("producer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("consumer", testPipeline("consumer",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		msg := ComposeSequenceChangedMsg{
			Sequence:   seq,
			Validation: ValidateSequence(seq),
			Parallel:   true,
			Stages:     [][]int{{0}, {1}}, // two stages
		}

		m, _ = m.Update(msg)
		assert.True(t, m.parallel)
		assert.Equal(t, 2, len(m.stages))
		// Stage-aware validation: cross-stage flow should exist
		require.Equal(t, 1, len(m.validation.Flows),
			"two stages should produce one cross-stage flow")
		assert.Equal(t, "Stage 1", m.validation.Flows[0].SourcePipeline)
		assert.Equal(t, "Stage 2", m.validation.Flows[0].TargetPipeline)
	})

	t.Run("ComposeSequenceChangedMsg with sequential uses pre-computed validation", func(t *testing.T) {
		m := NewComposeDetailModel()
		m.SetSize(80, 20)

		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		preComputed := ValidateSequence(seq)
		msg := ComposeSequenceChangedMsg{
			Sequence:   seq,
			Validation: preComputed,
			Parallel:   false,
			Stages:     nil,
		}

		m, _ = m.Update(msg)
		assert.False(t, m.parallel)
		// Sequential mode uses the pre-computed validation with pipeline names
		require.Equal(t, 1, len(m.validation.Flows))
		assert.Equal(t, "a", m.validation.Flows[0].SourcePipeline)
		assert.Equal(t, "b", m.validation.Flows[0].TargetPipeline)
	})

	t.Run("View shows content for parallel single-stage", func(t *testing.T) {
		m := NewComposeDetailModel()
		m.SetSize(80, 20)

		var seq Sequence
		seq.Add("a", nil)
		seq.Add("b", nil)

		msg := ComposeSequenceChangedMsg{
			Sequence:   seq,
			Validation: ValidateSequence(seq),
			Parallel:   true,
			Stages:     [][]int{{0, 1}},
		}

		m, _ = m.Update(msg)
		view := m.View()
		// Should show stage structure, not the placeholder
		assert.NotContains(t, view, "Add pipelines to see artifact flow",
			"parallel mode with stages should not show placeholder")
	})
}

// ===========================================================================
// ValidateSequenceWithStages tests
// ===========================================================================

func TestValidateSequenceWithStages(t *testing.T) {
	t.Run("single stage produces no flows", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0, 1}} // all in one stage
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityValid, result.Status)
		assert.Equal(t, 0, len(result.Flows), "single stage should produce no cross-stage flows")
		assert.True(t, result.IsReady())
	})

	t.Run("two stages with cross-stage compatible flow", func(t *testing.T) {
		var seq Sequence
		seq.Add("producer", testPipeline("producer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("consumer", testPipeline("consumer",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0}, {1}} // two stages
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityValid, result.Status)
		require.Equal(t, 1, len(result.Flows))
		assert.Equal(t, "Stage 1", result.Flows[0].SourcePipeline)
		assert.Equal(t, "Stage 2", result.Flows[0].TargetPipeline)
		assert.True(t, result.IsReady())

		// Should have a compatible match
		var compatible int
		for _, m := range result.Flows[0].Matches {
			if m.Status == MatchCompatible {
				compatible++
				assert.Equal(t, "spec", m.OutputName)
				assert.Equal(t, "spec", m.InputName)
				assert.Equal(t, "spec_info", m.InputAs)
			}
		}
		assert.Equal(t, 1, compatible)
	})

	t.Run("two stages with missing required input", func(t *testing.T) {
		var seq Sequence
		seq.Add("producer", testPipeline("producer",
			[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
			nil,
		))
		seq.Add("consumer", testPipeline("consumer",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0}, {1}}
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityError, result.Status)
		require.Equal(t, 1, len(result.Flows))
		assert.False(t, result.IsReady())
		assert.Equal(t, 1, len(result.Diagnostics))
		assert.Contains(t, result.Diagnostics[0], "missing required input 'spec'")
	})

	t.Run("multi-stage produces flows at each boundary", func(t *testing.T) {
		var seq Sequence
		seq.Add("A", testPipeline("A",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("B", testPipeline("B",
			[]pipeline.ArtifactDef{{Name: "plan", Path: "output/plan.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))
		seq.Add("C", testPipeline("C",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "plan", As: "plan_info"}},
		))

		stages := [][]int{{0}, {1}, {2}} // three stages
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityValid, result.Status)
		require.Equal(t, 2, len(result.Flows))
		assert.Equal(t, "Stage 1", result.Flows[0].SourcePipeline)
		assert.Equal(t, "Stage 2", result.Flows[0].TargetPipeline)
		assert.Equal(t, "Stage 2", result.Flows[1].SourcePipeline)
		assert.Equal(t, "Stage 3", result.Flows[1].TargetPipeline)
	})

	t.Run("aggregates outputs from multiple pipelines in source stage", func(t *testing.T) {
		var seq Sequence
		// Two pipelines in stage 1 producing different outputs
		seq.Add("producer-a", testPipeline("producer-a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("producer-b", testPipeline("producer-b",
			[]pipeline.ArtifactDef{{Name: "plan", Path: "output/plan.json"}},
			nil,
		))
		// One pipeline in stage 2 consuming both
		seq.Add("consumer", testPipeline("consumer",
			nil,
			[]pipeline.ArtifactRef{
				{Artifact: "spec", As: "spec_info"},
				{Artifact: "plan", As: "plan_info"},
			},
		))

		stages := [][]int{{0, 1}, {2}} // parallel stage 1, sequential stage 2
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityValid, result.Status)
		require.Equal(t, 1, len(result.Flows))

		// Both inputs should be compatible
		var compatible int
		for _, m := range result.Flows[0].Matches {
			if m.Status == MatchCompatible {
				compatible++
			}
		}
		assert.Equal(t, 2, compatible, "both inputs should match aggregated outputs from parallel stage")
	})

	t.Run("aggregates inputs from multiple pipelines in target stage", func(t *testing.T) {
		var seq Sequence
		// One pipeline producing output
		seq.Add("producer", testPipeline("producer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		// Two pipelines in stage 2 — one needs spec, one needs plan (missing)
		seq.Add("consumer-a", testPipeline("consumer-a",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))
		seq.Add("consumer-b", testPipeline("consumer-b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "plan", As: "plan_info"}},
		))

		stages := [][]int{{0}, {1, 2}} // sequential stage 1, parallel stage 2
		result := ValidateSequenceWithStages(seq, stages)

		assert.Equal(t, CompatibilityError, result.Status)
		require.Equal(t, 1, len(result.Flows))

		// One compatible, one missing
		var compatible, missing int
		for _, m := range result.Flows[0].Matches {
			switch m.Status {
			case MatchCompatible:
				compatible++
			case MatchMissing:
				missing++
			}
		}
		assert.Equal(t, 1, compatible, "spec should match")
		assert.Equal(t, 1, missing, "plan should be missing")
	})

	t.Run("empty stages produces no flows", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", nil)
		result := ValidateSequenceWithStages(seq, [][]int{})
		assert.Equal(t, CompatibilityValid, result.Status)
		assert.Equal(t, 0, len(result.Flows))
	})
}

// ===========================================================================
// Stage-aware rendering tests
// ===========================================================================

func TestRenderArtifactFlowParallel(t *testing.T) {
	t.Run("stage headers visible in parallel mode compact", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0}, {1}}
		result := ValidateSequenceWithStages(seq, stages)
		output := renderArtifactFlow(result, 80, true, seq, stages)

		assert.Contains(t, output, "Stage 1")
		assert.Contains(t, output, "Stage 2")
		assert.Contains(t, output, "a")
		assert.Contains(t, output, "b")
	})

	t.Run("stage headers visible in parallel mode full", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0}, {1}}
		result := ValidateSequenceWithStages(seq, stages)
		output := renderArtifactFlow(result, 120, true, seq, stages)

		assert.Contains(t, output, "Stage 1")
		assert.Contains(t, output, "Stage 2")
		assert.Contains(t, output, "┌")
		assert.Contains(t, output, "└")
	})

	t.Run("no intra-stage flows in parallel mode", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", testPipeline("a",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("b", testPipeline("b",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		// Both in same parallel stage — should have no flows
		stages := [][]int{{0, 1}}
		result := ValidateSequenceWithStages(seq, stages)
		output := renderArtifactFlow(result, 80, true, seq, stages)

		assert.NotContains(t, output, "(compatible)",
			"same-stage pipelines should not show artifact flows between them")
		assert.Contains(t, output, "Stage 1",
			"should still show stage structure")
		assert.Contains(t, output, "(parallel)",
			"should show parallel mode label")
	})

	t.Run("cross-stage flows correct in compact mode", func(t *testing.T) {
		var seq Sequence
		seq.Add("producer", testPipeline("producer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			nil,
		))
		seq.Add("consumer", testPipeline("consumer",
			nil,
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		))

		stages := [][]int{{0}, {1}}
		result := ValidateSequenceWithStages(seq, stages)
		output := renderArtifactFlow(result, 80, true, seq, stages)

		assert.Contains(t, output, "✓")
		assert.Contains(t, output, "(compatible)")
		assert.Contains(t, output, "spec")
	})

	t.Run("parallel stage shows branch characters", func(t *testing.T) {
		var seq Sequence
		seq.Add("a", nil)
		seq.Add("b", nil)
		seq.Add("c", nil)

		stages := [][]int{{0, 1}, {2}}
		result := ValidateSequenceWithStages(seq, stages)
		output := renderArtifactFlow(result, 80, true, seq, stages)

		assert.Contains(t, output, "┌─")
		assert.Contains(t, output, "└─")
		assert.Contains(t, output, "(parallel)")
		assert.Contains(t, output, "(sequential)")
	})
}

func TestRenderArtifactFlowSequentialUnchanged(t *testing.T) {
	t.Run("sequential mode produces same output with new signature", func(t *testing.T) {
		result := buildCompatibilityResult(
			"producer", "consumer",
			[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
			[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
		)

		output := renderArtifactFlow(result, 80, false, Sequence{}, nil)

		// Should contain the same elements as before
		assert.Contains(t, output, "producer → consumer")
		assert.Contains(t, output, "✓")
		assert.Contains(t, output, "(compatible)")
		assert.Contains(t, output, "All artifact flows compatible")
	})

	t.Run("sequential mode empty shows placeholder", func(t *testing.T) {
		result := CompatibilityResult{Status: CompatibilityValid}
		output := renderArtifactFlow(result, 80, false, Sequence{}, nil)
		assert.Equal(t, "Add pipelines to see artifact flow", output)
	})
}
