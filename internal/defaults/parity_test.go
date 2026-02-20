package defaults

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPersonaFilesParity asserts byte-identical parity between all files
// in internal/defaults/personas/ and .wave/personas/.
func TestPersonaFilesParity(t *testing.T) {
	embedded, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	// Find the repo root by walking up from the test package directory.
	// internal/defaults/ -> repo root is ../../
	repoRoot := filepath.Join("..", "..")
	waveDir := filepath.Join(repoRoot, ".wave", "personas")

	for name, embeddedContent := range embedded {
		wavePath := filepath.Join(waveDir, name)
		waveContent, err := os.ReadFile(wavePath)
		if err != nil {
			t.Errorf("file %q exists in internal/defaults/personas/ but not in .wave/personas/: %v", name, err)
			continue
		}
		if string(waveContent) != embeddedContent {
			t.Errorf("parity violation: internal/defaults/personas/%s differs from .wave/personas/%s", name, name)
		}
	}

	// Also check for files in .wave/personas/ that are NOT in embedded defaults
	entries, err := os.ReadDir(waveDir)
	if err != nil {
		t.Fatalf("failed to read .wave/personas/: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := embedded[entry.Name()]; !ok {
			t.Errorf("file %q exists in .wave/personas/ but not in internal/defaults/personas/", entry.Name())
		}
	}
}
