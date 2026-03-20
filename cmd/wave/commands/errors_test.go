package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIError_Error(t *testing.T) {
	err := NewCLIError(CodePipelineNotFound, "pipeline 'foo' not found", "Run 'wave list pipelines'")
	assert.Equal(t, "pipeline 'foo' not found", err.Error())
}

func TestCLIError_JSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		err      *CLIError
		wantCode string
		wantMsg  string
		hasDebug bool
	}{
		{
			name:     "basic error",
			err:      NewCLIError(CodePipelineNotFound, "not found", "try list"),
			wantCode: CodePipelineNotFound,
			wantMsg:  "not found",
		},
		{
			name:     "debug omitted when empty",
			err:      NewCLIError(CodeInternalError, "failed", "retry"),
			wantCode: CodeInternalError,
			wantMsg:  "failed",
			hasDebug: false,
		},
		{
			name: "debug included when set",
			err: &CLIError{
				Message:    "failed",
				Code:       CodeInternalError,
				Suggestion: "retry",
				Debug:      "stack trace here",
			},
			wantCode: CodeInternalError,
			wantMsg:  "failed",
			hasDebug: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.err)
			require.NoError(t, err)

			var result map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &result))

			assert.Equal(t, tt.wantMsg, result["error"])
			assert.Equal(t, tt.wantCode, result["code"])
			assert.NotEmpty(t, result["suggestion"])

			if tt.hasDebug {
				assert.NotEmpty(t, result["debug"])
			} else {
				_, hasDebug := result["debug"]
				assert.False(t, hasDebug, "debug field should be omitted when empty")
			}
		})
	}
}

func TestRenderJSONError_CLIError(t *testing.T) {
	var buf bytes.Buffer
	cliErr := NewCLIError(CodePipelineNotFound, "pipeline not found", "wave list pipelines")

	RenderJSONError(&buf, cliErr, false)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "pipeline not found", result["error"])
	assert.Equal(t, CodePipelineNotFound, result["code"])
	assert.Equal(t, "wave list pipelines", result["suggestion"])
}

func TestRenderJSONError_PlainError(t *testing.T) {
	var buf bytes.Buffer
	plainErr := errors.New("something went wrong")

	RenderJSONError(&buf, plainErr, false)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "something went wrong", result["error"])
	assert.Equal(t, CodeInternalError, result["code"])
}

func TestRenderJSONError_DebugIncluded(t *testing.T) {
	var buf bytes.Buffer
	cliErr := &CLIError{
		Message:    "failed",
		Code:       CodeInternalError,
		Suggestion: "retry",
		Debug:      "detailed stack",
	}

	RenderJSONError(&buf, cliErr, true)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "detailed stack", result["debug"])
}

func TestRenderJSONError_DebugExcluded(t *testing.T) {
	var buf bytes.Buffer
	cliErr := &CLIError{
		Message:    "failed",
		Code:       CodeInternalError,
		Suggestion: "retry",
		Debug:      "detailed stack",
	}

	RenderJSONError(&buf, cliErr, false)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	_, hasDebug := result["debug"]
	assert.False(t, hasDebug, "debug should not be present when debug=false")
}

func TestRenderTextError_CLIError(t *testing.T) {
	var buf bytes.Buffer
	cliErr := NewCLIError(CodePipelineNotFound, "pipeline not found", "wave list pipelines")

	RenderTextError(&buf, cliErr, false)

	output := buf.String()
	assert.Contains(t, output, "Error: pipeline not found")
	assert.Contains(t, output, "Suggestion: wave list pipelines")
}

func TestRenderTextError_PlainError(t *testing.T) {
	var buf bytes.Buffer
	plainErr := errors.New("generic failure")

	RenderTextError(&buf, plainErr, false)

	output := buf.String()
	assert.Contains(t, output, "Error: generic failure")
	assert.NotContains(t, output, "Suggestion:")
}

func TestRenderTextError_WithDebug(t *testing.T) {
	var buf bytes.Buffer
	cliErr := &CLIError{
		Message:    "failed",
		Code:       CodeInternalError,
		Suggestion: "retry",
		Debug:      "stack trace",
	}

	RenderTextError(&buf, cliErr, true)

	output := buf.String()
	assert.Contains(t, output, "Debug: stack trace")
}

func TestRenderTextError_DebugHidden(t *testing.T) {
	var buf bytes.Buffer
	cliErr := &CLIError{
		Message:    "failed",
		Code:       CodeInternalError,
		Suggestion: "retry",
		Debug:      "stack trace",
	}

	RenderTextError(&buf, cliErr, false)

	output := buf.String()
	assert.NotContains(t, output, "Debug:")
}
