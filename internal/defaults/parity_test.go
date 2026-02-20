package defaults

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// findProjectRoot walks up from the current directory to find go.mod.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// TestPersonaParityEmbedVsWorkspace asserts byte-identical parity between
// internal/defaults/personas/ (embedded) and .wave/personas/ (workspace copy).
func TestPersonaParityEmbedVsWorkspace(t *testing.T) {
	root := findProjectRoot(t)
	wavePersonasDir := filepath.Join(root, ".wave", "personas")

	embedded, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	for filename, embeddedContent := range embedded {
		workspacePath := filepath.Join(wavePersonasDir, filename)
		workspaceContent, err := os.ReadFile(workspacePath)
		if err != nil {
			t.Errorf("file %s exists in internal/defaults/personas/ but not in .wave/personas/: %v", filename, err)
			continue
		}
		if string(workspaceContent) != embeddedContent {
			t.Errorf("parity violation: %s differs between internal/defaults/personas/ and .wave/personas/", filename)
		}
	}

	// Also check for files in .wave/personas/ that aren't in the embedded set
	entries, err := os.ReadDir(wavePersonasDir)
	if err != nil {
		t.Fatalf("failed to read .wave/personas/: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if _, ok := embedded[entry.Name()]; !ok {
			t.Errorf("file %s exists in .wave/personas/ but not in internal/defaults/personas/", entry.Name())
		}
	}
}

// TestPersonasLanguageAgnostic verifies no programming language-specific
// references exist in any persona file or base-protocol.md.
func TestPersonasLanguageAgnostic(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	// Match whole words only to avoid false positives (e.g., "Rust" in "trustworthy")
	langPatterns := regexp.MustCompile(`(?i)\b(Go|Golang|Python|TypeScript|JavaScript|Java|Rust|Ruby|Swift|Kotlin|C\+\+|C#)\b`)

	for filename, content := range personas {
		matches := langPatterns.FindAllString(content, -1)
		if len(matches) > 0 {
			t.Errorf("persona %s contains language-specific references: %v", filename, matches)
		}
	}
}

// TestPersonaStructuralElements verifies each persona file contains the three
// mandatory structural elements: H1 identity heading, responsibilities section,
// and output section.
func TestPersonaStructuralElements(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	for filename, content := range personas {
		// Skip base-protocol.md — it's not a persona
		if filename == "base-protocol.md" {
			continue
		}

		if !strings.HasPrefix(content, "# ") {
			t.Errorf("persona %s missing H1 identity heading", filename)
		}

		hasResponsibilities := strings.Contains(content, "## Responsibilities") ||
			strings.Contains(content, "## Mandatory Rules") ||
			strings.Contains(content, "## Instructions")
		if !hasResponsibilities {
			t.Errorf("persona %s missing responsibilities section", filename)
		}

		hasOutput := strings.Contains(content, "## Output Format") ||
			strings.Contains(content, "## Output")
		if !hasOutput {
			t.Errorf("persona %s missing output format section", filename)
		}
	}
}

// TestBaseProtocolExists verifies that base-protocol.md is included in the
// embedded persona files.
func TestBaseProtocolExists(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	content, ok := personas["base-protocol.md"]
	if !ok {
		t.Fatal("base-protocol.md not found in embedded personas")
	}

	if !strings.Contains(content, "# Wave Agent Protocol") {
		t.Error("base-protocol.md missing expected heading")
	}
}
