package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFilterCombination(t *testing.T) {
	tests := []struct {
		name      string
		config    StepFilterConfig
		fromStep  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:   "empty config with no from-step",
			config: StepFilterConfig{},
			wantErr: false,
		},
		{
			name:   "include only",
			config: StepFilterConfig{Include: []string{"step1"}},
			wantErr: false,
		},
		{
			name:   "exclude only",
			config: StepFilterConfig{Exclude: []string{"step1"}},
			wantErr: false,
		},
		{
			name:      "include and exclude = error",
			config:    StepFilterConfig{Include: []string{"step1"}, Exclude: []string{"step2"}},
			wantErr:   true,
			errSubstr: "mutually exclusive",
		},
		{
			name:     "from-step with exclude = ok",
			config:   StepFilterConfig{Exclude: []string{"step3"}},
			fromStep: "step2",
			wantErr:  false,
		},
		{
			name:      "from-step with include = error",
			config:    StepFilterConfig{Include: []string{"step2"}},
			fromStep:  "step1",
			wantErr:   true,
			errSubstr: "--from-step and --steps cannot be combined",
		},
		{
			name:     "from-step alone = ok",
			config:   StepFilterConfig{},
			fromStep: "step1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterCombination(tt.config, tt.fromStep)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStepNames(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "fetch"},
			{ID: "plan"},
			{ID: "implement"},
			{ID: "create-pr"},
		},
	}

	tests := []struct {
		name      string
		names     []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "all valid names",
			names:   []string{"fetch", "plan", "implement"},
			wantErr: false,
		},
		{
			name:    "single valid name",
			names:   []string{"plan"},
			wantErr: false,
		},
		{
			name:      "one invalid name",
			names:     []string{"plan", "nonexistent"},
			wantErr:   true,
			errSubstr: "nonexistent",
		},
		{
			name:      "all invalid names",
			names:     []string{"foo", "bar"},
			wantErr:   true,
			errSubstr: "foo, bar",
		},
		{
			name:      "invalid name shows available steps",
			names:     []string{"nope"},
			wantErr:   true,
			errSubstr: "fetch, plan, implement, create-pr",
		},
		{
			name:    "empty list",
			names:   []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStepNames(tt.names, p)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyStepFilter(t *testing.T) {
	steps := []*Step{
		{ID: "fetch"},
		{ID: "plan"},
		{ID: "implement"},
		{ID: "create-pr"},
	}

	tests := []struct {
		name        string
		config      StepFilterConfig
		wantIDs     []string
		wantSkipped []string
		wantErr     bool
		errSubstr   string
	}{
		{
			name:    "empty config returns all steps",
			config:  StepFilterConfig{},
			wantIDs: []string{"fetch", "plan", "implement", "create-pr"},
		},
		{
			name:        "include single step",
			config:      StepFilterConfig{Include: []string{"plan"}},
			wantIDs:     []string{"plan"},
			wantSkipped: []string{"fetch", "implement", "create-pr"},
		},
		{
			name:        "include multiple steps",
			config:      StepFilterConfig{Include: []string{"plan", "implement"}},
			wantIDs:     []string{"plan", "implement"},
			wantSkipped: []string{"fetch", "create-pr"},
		},
		{
			name:        "exclude single step",
			config:      StepFilterConfig{Exclude: []string{"create-pr"}},
			wantIDs:     []string{"fetch", "plan", "implement"},
			wantSkipped: []string{"create-pr"},
		},
		{
			name:        "exclude multiple steps",
			config:      StepFilterConfig{Exclude: []string{"implement", "create-pr"}},
			wantIDs:     []string{"fetch", "plan"},
			wantSkipped: []string{"implement", "create-pr"},
		},
		{
			name:      "exclude all steps = error",
			config:    StepFilterConfig{Exclude: []string{"fetch", "plan", "implement", "create-pr"}},
			wantErr:   true,
			errSubstr: "exclude all steps",
		},
		{
			name:      "include nonexistent step (no match) = error",
			config:    StepFilterConfig{Include: []string{"nonexistent"}},
			wantErr:   true,
			errSubstr: "exclude all steps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, skipped, err := ApplyStepFilter(steps, tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)

			var filteredIDs []string
			for _, s := range filtered {
				filteredIDs = append(filteredIDs, s.ID)
			}
			assert.Equal(t, tt.wantIDs, filteredIDs)

			if tt.wantSkipped != nil {
				assert.Equal(t, tt.wantSkipped, skipped)
			} else {
				assert.Nil(t, skipped)
			}
		})
	}
}

func TestValidateFilteredArtifacts(t *testing.T) {
	tests := []struct {
		name       string
		remaining  []*Step
		skippedIDs []string
		wantErr    bool
		errSubstr  string
	}{
		{
			name: "no skipped steps",
			remaining: []*Step{
				{ID: "plan"},
			},
			skippedIDs: nil,
			wantErr:    false,
		},
		{
			name: "remaining step depends on skipped step",
			remaining: []*Step{
				{
					ID: "implement",
					Memory: MemoryConfig{
						InjectArtifacts: []ArtifactRef{
							{Step: "plan", Artifact: "spec", As: "spec.md"},
						},
					},
				},
			},
			skippedIDs: []string{"plan"},
			wantErr:    true,
			errSubstr:  "requires artifact",
		},
		{
			name: "remaining step depends on non-skipped step",
			remaining: []*Step{
				{
					ID: "implement",
					Memory: MemoryConfig{
						InjectArtifacts: []ArtifactRef{
							{Step: "plan", Artifact: "spec", As: "spec.md"},
						},
					},
				},
			},
			skippedIDs: []string{"create-pr"},
			wantErr:    false,
		},
		{
			name: "optional artifact from skipped step is ok",
			remaining: []*Step{
				{
					ID: "implement",
					Memory: MemoryConfig{
						InjectArtifacts: []ArtifactRef{
							{Step: "plan", Artifact: "spec", As: "spec.md", Optional: true},
						},
					},
				},
			},
			skippedIDs: []string{"plan"},
			wantErr:    false,
		},
		{
			name: "cross-pipeline artifact ref is not affected",
			remaining: []*Step{
				{
					ID: "implement",
					Memory: MemoryConfig{
						InjectArtifacts: []ArtifactRef{
							{Pipeline: "other-pipeline", Artifact: "spec", As: "spec.md"},
						},
					},
				},
			},
			skippedIDs: []string{"plan"},
			wantErr:    false,
		},
	}

	p := &Pipeline{
		Steps: []Step{
			{ID: "plan"},
			{ID: "implement"},
			{ID: "create-pr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilteredArtifacts(tt.remaining, tt.skippedIDs, p, "")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseStepList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty string", input: "", want: nil},
		{name: "single step", input: "plan", want: []string{"plan"}},
		{name: "multiple steps", input: "plan,implement", want: []string{"plan", "implement"}},
		{name: "with spaces", input: " plan , implement , create-pr ", want: []string{"plan", "implement", "create-pr"}},
		{name: "trailing comma", input: "plan,", want: []string{"plan"}},
		{name: "leading comma", input: ",plan", want: []string{"plan"}},
		{name: "double comma", input: "plan,,implement", want: []string{"plan", "implement"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStepList(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestStepFilterConfigIsEmpty(t *testing.T) {
	assert.True(t, StepFilterConfig{}.IsEmpty())
	assert.False(t, StepFilterConfig{Include: []string{"a"}}.IsEmpty())
	assert.False(t, StepFilterConfig{Exclude: []string{"a"}}.IsEmpty())
}
