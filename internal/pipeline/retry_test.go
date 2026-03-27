package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name:     "custom max_delay caps exponential",
			config:   RetryConfig{Backoff: "exponential", BaseDelay: "1s", MaxDelay: "5s"},
			attempt:  10,
			expected: 5 * time.Second,
		},
		{
			name:     "custom max_delay caps linear",
			config:   RetryConfig{Backoff: "linear", BaseDelay: "2s", MaxDelay: "8s"},
			attempt:  10,
			expected: 8 * time.Second,
		},
		{
			name:     "custom max_delay caps fixed (no-op when base < max)",
			config:   RetryConfig{Backoff: "fixed", BaseDelay: "3s", MaxDelay: "10s"},
			attempt:  5,
			expected: 3 * time.Second,
		},
		{
			name:     "custom max_delay caps fixed (base > max)",
			config:   RetryConfig{Backoff: "fixed", BaseDelay: "15s", MaxDelay: "10s"},
			attempt:  1,
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.ComputeDelay(tt.attempt))
		})
	}
}

func TestRetryConfig_ParseMaxDelay(t *testing.T) {
	tests := []struct {
		name     string
		config   RetryConfig
		expected time.Duration
	}{
		{
			name:     "empty defaults to timeouts.RetryMaxDelay",
			config:   RetryConfig{},
			expected: 60 * time.Second,
		},
		{
			name:     "valid duration string",
			config:   RetryConfig{MaxDelay: "30s"},
			expected: 30 * time.Second,
		},
		{
			name:     "valid minute duration",
			config:   RetryConfig{MaxDelay: "2m"},
			expected: 2 * time.Minute,
		},
		{
			name:     "invalid duration defaults to timeouts.RetryMaxDelay",
			config:   RetryConfig{MaxDelay: "not-a-duration"},
			expected: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.ParseMaxDelay())
		})
	}
}

func TestRetryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		wantErr string
	}{
		{
			name:   "empty config is valid",
			config: RetryConfig{},
		},
		{
			name:   "on_failure=fail is valid",
			config: RetryConfig{OnFailure: "fail"},
		},
		{
			name:   "on_failure=skip is valid",
			config: RetryConfig{OnFailure: "skip"},
		},
		{
			name:   "on_failure=continue is valid",
			config: RetryConfig{OnFailure: "continue"},
		},
		{
			name: "on_failure=rework with rework_step is valid",
			config: RetryConfig{
				OnFailure:  "rework",
				ReworkStep: "fallback",
			},
		},
		{
			name:    "on_failure=rework without rework_step is invalid",
			config:  RetryConfig{OnFailure: "rework"},
			wantErr: "rework_step is required when on_failure is \"rework\"",
		},
		{
			name: "rework_step set without on_failure=rework is invalid",
			config: RetryConfig{
				OnFailure:  "fail",
				ReworkStep: "fallback",
			},
			wantErr: "rework_step is set but on_failure is \"fail\" (must be \"rework\")",
		},
		{
			name: "rework_step set with empty on_failure is invalid",
			config: RetryConfig{
				ReworkStep: "fallback",
			},
			wantErr: "rework_step is set but on_failure is \"\" (must be \"rework\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetryConfig_ResolvePolicy_Standard(t *testing.T) {
	rc := RetryConfig{Policy: "standard"}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 3, rc.MaxAttempts)
	assert.Equal(t, "exponential", rc.Backoff)
	assert.Equal(t, "1s", rc.BaseDelay)
	assert.Equal(t, "30s", rc.MaxDelay)
}

func TestRetryConfig_ResolvePolicy_None(t *testing.T) {
	rc := RetryConfig{Policy: "none"}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 1, rc.MaxAttempts)
	assert.Equal(t, "fixed", rc.Backoff)
	assert.Equal(t, "0s", rc.BaseDelay)
}

func TestRetryConfig_ResolvePolicy_Aggressive(t *testing.T) {
	rc := RetryConfig{Policy: "aggressive"}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 5, rc.MaxAttempts)
	assert.Equal(t, "exponential", rc.Backoff)
	assert.Equal(t, "200ms", rc.BaseDelay)
}

func TestRetryConfig_ResolvePolicy_Patient(t *testing.T) {
	rc := RetryConfig{Policy: "patient"}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 3, rc.MaxAttempts)
	assert.Equal(t, "exponential", rc.Backoff)
	assert.Equal(t, "5s", rc.BaseDelay)
	assert.Equal(t, "90s", rc.MaxDelay)
}

func TestRetryConfig_ResolvePolicy_ExplicitOverride(t *testing.T) {
	rc := RetryConfig{Policy: "standard", MaxAttempts: 10}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 10, rc.MaxAttempts, "explicit MaxAttempts should not be overridden by policy")
	assert.Equal(t, "exponential", rc.Backoff, "Backoff should come from policy when not explicitly set")
}

func TestRetryConfig_ResolvePolicy_UnknownPolicy(t *testing.T) {
	rc := RetryConfig{Policy: "foobar"}
	err := rc.ResolvePolicy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown retry policy")
}

func TestRetryConfig_ResolvePolicy_EmptyPolicy(t *testing.T) {
	rc := RetryConfig{}
	err := rc.ResolvePolicy()
	require.NoError(t, err)
	assert.Equal(t, 0, rc.MaxAttempts, "no fields should be changed for empty policy")
	assert.Equal(t, "", rc.Backoff)
	assert.Equal(t, "", rc.BaseDelay)
	assert.Equal(t, "", rc.MaxDelay)
}

func TestRetryConfig_Validate_UnknownPolicy(t *testing.T) {
	rc := RetryConfig{Policy: "bad"}
	err := rc.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown retry policy")
}
