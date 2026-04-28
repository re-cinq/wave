package commands

import (
	"reflect"
	"testing"
)

// TestBuildDetachedArgsAllFlagsPresent constructs a RunOptions value with every
// non-zero field, runs the argv builder, and asserts that every flag declared
// in detachFlagSpecs appears in the produced argv. This is the regression test
// for issue #1500 — Continuous, Source, MaxIterations, Delay, OnFailure, and
// NoRetro were silently dropped from the detached subprocess invocation.
func TestBuildDetachedArgsAllFlagsPresent(t *testing.T) {
	opts := RunOptions{
		Pipeline:          "impl-issue",
		Input:             "fix login bug",
		FromStep:          "implement",
		Force:             true,
		Timeout:           42,
		Manifest:          "custom.yaml",
		Mock:              true,
		Model:             "haiku",
		ForceModel:        true,
		Adapter:           "claude",
		PreserveWorkspace: true,
		Steps:             "plan,implement",
		Exclude:           "create-pr",
		Continuous:        true,
		Source:            "github:label=bug",
		MaxIterations:     7,
		Delay:             "30s",
		OnFailure:         "skip",
		AutoApprove:       true,
		NoRetro:           true,
	}
	opts.Output.Verbose = true

	args := buildDetachedArgs(opts, "run-xyz")

	// Always-emitted prefix.
	if got, want := args[0], "run"; got != want {
		t.Fatalf("argv[0] = %q, want %q", got, want)
	}
	mustContainPair(t, args, "--pipeline", "impl-issue")
	mustContainPair(t, args, "--run", "run-xyz")

	// Every spec entry should produce its flag for these inputs.
	for _, spec := range detachFlagSpecs {
		if !containsFlag(args, "--"+spec.flag) {
			t.Errorf("detached argv missing --%s (RunOptions field %s)", spec.flag, spec.field)
		}
	}

	// OutputConfig.Verbose is special-cased outside the spec list.
	if !containsFlag(args, "--verbose") {
		t.Errorf("detached argv missing --verbose")
	}

	// Spot-check the six fields explicitly named in the bug report all
	// appear with their expected values.
	mustContainFlag(t, args, "--continuous")
	mustContainPair(t, args, "--source", "github:label=bug")
	mustContainPair(t, args, "--max-iterations", "7")
	mustContainPair(t, args, "--delay", "30s")
	mustContainPair(t, args, "--on-failure", "skip")
	mustContainFlag(t, args, "--no-retro")
}

// TestBuildDetachedArgsZeroValuesOmitted asserts that a near-empty RunOptions
// (just pipeline + runID) does not emit conditional flags.
func TestBuildDetachedArgsZeroValuesOmitted(t *testing.T) {
	opts := RunOptions{Pipeline: "impl-issue", Manifest: "wave.yaml"}
	args := buildDetachedArgs(opts, "run-xyz")

	mustContainPair(t, args, "--pipeline", "impl-issue")
	mustContainPair(t, args, "--run", "run-xyz")

	// None of the conditional flags should appear.
	for _, spec := range detachFlagSpecs {
		if containsFlag(args, "--"+spec.flag) {
			t.Errorf("zero-value RunOptions still emitted --%s", spec.flag)
		}
	}
	if containsFlag(args, "--verbose") {
		t.Errorf("zero-value RunOptions still emitted --verbose")
	}
}

// TestBuildDetachedArgsManifestDefaultOmitted verifies that a manifest set to
// the default "wave.yaml" is not forwarded — matches the legacy behaviour.
func TestBuildDetachedArgsManifestDefaultOmitted(t *testing.T) {
	opts := RunOptions{Pipeline: "p", Manifest: "wave.yaml"}
	args := buildDetachedArgs(opts, "rid")
	if containsFlag(args, "--manifest") {
		t.Errorf("default manifest 'wave.yaml' should not be forwarded; got %v", args)
	}
}

// TestDetachedArgsExhaustive walks the RunOptions struct via reflection and
// asserts that every field is either:
//   - registered in detachFlagSpecs by name, or
//   - explicitly skipped via detachFlagSkippedFields with a reason.
//
// This guards against future RunOptions fields silently dropping out of the
// detached subprocess invocation (the original bug in #1500).
func TestDetachedArgsExhaustive(t *testing.T) {
	registered := make(map[string]bool, len(detachFlagSpecs))
	for _, spec := range detachFlagSpecs {
		if registered[spec.field] {
			t.Errorf("duplicate detachFlagSpec for field %q", spec.field)
		}
		registered[spec.field] = true
	}

	rt := reflect.TypeOf(RunOptions{})
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if registered[name] {
			continue
		}
		if _, skipped := detachFlagSkippedFields[name]; skipped {
			continue
		}
		t.Errorf("RunOptions field %q is neither registered in detachFlagSpecs "+
			"nor listed in detachFlagSkippedFields — add it to one of them so "+
			"runDetached forwards (or explicitly drops) the flag", name)
	}

	// Inverse check: skipped fields must actually exist on RunOptions, so
	// stale entries surface as failures during refactors.
	for name := range detachFlagSkippedFields {
		if _, ok := rt.FieldByName(name); !ok {
			t.Errorf("detachFlagSkippedFields references unknown RunOptions field %q", name)
		}
	}
}

// containsFlag reports whether args contains the given flag token.
func containsFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func mustContainFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	if !containsFlag(args, flag) {
		t.Errorf("argv missing %s; full argv: %v", flag, args)
	}
}

// mustContainPair asserts that argv contains "flag value" as adjacent tokens.
func mustContainPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return
		}
	}
	t.Errorf("argv missing pair %s %s; full argv: %v", flag, value, args)
}
