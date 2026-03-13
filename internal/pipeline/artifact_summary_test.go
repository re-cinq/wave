package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeArtifact_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte(`{"name":"test","score":85,"passed":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(summary, "name") {
		t.Error("summary missing 'name' key")
	}
	if !strings.Contains(summary, "score") {
		t.Error("summary missing 'score' key")
	}
}

func TestSummarizeArtifact_LargeJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.json")

	// Create a large JSON object
	content := `{"key1":"` + strings.Repeat("a", 500) + `","key2":"` + strings.Repeat("b", 500) + `"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary) > 200 {
		t.Errorf("summary exceeds maxBytes: got %d, want <= 200", len(summary))
	}
}

func TestSummarizeArtifact_Markdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	content := "# Title\n\nFirst paragraph here.\n\n## Section 2\n\nMore content.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(summary, "# Title") {
		t.Error("summary missing heading")
	}
	if !strings.Contains(summary, "First paragraph") {
		t.Error("summary missing first paragraph")
	}
}

func TestSummarizeArtifact_PlainText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	content := "line 1\nline 2\nline 3\nline 4\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(summary, "line 1") {
		t.Error("summary missing first line")
	}
}

func TestSummarizeArtifact_Binary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x01, 0x02}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(summary, "binary file") {
		t.Errorf("expected binary detection, got: %s", summary)
	}
	if !strings.Contains(summary, "8 bytes") {
		t.Errorf("expected size in summary, got: %s", summary)
	}
}

func TestSummarizeArtifact_MissingFile(t *testing.T) {
	_, err := SummarizeArtifact("/nonexistent/file.json", 4096)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSummarizeArtifact_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary != "[empty file]" {
		t.Errorf("expected '[empty file]', got: %s", summary)
	}
}

func TestSummarizeArtifact_Truncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.txt")
	content := strings.Repeat("Hello world. ", 1000)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := SummarizeArtifact(path, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary) > 200 {
		t.Errorf("summary exceeds maxBytes: got %d, want <= 200", len(summary))
	}
	if !strings.Contains(summary, "truncated") {
		t.Error("expected truncation note")
	}
}
