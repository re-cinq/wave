package contract

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
)

// ReviewContextSource defines a context item provided to the reviewing agent.
type ReviewContextSource struct {
	Source   string `json:"source,omitempty"   yaml:"source,omitempty"`    // "git_diff" or "artifact"
	Artifact string `json:"artifact,omitempty" yaml:"artifact,omitempty"`  // Artifact name when source is "artifact"
	MaxSize  int    `json:"maxSize,omitempty"  yaml:"max_size,omitempty"`   // Max bytes; 0 = default (50KB)
	DiffBase string `json:"diffBase,omitempty" yaml:"diff_base,omitempty"` // Git ref to diff against (e.g. "main"); auto-detected if empty
}

// ReviewIssue is a single issue found during an agent review.
type ReviewIssue struct {
	Severity    string `json:"severity"`    // "critical", "major", "minor", "info"
	Description string `json:"description"` // Human-readable description
}

// ReviewFeedback is the structured output of an agent review.
// The reviewing agent is required to produce JSON matching this schema.
type ReviewFeedback struct {
	Verdict     string        `json:"verdict"`     // "pass", "fail", or "warn"
	Issues      []ReviewIssue `json:"issues"`      // Issues found (empty on pass)
	Suggestions []string      `json:"suggestions"` // Optional improvement suggestions
	Confidence  float64       `json:"confidence"`  // 0.0–1.0 reviewer confidence
	Summary     string        `json:"summary"`     // One-sentence summary
}

const defaultContextMaxSize = 50 * 1024 // 50KB

// reviewFeedbackSchema is injected into the reviewer's prompt so the LLM
// knows exactly what JSON structure to produce.
const reviewFeedbackSchema = `{
  "verdict": "pass" | "fail" | "warn",
  "issues": [
    {"severity": "critical" | "major" | "minor" | "info", "description": "<description>"}
  ],
  "suggestions": ["<suggestion>"],
  "confidence": 0.0-1.0,
  "summary": "<one-sentence summary>"
}`

// agentReviewValidator implements ContractValidator for the agent_review type.
type agentReviewValidator struct {
	runner   adapter.AdapterRunner
	manifest interface{} // *manifest.Manifest — stored as interface{} to avoid circular imports
}

// newAgentReviewValidator creates a new agent review validator.
func newAgentReviewValidator(runner adapter.AdapterRunner, manifest interface{}) *agentReviewValidator {
	return &agentReviewValidator{runner: runner, manifest: manifest}
}

// Validate implements ContractValidator. For agent_review, this is a no-op —
// callers must use ValidateWithRunner instead, which provides the runner context.
func (v *agentReviewValidator) Validate(_ ContractConfig, _ string) error {
	return &ValidationError{
		ContractType: "agent_review",
		Message:      "agent_review contracts require an adapter runner — use ValidateWithRunner()",
		Retryable:    false,
	}
}

// RunReview executes the agent review and returns structured feedback.
func (v *agentReviewValidator) RunReview(cfg ContractConfig, workspacePath string) (*ReviewFeedback, error) {
	// Load review criteria from file
	criteria, err := v.loadCriteria(cfg.CriteriaPath)
	if err != nil {
		return nil, err
	}

	// Assemble context from configured sources
	contextText := assembleContext(cfg.Context, cfg.ArtifactPaths, workspacePath)

	// Build reviewer prompt
	prompt := buildReviewPrompt(criteria, contextText)

	// Resolve timeout
	timeout := 120 * time.Second
	if cfg.Timeout != "" {
		d, parseErr := time.ParseDuration(cfg.Timeout)
		if parseErr == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runCfg := adapter.AdapterRunConfig{
		Persona:       cfg.Persona,
		WorkspacePath: workspacePath,
		Prompt:        prompt,
		Model:         cfg.Model,
		Timeout:       timeout,
	}

	result, err := v.runner.Run(ctx, runCfg)
	if err != nil {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("reviewer agent failed: %v", err),
			Retryable:    true,
		}
	}

	// Read stdout
	var stdoutStr string
	if result.Stdout != nil {
		data, err := io.ReadAll(io.LimitReader(result.Stdout, 1<<20)) // 1MB cap
		if err != nil {
			return nil, &ValidationError{
				ContractType: "agent_review",
				Message:      fmt.Sprintf("failed to read reviewer output: %v", err),
				Retryable:    true,
			}
		}
		stdoutStr = string(data)
	}

	// Token budget overrun is informational. Log to stderr so the run log
	// captures it, but never prepend the warning to stdoutStr — that breaks
	// JSON parsing when the reviewer subprocess emits a JSONL stream.
	if cfg.TokenBudget > 0 && result.TokensUsed > cfg.TokenBudget {
		fmt.Fprintf(os.Stderr, "[agent_review] reviewer used %d tokens, exceeding budget of %d\n",
			result.TokensUsed, cfg.TokenBudget)
	}

	feedback, err := parseReviewFeedback(stdoutStr)
	if err != nil {
		return nil, err
	}

	return feedback, nil
}

