package git

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

// stubRunner returns a Runner that records invocations and returns the
// next response from a sequence. Tests construct it inline to keep cases
// self-contained.
type stubResponse struct {
	out []byte
	err error
}

func newStubRunner(responses []stubResponse) (Runner, *[][]string) {
	var calls [][]string
	idx := 0
	r := func(args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if idx >= len(responses) {
			return nil, errors.New("no response queued")
		}
		resp := responses[idx]
		idx++
		return resp.out, resp.err
	}
	return r, &calls
}

func withRunner(t *testing.T, r Runner) {
	t.Helper()
	prev := SetRunner(r)
	t.Cleanup(func() { SetRunner(prev) })
}

func TestBranch(t *testing.T) {
	r, calls := newStubRunner([]stubResponse{{out: []byte("main\n")}})
	withRunner(t, r)

	got, err := Branch()
	if err != nil {
		t.Fatalf("Branch: %v", err)
	}
	if got != "main" {
		t.Errorf("got %q, want %q", got, "main")
	}
	want := [][]string{{"rev-parse", "--abbrev-ref", "HEAD"}}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls = %v, want %v", *calls, want)
	}
}

func TestBranchError(t *testing.T) {
	r, _ := newStubRunner([]stubResponse{{err: errors.New("boom")}})
	withRunner(t, r)

	if _, err := Branch(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestShortHash(t *testing.T) {
	r, calls := newStubRunner([]stubResponse{{out: []byte("abcdef0\n")}})
	withRunner(t, r)

	got, err := ShortHash()
	if err != nil {
		t.Fatalf("ShortHash: %v", err)
	}
	if got != "abcdef0" {
		t.Errorf("got %q", got)
	}
	if (*calls)[0][0] != "rev-parse" || (*calls)[0][1] != "--short" {
		t.Errorf("unexpected args: %v", *calls)
	}
}

func TestIsDirty(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{"clean", "", false},
		{"clean whitespace", "   \n", false},
		{"dirty", " M file.go\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newStubRunner([]stubResponse{{out: []byte(tt.out)}})
			withRunner(t, r)

			got, err := IsDirty()
			if err != nil {
				t.Fatalf("IsDirty: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFirstRemote(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
	}{
		{"origin", "origin\n", "origin"},
		{"multiple", "origin\nupstream\n", "origin"},
		{"none", "", ""},
		{"none whitespace", "  \n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newStubRunner([]stubResponse{{out: []byte(tt.out)}})
			withRunner(t, r)

			got, err := FirstRemote()
			if err != nil {
				t.Fatalf("FirstRemote: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerifyRefExists(t *testing.T) {
	r, calls := newStubRunner([]stubResponse{{out: []byte("abc\n")}})
	withRunner(t, r)

	ok, err := VerifyRef("feature/x")
	if err != nil {
		t.Fatalf("VerifyRef: %v", err)
	}
	if !ok {
		t.Errorf("expected true")
	}
	if (*calls)[0][2] != "feature/x" {
		t.Errorf("unexpected args: %v", *calls)
	}
}

func TestVerifyRefMissing(t *testing.T) {
	// Simulate exec.ExitError by running a real failing git command. We
	// allow the test to skip if git is unavailable to keep the package
	// hermetic on minimal builders.
	cmd := exec.Command("git", "rev-parse", "--verify", "this-ref-does-not-exist-xyz")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Skipf("git not available: %v", err)
		}
		r := func(args ...string) ([]byte, error) { return nil, exitErr }
		withRunner(t, r)

		ok, vErr := VerifyRef("missing")
		if vErr != nil {
			t.Fatalf("VerifyRef should swallow ExitError: %v", vErr)
		}
		if ok {
			t.Errorf("expected false for missing ref")
		}
	}
}

func TestVerifyRefOtherError(t *testing.T) {
	r, _ := newStubRunner([]stubResponse{{err: errors.New("io failure")}})
	withRunner(t, r)

	ok, err := VerifyRef("foo")
	if err == nil {
		t.Fatal("expected error to bubble up")
	}
	if ok {
		t.Error("expected false on error")
	}
}

func TestSetRunnerNilRestoresDefault(t *testing.T) {
	// Replace with a stub.
	stub := func(_ ...string) ([]byte, error) { return []byte("stub"), nil }
	prev := SetRunner(stub)
	defer SetRunner(prev)

	// Now reset to default by passing nil; subsequent run() should call exec.
	_ = SetRunner(nil)

	runnerMu.RLock()
	cur := runner
	runnerMu.RUnlock()

	// Compare function pointers via reflect.
	if reflect.ValueOf(cur).Pointer() != reflect.ValueOf(Runner(defaultRunner)).Pointer() {
		t.Errorf("SetRunner(nil) did not restore default runner")
	}
}
