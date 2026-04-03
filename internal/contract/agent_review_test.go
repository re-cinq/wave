package contract

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
)

// mockAdapterRunner records calls and returns preset results.
type mockAdapterRunner struct {
	stdout    string
	exitCode  int
	tokens    int
	returnErr error
}

func (m *mockAdapterRunner) Run(_ context.Context, _ adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	return &adapter.AdapterResult{
		ExitCode:   m.exitCode,
		Stdout:     strings.NewReader(m.stdout),
		TokensUsed: m.tokens,
	}, nil
}

// --- parseReviewFeedback tests (T015) ---

func TestParseReviewFeedback(t *testing.T) {
	tests := []struct {
		name        string
		stdout      string
		wantVerdict string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid pass JSON",
			stdout: `{"verdict":"pass","issues":[],"suggestions":[],"confidence":0.95,"summary":"looks good"}`,
			wantVerdict: "pass",
		},
		{
			name: "valid fail JSON with issues",
			stdout: `{"verdict":"fail","issues":[{"severity":"major","description":"Missing tests"}],"suggestions":["Add unit tests"],"confidence":0.8,"summary":"implementation is incomplete"}`,
			wantVerdict: "fail",
		},
		{
			name: "valid warn JSON",
			stdout: `{"verdict":"warn","issues":[{"severity":"minor","description":"Style issue"}],"suggestions":[],"confidence":0.7,"summary":"minor issues found"}`,
			wantVerdict: "warn",
		},
		{
			name: "JSON in markdown fences",
			stdout: "```json\n{\"verdict\":\"pass\",\"issues\":[],\"suggestions\":[],\"confidence\":0.9,\"summary\":\"ok\"}\n```",
			wantVerdict: "pass",
		},
		{
			name:        "empty output",
			stdout:      "",
			wantErr:     true,
			errContains: "no output",
		},
		{
			name:        "invalid verdict enum",
			stdout:      `{"verdict":"unknown","issues":[],"suggestions":[],"confidence":0.5,"summary":"x"}`,
			wantErr:     true,
			errContains: "invalid verdict",
		},
		{
			name:        "confidence out of range high",
			stdout:      `{"verdict":"pass","issues":[],"suggestions":[],"confidence":1.5,"summary":"x"}`,
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "confidence out of range low",
			stdout:      `{"verdict":"pass","issues":[],"suggestions":[],"confidence":-0.1,"summary":"x"}`,
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "completely unparseable output",
			stdout:      "This is not JSON at all, just plain text from the reviewer.",
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name:        "missing required fields",
			stdout:      `{"verdict":"pass"}`,
			wantVerdict: "pass", // partial OK — missing fields zero-value
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fb, err := parseReviewFeedback(tc.stdout)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
				return
			}
			if fb.Verdict != tc.wantVerdict {
				t.Errorf("verdict: got %q, want %q", fb.Verdict, tc.wantVerdict)
			}
		})
	}
}

// --- buildReviewPrompt tests (T016) ---

func TestBuildReviewPrompt(t *testing.T) {
	t.Run("includes criteria", func(t *testing.T) {
		prompt := buildReviewPrompt("Check for missing tests", "")
		if !strings.Contains(prompt, "Check for missing tests") {
			t.Errorf("prompt missing criteria content")
		}
	})

	t.Run("includes schema format", func(t *testing.T) {
		prompt := buildReviewPrompt("criteria", "")
		maxLen := 200
		if len(prompt) < maxLen {
			maxLen = len(prompt)
		}
		if !strings.Contains(prompt, "ReviewFeedback") && !strings.Contains(prompt, "verdict") {
			t.Errorf("prompt missing schema format, got: %s", prompt[:maxLen])
		}
	})

	t.Run("includes context when provided", func(t *testing.T) {
		prompt := buildReviewPrompt("criteria", "Some diff context here")
		if !strings.Contains(prompt, "Some diff context here") {
			t.Errorf("prompt missing context")
		}
	})

	t.Run("no context section when empty", func(t *testing.T) {
		prompt := buildReviewPrompt("criteria", "")
		if strings.Contains(prompt, "## Context") {
			t.Errorf("prompt should not include Context section when context is empty")
		}
	})
}


// --- agentReviewValidator.RunReview tests (T017) ---

