package ontology

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureEmitter records every Emit call for assertion.
type captureEmitter struct {
	mu     sync.Mutex
	events []event.Event
}

func (c *captureEmitter) Emit(evt event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
}

func (c *captureEmitter) states() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := make([]string, len(c.events))
	for i, e := range c.events {
		s[i] = e.State
	}
	return s
}

// captureSink records audit-log lines.
type captureSink struct {
	mu   sync.Mutex
	logs []string
}

func (c *captureSink) LogEvent(kind, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = append(c.logs, kind+" "+body)
	return nil
}

// nilStore satisfies state.StateStore's RecordOntologyUsage via a panic path
// if called — tests that need store semantics should use state's in-memory
// helpers. These tests cover the Enabled=false branch plus event emission.

func TestEnabledFromManifest(t *testing.T) {
	assert.False(t, EnabledFromManifest(nil))
	assert.False(t, EnabledFromManifest(&manifest.Manifest{}))
	assert.False(t, EnabledFromManifest(&manifest.Manifest{Ontology: &manifest.Ontology{}}))
	assert.True(t, EnabledFromManifest(&manifest.Manifest{
		Ontology: &manifest.Ontology{Contexts: []manifest.OntologyContext{{Name: "x"}}},
	}))
}

func TestNoOp_AllMethodsAreSafe(t *testing.T) {
	var svc Service = NoOp{}
	assert.False(t, svc.Enabled())
	assert.Equal(t, "", svc.CheckStaleness())
	assert.Equal(t, "", svc.BuildStepSection("p", "s", []string{"a"}))
	svc.RecordUsage("p", "s", []string{"a"}, true, "success")
	assert.Nil(t, svc.ValidateManifest(nil))
	assert.NoError(t, svc.InstallStalenessHook())
}

func TestNew_DisabledReturnsNoOp(t *testing.T) {
	svc := New(Config{Enabled: false}, Deps{})
	_, ok := svc.(NoOp)
	assert.True(t, ok)
}

func TestNew_EnabledReturnsRealService(t *testing.T) {
	m := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{{Name: "billing"}},
	}}
	svc := New(Config{Enabled: true}, Deps{Manifest: m})
	assert.True(t, svc.Enabled())
}

func TestBuildStepSection_EmitsInjectEvent(t *testing.T) {
	em := &captureEmitter{}
	sink := &captureSink{}
	m := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{
			{Name: "billing", Description: "money", Invariants: []string{"must balance"}},
		},
	}}
	svc := New(Config{Enabled: true}, Deps{Manifest: m, Emitter: em, AuditSink: sink})

	section := svc.BuildStepSection("run-1", "step-1", []string{"billing"})
	require.NotEmpty(t, section)

	states := em.states()
	assert.Contains(t, states, event.StateOntologyInject)
}

func TestBuildStepSection_WarnsOnUndefinedContext(t *testing.T) {
	em := &captureEmitter{}
	sink := &captureSink{}
	m := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{{Name: "billing"}},
	}}
	svc := New(Config{Enabled: true}, Deps{Manifest: m, Emitter: em, AuditSink: sink})

	svc.BuildStepSection("run-1", "step-1", []string{"unknown"})
	assert.Contains(t, em.states(), event.StateOntologyWarn)
}

func TestCheckStaleness_SentinelClearedAfterRead(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.MkdirAll(".agents", 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(".agents", ".ontology-stale"), nil, 0o644))

	m := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{{Name: "x"}},
	}}
	svc := New(Config{Enabled: true}, Deps{Manifest: m})

	msg := svc.CheckStaleness()
	assert.Contains(t, msg, "stale")

	// Sentinel is removed after the first read.
	_, err := os.Stat(filepath.Join(".agents", ".ontology-stale"))
	assert.True(t, os.IsNotExist(err), "sentinel should be cleared after CheckStaleness")
}

func TestIsStaleInDir(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, IsStaleInDir(dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ontology-stale"), nil, 0o644))
	assert.True(t, IsStaleInDir(dir))
}

func TestInstallStalenessHookAt_CreatesHook(t *testing.T) {
	dir := t.TempDir()
	hookDir := filepath.Join(dir, "hooks")
	require.NoError(t, os.MkdirAll(hookDir, 0o755))

	require.NoError(t, InstallStalenessHookAt(hookDir))

	data, err := os.ReadFile(filepath.Join(hookDir, "post-merge"))
	require.NoError(t, err)
	assert.Contains(t, string(data), hookMarker)
	assert.Contains(t, string(data), ".ontology-stale")

	// Idempotent: second install must not duplicate.
	require.NoError(t, InstallStalenessHookAt(hookDir))
	data2, err := os.ReadFile(filepath.Join(hookDir, "post-merge"))
	require.NoError(t, err)
	assert.Equal(t, string(data), string(data2))
}

func TestInstallStalenessHookAt_AppendsWhenHookExists(t *testing.T) {
	dir := t.TempDir()
	hookDir := filepath.Join(dir, "hooks")
	require.NoError(t, os.MkdirAll(hookDir, 0o755))

	existing := "#!/bin/sh\necho existing\n"
	require.NoError(t, os.WriteFile(filepath.Join(hookDir, "post-merge"), []byte(existing), 0o755))

	require.NoError(t, InstallStalenessHookAt(hookDir))

	data, err := os.ReadFile(filepath.Join(hookDir, "post-merge"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "echo existing")
	assert.Contains(t, string(data), hookMarker)
}

func TestValidateManifestShape(t *testing.T) {
	// nil -> no errs
	assert.Nil(t, ValidateManifestShape(nil))

	// duplicate name
	m := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{
			{Name: "a"},
			{Name: "a"},
		},
	}}
	errs := ValidateManifestShape(m)
	require.Len(t, errs, 1)

	// empty name
	m2 := &manifest.Manifest{Ontology: &manifest.Ontology{
		Contexts: []manifest.OntologyContext{{Name: "   "}},
	}}
	errs = ValidateManifestShape(m2)
	require.Len(t, errs, 1)
}
