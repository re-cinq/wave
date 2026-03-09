package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryConfig_EffectiveMaxAttempts(t *testing.T) {
	tests := []struct {
		name     string
		config   RetryConfig
		expected int
	}{
		{
			name:     "default returns 1",
			config:   RetryConfig{},
			expected: 1,
		},
		{
			name:     "explicit value",
			config:   RetryConfig{MaxAttempts: 3},
			expected: 3,
		},
		{
			name:     "zero returns 1",
			config:   RetryConfig{MaxAttempts: 0},
			expected: 1,
		},
		{
			name:     "negative returns 1",
			config:   RetryConfig{MaxAttempts: -1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.EffectiveMaxAttempts())
		})
	}
}

func TestRetryConfig_ParseBaseDelay(t *testing.T) {
	tests := []struct {
		name     string
		config   RetryConfig
		expected time.Duration
	}{
		{
			name:     "empty defaults to 1s",
			config:   RetryConfig{},
			expected: time.Second,
		},
		{
			name:     "valid duration string",
			config:   RetryConfig{BaseDelay: "2s"},
			expected: 2 * time.Second,
		},
		{
			name:     "valid millisecond duration",
			config:   RetryConfig{BaseDelay: "500ms"},
			expected: 500 * time.Millisecond,
		},
		{
			name:     "invalid duration defaults to 1s",
			config:   RetryConfig{BaseDelay: "not-a-duration"},
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.ParseBaseDelay())
		})
	}
}

func TestRetryConfig_ComputeDelay(t *testing.T) {
	tests := []struct {
		name     string
		config   RetryConfig
		attempt  int
		expected time.Duration
	}{
		{
			name:     "linear default attempt 1",
			config:   RetryConfig{},
			attempt:  1,
			expected: time.Second,
		},
		{
			name:     "linear default attempt 3",
			config:   RetryConfig{},
			attempt:  3,
			expected: 3 * time.Second,
		},
		{
			name:     "linear with custom base",
			config:   RetryConfig{Backoff: "linear", BaseDelay: "2s"},
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "fixed backoff always returns base",
			config:   RetryConfig{Backoff: "fixed", BaseDelay: "5s"},
			attempt:  1,
			expected: 5 * time.Second,
		},
		{
			name:     "fixed backoff attempt 3",
			config:   RetryConfig{Backoff: "fixed", BaseDelay: "5s"},
			attempt:  3,
			expected: 5 * time.Second,
		},
		{
			name:     "exponential attempt 1",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s"},
			attempt:  1,
			expected: time.Second,
		},
		{
			name:     "exponential attempt 2",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s"},
			attempt:  2,
			expected: 2 * time.Second,
		},
		{
			name:     "exponential attempt 3",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s"},
			attempt:  3,
			expected: 4 * time.Second,
		},
		{
			name:     "exponential attempt 4",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s"},
			attempt:  4,
			expected: 8 * time.Second,
		},
		{
			name:     "exponential caps at 60s",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "10s"},
			attempt:  5,
			expected: 60 * time.Second,
		},
		{
			name:     "exponential with large attempt caps at 60s",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s"},
			attempt:  10,
			expected: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.ComputeDelay(tt.attempt))
		})
	}
}
