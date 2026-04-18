package contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newMockAnthropicServer creates an httptest server that mimics the Anthropic Messages API.
// It returns the server and a cleanup function that restores the original API URL.
func newMockAnthropicServer(t *testing.T, judgeResp JudgeResponse) *httptest.Server {
	t.Helper()
	respJSON, err := json.Marshal(judgeResp)
	if err != nil {
		t.Fatalf("failed to marshal judge response: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "missing version", http.StatusBadRequest)
			return
		}

		apiResp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": string(respJSON)},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiResp)
	}))

	return server
}

func setupLLMJudgeTest(t *testing.T, content string) string {
	t.Helper()
	workspacePath := t.TempDir()
	waveDir := filepath.Join(workspacePath, ".agents")
	_ = os.MkdirAll(waveDir, 0755)
	_ = os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(content), 0644)
	return workspacePath
}

func TestLLMJudge_AllCriteriaPass(t *testing.T) {
	judgeResp := JudgeResponse{
		CriteriaResults: []CriterionResult{
			{Criterion: "Code follows patterns", Pass: true, Reasoning: "Consistent style"},
			{Criterion: "No security issues", Pass: true, Reasoning: "No vulnerabilities found"},
			{Criterion: "Error handling", Pass: true, Reasoning: "All errors checked"},
		},
		OverallPass: true,
		Score:       1.0,
		Summary:     "All criteria met",
	}

	server := newMockAnthropicServer(t, judgeResp)
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "good code"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:      "llm_judge",
		Criteria:  []string{"Code follows patterns", "No security issues", "Error handling"},
		Threshold: 0.8,
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}
}

func TestLLMJudge_ThresholdFailure(t *testing.T) {
	judgeResp := JudgeResponse{
		CriteriaResults: []CriterionResult{
			{Criterion: "Code follows patterns", Pass: true, Reasoning: "Good"},
			{Criterion: "No security issues", Pass: false, Reasoning: "SQL injection found"},
			{Criterion: "Error handling", Pass: false, Reasoning: "Errors swallowed"},
			{Criterion: "Single responsibility", Pass: false, Reasoning: "God function"},
		},
		OverallPass: false,
		Score:       0.25,
		Summary:     "Multiple issues found",
	}

	server := newMockAnthropicServer(t, judgeResp)
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "bad code"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:      "llm_judge",
		Criteria:  []string{"Code follows patterns", "No security issues", "Error handling", "Single responsibility"},
		Threshold: 0.8,
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for threshold failure")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if validErr.ContractType != "llm_judge" {
		t.Errorf("expected contract type llm_judge, got %s", validErr.ContractType)
	}

	if !validErr.Retryable {
		t.Error("expected Retryable to be true")
	}

	if !strings.Contains(validErr.Message, "25%") {
		t.Errorf("expected score percentage in message, got: %s", validErr.Message)
	}

	// Check per-criterion details are included
	errStr := validErr.Error()
	if !strings.Contains(errStr, "FAIL") {
		t.Errorf("expected FAIL markers in details, got: %s", errStr)
	}
	if !strings.Contains(errStr, "SQL injection found") {
		t.Errorf("expected criterion reasoning in details, got: %s", errStr)
	}
}

func TestLLMJudge_ThresholdBoundary(t *testing.T) {
	// 2/4 pass = 0.5, threshold = 0.5 — should pass (score >= threshold)
	judgeResp := JudgeResponse{
		CriteriaResults: []CriterionResult{
			{Criterion: "A", Pass: true, Reasoning: "OK"},
			{Criterion: "B", Pass: true, Reasoning: "OK"},
			{Criterion: "C", Pass: false, Reasoning: "Not OK"},
			{Criterion: "D", Pass: false, Reasoning: "Not OK"},
		},
		OverallPass: false,
		Score:       0.5,
		Summary:     "Mixed",
	}

	server := newMockAnthropicServer(t, judgeResp)
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "mixed code"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:      "llm_judge",
		Criteria:  []string{"A", "B", "C", "D"},
		Threshold: 0.5,
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected pass when score equals threshold, got error: %v", err)
	}
}

func TestLLMJudge_DefaultThreshold(t *testing.T) {
	// 2/3 pass = 0.667, default threshold = 1.0 — should fail
	judgeResp := JudgeResponse{
		CriteriaResults: []CriterionResult{
			{Criterion: "A", Pass: true, Reasoning: "OK"},
			{Criterion: "B", Pass: true, Reasoning: "OK"},
			{Criterion: "C", Pass: false, Reasoning: "Nope"},
		},
		OverallPass: false,
		Score:       0.67,
		Summary:     "Almost",
	}

	server := newMockAnthropicServer(t, judgeResp)
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "almost"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:     "llm_judge",
		Criteria: []string{"A", "B", "C"},
		// Threshold intentionally omitted — defaults to 1.0
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected failure when score < default threshold 1.0")
	}
}

