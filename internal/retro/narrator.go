package retro

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/adapter"
)

// Narrator uses an LLM adapter to generate qualitative narrative
// assessments from quantitative pipeline run data.
type Narrator struct {
	runner adapter.AdapterRunner
	model  string
}

// NewNarrator creates a Narrator that uses the given adapter runner and model
// (e.g. "claude-haiku-4-5") to generate retrospective narratives.
func NewNarrator(runner adapter.AdapterRunner, model string) *Narrator {
	return &Narrator{runner: runner, model: model}
}

// Narrate generates a NarrativeData assessment for the given retrospective
// by invoking the LLM with a structured prompt built from quantitative data
// and the original pipeline input.
func (n *Narrator) Narrate(ctx context.Context, retro *Retrospective, input string) (*NarrativeData, error) {
	prompt := buildPrompt(retro, input)

	result, err := n.runner.Run(ctx, adapter.AdapterRunConfig{
		Adapter:       "claude",
		Model:         n.model,
		Prompt:        prompt,
		WorkspacePath: os.TempDir(),
	})
	if err != nil {
		return nil, fmt.Errorf("narrator adapter run failed: %w", err)
	}

	narrative, err := parseNarrative(result.ResultContent)
	if err != nil {
		return nil, fmt.Errorf("narrator failed to parse response: %w", err)
	}

	return narrative, nil
}

// buildPrompt constructs the structured prompt sent to the LLM.
func buildPrompt(retro *Retrospective, input string) string {
	var b strings.Builder

	b.WriteString("You are analyzing a pipeline run. Generate a retrospective narrative as JSON.\n\n")
	fmt.Fprintf(&b, "Pipeline: %s\n", retro.Pipeline)
	fmt.Fprintf(&b, "Input: %s\n", input)
	fmt.Fprintf(&b, "Total Duration: %dms\n", retro.Quantitative.TotalDurationMs)
	b.WriteString("Steps:\n")

	for _, s := range retro.Quantitative.Steps {
		fmt.Fprintf(&b, "- %s: %s (%dms, %d retries, %d tokens)\n",
			s.Name, s.Status, s.DurationMs, s.Retries, s.TokensUsed)
	}

	b.WriteString(`
Respond with ONLY valid JSON matching this schema:
{
  "smoothness": "effortless|smooth|bumpy|struggled|failed",
  "intent": "what the pipeline was trying to accomplish",
  "outcome": "what actually happened",
  "friction_points": [{"type": "retry|timeout|wrong_approach|tool_failure|ambiguity|contract_failure", "step": "step name", "detail": "description"}],
  "learnings": [{"category": "repo|code|workflow|tool", "detail": "description"}],
  "open_items": [{"type": "tech_debt|follow_up|investigation|test_gap", "detail": "description"}]
}`)

	return b.String()
}

// parseNarrative attempts to parse JSON from the LLM response. If direct
// parsing fails, it tries to extract JSON from markdown code blocks.
func parseNarrative(content string) (*NarrativeData, error) {
	content = strings.TrimSpace(content)

	// Try direct parse first.
	var narrative NarrativeData
	if err := json.Unmarshal([]byte(content), &narrative); err == nil {
		return &narrative, nil
	}

	// Try extracting from markdown code blocks (```json ... ``` or ``` ... ```).
	extracted := extractJSONFromCodeBlock(content)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &narrative); err == nil {
			return &narrative, nil
		}
	}

	return nil, fmt.Errorf("response is not valid JSON: %.200s", content)
}

// extractJSONFromCodeBlock attempts to extract JSON content from a markdown
// fenced code block.
func extractJSONFromCodeBlock(s string) string {
	// Look for ```json\n...\n``` or ```\n...\n```
	start := strings.Index(s, "```")
	if start == -1 {
		return ""
	}

	// Skip the opening fence and any language tag on the same line.
	afterFence := s[start+3:]
	newline := strings.Index(afterFence, "\n")
	if newline == -1 {
		return ""
	}
	body := afterFence[newline+1:]

	// Find the closing fence.
	end := strings.Index(body, "```")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(body[:end])
}
