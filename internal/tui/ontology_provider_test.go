package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ontologyMockStore overrides GetOntologyStatsAll for provider tests.
// ---------------------------------------------------------------------------

type ontologyMockStore struct {
	baseStateStore
	allStats    []state.OntologyStats
	allStatsErr error
}

func (m *ontologyMockStore) GetOntologyStatsAll() ([]state.OntologyStats, error) {
	return m.allStats, m.allStatsErr
}

// chdirTempOntology changes to a fresh temp directory for the duration of the
// test and restores the original directory via t.Cleanup.
func chdirTempOntology(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return tmpDir
}

// ---------------------------------------------------------------------------
// NewDefaultOntologyDataProvider
// ---------------------------------------------------------------------------

// TestNewDefaultOntologyDataProvider_ReturnsProvider verifies the constructor
// wires the provided arguments and returns a non-nil provider.
func TestNewDefaultOntologyDataProvider_ReturnsProvider(t *testing.T) {
	m := &manifest.Manifest{}
	store := &ontologyMockStore{}
	p := NewDefaultOntologyDataProvider(m, ".agents/skills", store)
	require.NotNil(t, p)
}

// ---------------------------------------------------------------------------
// FetchOntology — nil/empty manifest cases
// ---------------------------------------------------------------------------

// TestFetchOntology_NilManifest_ReturnsEmptyOverview verifies that when the
// manifest is nil FetchOntology returns an empty overview without error.
func TestFetchOntology_NilManifest_ReturnsEmptyOverview(t *testing.T) {
	p := NewDefaultOntologyDataProvider(nil, ".agents/skills", nil)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.NotNil(t, overview)
	assert.Empty(t, overview.Telos)
	assert.Empty(t, overview.Contexts)
	assert.False(t, overview.Stale)
}

// TestFetchOntology_NilOntology_ReturnsEmptyOverview verifies that when
// manifest.Ontology is nil the result is an empty overview.
func TestFetchOntology_NilOntology_ReturnsEmptyOverview(t *testing.T) {
	p := NewDefaultOntologyDataProvider(&manifest.Manifest{Ontology: nil}, ".agents/skills", nil)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.NotNil(t, overview)
	assert.Empty(t, overview.Telos)
	assert.Empty(t, overview.Contexts)
}

// ---------------------------------------------------------------------------
// FetchOntology — staleness sentinel
// ---------------------------------------------------------------------------

// TestFetchOntology_StaleFile_SetsStale verifies that when
// .agents/.ontology-stale exists the overview has Stale=true.
func TestFetchOntology_StaleFile_SetsStale(t *testing.T) {
	tmpDir := chdirTempOntology(t)

	waveDir := filepath.Join(tmpDir, ".agents")
	require.NoError(t, os.MkdirAll(waveDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(waveDir, ".ontology-stale"), []byte(""), 0o644))

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{Telos: "test"}},
		filepath.Join(tmpDir, ".agents", "skills"),
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	assert.True(t, overview.Stale, "Stale should be true when sentinel file exists")
}

// TestFetchOntology_NoStaleFile_StaleIsFalse verifies that without the sentinel
// Stale remains false.
func TestFetchOntology_NoStaleFile_StaleIsFalse(t *testing.T) {
	chdirTempOntology(t)

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{Telos: "test"}},
		t.TempDir(),
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	assert.False(t, overview.Stale)
}

// ---------------------------------------------------------------------------
// FetchOntology — store interaction
// ---------------------------------------------------------------------------

// TestFetchOntology_NilStore_StatsMapEmpty verifies that when the store is nil
// contexts have HasLineage=false.
func TestFetchOntology_NilStore_StatsMapEmpty(t *testing.T) {
	chdirTempOntology(t)

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Telos: "test",
			Contexts: []manifest.OntologyContext{
				{Name: "billing"},
			},
		}},
		t.TempDir(),
		nil, // nil store
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	assert.False(t, overview.Contexts[0].HasLineage)
}

// TestFetchOntology_StoreError_StatsMapEmpty verifies that when GetOntologyStatsAll
// returns an error the contexts have HasLineage=false (stats silently ignored).
func TestFetchOntology_StoreError_StatsMapEmpty(t *testing.T) {
	chdirTempOntology(t)

	store := &ontologyMockStore{allStatsErr: errors.New("db error")}
	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Telos: "test",
			Contexts: []manifest.OntologyContext{
				{Name: "billing"},
			},
		}},
		t.TempDir(),
		store,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	assert.False(t, overview.Contexts[0].HasLineage, "store error should leave HasLineage=false")
}

