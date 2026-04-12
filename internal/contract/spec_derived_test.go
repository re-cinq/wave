package contract

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
)

// mockSpecDerivedRunner records calls and returns preset results.
type mockSpecDerivedRunner struct {
	stdout    string
	exitCode  int
	tokens    int
	returnErr error
}

func (m *mockSpecDerivedRunner) Run(_ context.Context, _ adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	return &adapter.AdapterResult{
		ExitCode:   m.exitCode,
		Stdout:     strings.NewReader(m.stdout),
		TokensUsed: m.tokens,
	}, nil
}

// --- Config validation tests (Task 3.1) ---

func TestValidateSpecDerivedConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ContractConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "all fields present",
			cfg: ContractConfig{
				Type:               "spec_derived_test",
				SpecArtifact:       "spec.md",
				TestPersona:        "test-author",
				ImplementationStep: "implement",
			},
			wantErr: false,
		},
		{
			name: "missing spec_artifact",
			cfg: ContractConfig{
				Type:               "spec_derived_test",
				TestPersona:        "test-author",
				ImplementationStep: "implement",
			},
			wantErr:     true,
			errContains: "spec_artifact",
		},
		{
			name: "missing test_persona",
			cfg: ContractConfig{
				Type:               "spec_derived_test",
				SpecArtifact:       "spec.md",
				ImplementationStep: "implement",
			},
			wantErr:     true,
			errContains: "test_persona",
		},
		{
			name: "missing implementation_step",
			cfg: ContractConfig{
				Type:               "spec_derived_test",
				SpecArtifact:       "spec.md",
				TestPersona:        "test-author",
			},
			wantErr:     true,
			errContains: "implementation_step",
		},
		{
			name: "all fields missing",
			cfg: ContractConfig{
				Type: "spec_derived_test",
			},
			wantErr:     true,
			errContains: "spec_artifact",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSpecDerivedConfig(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// --- Persona separation tests (Task 3.2) ---

func TestCheckPersonaSeparation(t *testing.T) {
	tests := []struct {
		name               string
		testPersona        string
		implementerPersona string
		wantErr            bool
		errContains        string
	}{
		{
			name:               "different personas accepted",
			testPersona:        "test-author",
			implementerPersona: "implementer",
			wantErr:            false,
		},
		{
			name:               "same persona rejected",
			testPersona:        "developer",
			implementerPersona: "developer",
			wantErr:            true,
			errContains:        "must differ",
		},
		{
			name:               "empty vs non-empty accepted",
			testPersona:        "test-author",
			implementerPersona: "",
			wantErr:            false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPersonaSeparation(tc.testPersona, tc.implementerPersona)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
				// Verify it's a hard error (not retryable)
				if validErr, ok := err.(*ValidationError); ok {
					if validErr.Retryable {
						t.Error("persona separation failure should not be retryable")
					}
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// --- Spec artifact loading tests (Task 3.3) ---

func TestLoadSpecArtifact(t *testing.T) {
	t.Run("valid file loaded", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "spec.md")
		if err := os.WriteFile(specFile, []byte("# Spec Content\nHello"), 0o644); err != nil {
			t.Fatal(err)
		}
		content, err := loadSpecArtifact("spec.md", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(content, "Spec Content") {
			t.Errorf("expected spec content, got: %q", content)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := loadSpecArtifact("nonexistent.md", dir)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "failed to read") {
			t.Errorf("error %q should mention 'failed to read'", err.Error())
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		dir := t.TempDir()
		_, err := loadSpecArtifact("../../../etc/passwd", dir)
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
		if !strings.Contains(err.Error(), "path traversal") {
			t.Errorf("error %q should mention 'path traversal'", err.Error())
		}
	})

	t.Run("empty file returns error", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "empty.md")
		if err := os.WriteFile(specFile, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := loadSpecArtifact("empty.md", dir)
		if err == nil {
			t.Fatal("expected error for empty file")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("error %q should mention 'empty'", err.Error())
		}
	})

	t.Run("absolute path works", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "abs-spec.md")
		if err := os.WriteFile(specFile, []byte("absolute spec"), 0o644); err != nil {
			t.Fatal(err)
		}
		content, err := loadSpecArtifact(specFile, "/unused")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "absolute spec" {
			t.Errorf("expected 'absolute spec', got: %q", content)
		}
	})
}

// --- NewValidator returns nil for spec_derived_test (Task 3.4) ---

func TestNewValidator_SpecDerivedTest(t *testing.T) {
	validator := NewValidator(ContractConfig{Type: "spec_derived_test"})
	if validator != nil {
		t.Error("expected nil validator for spec_derived_test (runner-dependent)")
	}
}

// --- Validate() returns runner-required error (Task 3.5) ---

func TestSpecDerivedValidator_Validate(t *testing.T) {
	v := &specDerivedValidator{}
	err := v.Validate(ContractConfig{}, "")
	if err == nil {
		t.Fatal("expected error from Validate()")
	}
	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if validErr.ContractType != "spec_derived_test" {
		t.Errorf("expected contract type spec_derived_test, got %s", validErr.ContractType)
	}
	if !strings.Contains(validErr.Message, "ValidateSpecDerived") {
		t.Errorf("error should mention ValidateSpecDerived, got: %s", validErr.Message)
	}
	if validErr.Retryable {
		t.Error("should not be retryable")
	}
}

// --- parseSpecDerivedResult tests ---

func TestParseSpecDerivedResult(t *testing.T) {
	tests := []struct {
		name        string
		stdout      string
		wantVerdict string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid pass",
			stdout:      `{"verdict":"pass","tests":[{"name":"test1","pass":true,"description":"checks X","reason":"correct"}],"summary":"all good"}`,
			wantVerdict: "pass",
		},
		{
			name:        "valid fail",
			stdout:      `{"verdict":"fail","tests":[{"name":"test1","pass":false,"description":"checks X","reason":"wrong output"}],"summary":"failed"}`,
			wantVerdict: "fail",
		},
		{
			name:        "JSON in markdown fences",
			stdout:      "```json\n{\"verdict\":\"pass\",\"tests\":[],\"summary\":\"ok\"}\n```",
			wantVerdict: "pass",
		},
		{
			name:        "empty output",
			stdout:      "",
			wantErr:     true,
			errContains: "no output",
		},
		{
			name:        "invalid verdict",
			stdout:      `{"verdict":"unknown","tests":[],"summary":"x"}`,
			wantErr:     true,
			errContains: "invalid verdict",
		},
		{
			name:        "unparseable output",
			stdout:      "This is not JSON at all.",
			wantErr:     true,
			errContains: "failed to parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseSpecDerivedResult(tc.stdout)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Verdict != tc.wantVerdict {
				t.Errorf("verdict: got %q, want %q", result.Verdict, tc.wantVerdict)
			}
		})
	}
}

