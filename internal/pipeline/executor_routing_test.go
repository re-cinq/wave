package pipeline

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 1. resolveModel — four-tier precedence
// ---------------------------------------------------------------------------

func TestResolveModel_FourTierPrecedence(t *testing.T) {
	tests := []struct {
		name          string
		modelOverride string // CLI --model
		stepModel     string
		personaModel  string
		want          string
	}{
		{"all empty", "", "", "", ""},
		{"persona only", "", "", "opus", "opus"},
		{"step overrides persona", "", "haiku", "opus", "haiku"},
		{"CLI overrides all", "sonnet", "haiku", "opus", "sonnet"},
		{"step only", "", "haiku", "", "haiku"},
		{"CLI with empty step and persona", "sonnet", "", "", "sonnet"},
		{"CLI overrides step only", "sonnet", "haiku", "", "sonnet"},
		{"persona without step or CLI", "", "", "opus", "opus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := &DefaultPipelineExecutor{modelOverride: tt.modelOverride}
			step := &Step{Model: tt.stepModel}
			persona := &manifest.Persona{Model: tt.personaModel}

			got := ex.resolveModel(step, persona)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// 2. resolveRunner — step.Adapter > persona.Adapter > fallback to e.runner
// ---------------------------------------------------------------------------

// namedMockRunner wraps a MockAdapter and records its own name so tests can
// verify which runner was selected by resolveRunner.
type namedMockRunner struct {
	name string
	*adapter.MockAdapter
}

func newNamedMockRunner(name string) *namedMockRunner {
	return &namedMockRunner{
		name: name,
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status":"success"}`),
			adapter.WithTokensUsed(100),
		),
	}
}

func TestResolveRunner_StepOverridesPersona(t *testing.T) {
	claudeRunner := newNamedMockRunner("claude")
	opencodeRunner := newNamedMockRunner("opencode")
	defaultRunner := newNamedMockRunner("default")

	reg := adapter.NewAdapterRegistry()
	reg.Register("claude", claudeRunner)
	reg.Register("opencode", opencodeRunner)

	ex := NewDefaultPipelineExecutor(defaultRunner, WithAdapterRegistry(reg))

	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"claude":   {Binary: "claude"},
			"opencode": {Binary: "opencode"},
		},
	}

	// Step specifies "opencode", persona specifies "claude" — step wins
	step := &Step{Adapter: "opencode"}
	persona := &manifest.Persona{Adapter: "claude"}

	runner := ex.resolveRunner(step, persona, m)
	// The resolved runner should be the opencode runner
	named, ok := runner.(*namedMockRunner)
	require.True(t, ok, "runner should be *namedMockRunner")
	assert.Equal(t, "opencode", named.name)
}

func TestResolveRunner_PersonaUsedWhenStepEmpty(t *testing.T) {
	claudeRunner := newNamedMockRunner("claude")
	opencodeRunner := newNamedMockRunner("opencode")
	defaultRunner := newNamedMockRunner("default")

	reg := adapter.NewAdapterRegistry()
	reg.Register("claude", claudeRunner)
	reg.Register("opencode", opencodeRunner)

	ex := NewDefaultPipelineExecutor(defaultRunner, WithAdapterRegistry(reg))

	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"claude":   {Binary: "claude"},
			"opencode": {Binary: "opencode"},
		},
	}

	// Step has no adapter override — persona adapter "claude" should be used
	step := &Step{Adapter: ""}
	persona := &manifest.Persona{Adapter: "claude"}

	runner := ex.resolveRunner(step, persona, m)
	named, ok := runner.(*namedMockRunner)
	require.True(t, ok, "runner should be *namedMockRunner")
	assert.Equal(t, "claude", named.name)
}

func TestResolveRunner_FallsBackToDefaultRunner(t *testing.T) {
	defaultRunner := newNamedMockRunner("default")

	reg := adapter.NewAdapterRegistry()
	// Registry is empty — no adapters registered

	ex := NewDefaultPipelineExecutor(defaultRunner, WithAdapterRegistry(reg))

	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
	}

	step := &Step{Adapter: ""}
	persona := &manifest.Persona{Adapter: "claude"}

	// "claude" not in registry — should fall back to default runner
	runner := ex.resolveRunner(step, persona, m)
	named, ok := runner.(*namedMockRunner)
	require.True(t, ok, "runner should be *namedMockRunner")
	assert.Equal(t, "default", named.name)
}

