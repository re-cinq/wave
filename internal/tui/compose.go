package tui

import (
	"fmt"

	"github.com/recinq/wave/internal/pipeline"
)

// MatchStatus indicates the result of matching an artifact.
type MatchStatus int

const (
	MatchCompatible MatchStatus = iota // Output name matches inject artifact name
	MatchMissing                       // Inject artifact expected but no matching output
	MatchUnmatched                     // Output produced but not consumed by next pipeline
)

// CompatibilityStatus indicates the overall sequence readiness.
type CompatibilityStatus int

const (
	CompatibilityValid   CompatibilityStatus = iota // All flows compatible
	CompatibilityWarning                            // Optional mismatches only
	CompatibilityError                              // Required inputs missing
)

// Sequence represents an ordered list of pipelines to execute in series.
type Sequence struct {
	Entries []SequenceEntry
}

// SequenceEntry is a single pipeline in a sequence.
type SequenceEntry struct {
	PipelineName string
	Pipeline     *pipeline.Pipeline
}

// Add appends a pipeline to the sequence.
func (s *Sequence) Add(name string, p *pipeline.Pipeline) {
	s.Entries = append(s.Entries, SequenceEntry{PipelineName: name, Pipeline: p})
}

// Remove removes the entry at index, adjusting the slice.
func (s *Sequence) Remove(index int) {
	if index < 0 || index >= len(s.Entries) {
		return
	}
	s.Entries = append(s.Entries[:index], s.Entries[index+1:]...)
}

// MoveUp swaps the entry at index with the one above it.
func (s *Sequence) MoveUp(index int) {
	if index <= 0 || index >= len(s.Entries) {
		return
	}
	s.Entries[index], s.Entries[index-1] = s.Entries[index-1], s.Entries[index]
}

// MoveDown swaps the entry at index with the one below it.
func (s *Sequence) MoveDown(index int) {
	if index < 0 || index >= len(s.Entries)-1 {
		return
	}
	s.Entries[index], s.Entries[index+1] = s.Entries[index+1], s.Entries[index]
}

// Len returns the number of entries.
func (s *Sequence) Len() int {
	return len(s.Entries)
}

// IsEmpty returns true when there are no entries.
func (s *Sequence) IsEmpty() bool {
	return len(s.Entries) == 0
}

// IsSingle returns true when there is exactly one entry.
func (s *Sequence) IsSingle() bool {
	return len(s.Entries) == 1
}

// ArtifactFlow describes the artifact compatibility at one boundary
// between pipeline N and pipeline N+1 in a sequence.
type ArtifactFlow struct {
	SourcePipeline string
	TargetPipeline string
	Outputs        []pipeline.ArtifactDef
	Inputs         []pipeline.ArtifactRef
	Matches        []FlowMatch
}

// FlowMatch describes the match status of a single artifact flow.
type FlowMatch struct {
	OutputName string
	InputName  string
	InputAs    string
	Status     MatchStatus
	Optional   bool
}

// CompatibilityResult is the aggregated result of validating all
// artifact flows across a sequence.
type CompatibilityResult struct {
	Flows       []ArtifactFlow
	Status      CompatibilityStatus
	Diagnostics []string
}

// IsReady returns true if the sequence can be started without hard errors.
func (r CompatibilityResult) IsReady() bool {
	return r.Status == CompatibilityValid || r.Status == CompatibilityWarning
}

// ValidateSequence validates artifact compatibility across all adjacent pipeline
// boundaries in a sequence. It returns a CompatibilityResult with per-boundary
// flows and an overall status.
func ValidateSequence(seq Sequence) CompatibilityResult {
	result := CompatibilityResult{Status: CompatibilityValid}

	for i := 0; i < len(seq.Entries)-1; i++ {
		source := seq.Entries[i]
		target := seq.Entries[i+1]

		flow := ArtifactFlow{
			SourcePipeline: source.PipelineName,
			TargetPipeline: target.PipelineName,
		}

		// Get last step outputs from source pipeline
		var outputs []pipeline.ArtifactDef
		if source.Pipeline != nil && len(source.Pipeline.Steps) > 0 {
			lastStep := source.Pipeline.Steps[len(source.Pipeline.Steps)-1]
			outputs = lastStep.OutputArtifacts
		}
		flow.Outputs = outputs

		// Get first step inputs from target pipeline
		var inputs []pipeline.ArtifactRef
		if target.Pipeline != nil && len(target.Pipeline.Steps) > 0 {
			firstStep := target.Pipeline.Steps[0]
			inputs = firstStep.Memory.InjectArtifacts
		}
		flow.Inputs = inputs

		// Track which outputs are consumed
		outputConsumed := make(map[string]bool)

		// Match each input to an output
		for _, input := range inputs {
			found := false
			for _, output := range outputs {
				if output.Name == input.Artifact {
					flow.Matches = append(flow.Matches, FlowMatch{
						OutputName: output.Name,
						InputName:  input.Artifact,
						InputAs:    input.As,
						Status:     MatchCompatible,
						Optional:   input.Optional,
					})
					outputConsumed[output.Name] = true
					found = true
					break
				}
			}
			if !found {
				flow.Matches = append(flow.Matches, FlowMatch{
					InputName: input.Artifact,
					InputAs:   input.As,
					Status:    MatchMissing,
					Optional:  input.Optional,
				})
				if input.Optional {
					if result.Status < CompatibilityWarning {
						result.Status = CompatibilityWarning
					}
					result.Diagnostics = append(result.Diagnostics,
						fmt.Sprintf("%s → %s: optional input '%s' has no matching output",
							source.PipelineName, target.PipelineName, input.Artifact))
				} else {
					result.Status = CompatibilityError
					result.Diagnostics = append(result.Diagnostics,
						fmt.Sprintf("%s → %s: missing required input '%s'",
							source.PipelineName, target.PipelineName, input.Artifact))
				}
			}
		}

		// Mark unmatched outputs
		for _, output := range outputs {
			if !outputConsumed[output.Name] {
				flow.Matches = append(flow.Matches, FlowMatch{
					OutputName: output.Name,
					Status:     MatchUnmatched,
				})
			}
		}

		result.Flows = append(result.Flows, flow)
	}

	return result
}

