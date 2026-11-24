package commands

import (
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaAdapterName_Deterministic(t *testing.T) {
	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{"zeta": {}, "alpha": {}, "mid": {}},
		Personas: map[string]manifest.Persona{"philosopher": {Adapter: "mid"}},
	}
	assert.Equal(t, "mid", metaAdapterName(m), "philosopher adapter wins")

	delete(m.Personas, "philosopher")
	for i := 0; i < 20; i++ {
		assert.Equal(t, "alpha", metaAdapterName(m), "fallback is the alphabetically first adapter")
	}

	assert.Equal(t, "", metaAdapterName(&manifest.Manifest{}))
}

func TestLoadFailureContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")
	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	runID, err := store.CreateRun("does-not-exist-pipeline", "add a health endpoint")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "failed", "step \"review\" failed: contract validation failed", 0))
	require.NoError(t, store.LogEvent(runID, "review", "contract_failed", "craftsman", "contract validation failed [json_schema]: missing required field summary", 0, 0, "", "", ""))
	require.NoError(t, store.Close())

	task, fc, err := loadFailureContext(dbPath, runID)
	require.NoError(t, err)
	assert.Equal(t, "add a health endpoint", task)
	assert.Contains(t, fc, "Pipeline: does-not-exist-pipeline")
	assert.Contains(t, fc, "Status: failed")
	assert.Contains(t, fc, "contract validation failed")
	assert.Contains(t, fc, "missing required field summary")

	_, _, err = loadFailureContext(dbPath, "no-such-run")
	require.Error(t, err)
}
