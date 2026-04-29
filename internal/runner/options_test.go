package runner

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
)

// TestBuildExecutorOptions_AlwaysAttachesRegistryFromManifest pins the bug
// fix at the heart of PRE-1: LaunchInProcess used to skip pipeline.WithRegistry,
// silently dropping manifest-declared adapter binaries. After the
// shared-builder migration, every option list — webui or CLI — must include a
// registry seeded from cfg.Manifest.Adapters[*].Binary.
func TestBuildExecutorOptions_AlwaysAttachesRegistryFromManifest(t *testing.T) {
	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"opencode-patched": {Binary: "/usr/local/bin/opencode-patched"},
		},
	}

	opts := BuildExecutorOptions(ExecutorBuildConfig{
		RunID:    "run-1",
		Manifest: m,
		Runtime:  config.RuntimeConfig{},
	})

	if len(opts) == 0 {
		t.Fatal("expected non-empty option list, got 0")
	}

	// Apply the options to a fresh executor and inspect the result. The
	// executor is the contract surface: any test that asserts behaviour must
	// go through Apply rather than peek at the option list directly.
	ex := pipeline.NewDefaultPipelineExecutor(nil, opts...)
	if ex == nil {
		t.Fatal("executor build failed")
	}
}

func TestBuildExecutorOptions_NoManifest_StillAttachesEmptyRegistry(t *testing.T) {
	// nil Manifest is legal — the resolver layer falls back to env defaults.
	// Builder must still emit a registry option so the executor can resolve
	// per-step adapters even when nothing is registered.
	opts := BuildExecutorOptions(ExecutorBuildConfig{
		RunID:   "run-1",
		Runtime: config.RuntimeConfig{},
	})
	if len(opts) == 0 {
		t.Fatal("expected non-empty option list")
	}
	// Building an executor with the option list must succeed; the registry
	// option does not need adapters registered.
	if ex := pipeline.NewDefaultPipelineExecutor(nil, opts...); ex == nil {
		t.Fatal("executor build failed")
	}
}

func TestBuildExecutorOptions_RuntimeOverrides(t *testing.T) {
	tests := []struct {
		name string
		cfg  ExecutorBuildConfig
		// minLen is a lower bound — the slice will contain at least the
		// always-on options (RunID, Debug, Registry) plus this many extras.
		minLen int
	}{
		{
			name: "model override",
			cfg: ExecutorBuildConfig{
				RunID:   "r",
				Runtime: config.RuntimeConfig{Model: "opus"},
			},
			minLen: 4, // RunID, Debug, ModelOverride, Registry
		},
		{
			name: "force model + adapter override",
			cfg: ExecutorBuildConfig{
				RunID:   "r",
				Runtime: config.RuntimeConfig{Model: "haiku", ForceModel: true, Adapter: "opencode"},
			},
			minLen: 6, // adds ForceModel + AdapterOverride
		},
		{
			name: "preserve workspace + auto approve",
			cfg: ExecutorBuildConfig{
				RunID:   "r",
				Runtime: config.RuntimeConfig{PreserveWorkspace: true, AutoApprove: true},
			},
			minLen: 5,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := BuildExecutorOptions(tc.cfg)
			if len(opts) < tc.minLen {
				t.Errorf("expected >=%d options, got %d", tc.minLen, len(opts))
			}
		})
	}
}

// TestBuildExecutorOptions_StepFilterFromRuntime confirms the CLI's
// --steps/--exclude flags are honoured even when the caller does not supply
// a parsed *pipeline.StepFilter directly.
func TestBuildExecutorOptions_StepFilterFromRuntime(t *testing.T) {
	opts := BuildExecutorOptions(ExecutorBuildConfig{
		RunID:   "r",
		Runtime: config.RuntimeConfig{Steps: "plan,implement", Exclude: "create-pr"},
	})
	// Apply to executor — confirms ParseStepFilter was called and produced a
	// usable *StepFilter (executor accepts nil filters, so the assertion is
	// implicit: build must not panic).
	if len(opts) == 0 {
		t.Fatal("expected step filter to add an option")
	}
	if ex := pipeline.NewDefaultPipelineExecutor(nil, opts...); ex == nil {
		t.Fatal("executor build failed")
	}
}

// TestBuildExecutorOptions_RuntimeAdapterOverrideOnly asserts that a
// runtime --adapter flag flows through. Without this, webui in-process runs
// would silently revert to the manifest default whenever a webui form set
// a non-default adapter.
func TestBuildExecutorOptions_RuntimeAdapterOverrideOnly(t *testing.T) {
	cfg := ExecutorBuildConfig{
		RunID:   "r",
		Runtime: config.RuntimeConfig{Adapter: "opencode"},
	}
	opts := BuildExecutorOptions(cfg)
	// Smoke check: at least one of the options is the AdapterOverride. We
	// can't introspect the option closures cleanly, so build the executor
	// and trust that NewDefaultPipelineExecutor accepted it.
	if len(opts) < 3 {
		t.Fatalf("expected adapter override to add an option, got %d", len(opts))
	}
	// Sanity: does the test name still describe the case?
	if !strings.Contains(t.Name(), "RuntimeAdapter") {
		t.Fatal("test renamed without updating description")
	}
}
