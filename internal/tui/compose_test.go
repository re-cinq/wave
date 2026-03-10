package tui

import (
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helper
// ---------------------------------------------------------------------------

// testPipeline builds a minimal Pipeline for compose/sequence testing.
//
//   - If only outputs are provided: one step (last) with OutputArtifacts.
//   - If only inputs are provided: one step (first) with Memory.InjectArtifacts.
//   - If both outputs and inputs are provided: two steps — first with inputs,
//     second with outputs.
//   - The pipeline name is set via Metadata.Name.
func testPipeline(name string, outputs []pipeline.ArtifactDef, inputs []pipeline.ArtifactRef) *pipeline.Pipeline {
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{Name: name},
	}

	hasOutputs := len(outputs) > 0
	hasInputs := len(inputs) > 0

	switch {
	case hasOutputs && hasInputs:
		// Two steps: first with inputs, second with outputs.
		p.Steps = []pipeline.Step{
			{
				ID: name + "-step-1",
				Memory: pipeline.MemoryConfig{
					InjectArtifacts: inputs,
				},
			},
			{
				ID:              name + "-step-2",
				OutputArtifacts: outputs,
			},
		}
	case hasOutputs:
		// One step with outputs only.
		p.Steps = []pipeline.Step{
			{
				ID:              name + "-step-1",
				OutputArtifacts: outputs,
			},
		}
	case hasInputs:
		// One step with inputs only.
		p.Steps = []pipeline.Step{
			{
				ID: name + "-step-1",
				Memory: pipeline.MemoryConfig{
					InjectArtifacts: inputs,
				},
			},
		}
	default:
		// No outputs or inputs — single empty step.
		p.Steps = []pipeline.Step{
			{ID: name + "-step-1"},
		}
	}

	return p
}

// ===========================================================================
// ValidateSequence tests
// ===========================================================================