// ValidateSequenceWithStages validates artifact compatibility across stage
// boundaries in a parallel sequence. Unlike ValidateSequence which checks every
// adjacent pair, this only produces flows at stage boundaries: aggregating all
// outputs from all pipelines in stage N and matching against all inputs from all
// pipelines in stage N+1. Pipelines within the same stage are independent and
// produce no inter-flow artifacts.
func ValidateSequenceWithStages(seq Sequence, stages [][]int) CompatibilityResult {
	result := CompatibilityResult{Status: CompatibilityValid}

	for i := 0; i < len(stages)-1; i++ {
		sourceStage := stages[i]
		targetStage := stages[i+1]

		sourceName := fmt.Sprintf("Stage %d", i+1)
		targetName := fmt.Sprintf("Stage %d", i+2)

		flow := ArtifactFlow{
			SourcePipeline: sourceName,
			TargetPipeline: targetName,
		}

		// Aggregate outputs from all pipelines in source stage
		var outputs []pipeline.ArtifactDef
		for _, idx := range sourceStage {
			if idx < len(seq.Entries) {
				entry := seq.Entries[idx]
				if entry.Pipeline != nil && len(entry.Pipeline.Steps) > 0 {
					lastStep := entry.Pipeline.Steps[len(entry.Pipeline.Steps)-1]
					outputs = append(outputs, lastStep.OutputArtifacts...)
				}
			}
		}
		flow.Outputs = outputs

		// Aggregate inputs from all pipelines in target stage
		var inputs []pipeline.ArtifactRef
		for _, idx := range targetStage {
			if idx < len(seq.Entries) {
				entry := seq.Entries[idx]
				if entry.Pipeline != nil && len(entry.Pipeline.Steps) > 0 {
					firstStep := entry.Pipeline.Steps[0]
					inputs = append(inputs, firstStep.Memory.InjectArtifacts...)
				}
			}
		}
		flow.Inputs = inputs

		// Track which outputs are consumed
		outputConsumed := make(map[string]bool)

		// Match each input to an output
		for _, input := range inputs {
			found := false
			for _, output := range outputs {
				if output.Name == input.Artifact {
					flow.Matches = append(flow.Matches, FlowMatch{
						OutputName: output.Name,
						InputName:  input.Artifact,
						InputAs:    input.As,
						Status:     MatchCompatible,
						Optional:   input.Optional,
					})
					outputConsumed[output.Name] = true
					found = true
					break
				}
			}
			if !found {
				flow.Matches = append(flow.Matches, FlowMatch{
					InputName: input.Artifact,
					InputAs:   input.As,
					Status:    MatchMissing,
					Optional:  input.Optional,
				})
				if input.Optional {
					if result.Status < CompatibilityWarning {
						result.Status = CompatibilityWarning
					}
					result.Diagnostics = append(result.Diagnostics,
						fmt.Sprintf("%s → %s: optional input '%s' has no matching output",
							sourceName, targetName, input.Artifact))
				} else {
					result.Status = CompatibilityError
					result.Diagnostics = append(result.Diagnostics,
						fmt.Sprintf("%s → %s: missing required input '%s'",
							sourceName, targetName, input.Artifact))
				}
			}
		}

		// Mark unmatched outputs
		for _, output := range outputs {
			if !outputConsumed[output.Name] {
				flow.Matches = append(flow.Matches, FlowMatch{
					OutputName: output.Name,
					Status:     MatchUnmatched,
				})
			}
		}

		result.Flows = append(result.Flows, flow)
	}

	return result
}
