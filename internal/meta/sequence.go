package meta

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// PipelineRunner abstracts single-pipeline execution for testability.
type PipelineRunner interface {
	// RunPipeline executes a named pipeline and returns artifact paths on success.
	RunPipeline(ctx context.Context, pipelineName string, input string, artifactDir string) (artifactPaths map[string]string, err error)
}

// SequenceExecutor runs pipelines in order, managing cross-pipeline artifact handoff.
type SequenceExecutor struct {
	runner PipelineRunner
	wsRoot string
}

// SequenceResult holds the outcome of a multi-pipeline sequence.
type SequenceResult struct {
	Pipelines     []SequencePipelineResult `json:"pipelines"`
	TotalDuration time.Duration            `json:"total_duration_ms"`
	FailedAt      int                      `json:"failed_at"` // -1 if all succeeded
}

// SequencePipelineResult holds the result of one pipeline in a sequence.
type SequencePipelineResult struct {
	PipelineName  string            `json:"pipeline_name"`
	Status        string            `json:"status"` // "completed", "failed", "skipped"
	Duration      time.Duration     `json:"duration_ms"`
	Error         string            `json:"error,omitempty"`
	ArtifactPaths map[string]string `json:"artifact_paths,omitempty"`
}

// NewSequenceExecutor creates a new SequenceExecutor.
func NewSequenceExecutor(runner PipelineRunner, wsRoot string) *SequenceExecutor {
	return &SequenceExecutor{
		runner: runner,
		wsRoot: wsRoot,
	}
}

// Execute runs the given pipelines in sequence, passing artifacts between them.
func (s *SequenceExecutor) Execute(ctx context.Context, pipelineNames []string, input string) (*SequenceResult, error) {
	if len(pipelineNames) == 0 {
		return &SequenceResult{
			FailedAt: -1,
		}, nil
	}

	totalStart := time.Now()
	result := &SequenceResult{
		Pipelines: make([]SequencePipelineResult, len(pipelineNames)),
		FailedAt:  -1,
	}

	var prevArtifacts map[string]string

	for i, name := range pipelineNames {
		// Check for context cancellation before starting next pipeline.
		if err := ctx.Err(); err != nil {
			// Mark this and remaining pipelines as skipped.
			for j := i; j < len(pipelineNames); j++ {
				result.Pipelines[j] = SequencePipelineResult{
					PipelineName: pipelineNames[j],
					Status:       "skipped",
					Error:        "context cancelled",
				}
			}
			result.FailedAt = i
			result.TotalDuration = time.Since(totalStart)
			return result, nil
		}

		// Create artifact directory for this pipeline.
		artifactDir := filepath.Join(s.wsRoot, "sequence", name, ".wave", "artifacts")
		if err := os.MkdirAll(artifactDir, 0o755); err != nil {
			result.Pipelines[i] = SequencePipelineResult{
				PipelineName: name,
				Status:       "failed",
				Error:        fmt.Sprintf("failed to create artifact directory: %v", err),
			}
			// Mark remaining as skipped.
			for j := i + 1; j < len(pipelineNames); j++ {
				result.Pipelines[j] = SequencePipelineResult{
					PipelineName: pipelineNames[j],
					Status:       "skipped",
				}
			}
			result.FailedAt = i
			result.TotalDuration = time.Since(totalStart)
			return result, nil
		}

		// Copy artifacts from previous pipeline if not the first.
		if i > 0 && prevArtifacts != nil {
			if err := copyArtifacts(prevArtifacts, artifactDir); err != nil {
				result.Pipelines[i] = SequencePipelineResult{
					PipelineName: name,
					Status:       "failed",
					Error:        fmt.Sprintf("artifact copy failed: %v", err),
				}
				// Mark remaining as skipped.
				for j := i + 1; j < len(pipelineNames); j++ {
					result.Pipelines[j] = SequencePipelineResult{
						PipelineName: pipelineNames[j],
						Status:       "skipped",
					}
				}
				result.FailedAt = i
				result.TotalDuration = time.Since(totalStart)
				return result, nil
			}
		}

		// Run the pipeline.
		stepStart := time.Now()
		artifactPaths, err := s.runner.RunPipeline(ctx, name, input, artifactDir)
		stepDuration := time.Since(stepStart)

		if err != nil {
			result.Pipelines[i] = SequencePipelineResult{
				PipelineName: name,
				Status:       "failed",
				Duration:     stepDuration,
				Error:        err.Error(),
			}
			// Mark remaining as skipped.
			for j := i + 1; j < len(pipelineNames); j++ {
				result.Pipelines[j] = SequencePipelineResult{
					PipelineName: pipelineNames[j],
					Status:       "skipped",
				}
			}
			result.FailedAt = i
			result.TotalDuration = time.Since(totalStart)
			return result, nil
		}

		result.Pipelines[i] = SequencePipelineResult{
			PipelineName:  name,
			Status:        "completed",
			Duration:      stepDuration,
			ArtifactPaths: artifactPaths,
		}
		prevArtifacts = artifactPaths
	}

	result.TotalDuration = time.Since(totalStart)
	return result, nil
}

// copyArtifacts copies files from source artifact paths into the destination directory.
// Each entry in src maps an artifact name to a file path on disk.
func copyArtifacts(src map[string]string, destDir string) error {
	for name, srcPath := range src {
		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("source artifact %q not found at %s: %w", name, srcPath, err)
		}

		destPath := filepath.Join(destDir, filepath.Base(srcPath))
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy artifact %q: %w", name, err)
		}
	}
	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
