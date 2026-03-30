package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
)

// ReviewContextSource defines a context item provided to the reviewing agent.
type ReviewContextSource struct {
	Source   string `json:"source,omitempty"`   // "git_diff" or "artifact"
	Artifact string `json:"artifact,omitempty"` // Artifact name when source is "artifact"
	MaxSize  int    `json:"maxSize,omitempty"`  // Max bytes; 0 = default (50KB)
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
	var stdout strings.Builder
	if result.Stdout != nil {
		buf := make([]byte, 1<<20) // 1MB cap
		n, _ := result.Stdout.Read(buf)
		stdout.Write(buf[:n])
	}

	feedback, err := parseReviewFeedback(stdout.String())
	if err != nil {
		return nil, err
	}

	// Enforce token budget if set
	if cfg.TokenBudget > 0 && result.TokensUsed > cfg.TokenBudget {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message: fmt.Sprintf("reviewer used %d tokens, exceeding budget of %d",
				result.TokensUsed, cfg.TokenBudget),
			Retryable: false,
		}
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
	data, err := os.ReadFile(criteriaPath)
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
func parseReviewFeedback(stdout string) (*ReviewFeedback, error) {
	if stdout == "" {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      "reviewer produced no output",
			Retryable:    true,
		}
	}

	cleaned := extractJSON(stdout)
	var feedback ReviewFeedback
	if err := json.Unmarshal([]byte(cleaned), &feedback); err != nil {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      "failed to parse ReviewFeedback from reviewer output",
			Details:      []string{err.Error(), stdout},
			Retryable:    true,
		}
	}

	// Validate verdict enum
	switch feedback.Verdict {
	case "pass", "fail", "warn":
		// valid
	default:
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("invalid verdict %q (must be pass, fail, or warn)", feedback.Verdict),
			Details:      []string{stdout},
			Retryable:    true,
		}
	}

	// Validate confidence range
	if feedback.Confidence < 0.0 || feedback.Confidence > 1.0 {
		return nil, &ValidationError{
			ContractType: "agent_review",
			Message:      fmt.Sprintf("confidence %f is out of range [0.0, 1.0]", feedback.Confidence),
			Retryable:    true,
		}
	}

	return &feedback, nil
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
			diff, err := fetchGitDiff(workspacePath, src.MaxSize)
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

// fetchGitDiff runs `git diff HEAD` in the workspace directory.
func fetchGitDiff(workspacePath string, maxSize int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	if len(out) == 0 {
		return "No uncommitted changes detected in workspace.", nil
	}

	return truncateContent(string(out), maxSize), nil
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
