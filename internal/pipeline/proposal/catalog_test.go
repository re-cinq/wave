package proposal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePipelineYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0644))
}

func TestNewCatalogDiscovery(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, "test-gen", `
kind: WavePipeline
metadata:
  name: test-gen
  description: Generate tests
  category: testing
input:
  source: cli
steps:
  - id: generate
    persona: craftsman
`)
	writePipelineYAML(t, dir, "doc-fix", `
kind: WavePipeline
metadata:
  name: doc-fix
  description: Fix documentation
input:
  source: cli
steps:
  - id: fix
    persona: craftsman
`)

	catalog, err := NewCatalog(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, catalog.Len())

	entries := catalog.Entries()
	// Should be sorted by name
	assert.Equal(t, "doc-fix", entries[0].Name)
	assert.Equal(t, "test-gen", entries[1].Name)
	assert.Equal(t, "testing", entries[1].Category)
}

func TestNewCatalogDedup(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	writePipelineYAML(t, dir1, "refactor", `
kind: WavePipeline
metadata:
  name: refactor
  description: First version
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
`)
	writePipelineYAML(t, dir2, "refactor", `
kind: WavePipeline
metadata:
  name: refactor
  description: Second version
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
`)

	catalog, err := NewCatalog(dir1, dir2)
	require.NoError(t, err)
	assert.Equal(t, 1, catalog.Len())
	// First directory wins
	assert.Equal(t, "First version", catalog.Entries()[0].Description)
}

func TestNewCatalogEmptyDir(t *testing.T) {
	dir := t.TempDir()
	catalog, err := NewCatalog(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, catalog.Len())
}

func TestNewCatalogNonexistentDir(t *testing.T) {
	catalog, err := NewCatalog("/nonexistent/path")
	require.NoError(t, err) // Should not error — nonexistent dirs are skipped
	assert.Equal(t, 0, catalog.Len())
}

func TestNewCatalogSkipsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, "good", `
kind: WavePipeline
metadata:
  name: good
  description: Valid pipeline
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
`)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "bad.yaml"),
		[]byte("{{invalid yaml content"),
		0644,
	))

	catalog, err := NewCatalog(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, catalog.Len())
	assert.Equal(t, "good", catalog.Entries()[0].Name)
}

func TestNewCatalogSkipsDisabled(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, "active", `
kind: WavePipeline
metadata:
  name: active
  description: Active pipeline
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
`)
	writePipelineYAML(t, dir, "disabled", `
kind: WavePipeline
metadata:
  name: disabled
  description: Disabled pipeline
  disabled: true
input:
  source: cli
steps:
  - id: step1
    persona: craftsman
`)

	catalog, err := NewCatalog(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, catalog.Len())
	assert.Equal(t, "active", catalog.Entries()[0].Name)
}

func TestNewCatalogSkipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, "valid", `
kind: WavePipeline
metadata:
  name: valid
input:
  source: cli
steps:
  - id: s
    persona: c
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0644))

	catalog, err := NewCatalog(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, catalog.Len())
}

func TestCatalogEntriesReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, "test", `
kind: WavePipeline
metadata:
  name: test
input:
  source: cli
steps:
  - id: s
    persona: c
`)

	catalog, err := NewCatalog(dir)
	require.NoError(t, err)

	entries1 := catalog.Entries()
	entries2 := catalog.Entries()
	entries1[0].Name = "modified"
	assert.Equal(t, "test", entries2[0].Name, "Entries should return a copy")
}
