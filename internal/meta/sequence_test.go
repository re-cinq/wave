package meta

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPipelineRunner implements PipelineRunner for testing.
type mockPipelineRunner struct {
	mu    sync.Mutex
	calls []mockRunnerCall

	// results maps pipeline name to pre-configured result.
	results map[string]mockRunnerResult
}

type mockRunnerCall struct {
	PipelineName string
	Input        string
	ArtifactDir  string
}

type mockRunnerResult struct {
	ArtifactPaths map[string]string
	Err           error
	// CreateFiles causes the runner to create actual files at the artifact paths.
	CreateFiles bool
	FileContent map[string]string // artifact name -> file content
}

func newMockRunner() *mockPipelineRunner {
	return &mockPipelineRunner{
		results: make(map[string]mockRunnerResult),
	}
}

func (m *mockPipelineRunner) RunPipeline(ctx context.Context, pipelineName string, input string, artifactDir string) (map[string]string, error) {
	m.mu.Lock()
	m.calls = append(m.calls, mockRunnerCall{
		PipelineName: pipelineName,
		Input:        input,
		ArtifactDir:  artifactDir,
	})
	m.mu.Unlock()

	result, ok := m.results[pipelineName]
	if !ok {
		return nil, fmt.Errorf("no mock result configured for pipeline %q", pipelineName)
	}

	if result.Err != nil {
		return nil, result.Err
	}

	// If CreateFiles is set, create actual files on disk.
	if result.CreateFiles {
		for name, path := range result.ArtifactPaths {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("mock: failed to create dir for %s: %w", name, err)
			}
			content := "mock content for " + name
			if c, ok := result.FileContent[name]; ok {
				content = c
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return nil, fmt.Errorf("mock: failed to write %s: %w", name, err)
			}
		}
	}

	return result.ArtifactPaths, nil
}

func (m *mockPipelineRunner) getCalls() []mockRunnerCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]mockRunnerCall, len(m.calls))
	copy(cp, m.calls)
	return cp
}

func TestSequenceExecutor_SuccessfulTwoPipelineHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	// First pipeline produces an artifact file on disk.
	firstOutputDir := filepath.Join(tmpDir, "first-output")
	require.NoError(t, os.MkdirAll(firstOutputDir, 0o755))
	specFile := filepath.Join(firstOutputDir, "spec.json")
	require.NoError(t, os.WriteFile(specFile, []byte(`{"feature":"auth"}`), 0o644))

	runner.results["research"] = mockRunnerResult{
		ArtifactPaths: map[string]string{
			"spec": specFile,
		},
	}
	runner.results["implement"] = mockRunnerResult{
		ArtifactPaths: map[string]string{
			"code": filepath.Join(tmpDir, "impl-output", "code.patch"),
		},
		CreateFiles: true,
		FileContent: map[string]string{
			"code": "diff --git a/main.go b/main.go",
		},
	}

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{"research", "implement"}, "implement auth")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, -1, result.FailedAt)
	require.Len(t, result.Pipelines, 2)

	// First pipeline completed.
	assert.Equal(t, "research", result.Pipelines[0].PipelineName)
	assert.Equal(t, "completed", result.Pipelines[0].Status)
	assert.Empty(t, result.Pipelines[0].Error)
	assert.Contains(t, result.Pipelines[0].ArtifactPaths, "spec")

	// Second pipeline completed.
	assert.Equal(t, "implement", result.Pipelines[1].PipelineName)
	assert.Equal(t, "completed", result.Pipelines[1].Status)
	assert.Empty(t, result.Pipelines[1].Error)

	// Verify artifact was copied to second pipeline's artifact dir.
	secondArtifactDir := filepath.Join(tmpDir, "sequence", "implement", ".wave", "artifacts")
	copiedSpec := filepath.Join(secondArtifactDir, "spec.json")
	content, err := os.ReadFile(copiedSpec)
	require.NoError(t, err)
	assert.Equal(t, `{"feature":"auth"}`, string(content))

	// Verify both pipelines were called in order.
	calls := runner.getCalls()
	require.Len(t, calls, 2)
	assert.Equal(t, "research", calls[0].PipelineName)
	assert.Equal(t, "implement", calls[1].PipelineName)
	assert.Equal(t, "implement auth", calls[0].Input)
	assert.Equal(t, "implement auth", calls[1].Input)

	// Verify total duration is positive.
	assert.Greater(t, result.TotalDuration.Nanoseconds(), int64(0))
}

func TestSequenceExecutor_FirstPipelineFailsHaltsSequence(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	runner.results["research"] = mockRunnerResult{
		Err: fmt.Errorf("research pipeline failed: missing data"),
	}
	// Second pipeline should never be called.
	runner.results["implement"] = mockRunnerResult{
		ArtifactPaths: map[string]string{},
	}

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{"research", "implement"}, "do stuff")

	require.NoError(t, err)
	require.NotNil(t, result)

	// Sequence failed at index 0.
	assert.Equal(t, 0, result.FailedAt)
	require.Len(t, result.Pipelines, 2)

	// First pipeline failed.
	assert.Equal(t, "research", result.Pipelines[0].PipelineName)
	assert.Equal(t, "failed", result.Pipelines[0].Status)
	assert.Contains(t, result.Pipelines[0].Error, "research pipeline failed")

	// Second pipeline skipped.
	assert.Equal(t, "implement", result.Pipelines[1].PipelineName)
	assert.Equal(t, "skipped", result.Pipelines[1].Status)

	// Only one pipeline was actually invoked.
	calls := runner.getCalls()
	assert.Len(t, calls, 1)
	assert.Equal(t, "research", calls[0].PipelineName)
}