// --- ValidateSpecDerived integration tests ---

func TestValidateSpecDerived(t *testing.T) {
	dir := t.TempDir()
	specFile := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specFile, []byte("# Feature Spec\nDo something useful."), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("pass verdict from runner", func(t *testing.T) {
		runner := &mockSpecDerivedRunner{
			stdout: `{"verdict":"pass","tests":[{"name":"t1","pass":true,"description":"checks feature","reason":"correct"}],"summary":"all tests pass"}`,
			tokens: 100,
		}
		cfg := ContractConfig{
			Type:               "spec_derived_test",
			SpecArtifact:       "spec.md",
			TestPersona:        "test-author",
			ImplementationStep: "implement",
		}
		result, err := ValidateSpecDerived(cfg, dir, runner, nil, "implementer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Verdict != "pass" {
			t.Errorf("expected pass, got %q", result.Verdict)
		}
	})

	t.Run("fail verdict from runner", func(t *testing.T) {
		runner := &mockSpecDerivedRunner{
			stdout: `{"verdict":"fail","tests":[{"name":"t1","pass":false,"description":"checks feature","reason":"wrong output"}],"summary":"tests failed"}`,
			tokens: 200,
		}
		cfg := ContractConfig{
			Type:               "spec_derived_test",
			SpecArtifact:       "spec.md",
			TestPersona:        "test-author",
			ImplementationStep: "implement",
		}
		result, err := ValidateSpecDerived(cfg, dir, runner, nil, "implementer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Verdict != "fail" {
			t.Errorf("expected fail, got %q", result.Verdict)
		}
	})

	t.Run("runner error propagated", func(t *testing.T) {
		runner := &mockSpecDerivedRunner{
			returnErr: context.DeadlineExceeded,
		}
		cfg := ContractConfig{
			Type:               "spec_derived_test",
			SpecArtifact:       "spec.md",
			TestPersona:        "test-author",
			ImplementationStep: "implement",
		}
		_, err := ValidateSpecDerived(cfg, dir, runner, nil, "implementer")
		if err == nil {
			t.Fatal("expected error from runner failure")
		}
	})

	t.Run("persona separation enforced", func(t *testing.T) {
		runner := &mockSpecDerivedRunner{
			stdout: `{"verdict":"pass","tests":[],"summary":"ok"}`,
		}
		cfg := ContractConfig{
			Type:               "spec_derived_test",
			SpecArtifact:       "spec.md",
			TestPersona:        "same-persona",
			ImplementationStep: "implement",
		}
		_, err := ValidateSpecDerived(cfg, dir, runner, nil, "same-persona")
		if err == nil {
			t.Fatal("expected persona separation error")
		}
		if !strings.Contains(err.Error(), "must differ") {
			t.Errorf("error should mention persona separation: %v", err)
		}
	})

	t.Run("missing config fields rejected", func(t *testing.T) {
		runner := &mockSpecDerivedRunner{}
		cfg := ContractConfig{
			Type: "spec_derived_test",
		}
		_, err := ValidateSpecDerived(cfg, dir, runner, nil, "implementer")
		if err == nil {
			t.Fatal("expected config validation error")
		}
		if !strings.Contains(err.Error(), "missing required") {
			t.Errorf("error should mention missing fields: %v", err)
		}
	})
}

// --- buildSpecDerivedPrompt tests ---

func TestBuildSpecDerivedPrompt(t *testing.T) {
	t.Run("includes spec content", func(t *testing.T) {
		prompt := buildSpecDerivedPrompt("Feature: login flow", "step-1")
		if !strings.Contains(prompt, "Feature: login flow") {
			t.Error("prompt missing spec content")
		}
	})

	t.Run("includes implementation step", func(t *testing.T) {
		prompt := buildSpecDerivedPrompt("spec", "step-impl")
		if !strings.Contains(prompt, "step-impl") {
			t.Error("prompt missing implementation step reference")
		}
	})

	t.Run("includes output schema", func(t *testing.T) {
		prompt := buildSpecDerivedPrompt("spec", "step-1")
		if !strings.Contains(prompt, "verdict") {
			t.Error("prompt missing output schema")
		}
	})
}