func TestAgentReviewValidator_RunReview(t *testing.T) {
	// Create temp criteria file
	dir := t.TempDir()
	criteriaFile := filepath.Join(dir, "criteria.md")
	if err := os.WriteFile(criteriaFile, []byte("# Review Criteria\nCheck correctness."), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("pass verdict from runner", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"pass","issues":[],"suggestions":[],"confidence":0.95,"summary":"ok"}`,
			tokens: 100,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
		}
		fb, _ := v.RunReview(cfg, dir)
		if fb.Verdict != "pass" {
			t.Errorf("expected pass, got %q", fb.Verdict)
		}
	})

	t.Run("fail verdict from runner", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"fail","issues":[{"severity":"critical","description":"No-op implementation"}],"suggestions":[],"confidence":0.9,"summary":"no actual changes"}`,
			tokens: 200,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
		}
		fb, _ := v.RunReview(cfg, dir)
		if fb.Verdict != "fail" {
			t.Errorf("expected fail, got %q", fb.Verdict)
		}
		if len(fb.Issues) == 0 {
			t.Error("expected at least one issue")
		}
	})

	t.Run("runner error is propagated", func(t *testing.T) {
		runner := &mockAdapterRunner{
			returnErr: context.DeadlineExceeded,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
		}
		_, err := v.RunReview(cfg, dir)
		if err == nil {
			t.Fatal("expected error from runner failure")
		}
	})

	t.Run("stdout parse failure", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: "I cannot provide a review at this time.",
			tokens: 50,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
		}
		_, err := v.RunReview(cfg, dir)
		if err == nil {
			t.Fatal("expected error for unparseable stdout")
		}
	})

	t.Run("warn verdict", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"warn","issues":[{"severity":"minor","description":"Nit"}],"suggestions":["Improve naming"],"confidence":0.75,"summary":"minor issues"}`,
			tokens: 150,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
		}
		fb, _ := v.RunReview(cfg, dir)
		if fb.Verdict != "warn" {
			t.Errorf("expected warn, got %q", fb.Verdict)
		}
	})
}

// --- Context assembly tests (T042) ---

func TestAssembleContext(t *testing.T) {
	dir := t.TempDir()

	t.Run("empty sources returns empty string", func(t *testing.T) {
		result := assembleContext(nil, nil, dir)
		if result != "" {
			t.Errorf("expected empty, got %q", result)
		}
	})

	t.Run("artifact found", func(t *testing.T) {
		artFile := filepath.Join(dir, "assessment.json")
		if err := os.WriteFile(artFile, []byte(`{"status":"ok"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		sources := []ReviewContextSource{{Source: "artifact", Artifact: "assessment"}}
		paths := map[string]string{"assessment": artFile}
		result := assembleContext(sources, paths, dir)
		if !strings.Contains(result, `{"status":"ok"}`) {
			t.Errorf("result missing artifact content: %q", result)
		}
	})

	t.Run("artifact missing emits warning in context", func(t *testing.T) {
		sources := []ReviewContextSource{{Source: "artifact", Artifact: "missing-art"}}
		result := assembleContext(sources, nil, dir)
		if !strings.Contains(result, "not found") {
			t.Errorf("expected 'not found' notice, got: %q", result)
		}
	})

	t.Run("multiple sources concatenated", func(t *testing.T) {
		art1 := filepath.Join(dir, "art1.txt")
		if err := os.WriteFile(art1, []byte("content1"), 0o644); err != nil {
			t.Fatal(err)
		}
		art2 := filepath.Join(dir, "art2.txt")
		if err := os.WriteFile(art2, []byte("content2"), 0o644); err != nil {
			t.Fatal(err)
		}
		sources := []ReviewContextSource{
			{Source: "artifact", Artifact: "art1"},
			{Source: "artifact", Artifact: "art2"},
		}
		paths := map[string]string{"art1": art1, "art2": art2}
		result := assembleContext(sources, paths, dir)
		if !strings.Contains(result, "content1") || !strings.Contains(result, "content2") {
			t.Errorf("expected both artifacts in result: %q", result)
		}
	})
}

// --- Token budget tests (T043) ---

func TestTokenBudget(t *testing.T) {
	dir := t.TempDir()
	criteriaFile := filepath.Join(dir, "criteria.md")
	if err := os.WriteFile(criteriaFile, []byte("criteria"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("within budget passes", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"pass","issues":[],"suggestions":[],"confidence":0.9,"summary":"ok"}`,
			tokens: 100,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
			TokenBudget:  500,
		}
		_, _ = v.RunReview(cfg, dir)
	})

	t.Run("over budget emits warning but still parses feedback", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"pass","issues":[],"suggestions":[],"confidence":0.9,"summary":"ok"}`,
			tokens: 1000,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
			TokenBudget:  500,
		}
		feedback, err := v.RunReview(cfg, dir)
		if err != nil {
			t.Fatalf("expected no error (budget overrun is a warning), got: %v", err)
		}
		if feedback == nil {
			t.Fatal("expected feedback to be returned despite budget overrun")
		}
		if feedback.Verdict != "pass" {
			t.Errorf("expected verdict 'pass', got %q", feedback.Verdict)
		}
	})

	t.Run("no budget set (unlimited) passes", func(t *testing.T) {
		runner := &mockAdapterRunner{
			stdout: `{"verdict":"pass","issues":[],"suggestions":[],"confidence":0.9,"summary":"ok"}`,
			tokens: 999999,
		}
		v := newAgentReviewValidator(runner, nil)
		cfg := ContractConfig{
			Type:         "agent_review",
			CriteriaPath: criteriaFile,
			Persona:      "navigator",
			TokenBudget:  0, // unlimited
		}
		_, _ = v.RunReview(cfg, dir)
	})
}

// --- Truncation tests ---

func TestTruncateContent(t *testing.T) {
	t.Run("content within limit not truncated", func(t *testing.T) {
		content := strings.Repeat("a", 1000)
		result := truncateContent(content, 2000)
		if result != content {
			t.Errorf("content should not be truncated")
		}
	})

	t.Run("content over limit truncated with notice", func(t *testing.T) {
		content := strings.Repeat("b", 100)
		result := truncateContent(content, 50)
		if len(result) <= 50 && !strings.Contains(result, "truncated") {
			t.Errorf("expected truncation notice, got: %q", result)
		}
		if !strings.Contains(result, "truncated") {
			t.Errorf("truncation notice missing: %q", result)
		}
	})

	t.Run("zero limit uses default 50KB", func(t *testing.T) {
		content := strings.Repeat("c", defaultContextMaxSize+100)
		result := truncateContent(content, 0)
		if !strings.Contains(result, "truncated") {
			t.Errorf("expected truncation with default limit")
		}
	})
}
