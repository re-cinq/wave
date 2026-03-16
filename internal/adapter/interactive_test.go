package adapter

import (
	"testing"
)

func TestBuildInteractiveArgs_Basic(t *testing.T) {
	args := buildInteractiveArgs(InteractiveOptions{
		Model: "sonnet",
	})

	if !containsArg(args, "--model") {
		t.Error("missing --model flag")
	}
	if !containsArg(args, "sonnet") {
		t.Error("missing model value")
	}
	if !containsArg(args, "--dangerously-skip-permissions") {
		t.Error("missing --dangerously-skip-permissions")
	}
}

func TestBuildInteractiveArgs_Resume(t *testing.T) {
	args := buildInteractiveArgs(InteractiveOptions{
		Resume: "session-abc123",
	})

	if !containsArgPair(args, "--resume", "session-abc123") {
		t.Error("missing or incorrect --resume flag")
	}
}

func TestBuildInteractiveArgs_Prompt(t *testing.T) {
	args := buildInteractiveArgs(InteractiveOptions{
		Prompt: "explain the plan artifact",
	})

	// Prompt should be the last argument (positional)
	last := args[len(args)-1]
	if last != "explain the plan artifact" {
		t.Errorf("expected prompt as last arg, got %q", last)
	}
}

func TestBuildInteractiveArgs_AllOptions(t *testing.T) {
	args := buildInteractiveArgs(InteractiveOptions{
		Model:        "opus",
		AllowedTools: []string{"Read", "Bash"},
		AddDirs:      []string{"/project", "/workspace"},
		SystemPrompt: "You are a helpful assistant",
		Resume:       "session-xyz",
		Prompt:       "what changed?",
	})

	if !containsArgPair(args, "--model", "opus") {
		t.Error("missing --model opus")
	}
	if !containsArg(args, "--allowedTools") {
		t.Error("missing --allowedTools")
	}
	if !containsArgPair(args, "--system-prompt", "You are a helpful assistant") {
		t.Error("missing --system-prompt")
	}
	if !containsArgPair(args, "--resume", "session-xyz") {
		t.Error("missing --resume")
	}

	// Count --add-dir flags
	addDirCount := 0
	for _, a := range args {
		if a == "--add-dir" {
			addDirCount++
		}
	}
	if addDirCount != 2 {
		t.Errorf("expected 2 --add-dir flags, got %d", addDirCount)
	}

	// Prompt should be the last argument
	last := args[len(args)-1]
	if last != "what changed?" {
		t.Errorf("expected prompt as last arg, got %q", last)
	}
}

func TestBuildInteractiveArgs_NoResumeOrPrompt(t *testing.T) {
	args := buildInteractiveArgs(InteractiveOptions{
		Model: "sonnet",
	})

	for _, arg := range args {
		if arg == "--resume" {
			t.Error("should not include --resume when empty")
		}
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "simple session ID",
			input:  "Session: abc123def456",
			expect: "abc123def456",
		},
		{
			name:   "session ID with surrounding output",
			input:  "some output\nSession: 01234567-89ab-cdef\nmore output",
			expect: "01234567-89ab-cdef",
		},
		{
			name:   "no session info",
			input:  "no session info here",
			expect: "",
		},
		{
			name:   "empty string",
			input:  "",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSessionID(tt.input)
			if got != tt.expect {
				t.Errorf("extractSessionID(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

func containsArgPair(args []string, flag, value string) bool {
	for i, a := range args {
		if a == flag && i+1 < len(args) && args[i+1] == value {
			return true
		}
	}
	return false
}
