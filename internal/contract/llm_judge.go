package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// anthropicAPIURL is the default Anthropic Messages API endpoint.
// Tests override this via the package-level variable.
var anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// judgeHTTPClient is used for all LLM judge API calls instead of
// http.DefaultClient so that callers sharing the process cannot observe
// or mutate transport-level settings (timeouts, cookies, redirects).
var judgeHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}

type llmJudgeValidator struct{}

// CriterionResult holds the evaluation result for a single criterion.
type CriterionResult struct {
	Criterion string `json:"criterion"`
	Pass      bool   `json:"pass"`
	Reasoning string `json:"reasoning"`
}

// JudgeResponse is the structured response from the judge LLM.
type JudgeResponse struct {
	CriteriaResults []CriterionResult `json:"criteria_results"`
	OverallPass     bool              `json:"overall_pass"`
	Score           float64           `json:"score"`
	Summary         string            `json:"summary"`
}

func (v *llmJudgeValidator) Validate(cfg ContractConfig, workspacePath string) error {
	if len(cfg.Criteria) == 0 {
		return &ValidationError{
			ContractType: "llm_judge",
			Message:      "no evaluation criteria provided",
			Details:      []string{"specify at least one criterion in the 'criteria' field"},
			Retryable:    false,
		}
	}

	// Read step output
	content, err := v.readStepOutput(cfg, workspacePath)
	if err != nil {
		return err
	}

	// Build prompts
	systemPrompt := v.buildSystemPrompt()
	userPrompt := v.buildUserPrompt(cfg.Criteria, content)

	model := cfg.Model
	if model == "" {
		model = "claude-haiku"
	}

	// Try API key first, fall back to Claude CLI for OAuth environments
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	var judgeResp *JudgeResponse
	if apiKey != "" {
		judgeResp, err = v.callAPI(apiKey, model, systemPrompt, userPrompt)
	} else {
		judgeResp, err = v.callViaCLI(model, systemPrompt, userPrompt)
	}
	if err != nil {
		return err
	}

	// Evaluate threshold
	return v.evaluateThreshold(cfg, judgeResp)
}

func (v *llmJudgeValidator) readStepOutput(cfg ContractConfig, workspacePath string) (string, error) {
	sourcePath := cfg.Source
	if sourcePath == "" {
		sourcePath = ".wave/artifact.json"
	}
	fullPath := filepath.Join(workspacePath, sourcePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", &ValidationError{
			ContractType: "llm_judge",
			Message:      fmt.Sprintf("failed to read step output: %s", fullPath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}
	return string(data), nil
}

func (v *llmJudgeValidator) buildSystemPrompt() string {
	return `You are an objective code quality judge. Evaluate the provided content against each criterion independently.

Return your assessment as a JSON object with this exact structure:
{
  "criteria_results": [
    {"criterion": "<criterion text>", "pass": true/false, "reasoning": "<brief explanation>"}
  ],
  "overall_pass": true/false,
  "score": 0.0-1.0,
  "summary": "<one sentence summary>"
}

Rules:
- Evaluate each criterion independently
- Be objective and specific in reasoning
- Set score = (number of passing criteria) / (total criteria)
- Set overall_pass = true if all criteria pass
- Return ONLY the JSON object, no other text`
}

func (v *llmJudgeValidator) buildUserPrompt(criteria []string, content string) string {
	var b strings.Builder
	b.WriteString("Evaluate the following content against these criteria:\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n--- Content to evaluate ---\n\n")
	// Truncate very large content to avoid token limits.
	// Use a valid UTF-8 boundary to avoid splitting multi-byte runes.
	if len(content) > 50000 {
		truncated := content[:50000]
		for !utf8.ValidString(truncated) && len(truncated) > 0 {
			truncated = truncated[:len(truncated)-1]
		}
		b.WriteString(truncated)
		b.WriteString("\n\n[... truncated ...]")
	} else {
		b.WriteString(content)
	}
	return b.String()
}

func (v *llmJudgeValidator) callAPI(apiKey, model, systemPrompt, userPrompt string) (*JudgeResponse, error) {
	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to marshal API request",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to create API request",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := judgeHTTPClient.Do(req)
	if err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "API request failed",
			Details:      []string{err.Error()},
			Retryable:    true,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to read API response",
			Details:      []string{err.Error()},
			Retryable:    true,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      fmt.Sprintf("API returned status %d", resp.StatusCode),
			Details:      []string{string(respBody)},
			Retryable:    true,
		}
	}

	// Parse Anthropic response to extract text content
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to parse API response",
			Details:      []string{err.Error(), string(respBody)},
			Retryable:    true,
		}
	}

	// Find the text content block
	var textContent string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			textContent = block.Text
			break
		}
	}

	if textContent == "" {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "API response contained no text content",
			Details:      []string{string(respBody)},
			Retryable:    true,
		}
	}

	// Parse the judge response JSON
	var judgeResp JudgeResponse
	// Try cleaning the JSON in case of markdown fences
	cleaned := extractJSON(textContent)
	if err := json.Unmarshal([]byte(cleaned), &judgeResp); err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to parse judge response",
			Details:      []string{err.Error(), textContent},
			Retryable:    true,
		}
	}

	return &judgeResp, nil
}