// loadCriteria reads review criteria from the specified file path.
func (v *agentReviewValidator) loadCriteria(criteriaPath string) (string, error) {
	if criteriaPath == "" {
		return "", &ValidationError{
			ContractType: "agent_review",
			Message:      "criteria_path is required for agent_review contracts",
			Retryable:    false,
		}
	}
	// Path traversal protection: ensure resolved path stays within project directory
	cleanPath := filepath.Clean(criteriaPath)
	if strings.Contains(cleanPath, "..") {
		return "", &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("criteria_path %q contains path traversal", criteriaPath),
			Retryable:    false,
		}
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("failed to read criteria file %q: %v", criteriaPath, err),
			Retryable:    false,
		}
	}
	return string(data), nil
}

// buildReviewPrompt assembles the user prompt for the reviewing agent.
func buildReviewPrompt(criteria, contextText string) string {
	var b strings.Builder
	b.WriteString("## Review Criteria\n\n")
	b.WriteString(criteria)
	b.WriteString("\n\n")
	if contextText != "" {
		b.WriteString("## Context\n\n")
		b.WriteString(contextText)
		b.WriteString("\n\n")
	}
	b.WriteString("## Required Output Format\n\n")
	b.WriteString("You MUST respond with a single JSON object matching this schema:\n\n")
	b.WriteString("```json\n")
	b.WriteString(reviewFeedbackSchema)
	b.WriteString("\n```\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- verdict MUST be exactly \"pass\", \"fail\", or \"warn\"\n")
	b.WriteString("- issues MUST be an array (empty if verdict is pass)\n")
	b.WriteString("- confidence MUST be a float between 0.0 and 1.0\n")
	b.WriteString("- Return ONLY the JSON object, no other text outside the JSON block\n")
	return b.String()
}

// parseReviewFeedback extracts ReviewFeedback from agent stdout.
//
// The reviewer subprocess (Claude Code) emits a JSONL event stream — one JSON
// object per line — with the reviewer's actual verdict surfacing either as the
// `result` field of a `{"type":"result",...}` event, or as a standalone
// `{"verdict":...}` object emitted via a text content block. We try a few
// strategies in order: direct unmarshal, brace-trim via extractJSON, and a
// JSONL-aware scan that returns the last object containing a `verdict` field.
// Validation (verdict enum + confidence range) runs once on the chosen feedback.
func parseReviewFeedback(stdout string) (*ReviewFeedback, error) {
	if stdout == "" {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      "reviewer produced no output",
			Retryable:    true,
		}
	}

	feedback := unmarshalFeedback(stdout)
	if feedback == nil {
		feedback = unmarshalFeedback(extractJSON(stdout))
	}
	if feedback == nil {
		feedback = lastVerdictObjectFromJSONL(stdout)
	}
	if feedback == nil {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      "failed to parse ReviewFeedback from reviewer output",
			Details:      []string{"no JSON object containing a 'verdict' field was found in the reviewer output", stdout},
			Retryable:    true,
		}
	}
	if err := validateFeedback(feedback); err != nil {
		// Decorate validation errors with the original stdout for diagnostics
		if ve, ok := err.(*ValidationError); ok {
			ve.Details = append(ve.Details, stdout)
		}
		return nil, err
	}
	return feedback, nil
}

// unmarshalFeedback attempts to unmarshal text into a ReviewFeedback. Returns
// nil if the text is not valid JSON or does not carry a `verdict` field. Does
// not enforce the verdict enum or confidence range — that is validateFeedback's
// job. Splitting the two lets the caller surface validation errors instead of
// silently rejecting otherwise-parseable feedback.
func unmarshalFeedback(text string) *ReviewFeedback {
	if text == "" {
		return nil
	}
	var feedback ReviewFeedback
	if err := json.Unmarshal([]byte(text), &feedback); err != nil {
		return nil
	}
	if feedback.Verdict == "" {
		return nil
	}
	return &feedback
}

// lastVerdictObjectFromJSONL scans stdout as a JSONL stream and returns the
// last top-level JSON object that carries a `verdict` field. Each line may
// itself be a Claude-Code event whose `result` field is a stringified
// ReviewFeedback — that envelope is unwrapped before checking.
func lastVerdictObjectFromJSONL(stdout string) *ReviewFeedback {
	var last *ReviewFeedback
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if raw, ok := envelope["result"]; ok {
			var inner string
			if err := json.Unmarshal(raw, &inner); err == nil {
				if fb := unmarshalFeedback(extractJSON(inner)); fb != nil {
					last = fb
					continue
				}
			}
		}
		if fb := unmarshalFeedback(line); fb != nil {
			last = fb
		}
	}
	return last
}