func TestValidateSequence(t *testing.T) {
	tests := []struct {
		name               string
		seq                Sequence
		wantStatus         CompatibilityStatus
		wantFlowCount      int
		wantDiagCount      int
		wantDiagSubstrings []string // partial strings expected in diagnostics
		wantReady          bool
		checkMatches       func(t *testing.T, result CompatibilityResult)
	}{
		{
			name:          "empty sequence",
			seq:           Sequence{},
			wantStatus:    CompatibilityValid,
			wantFlowCount: 0,
			wantDiagCount: 0,
			wantReady:     true,
		},
		{
			name: "single pipeline",
			seq: func() Sequence {
				var s Sequence
				s.Add("alpha", testPipeline("alpha",
					[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
					nil,
				))
				return s
			}(),
			wantStatus:    CompatibilityValid,
			wantFlowCount: 0,
			wantDiagCount: 0,
			wantReady:     true,
		},
		{
			name: "two compatible pipelines",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
					nil,
				))
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
				))
				return s
			}(),
			wantStatus:    CompatibilityValid,
			wantFlowCount: 1,
			wantDiagCount: 0,
			wantReady:     true,
			checkMatches: func(t *testing.T, result CompatibilityResult) {
				require.Len(t, result.Flows, 1)
				flow := result.Flows[0]
				assert.Equal(t, "producer", flow.SourcePipeline)
				assert.Equal(t, "consumer", flow.TargetPipeline)

				// Should have one compatible match
				var compatible int
				for _, m := range flow.Matches {
					if m.Status == MatchCompatible {
						compatible++
						assert.Equal(t, "spec", m.OutputName)
						assert.Equal(t, "spec", m.InputName)
						assert.Equal(t, "spec_info", m.InputAs)
					}
				}
				assert.Equal(t, 1, compatible)
			},
		},
		{
			name: "two incompatible pipelines with required input missing",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
					nil,
				))
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
				))
				return s
			}(),
			wantStatus:         CompatibilityError,
			wantFlowCount:      1,
			wantDiagCount:      1,
			wantDiagSubstrings: []string{"missing required input 'spec'"},
			wantReady:          false,
		},
		{
			name: "optional input missing",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{{Name: "report", Path: "output/report.json"}},
					nil,
				))
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "hints", As: "hints_info", Optional: true}},
				))
				return s
			}(),
			wantStatus:         CompatibilityWarning,
			wantFlowCount:      1,
			wantDiagCount:      1,
			wantDiagSubstrings: []string{"optional input 'hints' has no matching output"},
			wantReady:          true,
		},
		{
			name: "pipeline with no output artifacts warns for required inputs",
			seq: func() Sequence {
				var s Sequence
				// Producer has no output artifacts at all
				s.Add("empty-producer", testPipeline("empty-producer", nil, nil))
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "data", As: "data_info"}},
				))
				return s
			}(),
			wantStatus:         CompatibilityError,
			wantFlowCount:      1,
			wantDiagCount:      1,
			wantDiagSubstrings: []string{"missing required input 'data'"},
			wantReady:          false,
		},
		{
			name: "pipeline with no input artifacts is valid",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
					nil,
				))
				// Consumer has no inputs — nothing to match
				s.Add("no-inputs", testPipeline("no-inputs", nil, nil))
				return s
			}(),
			wantStatus:    CompatibilityValid,
			wantFlowCount: 1,
			wantDiagCount: 0,
			wantReady:     true,
			checkMatches: func(t *testing.T, result CompatibilityResult) {
				require.Len(t, result.Flows, 1)
				flow := result.Flows[0]
				// The output "spec" should be unmatched since consumer has no inputs
				var unmatched int
				for _, m := range flow.Matches {
					if m.Status == MatchUnmatched {
						unmatched++
						assert.Equal(t, "spec", m.OutputName)
					}
				}
				assert.Equal(t, 1, unmatched)
			},
		},
		{
			name: "three pipeline chain with mixed compatibility",
			seq: func() Sequence {
				var s Sequence
				// A produces "spec"
				s.Add("A", testPipeline("A",
					[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
					nil,
				))
				// B consumes "spec" and produces "plan"
				s.Add("B", testPipeline("B",
					[]pipeline.ArtifactDef{{Name: "plan", Path: "output/plan.json"}},
					[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
				))
				// C requires "report" which B does not produce
				s.Add("C", testPipeline("C",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "report", As: "report_info"}},
				))
				return s
			}(),
			wantStatus:         CompatibilityError,
			wantFlowCount:      2,
			wantDiagCount:      1,
			wantDiagSubstrings: []string{"B → C: missing required input 'report'"},
			wantReady:          false,
			checkMatches: func(t *testing.T, result CompatibilityResult) {
				require.Len(t, result.Flows, 2)

				// A→B: ValidateSequence checks last step outputs of A vs first step inputs of B.
				// For testPipeline("B", outputs, inputs): first step has inputs, second has outputs.
				// So B's first step injects "spec".
				flowAB := result.Flows[0]
				assert.Equal(t, "A", flowAB.SourcePipeline)
				assert.Equal(t, "B", flowAB.TargetPipeline)
				var compatAB int
				for _, m := range flowAB.Matches {
					if m.Status == MatchCompatible {
						compatAB++
					}
				}
				assert.Equal(t, 1, compatAB, "A→B should have one compatible match")

				// B→C: B's last step (step-2) produces "plan"; C needs "report"
				flowBC := result.Flows[1]
				assert.Equal(t, "B", flowBC.SourcePipeline)
				assert.Equal(t, "C", flowBC.TargetPipeline)
				var missingBC int
				for _, m := range flowBC.Matches {
					if m.Status == MatchMissing {
						missingBC++
					}
				}
				assert.Equal(t, 1, missingBC, "B→C should have one missing match")
			},
		},
		{
			name: "unmatched outputs do not affect overall status",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{
						{Name: "spec", Path: "output/spec.json"},
						{Name: "extra", Path: "output/extra.json"},
					},
					nil,
				))
				// Consumer only consumes "spec"; "extra" is unmatched
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{{Artifact: "spec", As: "spec_info"}},
				))
				return s
			}(),
			wantStatus:    CompatibilityValid,
			wantFlowCount: 1,
			wantDiagCount: 0,
			wantReady:     true,
			checkMatches: func(t *testing.T, result CompatibilityResult) {
				require.Len(t, result.Flows, 1)
				flow := result.Flows[0]
				var unmatched int
				for _, m := range flow.Matches {
					if m.Status == MatchUnmatched {
						unmatched++
						assert.Equal(t, "extra", m.OutputName)
					}
				}
				assert.Equal(t, 1, unmatched, "one output should be unmatched")
			},
		},
		{
			name: "multiple inputs with partial match",
			seq: func() Sequence {
				var s Sequence
				s.Add("producer", testPipeline("producer",
					[]pipeline.ArtifactDef{{Name: "spec", Path: "output/spec.json"}},
					nil,
				))
				// Consumer requires "spec" (matched) and "plan" (missing required)
				s.Add("consumer", testPipeline("consumer",
					nil,
					[]pipeline.ArtifactRef{
						{Artifact: "spec", As: "spec_info"},
						{Artifact: "plan", As: "plan_info"},
					},
				))
				return s
			}(),
			wantStatus:         CompatibilityError,
			wantFlowCount:      1,
			wantDiagCount:      1,
			wantDiagSubstrings: []string{"missing required input 'plan'"},
			wantReady:          false,
			checkMatches: func(t *testing.T, result CompatibilityResult) {
				require.Len(t, result.Flows, 1)
				flow := result.Flows[0]
				var compatible, missing int
				for _, m := range flow.Matches {
					switch m.Status {
					case MatchCompatible:
						compatible++
					case MatchMissing:
						missing++
					}
				}
				assert.Equal(t, 1, compatible, "one input should match")
				assert.Equal(t, 1, missing, "one input should be missing")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSequence(tt.seq)

			assert.Equal(t, tt.wantStatus, result.Status, "unexpected compatibility status")
			assert.Equal(t, tt.wantFlowCount, len(result.Flows), "unexpected flow count")
			assert.Equal(t, tt.wantDiagCount, len(result.Diagnostics), "unexpected diagnostic count")
			assert.Equal(t, tt.wantReady, result.IsReady(), "unexpected IsReady result")

			for _, substr := range tt.wantDiagSubstrings {
				found := false
				for _, diag := range result.Diagnostics {
					if contains(diag, substr) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected diagnostic containing %q, got %v", substr, result.Diagnostics)
			}

			if tt.checkMatches != nil {
				tt.checkMatches(t, result)
			}
		})
	}
}