func TestLLMJudge_MissingCriteria(t *testing.T) {
	cfg := ContractConfig{
		Type: "llm_judge",
		// No criteria
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing criteria")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if !strings.Contains(validErr.Message, "no evaluation criteria") {
		t.Errorf("expected message about missing criteria, got: %s", validErr.Message)
	}

	if validErr.Retryable {
		t.Error("expected Retryable to be false for config error")
	}
}

func TestLLMJudge_NoAPIKey_FallsThroughToCLI(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0o755)
	_ = os.WriteFile(filepath.Join(tmpDir, ".agents", "artifact.json"), []byte(`{"test": true}`), 0o644)

	cfg := ContractConfig{
		Type:     "llm_judge",
		Criteria: []string{"Test criterion"},
		Source:   ".agents/artifact.json",
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, tmpDir)
	// Without API key, falls through to CLI. CLI likely not available in test.
	if err == nil {
		return // CLI worked (unlikely in test, but valid)
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	// Should NOT be an "API key missing" error — it should be a CLI error
	if strings.Contains(validErr.Message, "ANTHROPIC_API_KEY") {
		t.Errorf("should fall through to CLI, not fail on missing API key, got: %s", validErr.Message)
	}
}

func TestLLMJudge_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": "internal server error"}`)
	}))
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "test"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:     "llm_judge",
		Criteria: []string{"Test criterion"},
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for API failure")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if !strings.Contains(validErr.Message, "500") {
		t.Errorf("expected status code in message, got: %s", validErr.Message)
	}

	if !validErr.Retryable {
		t.Error("expected Retryable to be true for API errors")
	}
}

func TestLLMJudge_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiResp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "This is not valid JSON at all"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiResp)
	}))
	defer server.Close()

	origURL := anthropicAPIURL
	anthropicAPIURL = server.URL
	defer func() { anthropicAPIURL = origURL }()

	workspacePath := setupLLMJudgeTest(t, `{"result": "test"}`)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	cfg := ContractConfig{
		Type:     "llm_judge",
		Criteria: []string{"Test criterion"},
	}

	v := &llmJudgeValidator{}
	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for malformed response")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if !strings.Contains(validErr.Message, "failed to parse judge response") {
		t.Errorf("expected parse error message, got: %s", validErr.Message)
	}

	if !validErr.Retryable {
		t.Error("expected Retryable to be true for malformed response")
	}
}

func TestBuildUserPrompt_UTF8SafeTruncation(t *testing.T) {
	v := &llmJudgeValidator{}

	// Create content that ends with a multi-byte rune at the 50000 byte boundary.
	// U+1F600 (😀) is 4 bytes in UTF-8. Place it so that a naive byte slice at
	// 50000 would split the rune.
	base := strings.Repeat("a", 49999) // 49999 ASCII bytes
	base += "😀"                        // 4 bytes → total 50003 bytes; naive [:50000] splits the emoji

	prompt := v.buildUserPrompt([]string{"criterion"}, base)

	// The truncated portion must be valid UTF-8.
	if strings.Contains(prompt, "\uFFFD") {
		t.Error("truncated prompt contains replacement character, indicating invalid UTF-8")
	}
	if !strings.Contains(prompt, "[... truncated ...]") {
		t.Error("expected truncation marker in prompt")
	}
}

func TestResolveLLMJudgeModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "empty defaults to cheapest model",
			model: "",
			want:  "claude-haiku-4-5",
		},
		{
			name:  "cheapest tier resolves to haiku",
			model: "cheapest",
			want:  "claude-haiku-4-5",
		},
		{
			name:  "strongest tier resolves to opus",
			model: "strongest",
			want:  "claude-opus-4",
		},
		{
			name:  "balanced tier falls back to cheapest",
			model: "balanced",
			want:  "claude-haiku-4-5",
		},
		{
			name:  "literal model name returned unchanged",
			model: "claude-sonnet-4",
			want:  "claude-sonnet-4",
		},
		{
			name:  "unknown tier name returned as-is",
			model: "extreme",
			want:  "extreme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveLLMJudgeModel(tt.model)
			if got != tt.want {
				t.Errorf("resolveLLMJudgeModel(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON in code fence",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON in plain code fence",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with whitespace",
			input: "  \n{\"key\": \"value\"}\n  ",
			want:  `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
