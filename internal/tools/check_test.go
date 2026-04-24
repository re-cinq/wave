package tools

import (
	"errors"
	"testing"
)

func TestCheckOnPath_CustomLookPath(t *testing.T) {
	calls := []string{}
	lookup := func(name string) (string, error) {
		calls = append(calls, name)
		switch name {
		case "git", "go":
			return "/usr/bin/" + name, nil
		default:
			return "", errors.New("not found")
		}
	}

	got := CheckOnPath(lookup, []string{"git", "nope", "go"})

	want := []PathResult{
		{Name: "git", Found: true},
		{Name: "nope", Found: false},
		{Name: "go", Found: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if len(calls) != 3 {
		t.Errorf("lookPath called %d times, want 3", len(calls))
	}
}

func TestCheckOnPath_SkipsEmptyNames(t *testing.T) {
	lookup := func(string) (string, error) { return "/x", nil }
	got := CheckOnPath(lookup, []string{"", "tool", ""})
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	if got[0].Name != "tool" || !got[0].Found {
		t.Errorf("unexpected result: %+v", got[0])
	}
}

func TestCheckOnPath_EmptyInput(t *testing.T) {
	got := CheckOnPath(nil, nil)
	if len(got) != 0 {
		t.Errorf("got %d results, want 0", len(got))
	}
}

func TestCheckOnPath_NilLookPathUsesExec(t *testing.T) {
	// Passing nil lookPath exercises the default exec.LookPath branch.
	// Use a name that cannot exist on PATH so the result is deterministic.
	got := CheckOnPath(nil, []string{"wave-does-not-exist-xyzzy"})
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	if got[0].Found {
		t.Errorf("expected Found=false for nonexistent tool, got %+v", got[0])
	}
}

func TestCheckOnPath_PreservesOrder(t *testing.T) {
	lookup := func(string) (string, error) { return "/x", nil }
	names := []string{"c", "a", "b"}
	got := CheckOnPath(lookup, names)
	for i, r := range got {
		if r.Name != names[i] {
			t.Errorf("position %d: got %q want %q", i, r.Name, names[i])
		}
	}
}
