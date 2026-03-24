package defaults

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPersonaFilesParity asserts byte-identical parity between all files
// in internal/defaults/personas/ and .wave/personas/.
func TestPersonaFilesParity(t *testing.T) {
	assertDirParity(t, GetPersonas, "personas")
}

// TestPipelineFilesParity asserts byte-identical parity between all files
// in internal/defaults/pipelines/ and .wave/pipelines/.
func TestPipelineFilesParity(t *testing.T) {
	assertDirParity(t, GetPipelines, "pipelines")
}

// TestContractFilesParity asserts byte-identical parity between all files
// in internal/defaults/contracts/ and .wave/contracts/.
func TestContractFilesParity(t *testing.T) {
	assertDirParity(t, GetContracts, "contracts")
}

// assertDirParity is a shared helper that checks byte-identical parity between
// embedded defaults (internal/defaults/<kind>/) and working-tree files (.wave/<kind>/).
func assertDirParity(t *testing.T, getter func() (map[string]string, error), kind string) {
	t.Helper()

	embedded, err := getter()
	if err != nil {
		t.Fatalf("Get%s() error: %v", kind, err)
	}

	// Find the repo root by walking up from the test package directory.
	// internal/defaults/ -> repo root is ../../
	repoRoot := filepath.Join("..", "..")
	waveDir := filepath.Join(repoRoot, ".wave", kind)

	for name, embeddedContent := range embedded {
		wavePath := filepath.Join(waveDir, name)
		waveContent, err := os.ReadFile(wavePath)
		if err != nil {
			t.Errorf("file %q exists in internal/defaults/%s/ but not in .wave/%s/: %v", name, kind, kind, err)
			continue
		}
		if string(waveContent) != embeddedContent {
			t.Errorf("parity violation: internal/defaults/%s/%s differs from .wave/%s/%s", kind, name, kind, name)
		}
	}

	// Also check for files in .wave/<kind>/ that are NOT in embedded defaults.
	// Skip wave-* prefixed files — those are development-only pipelines/contracts
	// that are not shipped to users and don't need embedded defaults.
	entries, err := os.ReadDir(waveDir)
	if err != nil {
		t.Fatalf("failed to read .wave/%s/: %v", kind, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), "wave-") {
			continue // Development-only, not shipped
		}
		if _, ok := embedded[entry.Name()]; !ok {
			t.Errorf("file %q exists in .wave/%s/ but not in internal/defaults/%s/", entry.Name(), kind, kind)
		}
	}
}
