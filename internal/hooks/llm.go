package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// executeLLMJudge invokes a single-turn LLM evaluation via the claude CLI.
func executeLLMJudge(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Interpolate event context into the prompt
	prompt := hook.Prompt
	prompt = strings.ReplaceAll(prompt, "{{pipeline_id}}", evt.PipelineID)
	prompt = strings.ReplaceAll(prompt, "{{step_id}}", evt.StepID)
	prompt = strings.ReplaceAll(prompt, "{{workspace}}", evt.Workspace)

	args := []string{"--print", "--output-format", "json", "-p", prompt}
	if hook.Model != "" {
		args = append(args, "--model", hook.Model)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("LLM judge execution failed: %v: %s", err, stderr.String()),
			Err:      err,
		}
	}

	// Parse the JSON response — look for {"ok": bool, "reason": "..."}
	// The response might be wrapped in a JSON object from Claude CLI
	output := stdout.String()

	// Try to extract JSON from the output
	var result struct {
		OK     bool   `json:"ok"`
		Reason string `json:"reason,omitempty"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// Try to find JSON in the output
		start := strings.Index(output, "{")
		end := strings.LastIndex(output, "}")
		if start >= 0 && end > start {
			jsonStr := output[start : end+1]
			if err2 := json.Unmarshal([]byte(jsonStr), &result); err2 != nil {
				return HookResult{
					HookName: hook.Name,
					Decision: DecisionBlock,
					Reason:   fmt.Sprintf("failed to parse LLM response: %v", err),
					Err:      err,
				}
			}
		} else {
			return HookResult{
				HookName: hook.Name,
				Decision: DecisionBlock,
				Reason:   fmt.Sprintf("failed to parse LLM response: %v", err),
				Err:      err,
			}
		}
	}

	if !result.OK {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   result.Reason,
		}
	}

	return HookResult{
		HookName: hook.Name,
		Decision: DecisionProceed,
	}
}
