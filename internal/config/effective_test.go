package config

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

// fakeManifest is a test-only ManifestDefaults that returns canned values.
// Used to keep the merge tests focused on precedence rules, not on coupling
// to the real internal/manifest.Manifest type.
type fakeManifest struct {
	firstAdapter   string
	defaultTimeout time.Duration
}

func (f fakeManifest) FirstAdapter() string          { return f.firstAdapter }
func (f fakeManifest) DefaultTimeout() time.Duration { return f.defaultTimeout }

func TestResolve_PrecedenceTable(t *testing.T) {
	tests := []struct {
		name string
		m    ManifestDefaults
		env  Env
		rt   RuntimeConfig
		want Effective
	}{
		{
			name: "manifest only — adapter and timeout from manifest",
			m: fakeManifest{
				firstAdapter:   "opencode",
				defaultTimeout: 12 * time.Minute,
			},
			env: Env{},
			rt:  RuntimeConfig{},
			want: Effective{
				Adapter:     "opencode",
				StepTimeout: 12 * time.Minute,
			},
		},
		{
			name: "env overrides manifest for NoColor",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{NoColor: "1"},
			rt:  RuntimeConfig{},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 5 * time.Minute,
				NoColor:     true,
			},
		},
		{
			name: "runtime adapter overrides manifest",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Adapter: "gemini",
			},
			want: Effective{
				Adapter:     "gemini",
				StepTimeout: 5 * time.Minute,
			},
		},
		{
			name: "runtime timeout overrides manifest",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Timeout: 30, // minutes
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 30 * time.Minute,
			},
		},
		{
			name: "runtime zero-value timeout defers to manifest",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 7 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Timeout: 0,
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 7 * time.Minute,
			},
		},
		{
			name: "runtime empty adapter defers to manifest",
			m: fakeManifest{
				firstAdapter:   "codex",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Adapter: "",
			},
			want: Effective{
				Adapter:     "codex",
				StepTimeout: 5 * time.Minute,
			},
		},
		{
			name: "nil manifest tolerated — adapter falls through to claude, timeout to default",
			m:    nil,
			env:  Env{},
			rt:   RuntimeConfig{},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: timeouts.StepDefault,
			},
		},
		{
			name: "nil manifest with runtime overrides — runtime still wins",
			m:    nil,
			env:  Env{},
			rt: RuntimeConfig{
				Adapter: "gemini",
				Timeout: 15,
			},
			want: Effective{
				Adapter:     "gemini",
				StepTimeout: 15 * time.Minute,
			},
		},
		{
			name: "model passes through from runtime — no manifest layer for run-wide model",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Model: "haiku",
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 5 * time.Minute,
				Model:       "haiku",
			},
		},
		{
			name: "runtime-only flags bundled (auto-approve / no-retro / force-model)",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				AutoApprove: true,
				NoRetro:     true,
				ForceModel:  true,
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 5 * time.Minute,
				AutoApprove: true,
				NoRetro:     true,
				ForceModel:  true,
			},
		},
		{
			name: "runtime Output.NoColor sets NoColor even when env unset",
			m: fakeManifest{
				firstAdapter:   "claude",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt: RuntimeConfig{
				Output: OutputConfig{NoColor: true},
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 5 * time.Minute,
				NoColor:     true,
			},
		},
		{
			name: "empty manifest first adapter falls through to claude",
			m: fakeManifest{
				firstAdapter:   "",
				defaultTimeout: 5 * time.Minute,
			},
			env: Env{},
			rt:  RuntimeConfig{},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 5 * time.Minute,
			},
		},
		{
			name: "all three layers contribute — runtime adapter, env nocolor, manifest timeout",
			m: fakeManifest{
				firstAdapter:   "opencode",
				defaultTimeout: 9 * time.Minute,
			},
			env: Env{NoColor: "true"},
			rt: RuntimeConfig{
				Adapter: "claude",
				Model:   "opus",
			},
			want: Effective{
				Adapter:     "claude",
				StepTimeout: 9 * time.Minute,
				Model:       "opus",
				NoColor:     true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.m, tc.env, tc.rt)
			if got != tc.want {
				t.Errorf("Resolve() mismatch\n got = %+v\nwant = %+v", got, tc.want)
			}
		})
	}
}

func TestManifestDefaultsFunc_NilFunctionsSafe(t *testing.T) {
	// A zero-valued ManifestDefaultsFunc must be safe to call — callers that
	// only know one of the two values should not have to populate both.
	var f ManifestDefaultsFunc

	if got := f.FirstAdapter(); got != "" {
		t.Errorf("FirstAdapter() with nil fn = %q, want empty", got)
	}
	if got := f.DefaultTimeout(); got != timeouts.StepDefault {
		t.Errorf("DefaultTimeout() with nil fn = %v, want %v", got, timeouts.StepDefault)
	}
}

func TestManifestDefaultsFunc_DelegatesToFunctions(t *testing.T) {
	f := ManifestDefaultsFunc{
		FirstAdapterFn:   func() string { return "gemini" },
		DefaultTimeoutFn: func() time.Duration { return 42 * time.Second },
	}

	if got := f.FirstAdapter(); got != "gemini" {
		t.Errorf("FirstAdapter() = %q, want gemini", got)
	}
	if got := f.DefaultTimeout(); got != 42*time.Second {
		t.Errorf("DefaultTimeout() = %v, want 42s", got)
	}
}
