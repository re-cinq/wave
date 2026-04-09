package retro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
)

// Narrator generates LLM-powered narrative analysis from quantitative data.
type Narrator struct {
	runner  adapter.AdapterRunner
	model   string
	timeout time.Duration
}

// NewNarrator creates a new Narrator.
func NewNarrator(runner adapter.AdapterRunner, model string) *Narrator {
	return &Narrator{
		runner:  runner,
		model:   model,
		timeout: 2 * time.Minute,
	}
}

// Narrate generates a narrative from quantitative data.
func (n *Narrator) Narrate(ctx context.Context, runID string, pipeline string, quant *QuantitativeData) (*Narrative, error) {
	prompt := n.buildPrompt(runID, pipeline, quant)

	cfg := adapter.AdapterRunConfig{
		Prompt:  prompt,
		Model:   n.model,
		Timeout: n.timeout,
	}

	result, err := n.runner.Run(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("narrator adapter failed: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("narrator adapter exited with code %d", result.ExitCode)
	}

	// Read output
	var output string
	if result.ResultContent != "" {
		output = result.ResultContent
	} else if result.Stdout != nil {
		data, err := io.ReadAll(result.Stdout)
		if err != nil {
			return nil, fmt.Errorf("failed to read narrator output: %w", err)
		}
		output = string(data)
	}

	narrative, err := parseNarrativeResponse(output)
	if err != nil {
		log.Printf("[retro] warning: failed to parse narrative response, using fallback: %v", err)
		return n.fallbackNarrative(quant), nil
	}

	return narrative, nil
}

// buildPrompt constructs the narrator prompt from quantitative data.
func (n *Narrator) buildPrompt(runID string, pipeline string, quant *QuantitativeData) string {
	var b strings.Builder

	b.WriteString("Analyze this pipeline run and produce a structured retrospective narrative.\n\n")
	b.WriteString(fmt.Sprintf("Run ID: %s\n", runID))
	b.WriteString(fmt.Sprintf("Pipeline: %s\n", pipeline))
	b.WriteString(fmt.Sprintf("Total Duration: %dms\n", quant.TotalDurationMs))
	b.WriteString(fmt.Sprintf("Steps: %d total, %d succeeded, %d failed\n", quant.TotalSteps, quant.SuccessCount, quant.FailureCount))
	b.WriteString(fmt.Sprintf("Total Retries: %d\n", quant.TotalRetries))
	b.WriteString(fmt.Sprintf("Total Tokens: %d\n\n", quant.TotalTokens))

	if len(quant.Steps) > 0 {
		b.WriteString("Step Details:\n")
		for _, s := range quant.Steps {
			b.WriteString(fmt.Sprintf("  - %s: %dms, status=%s, retries=%d, tokens=%d, files=%d\n",
				s.Name, s.DurationMs, s.Status, s.Retries, s.TokensUsed, s.FilesChanged))
		}
		b.WriteString("\n")
	}

	b.WriteString(`Respond with ONLY a JSON object (no markdown, no explanation) with this exact structure:
{
  "smoothness": "<effortless|smooth|bumpy|struggled|failed>",
  "intent": "<one sentence describing what this run was trying to accomplish>",
  "outcome": "<one sentence describing the result>",
  "friction_points": [{"type": "<retry|timeout|wrong_approach|tool_failure|ambiguity|contract_failure|review_rework>", "step": "<step name>", "detail": "<brief description>"}],
  "learnings": [{"category": "<repo|code|workflow|tool>", "detail": "<what was learned>"}],
  "open_items": [{"type": "<tech_debt|follow_up|investigation|test_gap>", "detail": "<what needs attention>"}],
  "recommendations": ["<concrete suggestion for improvement>"]
}`)

	return b.String()
}

// parseNarrativeResponse extracts a Narrative from the LLM's JSON response.
func parseNarrativeResponse(output string) (*Narrative, error) {
	// Try to extract JSON from the response (LLMs sometimes wrap in markdown)
	jsonStr := output
	if idx := strings.Index(output, "{"); idx >= 0 {
		if end := strings.LastIndex(output, "}"); end > idx {
			jsonStr = output[idx : end+1]
		}
	}

	var narrative Narrative
	if err := json.Unmarshal([]byte(jsonStr), &narrative); err != nil {
		return nil, fmt.Errorf("failed to parse narrative JSON: %w", err)
	}

	if !ValidSmoothness(narrative.Smoothness) {
		narrative.Smoothness = SmoothnessBumpy // safe default
	}

	return &narrative, nil
}

// fallbackNarrative generates a basic narrative when LLM parsing fails.
func (n *Narrator) fallbackNarrative(quant *QuantitativeData) *Narrative {
	smoothness := SmoothnessSmooth
	switch {
	case quant.FailureCount > 0:
		smoothness = SmoothnessFailed
	case quant.TotalRetries > 2:
		smoothness = SmoothnessStruggled
	case quant.TotalRetries > 0:
		smoothness = SmoothnessBumpy
	}

	outcome := fmt.Sprintf("%d/%d steps completed", quant.SuccessCount, quant.TotalSteps)
	if quant.TotalRetries > 0 {
		outcome += fmt.Sprintf(" with %d retries", quant.TotalRetries)
	}

	var frictionPoints []FrictionPoint
	for _, s := range quant.Steps {
		if s.Retries > 0 {
			frictionPoints = append(frictionPoints, FrictionPoint{
				Type:   FrictionRetry,
				Step:   s.Name,
				Detail: fmt.Sprintf("%d retries", s.Retries),
			})
		}
	}

	return &Narrative{
		Smoothness:     smoothness,
		Intent:         "Pipeline execution",
		Outcome:        outcome,
		FrictionPoints: frictionPoints,
	}
}