// contains checks if s contains substr using a simple scan.
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}

// ===========================================================================
// Sequence method tests
// ===========================================================================

func TestSequence(t *testing.T) {
	t.Run("Add appends entries correctly", func(t *testing.T) {
		var s Sequence
		p1 := testPipeline("alpha", nil, nil)
		p2 := testPipeline("beta", nil, nil)

		s.Add("alpha", p1)
		assert.Equal(t, 1, s.Len())
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
		assert.Equal(t, p1, s.Entries[0].Pipeline)

		s.Add("beta", p2)
		assert.Equal(t, 2, s.Len())
		assert.Equal(t, "beta", s.Entries[1].PipelineName)
		assert.Equal(t, p2, s.Entries[1].Pipeline)
	})

	t.Run("Remove at valid index", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))
		s.Add("gamma", testPipeline("gamma", nil, nil))

		s.Remove(1)
		require.Equal(t, 2, s.Len())
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
		assert.Equal(t, "gamma", s.Entries[1].PipelineName)
	})

	t.Run("Remove first element", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.Remove(0)
		require.Equal(t, 1, s.Len())
		assert.Equal(t, "beta", s.Entries[0].PipelineName)
	})

	t.Run("Remove last element", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.Remove(1)
		require.Equal(t, 1, s.Len())
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
	})

	t.Run("Remove at negative index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.Remove(-1)
		assert.Equal(t, 1, s.Len())
	})

	t.Run("Remove at out-of-bounds index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.Remove(5)
		assert.Equal(t, 1, s.Len())
	})

	t.Run("Remove from empty sequence is no-op", func(t *testing.T) {
		var s Sequence
		s.Remove(0)
		assert.Equal(t, 0, s.Len())
	})

	t.Run("MoveUp swaps with previous", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))
		s.Add("gamma", testPipeline("gamma", nil, nil))

		s.MoveUp(2)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
		assert.Equal(t, "gamma", s.Entries[1].PipelineName)
		assert.Equal(t, "beta", s.Entries[2].PipelineName)
	})

	t.Run("MoveUp at index 1", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.MoveUp(1)
		assert.Equal(t, "beta", s.Entries[0].PipelineName)
		assert.Equal(t, "alpha", s.Entries[1].PipelineName)
	})

	t.Run("MoveUp at index 0 is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.MoveUp(0)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
		assert.Equal(t, "beta", s.Entries[1].PipelineName)
	})

	t.Run("MoveUp at negative index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.MoveUp(-1)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
	})

	t.Run("MoveUp at out-of-bounds index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.MoveUp(5)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
	})

	t.Run("MoveDown swaps with next", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))
		s.Add("gamma", testPipeline("gamma", nil, nil))

		s.MoveDown(0)
		assert.Equal(t, "beta", s.Entries[0].PipelineName)
		assert.Equal(t, "alpha", s.Entries[1].PipelineName)
		assert.Equal(t, "gamma", s.Entries[2].PipelineName)
	})

	t.Run("MoveDown at second-to-last index", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.MoveDown(0)
		assert.Equal(t, "beta", s.Entries[0].PipelineName)
		assert.Equal(t, "alpha", s.Entries[1].PipelineName)
	})

	t.Run("MoveDown at last index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))

		s.MoveDown(1)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
		assert.Equal(t, "beta", s.Entries[1].PipelineName)
	})

	t.Run("MoveDown at negative index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.MoveDown(-1)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
	})

	t.Run("MoveDown at out-of-bounds index is no-op", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))

		s.MoveDown(5)
		assert.Equal(t, "alpha", s.Entries[0].PipelineName)
	})

	t.Run("Len returns correct count", func(t *testing.T) {
		var s Sequence
		assert.Equal(t, 0, s.Len())

		s.Add("alpha", testPipeline("alpha", nil, nil))
		assert.Equal(t, 1, s.Len())

		s.Add("beta", testPipeline("beta", nil, nil))
		assert.Equal(t, 2, s.Len())

		s.Add("gamma", testPipeline("gamma", nil, nil))
		assert.Equal(t, 3, s.Len())
	})

	t.Run("IsEmpty on empty sequence", func(t *testing.T) {
		var s Sequence
		assert.True(t, s.IsEmpty())
	})

	t.Run("IsEmpty on non-empty sequence", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		assert.False(t, s.IsEmpty())
	})

	t.Run("IsSingle on empty sequence", func(t *testing.T) {
		var s Sequence
		assert.False(t, s.IsSingle())
	})

	t.Run("IsSingle on single-entry sequence", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		assert.True(t, s.IsSingle())
	})

	t.Run("IsSingle on multi-entry sequence", func(t *testing.T) {
		var s Sequence
		s.Add("alpha", testPipeline("alpha", nil, nil))
		s.Add("beta", testPipeline("beta", nil, nil))
		assert.False(t, s.IsSingle())
	})

	t.Run("IsReady with CompatibilityValid", func(t *testing.T) {
		r := CompatibilityResult{Status: CompatibilityValid}
		assert.True(t, r.IsReady())
	})

	t.Run("IsReady with CompatibilityWarning", func(t *testing.T) {
		r := CompatibilityResult{Status: CompatibilityWarning}
		assert.True(t, r.IsReady())
	})

	t.Run("IsReady with CompatibilityError", func(t *testing.T) {
		r := CompatibilityResult{Status: CompatibilityError}
		assert.False(t, r.IsReady())
	})
}
