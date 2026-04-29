// Package runner is the single source of truth for launching pipeline runs —
// either in-process (LaunchInProcess) or as a fully-detached subprocess
// (Detach). It is shared by cmd/wave/commands and internal/webui so the two
// paths produce identical runtime behaviour.
//
// The package intentionally exposes a small surface:
//
//	config.RuntimeConfig — every CLI-parity input (mirror of `wave run` flags),
//	                       defined in internal/config alongside the env snapshot
//	Detach               — spawn a `wave run` subprocess (Setsid + Process.Release)
//	LaunchInProcess      — wire up DefaultPipelineExecutor and run a goroutine
//
// The detach flag-spec table (DetachFlagSpecs / DetachFlagSkippedFields) lives
// in detach.go and is exercised by TestDetachedArgsExhaustive — adding a new
// config.RuntimeConfig field requires registering it (or explicitly skipping
// it) in that table, otherwise the test fails.
package runner