// callViaCLI invokes the Claude CLI for environments without ANTHROPIC_API_KEY
// (e.g., OAuth-authenticated Claude Code). This allows llm_judge to work in
// sandbox environments where only the adapter binary has auth.
func (v *llmJudgeValidator) callViaCLI(model, systemPrompt, userPrompt string) (*JudgeResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	prompt := systemPrompt + "\n\n" + userPrompt

	args := []string{"--print", "--output-format", "text", "--model", model, "--prompt", prompt}
	cmd := exec.CommandContext(ctx, "claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      fmt.Sprintf("claude CLI failed: %v", err),
			Details:      []string{stderr.String()},
			Retryable:    true,
		}
	}

	textContent := stdout.String()
	cleaned := extractJSON(textContent)
	var judgeResp JudgeResponse
	if err := json.Unmarshal([]byte(cleaned), &judgeResp); err != nil {
		return nil, &ValidationError{
			ContractType: "llm_judge",
			Message:      "failed to parse judge response from CLI",
			Details:      []string{err.Error(), textContent},
			Retryable:    true,
		}
	}
	return &judgeResp, nil
}

// extractJSON strips markdown code fences and leading/trailing whitespace
// to extract raw JSON from an LLM response.
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	// Strip ```json ... ``` fences
	if strings.HasPrefix(text, "```") {
		lines := strings.SplitN(text, "\n", 2)
		if len(lines) == 2 {
			text = lines[1]
		}
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
	}
	text = strings.TrimSpace(text)
	// Handle LLM preamble: locate the first '{' and last '}'
	if firstBrace := strings.Index(text, "{"); firstBrace >= 0 {
		if lastBrace := strings.LastIndex(text, "}"); lastBrace > firstBrace {
			text = text[firstBrace : lastBrace+1]
		}
	}
	return strings.TrimSpace(text)
}

func (v *llmJudgeValidator) evaluateThreshold(cfg ContractConfig, resp *JudgeResponse) error {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = 1.0
	}

	// Calculate score from criteria results
	if len(resp.CriteriaResults) == 0 {
		return &ValidationError{
			ContractType: "llm_judge",
			Message:      "judge returned no criteria results",
			Retryable:    true,
		}
	}

	passed := 0
	for _, cr := range resp.CriteriaResults {
		if cr.Pass {
			passed++
		}
	}
	score := float64(passed) / float64(len(resp.CriteriaResults))

	if score >= threshold {
		return nil
	}

	// Build failure details with per-criterion reasoning
	details := make([]string, 0, len(resp.CriteriaResults)+2)
	details = append(details, fmt.Sprintf("Score: %.0f%% (threshold: %.0f%%)", score*100, threshold*100))
	details = append(details, fmt.Sprintf("Summary: %s", resp.Summary))
	for _, cr := range resp.CriteriaResults {
		status := "PASS"
		if !cr.Pass {
			status = "FAIL"
		}
		details = append(details, fmt.Sprintf("[%s] %s: %s", status, cr.Criterion, cr.Reasoning))
	}

	return &ValidationError{
		ContractType: "llm_judge",
		Message:      fmt.Sprintf("LLM judge score %.0f%% is below threshold %.0f%%", score*100, threshold*100),
		Details:      details,
		Retryable:    true,
	}
}