// ---------------------------------------------------------------------------
// FetchOntology — skill file detection
// ---------------------------------------------------------------------------

// TestFetchOntology_SkillFileExists_SetsHasSkill verifies that when a
// SKILL.md file exists for a context HasSkill=true and SkillPath is set.
func TestFetchOntology_SkillFileExists_SetsHasSkill(t *testing.T) {
	chdirTempOntology(t)

	skillsDir := t.TempDir()
	ctxSkillDir := filepath.Join(skillsDir, "wave-ctx-billing")
	require.NoError(t, os.MkdirAll(ctxSkillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ctxSkillDir, "SKILL.md"), []byte("# Billing"), 0o644))

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "billing"},
			},
		}},
		skillsDir,
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	ctx := overview.Contexts[0]
	assert.True(t, ctx.HasSkill, "HasSkill should be true when SKILL.md exists")
	assert.NotEmpty(t, ctx.SkillPath)
}

// TestFetchOntology_NoSkillFile_HasSkillFalse verifies that when no SKILL.md
// exists for a context HasSkill remains false.
func TestFetchOntology_NoSkillFile_HasSkillFalse(t *testing.T) {
	chdirTempOntology(t)

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "billing"},
			},
		}},
		t.TempDir(), // empty skills dir
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	assert.False(t, overview.Contexts[0].HasSkill)
}

// ---------------------------------------------------------------------------
// FetchOntology — lineage stats
// ---------------------------------------------------------------------------

// TestFetchOntology_LineageTotalRunsZero_HasLineageFalse verifies that a context
// with TotalRuns=0 in the stats has HasLineage=false.
func TestFetchOntology_LineageTotalRunsZero_HasLineageFalse(t *testing.T) {
	chdirTempOntology(t)

	store := &ontologyMockStore{
		allStats: []state.OntologyStats{
			{ContextName: "billing", TotalRuns: 0},
		},
	}
	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{{Name: "billing"}},
		}},
		t.TempDir(),
		store,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	assert.False(t, overview.Contexts[0].HasLineage, "TotalRuns=0 should give HasLineage=false")
}

// TestFetchOntology_LineageTotalRunsNonZero_HasLineageTrue verifies that a
// context with TotalRuns>0 has HasLineage=true and stats are populated.
func TestFetchOntology_LineageTotalRunsNonZero_HasLineageTrue(t *testing.T) {
	chdirTempOntology(t)

	store := &ontologyMockStore{
		allStats: []state.OntologyStats{
			{ContextName: "billing", TotalRuns: 10, Successes: 8, Failures: 2, SuccessRate: 80.0},
		},
	}
	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{{Name: "billing"}},
		}},
		t.TempDir(),
		store,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 1)
	ctx := overview.Contexts[0]
	assert.True(t, ctx.HasLineage)
	assert.Equal(t, 10, ctx.TotalRuns)
	assert.Equal(t, 8, ctx.Successes)
	assert.Equal(t, 80.0, ctx.SuccessRate)
}

// ---------------------------------------------------------------------------
// FetchOntology — sorting and conventions
// ---------------------------------------------------------------------------

// TestFetchOntology_MultipleContexts_SortedAlphabetically verifies that contexts
// are returned sorted by Name regardless of manifest order.
func TestFetchOntology_MultipleContexts_SortedAlphabetically(t *testing.T) {
	chdirTempOntology(t)

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "zebra"},
				{Name: "alpha"},
				{Name: "middle"},
			},
		}},
		t.TempDir(),
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.Len(t, overview.Contexts, 3)
	assert.Equal(t, "alpha", overview.Contexts[0].Name)
	assert.Equal(t, "middle", overview.Contexts[1].Name)
	assert.Equal(t, "zebra", overview.Contexts[2].Name)
}

// TestFetchOntology_Conventions_Propagated verifies that the Conventions map
// from the manifest is included in the overview.
func TestFetchOntology_Conventions_Propagated(t *testing.T) {
	chdirTempOntology(t)

	p := NewDefaultOntologyDataProvider(
		&manifest.Manifest{Ontology: &manifest.Ontology{
			Telos: "Build great software",
			Conventions: map[string]string{
				"naming": "kebab-case",
			},
		}},
		t.TempDir(),
		nil,
	)

	overview, err := p.FetchOntology()
	require.NoError(t, err)
	require.NotNil(t, overview.Conventions)
	assert.Equal(t, "kebab-case", overview.Conventions["naming"])
}