func TestSequenceExecutor_ArtifactCopyMissingSourceDetected(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	// First pipeline claims an artifact path that does NOT exist on disk.
	nonexistentFile := filepath.Join(tmpDir, "does-not-exist", "phantom.json")
	runner.results["research"] = mockRunnerResult{
		ArtifactPaths: map[string]string{
			"spec": nonexistentFile,
		},
	}
	runner.results["implement"] = mockRunnerResult{
		ArtifactPaths: map[string]string{},
	}

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{"research", "implement"}, "go")

	require.NoError(t, err)
	require.NotNil(t, result)

	// The sequence should fail at the second pipeline due to missing artifact.
	assert.Equal(t, 1, result.FailedAt)
	require.Len(t, result.Pipelines, 2)

	assert.Equal(t, "completed", result.Pipelines[0].Status)
	assert.Equal(t, "failed", result.Pipelines[1].Status)
	assert.Contains(t, result.Pipelines[1].Error, "artifact copy failed")
	assert.Contains(t, result.Pipelines[1].Error, "not found")

	// Only the first pipeline was actually run; second failed before running.
	calls := runner.getCalls()
	assert.Len(t, calls, 1)
}

func TestSequenceExecutor_SinglePipeline(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	outputFile := filepath.Join(tmpDir, "output", "result.json")
	runner.results["build"] = mockRunnerResult{
		ArtifactPaths: map[string]string{
			"result": outputFile,
		},
		CreateFiles: true,
		FileContent: map[string]string{
			"result": `{"status":"ok"}`,
		},
	}

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{"build"}, "build it")

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, -1, result.FailedAt)
	require.Len(t, result.Pipelines, 1)
	assert.Equal(t, "build", result.Pipelines[0].PipelineName)
	assert.Equal(t, "completed", result.Pipelines[0].Status)

	calls := runner.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "build", calls[0].PipelineName)
	assert.Equal(t, "build it", calls[0].Input)
}

func TestSequenceExecutor_EmptyPipelineList(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{}, "nothing")

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, -1, result.FailedAt)
	assert.Empty(t, result.Pipelines)

	// No pipelines should have been called.
	calls := runner.getCalls()
	assert.Empty(t, calls)
}

func TestNewSequenceExecutor(t *testing.T) {
	runner := newMockRunner()
	executor := NewSequenceExecutor(runner, "/tmp/test")

	assert.NotNil(t, executor)
	assert.Equal(t, "/tmp/test", executor.wsRoot)
	assert.Equal(t, runner, executor.runner)
}

func TestCopyArtifacts_ValidFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files.
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.json"), []byte(`{"a":1}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.md"), []byte("# B"), 0o644))

	destDir := filepath.Join(tmpDir, "dest")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	src := map[string]string{
		"alpha": filepath.Join(srcDir, "a.json"),
		"beta":  filepath.Join(srcDir, "b.md"),
	}

	err := copyArtifacts(src, destDir)
	require.NoError(t, err)

	// Verify files were copied.
	contentA, err := os.ReadFile(filepath.Join(destDir, "a.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, string(contentA))

	contentB, err := os.ReadFile(filepath.Join(destDir, "b.md"))
	require.NoError(t, err)
	assert.Equal(t, "# B", string(contentB))
}

func TestCopyArtifacts_MissingSourceReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	src := map[string]string{
		"missing": filepath.Join(tmpDir, "nonexistent", "file.json"),
	}

	err := copyArtifacts(src, destDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "missing")
}

func TestCopyArtifacts_EmptyMap(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := copyArtifacts(map[string]string{}, destDir)
	require.NoError(t, err)
}

func TestSequenceExecutor_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	runner.results["first"] = mockRunnerResult{
		ArtifactPaths: map[string]string{},
	}
	runner.results["second"] = mockRunnerResult{
		ArtifactPaths: map[string]string{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(ctx, []string{"first", "second"}, "go")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.FailedAt)

	// Both pipelines should be skipped.
	require.Len(t, result.Pipelines, 2)
	assert.Equal(t, "skipped", result.Pipelines[0].Status)
	assert.Equal(t, "skipped", result.Pipelines[1].Status)

	// No pipelines should have been called.
	calls := runner.getCalls()
	assert.Empty(t, calls)
}

func TestSequenceExecutor_SecondPipelineFailsThirdSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	runner := newMockRunner()

	outputFile := filepath.Join(tmpDir, "first-output", "data.json")
	runner.results["first"] = mockRunnerResult{
		ArtifactPaths: map[string]string{
			"data": outputFile,
		},
		CreateFiles: true,
		FileContent: map[string]string{
			"data": `{"step":1}`,
		},
	}
	runner.results["second"] = mockRunnerResult{
		Err: fmt.Errorf("second pipeline exploded"),
	}
	runner.results["third"] = mockRunnerResult{
		ArtifactPaths: map[string]string{},
	}

	executor := NewSequenceExecutor(runner, tmpDir)
	result, err := executor.Execute(context.Background(), []string{"first", "second", "third"}, "go")

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.FailedAt)
	require.Len(t, result.Pipelines, 3)

	assert.Equal(t, "completed", result.Pipelines[0].Status)
	assert.Equal(t, "failed", result.Pipelines[1].Status)
	assert.Contains(t, result.Pipelines[1].Error, "second pipeline exploded")
	assert.Equal(t, "skipped", result.Pipelines[2].Status)

	// Only first and second were called.
	calls := runner.getCalls()
	assert.Len(t, calls, 2)
}
