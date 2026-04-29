package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAllShippedPipelinesLoad asserts that every YAML pipeline in
// .agents/pipelines and internal/defaults/pipelines parses through
// YAMLPipelineLoader (which also runs ValidatePipelineIOTypes). This is a
// regression guard for the typed I/O protocol (docs/adr/010).
//
// Any pipeline that fails to load — due to unknown type names, broken step
// references, or malformed input_ref — will fail this test.
func TestAllShippedPipelinesLoad(t *testing.T) {
	// Walk up from this package to the repo root (two levels: ../..).
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	dirs := []string{
		filepath.Join(repoRoot, ".agents", "pipelines"),
		filepath.Join(repoRoot, "internal", "defaults", "embedfs", "pipelines"),
	}

	loader := &YAMLPipelineLoader{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			name := e.Name()
			path := filepath.Join(dir, name)
			t.Run(filepath.Base(dir)+"/"+name, func(t *testing.T) {
				if _, err := loader.Load(path); err != nil {
					t.Fatalf("load %s: %v", path, err)
				}
			})
		}
	}
}