func TestResolveRunner_NoRegistry(t *testing.T) {
	defaultRunner := newNamedMockRunner("default")

	// No registry set — should always return e.runner
	ex := NewDefaultPipelineExecutor(defaultRunner)

	m := &manifest.Manifest{}
	step := &Step{Adapter: "opencode"}
	persona := &manifest.Persona{Adapter: "claude"}

	runner := ex.resolveRunner(step, persona, m)
	named, ok := runner.(*namedMockRunner)
	require.True(t, ok, "runner should be *namedMockRunner")
	assert.Equal(t, "default", named.name)
}

// ---------------------------------------------------------------------------
// 3. Per-step adapter resolution through full Execute path
// ---------------------------------------------------------------------------

// adapterCapturingRunner records which adapter name was used for each step.
type adapterCapturingRunner struct {
	mu    sync.Mutex
	calls map[string]string // stepID -> adapter name inferred from workspace path
	inner adapter.AdapterRunner
	name  string
}

func newAdapterCapturingRunner(name string) *adapterCapturingRunner {
	return &adapterCapturingRunner{
		name:  name,
		calls: make(map[string]string),
		inner: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status":"success"}`),
			adapter.WithTokensUsed(100),
		),
	}
}

func (a *adapterCapturingRunner) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	stepID := filepath.Base(cfg.WorkspacePath)
	a.mu.Lock()
	a.calls[stepID] = a.name
	a.mu.Unlock()
	return a.inner.Run(ctx, cfg)
}

func (a *adapterCapturingRunner) getAdapterForStep(stepID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[stepID]
}