// validateFeedback enforces the verdict enum and confidence range invariants.
func validateFeedback(feedback *ReviewFeedback) error {
	switch feedback.Verdict {
	case "pass", "fail", "warn":
	default:
		return &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("invalid verdict %q (must be pass, fail, or warn)", feedback.Verdict),
			Retryable:    true,
		}
	}
	if feedback.Confidence < 0.0 || feedback.Confidence > 1.0 {
		return &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("confidence %f is out of range [0.0, 1.0]", feedback.Confidence),
			Retryable:    true,
		}
	}
	return nil
}

// assembleContext builds the context string for the reviewer from configured sources.
func assembleContext(sources []ReviewContextSource, artifactPaths map[string]string, workspacePath string) string {
	if len(sources) == 0 {
		return ""
	}

	var b strings.Builder
	for _, src := range sources {
		switch src.Source {
		case "git_diff":
			diff, err := fetchGitDiff(workspacePath, src.MaxSize, src.DiffBase)
			if err != nil {
				// Non-fatal — include error notice but continue
				b.WriteString("### Git Diff\n\n")
				b.WriteString(fmt.Sprintf("[Error fetching git diff: %v]\n\n", err))
			} else {
				b.WriteString("### Git Diff\n\n")
				b.WriteString(diff)
				b.WriteString("\n\n")
			}
		case "artifact":
			content, found := resolveArtifact(src.Artifact, artifactPaths)
			if !found {
				b.WriteString(fmt.Sprintf("### Artifact: %s\n\n[Warning: artifact %q not found]\n\n", src.Artifact, src.Artifact))
			} else {
				content = truncateContent(content, src.MaxSize)
				b.WriteString(fmt.Sprintf("### Artifact: %s\n\n", src.Artifact))
				b.WriteString(content)
				b.WriteString("\n\n")
			}
		default:
			// Unknown source type — skip
		}
	}
	return b.String()
}

// fetchGitDiff runs `git diff <base>..HEAD` in the workspace directory.
// When diffBase is empty it auto-detects the merge base by trying origin/main,
// origin/master, main, and master in order. Falls back to git diff HEAD if none resolve.
func fetchGitDiff(workspacePath string, maxSize int, diffBase string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	base := diffBase
	if base == "" {
		base = detectDiffBase(ctx, workspacePath)
	}

	var args []string
	if base != "" {
		args = []string{"diff", base + "..HEAD"}
	} else {
		args = []string{"diff", "HEAD"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	if len(out) == 0 {
		return "No changes detected relative to " + base + ".", nil
	}

	return truncateContent(string(out), maxSize), nil
}

// detectDiffBase tries common remote and local branch names to find a diff base.
// Returns empty string if none resolve.
func detectDiffBase(ctx context.Context, workspacePath string) string {
	candidates := []string{"origin/main", "origin/master", "main", "master"}
	for _, ref := range candidates {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", ref)
		cmd.Dir = workspacePath
		if err := cmd.Run(); err == nil {
			return ref
		}
	}
	return ""
}

// resolveArtifact looks up an artifact by name in the provided paths map.
// It also reads the file content if found.
func resolveArtifact(name string, artifactPaths map[string]string) (string, bool) {
	if artifactPaths == nil {
		return "", false
	}
	path, ok := artifactPaths[name]
	if !ok {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("[Error reading artifact %q: %v]", name, err), true
	}
	return string(data), true
}

// truncateContent truncates content at maxSize bytes with a notice.
// If maxSize <= 0, the default (50KB) is used.
func truncateContent(content string, maxSize int) string {
	limit := maxSize
	if limit <= 0 {
		limit = defaultContextMaxSize
	}
	if len(content) <= limit {
		return content
	}
	return content[:limit] + fmt.Sprintf("\n\n[... truncated at %d bytes ...]", limit)
}

// ValidateWithRunner runs an agent_review contract using the provided adapter runner.
// It dispatches to agentReviewValidator for agent_review type, and falls back to
// Validate() for all other contract types (returning nil ReviewFeedback).
func ValidateWithRunner(cfg ContractConfig, workspacePath string, runner adapter.AdapterRunner, manifest interface{}) (*ReviewFeedback, error) {
	if cfg.Type != "agent_review" {
		return nil, Validate(cfg, workspacePath)
	}
	v := newAgentReviewValidator(runner, manifest)
	return v.RunReview(cfg, workspacePath)
}
