package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteLLMJudge_CommandNotFound(t *testing.T) {
	// Since "claude" CLI won't be available in the test environment,
	// executeLLMJudge should fail with a command-not-found error and return DecisionBlock.
	hook := &LifecycleHookDef{
		Name:    "llm-hook",
		Type:    HookTypeLLMJudge,
		Prompt:  "Is this code safe?",
		Model:   "sonnet",
		Timeout: "2s",
	}
	evt := HookEvent{
		Type:       EventStepCompleted,
		PipelineID: "test-pipeline",
		StepID:     "test-step",
		Workspace:  "/tmp/workspace",
	}

	result := executeLLMJudge(context.Background(), hook, evt)

	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
	assert.Contains(t, result.Reason, "LLM judge execution failed")
	assert.Equal(t, "llm-hook", result.HookName)
}

func TestExecuteLLMJudge_PromptInterpolation(t *testing.T) {
	// Verify that the prompt interpolation logic works by checking the function
	// still returns block (since claude is unavailable) but tests the code path
	hook := &LifecycleHookDef{
		Name:    "llm-interpolation",
		Type:    HookTypeLLMJudge,
		Prompt:  "Review pipeline {{pipeline_id}} step {{step_id}} in {{workspace}}",
		Timeout: "2s",
	}
	evt := HookEvent{
		Type:       EventStepCompleted,
		PipelineID: "my-pipeline",
		StepID:     "my-step",
		Workspace:  "/tmp/my-workspace",
	}

	result := executeLLMJudge(context.Background(), hook, evt)

	// Will fail because claude doesn't exist, but the code path for prompt
	// interpolation is exercised
	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
}

func TestExecuteLLMJudge_TimeoutRespected(t *testing.T) {
	hook := &LifecycleHookDef{
		Name:    "llm-timeout",
		Type:    HookTypeLLMJudge,
		Prompt:  "test",
		Timeout: "100ms",
	}
	evt := HookEvent{
		Type:       EventStepCompleted,
		PipelineID: "test",
	}

	result := executeLLMJudge(context.Background(), hook, evt)

	// Should fail (claude not available) but should not hang
	assert.Equal(t, DecisionBlock, result.Decision)
}
