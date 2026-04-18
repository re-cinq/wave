package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/fileutil"
)

// injectSubPipelineArtifacts copies named artifacts from the parent execution's
// ArtifactPaths into the child workspace's .agents/artifacts/ directory.
// Only artifacts listed in cfg.Inject are copied.
func injectSubPipelineArtifacts(cfg *SubPipelineConfig, parentCtx *PipelineContext, childWorkspaceDir string) error {
	if cfg == nil || len(cfg.Inject) == 0 || parentCtx == nil {
		return nil
	}

	destDir := filepath.Join(childWorkspaceDir, ".agents", "artifacts")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create child artifacts dir: %w", err)
	}

	for _, name := range cfg.Inject {
		srcPath := parentCtx.GetArtifactPath(name)
		if srcPath == "" {
			return fmt.Errorf("artifact %q not found in parent context for injection", name)
		}

		destPath := filepath.Join(destDir, name)
		if err := fileutil.CopyPath(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to inject artifact %q: %w", name, err)
		}
	}

	return nil
}

// extractSubPipelineArtifacts copies named artifacts from the child execution's
// ArtifactPaths back to the parent execution's artifact directory.
// Extracted artifacts are namespaced by the child pipeline name.
func extractSubPipelineArtifacts(cfg *SubPipelineConfig, childCtx *PipelineContext, childPipelineName string, parentCtx *PipelineContext, parentWorkspaceDir string) error {
	if cfg == nil || len(cfg.Extract) == 0 || childCtx == nil || parentCtx == nil {
		return nil
	}

	destDir := filepath.Join(parentWorkspaceDir, ".agents", "artifacts")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent artifacts dir: %w", err)
	}

	for _, name := range cfg.Extract {
		srcPath := childCtx.GetArtifactPath(name)
		if srcPath == "" {
			// Try looking in the child workspace artifacts dir
			srcPath = filepath.Join(parentWorkspaceDir, ".agents", "artifacts", name)
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				return fmt.Errorf("artifact %q not found in child context for extraction", name)
			}
		}

		// Namespace extracted artifacts: childPipelineName.artifactName
		namespacedName := childPipelineName + "." + name
		destPath := filepath.Join(destDir, namespacedName)

		if err := fileutil.CopyPath(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to extract artifact %q: %w", name, err)
		}

		// Register the extracted artifact in parent context
		parentCtx.SetArtifactPath(namespacedName, destPath)
	}

	return nil
}

// evaluateStopCondition evaluates a stop condition template expression against
// the child pipeline context. Returns true if the condition is met.
func evaluateStopCondition(condition string, childCtx *PipelineContext) bool {
	if condition == "" || childCtx == nil {
		return false
	}

	// Resolve template placeholders
	resolved := childCtx.ResolvePlaceholders(condition)

	// Check for simple key=value format: "context.key=value"
	if parts := strings.SplitN(resolved, "=", 2); len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		// Look up the key in custom variables
		actual := ""
		if strings.HasPrefix(key, "context.") {
			varKey := strings.TrimPrefix(key, "context.")
			childCtx.mu.Lock()
			actual = childCtx.CustomVariables[varKey]
			childCtx.mu.Unlock()
		}
		return actual == expected
	}

	// Treat resolved value as boolean
	resolved = strings.TrimSpace(resolved)
	return resolved == "true" || resolved == "done" || resolved == "yes"
}

// subPipelineTimeout wraps a context with a timeout from SubPipelineConfig.
// Returns the wrapped context, cancel function, and any error.
// If no timeout is configured, returns the original context.
func subPipelineTimeout(ctx context.Context, cfg *SubPipelineConfig) (context.Context, context.CancelFunc) {
	if cfg == nil {
		return ctx, func() {}
	}
	timeout := cfg.ParseTimeout()
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
