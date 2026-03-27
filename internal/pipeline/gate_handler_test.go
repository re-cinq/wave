package pipeline

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// AutoApproveHandler tests
// ---------------------------------------------------------------------------

func TestAutoApproveHandler_ReturnsDefaultChoice(t *testing.T) {
	handler := &AutoApproveHandler{}
	gate := &GateConfig{
		Default: "a",
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
			{Key: "r", Label: "Reject"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Choice != "a" {
		t.Errorf("Choice = %q, want %q", decision.Choice, "a")
	}
	if decision.Label != "Approve" {
		t.Errorf("Label = %q, want %q", decision.Label, "Approve")
	}
	if decision.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAutoApproveHandler_ErrorWhenNoDefault(t *testing.T) {
	handler := &AutoApproveHandler{}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	_, err := handler.Prompt(context.Background(), gate)
	if err == nil {
		t.Fatal("expected error when no default is set")
	}
	if !strings.Contains(err.Error(), "default choice") {
		t.Errorf("error should mention default choice, got: %v", err)
	}
}

func TestAutoApproveHandler_ErrorWhenDefaultKeyNotFound(t *testing.T) {
	handler := &AutoApproveHandler{}
	gate := &GateConfig{
		Default: "x",
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
			{Key: "r", Label: "Reject"},
		},
	}

	_, err := handler.Prompt(context.Background(), gate)
	if err == nil {
		t.Fatal("expected error when default key doesn't match any choice")
	}
	if !strings.Contains(err.Error(), "x") {
		t.Errorf("error should mention the missing key %q, got: %v", "x", err)
	}
}

func TestAutoApproveHandler_ReturnsCorrectTarget(t *testing.T) {
	handler := &AutoApproveHandler{}
	gate := &GateConfig{
		Default: "a",
		Choices: []GateChoice{
			{Key: "a", Label: "Approve", Target: "implement"},
			{Key: "r", Label: "Reject", Target: "_fail"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Target != "implement" {
		t.Errorf("Target = %q, want %q", decision.Target, "implement")
	}
}

// ---------------------------------------------------------------------------
// CLIGateHandler tests
// ---------------------------------------------------------------------------

func TestCLIGateHandler_ReadsValidChoice(t *testing.T) {
	in := strings.NewReader("a\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Prompt: "Approve this step?",
		Choices: []GateChoice{
			{Key: "a", Label: "Approve", Target: "next-step"},
			{Key: "r", Label: "Reject", Target: "_fail"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Choice != "a" {
		t.Errorf("Choice = %q, want %q", decision.Choice, "a")
	}
	if decision.Label != "Approve" {
		t.Errorf("Label = %q, want %q", decision.Label, "Approve")
	}
	if decision.Target != "next-step" {
		t.Errorf("Target = %q, want %q", decision.Target, "next-step")
	}
	if decision.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Verify prompt was displayed
	output := out.String()
	if !strings.Contains(output, "Approve this step?") {
		t.Errorf("output should contain prompt text, got: %q", output)
	}
	if !strings.Contains(output, "[a] Approve") {
		t.Errorf("output should display choice [a], got: %q", output)
	}
	if !strings.Contains(output, "[r] Reject") {
		t.Errorf("output should display choice [r], got: %q", output)
	}
}

func TestCLIGateHandler_RejectsInvalidThenAcceptsValid(t *testing.T) {
	// First line is invalid, second is valid
	in := strings.NewReader("x\na\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
			{Key: "r", Label: "Reject"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Choice != "a" {
		t.Errorf("Choice = %q, want %q", decision.Choice, "a")
	}

	output := out.String()
	if !strings.Contains(output, "Invalid choice") {
		t.Errorf("output should contain invalid choice message, got: %q", output)
	}
}

func TestCLIGateHandler_FreeformText(t *testing.T) {
	// First line: choice, second line: freeform text
	in := strings.NewReader("a\nPlease fix the typo in main.go\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Freeform: true,
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Choice != "a" {
		t.Errorf("Choice = %q, want %q", decision.Choice, "a")
	}
	if decision.Text != "Please fix the typo in main.go" {
		t.Errorf("Text = %q, want %q", decision.Text, "Please fix the typo in main.go")
	}

	output := out.String()
	if !strings.Contains(output, "Additional notes") {
		t.Errorf("output should contain freeform prompt, got: %q", output)
	}
}

func TestCLIGateHandler_FreeformEmptySkipsText(t *testing.T) {
	// First line: choice, second line: empty (user presses Enter to skip)
	in := strings.NewReader("a\n\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Freeform: true,
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Text != "" {
		t.Errorf("Text = %q, want empty string for skipped freeform", decision.Text)
	}
}

func TestCLIGateHandler_UnexpectedEndOfInput(t *testing.T) {
	// Empty reader — no input at all
	in := strings.NewReader("")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	_, err := handler.Prompt(context.Background(), gate)
	if err == nil {
		t.Fatal("expected error on unexpected end of input")
	}
	if !strings.Contains(err.Error(), "unexpected end of input") {
		t.Errorf("error should mention unexpected end of input, got: %v", err)
	}
}

func TestCLIGateHandler_ContextCancellation(t *testing.T) {
	// Use a pipe that blocks on read — cancel context to unblock
	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()

	out := &bytes.Buffer{}
	handler := &CLIGateHandler{In: pr, Out: out}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// The scanner.Scan() blocks on the pipe, but context cancellation is checked
	// before each scan attempt. We cancel immediately and check in a goroutine.
	done := make(chan error, 1)
	go func() {
		_, err := handler.Prompt(ctx, gate)
		done <- err
	}()

	// Give the goroutine a moment to start, then cancel
	time.Sleep(20 * time.Millisecond)
	cancel()
	// Unblock the scan by closing the write end
	pw.Close()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from context cancellation or closed pipe")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Prompt to return after context cancel")
	}
}

func TestCLIGateHandler_DisplaysMessageWhenNoPrompt(t *testing.T) {
	in := strings.NewReader("a\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Message: "Review the plan before proceeding",
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	_, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Review the plan before proceeding") {
		t.Errorf("output should contain message text, got: %q", output)
	}
}

func TestCLIGateHandler_DisplaysAbortSuffix(t *testing.T) {
	in := strings.NewReader("a\n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
			{Key: "x", Label: "Abort", Target: "_fail"},
		},
	}

	_, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "(abort)") {
		t.Errorf("output should contain (abort) suffix for _fail target, got: %q", output)
	}
}

func TestCLIGateHandler_WhitespaceTrimming(t *testing.T) {
	in := strings.NewReader("  a  \n")
	out := &bytes.Buffer{}

	handler := &CLIGateHandler{In: in, Out: out}
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve"},
		},
	}

	decision, err := handler.Prompt(context.Background(), gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Choice != "a" {
		t.Errorf("Choice = %q, want %q (whitespace should be trimmed)", decision.Choice, "a")
	}
}

// ---------------------------------------------------------------------------
// GateConfig.Validate tests
// ---------------------------------------------------------------------------

func TestGateConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		gate    GateConfig
		stepIDs map[string]bool
		wantErr string // empty means no error expected
	}{
		{
			name: "valid config with unique keys",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: "Approve", Target: "implement"},
					{Key: "r", Label: "Reject", Target: "_fail"},
				},
			},
			stepIDs: map[string]bool{"implement": true},
			wantErr: "",
		},
		{
			name: "duplicate keys fail",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: "Approve"},
					{Key: "a", Label: "Also approve"},
				},
			},
			stepIDs: nil,
			wantErr: "duplicate key",
		},
		{
			name: "missing key fails",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "", Label: "Approve"},
				},
			},
			stepIDs: nil,
			wantErr: "key is required",
		},
		{
			name: "missing label fails",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: ""},
				},
			},
			stepIDs: nil,
			wantErr: "label is required",
		},
		{
			name: "invalid target step fails",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: "Approve", Target: "nonexistent-step"},
				},
			},
			stepIDs: map[string]bool{"implement": true},
			wantErr: "not a valid step ID",
		},
		{
			name: "target _fail is always valid",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "x", Label: "Abort", Target: "_fail"},
				},
			},
			stepIDs: map[string]bool{},
			wantErr: "",
		},
		{
			name: "target with nil stepIDs skips validation",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: "Approve", Target: "any-step"},
				},
			},
			stepIDs: nil,
			wantErr: "",
		},
		{
			name: "default referencing nonexistent key fails",
			gate: GateConfig{
				Default: "z",
				Choices: []GateChoice{
					{Key: "a", Label: "Approve"},
					{Key: "r", Label: "Reject"},
				},
			},
			stepIDs: nil,
			wantErr: "does not match any choice key",
		},
		{
			name: "valid default key passes",
			gate: GateConfig{
				Default: "a",
				Choices: []GateChoice{
					{Key: "a", Label: "Approve"},
					{Key: "r", Label: "Reject"},
				},
			},
			stepIDs: nil,
			wantErr: "",
		},
		{
			name:    "no choices passes (legacy gate)",
			gate:    GateConfig{Type: "timer"},
			stepIDs: nil,
			wantErr: "",
		},
		{
			name: "empty default with choices is valid",
			gate: GateConfig{
				Choices: []GateChoice{
					{Key: "a", Label: "Approve"},
				},
			},
			stepIDs: nil,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gate.Validate(tt.stepIDs)

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GateConfig.FindChoiceByKey tests
// ---------------------------------------------------------------------------

func TestGateConfig_FindChoiceByKey(t *testing.T) {
	gate := &GateConfig{
		Choices: []GateChoice{
			{Key: "a", Label: "Approve", Target: "implement"},
			{Key: "r", Label: "Reject", Target: "_fail"},
			{Key: "e", Label: "Edit", Target: "revise"},
		},
	}

	tests := []struct {
		name      string
		key       string
		wantLabel string
		wantNil   bool
	}{
		{
			name:      "finds first key",
			key:       "a",
			wantLabel: "Approve",
		},
		{
			name:      "finds middle key",
			key:       "r",
			wantLabel: "Reject",
		},
		{
			name:      "finds last key",
			key:       "e",
			wantLabel: "Edit",
		},
		{
			name:    "returns nil for unknown key",
			key:     "z",
			wantNil: true,
		},
		{
			name:    "returns nil for empty key",
			key:     "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice := gate.FindChoiceByKey(tt.key)

			if tt.wantNil {
				if choice != nil {
					t.Errorf("expected nil, got choice with key %q", choice.Key)
				}
				return
			}

			if choice == nil {
				t.Fatalf("expected choice with label %q, got nil", tt.wantLabel)
			}
			if choice.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", choice.Label, tt.wantLabel)
			}
		})
	}
}

func TestGateConfig_FindChoiceByKey_EmptyChoices(t *testing.T) {
	gate := &GateConfig{}
	choice := gate.FindChoiceByKey("a")
	if choice != nil {
		t.Errorf("expected nil for empty choices list, got choice with key %q", choice.Key)
	}
}
