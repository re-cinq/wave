package pipeline

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRunID_Format(t *testing.T) {
	id := GenerateRunID("my-pipeline", 8)

	// Should match format: name-hexsuffix
	re := regexp.MustCompile(`^my-pipeline-[0-9a-f]{8}$`)
	assert.Regexp(t, re, id, "ID should match format name-hexsuffix")
}

func TestGenerateRunID_Length(t *testing.T) {
	tests := []struct {
		name       string
		hashLength int
		wantLen    int
	}{
		{"length 4", 4, 4},
		{"length 8 (default)", 8, 8},
		{"length 12", 12, 12},
		{"length 16", 16, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateRunID("test", tt.hashLength)
			// Extract suffix after "test-"
			suffix := id[len("test-"):]
			assert.Len(t, suffix, tt.wantLen, "suffix should be %d hex chars", tt.wantLen)

			// Verify suffix is valid hex
			re := regexp.MustCompile(`^[0-9a-f]+$`)
			assert.Regexp(t, re, suffix, "suffix should be valid hex")
		})
	}
}

func TestGenerateRunID_DefaultLength(t *testing.T) {
	// When hashLength is 0, should default to 8
	id := GenerateRunID("test", 0)
	suffix := id[len("test-"):]
	assert.Len(t, suffix, 8, "default hash length should be 8")
}

func TestGenerateRunID_NegativeLength(t *testing.T) {
	// Negative length should also default to 8
	id := GenerateRunID("test", -1)
	suffix := id[len("test-"):]
	assert.Len(t, suffix, 8, "negative hash length should default to 8")
}

func TestGenerateRunID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id := GenerateRunID("pipeline", 8)
		require.False(t, seen[id], "ID collision detected on iteration %d: %s", i, id)
		seen[id] = true
	}

	assert.Len(t, seen, count, "all generated IDs should be unique")
}

func TestGenerateRunID_PreservesName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPfx  string
	}{
		{"simple name", "my-pipeline", "my-pipeline-"},
		{"dashes in name", "github-issue-enhancer", "github-issue-enhancer-"},
		{"single word", "pipeline", "pipeline-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateRunID(tt.input, 8)
			assert.True(t, len(id) > len(tt.wantPfx), "ID should be longer than prefix")
			assert.Equal(t, tt.wantPfx, id[:len(tt.wantPfx)], "ID should start with name-")
		})
	}
}

func TestTimestampFallback(t *testing.T) {
	// Test the fallback function directly
	result := timestampFallback(8)
	assert.Len(t, result, 8, "fallback should return correct length")

	re := regexp.MustCompile(`^[0-9a-f]+$`)
	assert.Regexp(t, re, result, "fallback should return valid hex")
}