func TestPerStepAdapterResolution_Integration(t *testing.T) {
	claudeRunner := newAdapterCapturingRunner("claude")
	opencodeRunner := newAdapterCapturingRunner("opencode")
	defaultRunner := newAdapterCapturingRunner("default")

	reg := adapter.NewAdapterRegistry()
	reg.Register("claude", claudeRunner)
	reg.Register("opencode", opencodeRunner)

	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(defaultRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "adapter-routing-test"},
		Adapters: map[string]manifest.Adapter{
			"claude":   {Binary: "claude", Mode: "headless"},
			"opencode": {Binary: "opencode", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"nav": {
				Adapter:     "claude",
				Temperature: 0.1,
			},
			"dev": {
				Adapter:     "opencode",
				Temperature: 0.7,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "adapter-routing"},
		Steps: []Step{
			{
				ID:      "navigate",
				Persona: "nav",
				// No adapter override — should use persona adapter "claude"
				Exec: ExecConfig{Source: "navigate"},
			},
			{
				ID:           "implement",
				Persona:      "nav",
				Adapter:      "opencode", // Override: persona is "nav" (claude) but step forces opencode
				Dependencies: []string{"navigate"},
				Exec:         ExecConfig{Source: "implement"},
			},
			{
				ID:           "review",
				Persona:      "dev",
				Dependencies: []string{"implement"},
				// No adapter override — should use persona adapter "opencode"
				Exec: ExecConfig{Source: "review"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test routing")
	require.NoError(t, err)

	// Step "navigate": persona "nav" -> adapter "claude"
	assert.Equal(t, "claude", claudeRunner.getAdapterForStep("navigate"),
		"navigate step should use claude adapter from persona")

	// Step "implement": step.Adapter="opencode" overrides persona "nav" (claude)
	assert.Equal(t, "opencode", opencodeRunner.getAdapterForStep("implement"),
		"implement step should use opencode adapter from step override")

	// Step "review": persona "dev" -> adapter "opencode"
	assert.Equal(t, "opencode", opencodeRunner.getAdapterForStep("review"),
		"review step should use opencode adapter from persona")
}

// ---------------------------------------------------------------------------
// 4. Model resolution through full Execute path (four-tier)
// ---------------------------------------------------------------------------

func TestModelResolution_FourTier_Integration(t *testing.T) {
	capturer := newModelCapturingAdapter()
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(capturer,
		WithEmitter(collector),
		WithModelOverride("cli-model"),
	)

	tmpDir := t.TempDir()
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "model-four-tier"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"pinned": {
				Adapter:     "claude",
				Model:       "persona-model",
				Temperature: 0.1,
			},
			"unpinned": {
				Adapter:     "claude",
				Temperature: 0.1,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "model-four-tier"},
		Steps: []Step{
			{
				ID:      "step-cli-wins",
				Persona: "pinned",
				Model:   "step-model",
				Exec:    ExecConfig{Source: "test"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test model")
	require.NoError(t, err)

	// CLI override wins over both step and persona model
	assert.Equal(t, "cli-model", capturer.getModel("step-cli-wins"))
}

func TestModelResolution_StepOverridesPersona_Integration(t *testing.T) {
	capturer := newModelCapturingAdapter()
	collector := testutil.NewEventCollector()

	// No CLI model override
	executor := NewDefaultPipelineExecutor(capturer,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "model-step-over-persona"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"pinned": {
				Adapter:     "claude",
				Model:       "persona-model",
				Temperature: 0.1,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "model-step-over-persona"},
		Steps: []Step{
			{
				ID:      "step-model-wins",
				Persona: "pinned",
				Model:   "step-model",
				Exec:    ExecConfig{Source: "test"},
			},
			{
				ID:           "persona-model-used",
				Persona:      "pinned",
				Dependencies: []string{"step-model-wins"},
				// No step model — persona model should be used
				Exec: ExecConfig{Source: "test"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test model")
	require.NoError(t, err)

	assert.Equal(t, "step-model", capturer.getModel("step-model-wins"),
		"step model should override persona model")
	assert.Equal(t, "persona-model", capturer.getModel("persona-model-used"),
		"persona model should be used when step has no model")
}

// ---------------------------------------------------------------------------
// 5. Fallback chain triggered on transient errors
// ---------------------------------------------------------------------------

// failingMockRunner returns a configurable failure on first N calls, then succeeds.
type failingMockRunner struct {
	mu         sync.Mutex
	callCount  int
	failUntil  int
	failReason string // If non-empty, return a result with this FailureReason instead of an error
	failErr    error  // If non-nil, return this error
	name       string
	called     bool
}

func (f *failingMockRunner) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	f.mu.Lock()
	f.callCount++
	count := f.callCount
	f.called = true
	f.mu.Unlock()

	if count <= f.failUntil {
		if f.failErr != nil {
			return nil, f.failErr
		}
		return &adapter.AdapterResult{
			ExitCode:      1,
			Stdout:        bytes.NewReader([]byte("error")),
			ResultContent: "error",
			FailureReason: f.failReason,
		}, nil
	}

	return &adapter.AdapterResult{
		ExitCode:      0,
		Stdout:        bytes.NewReader([]byte(`{"status":"success"}`)),
		ResultContent: `{"status":"success"}`,
		TokensUsed:    100,
	}, nil
}

func (f *failingMockRunner) wasCalled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.called
}

func TestRunWithFallback_TransientError_TriesFallback(t *testing.T) {
	// Primary adapter fails with rate_limit (transient) — fallback should be tried
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	fallbackRunner := newNamedMockRunner("fallback")

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunWithFallback_TimeoutError_TriesFallback(t *testing.T) {
	// Primary adapter fails with timeout (transient) — fallback should be tried
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonTimeout,
	}

	fallbackRunner := newNamedMockRunner("fallback")

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunWithFallback_ErrorReturn_TriesFallback(t *testing.T) {
	// Primary adapter returns a Go error (not a result failure) — should retry
	primaryRunner := &failingMockRunner{
		name:      "primary",
		failUntil: 1,
		failErr:   errors.New("connection refused"),
	}

	fallbackRunner := newNamedMockRunner("fallback")

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

// ---------------------------------------------------------------------------
// 6. Fallback skipped on permanent errors
// ---------------------------------------------------------------------------

func TestRunWithFallback_PermanentError_NoFallback(t *testing.T) {
	// Primary adapter fails with general_error (permanent) — fallback should NOT be tried
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonGeneralError,
	}

	fallbackRunner := &failingMockRunner{
		name:      "fallback",
		failUntil: 0, // would succeed if called
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	// Should return the primary result without trying fallback
	require.NoError(t, err)
	assert.Equal(t, adapter.FailureReasonGeneralError, result.FailureReason,
		"permanent error result should be returned as-is")
	assert.False(t, fallbackRunner.wasCalled(),
		"fallback runner should not be called for permanent errors")
}

func TestRunWithFallback_ContextExhaustion_NoFallback(t *testing.T) {
	// context_exhaustion is also a permanent error
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonContextExhaustion,
	}

	fallbackRunner := &failingMockRunner{
		name:      "fallback",
		failUntil: 0,
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, adapter.FailureReasonContextExhaustion, result.FailureReason)
	assert.False(t, fallbackRunner.wasCalled(),
		"fallback should not be called for context exhaustion")
}

func TestRunWithFallback_NoFallbacksConfigured(t *testing.T) {
	// Even if adapter fails transiently, no fallbacks configured means original error returned
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	// Empty fallback map
	fallbacks := map[string][]string{}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, adapter.FailureReasonRateLimit, result.FailureReason,
		"rate limit result should be returned when no fallbacks configured")
}

func TestRunWithFallback_CancelledContext_NoFallback(t *testing.T) {
	// If context is already cancelled, do not try fallbacks
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	primaryRunner := &failingMockRunner{
		name:      "primary",
		failUntil: 1,
		failErr:   ctx.Err(),
	}

	fallbackRunner := &failingMockRunner{
		name:      "fallback",
		failUntil: 0,
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	_, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.Error(t, err)
	assert.False(t, fallbackRunner.wasCalled(),
		"fallback should not be attempted when context is cancelled")
}

func TestRunWithFallback_MultipleFallbacks_TriesInOrder(t *testing.T) {
	// Primary fails, first fallback also fails transiently, second fallback succeeds
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	fallback1 := &failingMockRunner{
		name:       "fallback1",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	fallback2 := newNamedMockRunner("fallback2")

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback1", fallback1)
	reg.Register("fallback2", fallback2)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback1", "fallback2"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode, "second fallback should have succeeded")
	assert.True(t, fallback1.wasCalled(), "first fallback should have been tried")
}

func TestRunWithFallback_AllFallbacksExhausted(t *testing.T) {
	// Primary and all fallbacks fail with transient errors
	primaryRunner := &failingMockRunner{
		name:      "primary",
		failUntil: 1,
		failErr:   errors.New("primary down"),
	}

	fallback1 := &failingMockRunner{
		name:       "fb1",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	fallback2 := &failingMockRunner{
		name:       "fb2",
		failUntil:  1,
		failReason: adapter.FailureReasonTimeout,
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fb1", fallback1)
	reg.Register("fb2", fallback2)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fb1", "fb2"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	_, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all fallback adapters exhausted",
		"error should indicate all fallbacks were exhausted")
	assert.Contains(t, err.Error(), "primary down",
		"error should wrap the original primary error")
}

func TestRunWithFallback_PrimarySucceeds_NoFallbackAttempted(t *testing.T) {
	primaryRunner := newNamedMockRunner("primary")

	fallbackRunner := &failingMockRunner{
		name:      "fallback",
		failUntil: 0,
	}

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, fallbackRunner.wasCalled(),
		"fallback should not be attempted when primary succeeds")
}

func TestRunWithFallback_NoRegistrySet(t *testing.T) {
	// When no registry is set, fallback is not possible even with fallback config
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	// No registry set on executor
	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	result, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)

	// Should return the primary failure without trying any fallback
	require.NoError(t, err)
	assert.Equal(t, adapter.FailureReasonRateLimit, result.FailureReason)
}

// ---------------------------------------------------------------------------
// 7. Fallback emits events
// ---------------------------------------------------------------------------

func TestRunWithFallback_EmitsFallbackEvent(t *testing.T) {
	primaryRunner := &failingMockRunner{
		name:       "primary",
		failUntil:  1,
		failReason: adapter.FailureReasonRateLimit,
	}

	fallbackRunner := newNamedMockRunner("fallback")

	reg := adapter.NewAdapterRegistry()
	reg.Register("primary", primaryRunner)
	reg.Register("fallback", fallbackRunner)

	collector := testutil.NewEventCollector()
	ex := NewDefaultPipelineExecutor(primaryRunner,
		WithAdapterRegistry(reg),
		WithEmitter(collector),
	)

	fallbacks := map[string][]string{
		"primary": {"fallback"},
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       "primary",
		WorkspacePath: "/tmp/test",
		Prompt:        "test",
	}

	ctx := context.Background()
	_, err := ex.runWithFallback(ctx, cfg, primaryRunner, "primary", fallbacks)
	require.NoError(t, err)

	// Verify a fallback event was emitted
	events := collector.GetEvents()
	var foundFallbackEvent bool
	for _, e := range events {
		if e.State == "fallback" {
			foundFallbackEvent = true
			assert.Contains(t, e.Message, "primary")
			assert.Contains(t, e.Message, "fallback")
			break
		}
	}
	assert.True(t, foundFallbackEvent, "should emit a fallback event when switching adapters")
}
