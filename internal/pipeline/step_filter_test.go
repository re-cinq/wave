package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPipeline() *Pipeline {
	return &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "specify"},
			{ID: "clarify", Dependencies: []string{"specify"}},
			{ID: "plan", Dependencies: []string{"clarify"}},
			{ID: "tasks", Dependencies: []string{"plan"}},
			{ID: "implement", Dependencies: []string{"tasks"}},
			{ID: "create-pr", Dependencies: []string{"implement"}},
		},
	}
}

func stepPtrs(p *Pipeline) []*Step {
	ptrs := make([]*Step, len(p.Steps))
	for i := range p.Steps {
		ptrs[i] = &p.Steps[i]
	}
	return ptrs
}

func stepIDs(steps []*Step) []string {
	ids := make([]string, len(steps))
	for i, s := range steps {
		ids[i] = s.ID
	}
	return ids
}

func TestStepFilter_Validate(t *testing.T) {
	p := testPipeline()

	tests := []struct {
		name          string
		filter        *StepFilter
		expectError   bool
		errorContains string
	}{
		{
			name:   "nil filter is valid",
			filter: nil,
		},
		{
			name:   "empty filter is valid",
			filter: &StepFilter{},
		},
		{
			name:   "valid include filter",
			filter: &StepFilter{Include: []string{"plan", "tasks"}},
		},
		{
			name:   "valid exclude filter",
			filter: &StepFilter{Exclude: []string{"implement", "create-pr"}},
		},
		{
			name:          "include and exclude are mutually exclusive",
			filter:        &StepFilter{Include: []string{"plan"}, Exclude: []string{"implement"}},
			expectError:   true,
			errorContains: "mutually exclusive",
		},
		{
			name:          "invalid include step name",
			filter:        &StepFilter{Include: []string{"nonexistent"}},
			expectError:   true,
			errorContains: "unknown step",
		},
		{
			name:          "invalid exclude step name",
			filter:        &StepFilter{Exclude: []string{"bogus"}},
			expectError:   true,
			errorContains: "unknown step",
		},
		{
			name:          "invalid include step lists available steps",
			filter:        &StepFilter{Include: []string{"bad-step"}},
			expectError:   true,
			errorContains: "specify",
		},
		{
			name:   "single valid include step",
			filter: &StepFilter{Include: []string{"specify"}},
		},
		{
			name:   "all steps included",
			filter: &StepFilter{Include: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate(p)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepFilter_ValidateCombinations(t *testing.T) {
	tests := []struct {
		name          string
		filter        *StepFilter
		fromStep      string
		expectError   bool
		errorContains string
	}{
		{
			name:     "nil filter with fromStep is valid",
			filter:   nil,
			fromStep: "plan",
		},
		{
			name:     "exclude with fromStep is valid",
			filter:   &StepFilter{Exclude: []string{"create-pr"}},
			fromStep: "plan",
		},
		{
			name:          "include with fromStep is rejected",
			filter:        &StepFilter{Include: []string{"plan"}},
			fromStep:      "clarify",
			expectError:   true,
			errorContains: "mutually exclusive",
		},
		{
			name:   "include without fromStep is valid",
			filter: &StepFilter{Include: []string{"plan"}},
		},
		{
			name:   "exclude without fromStep is valid",
			filter: &StepFilter{Exclude: []string{"plan"}},
		},
		{
			name:     "empty filter with fromStep is valid",
			filter:   &StepFilter{},
			fromStep: "plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateCombinations(tt.fromStep)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepFilter_Apply(t *testing.T) {
	p := testPipeline()
	allSteps := stepPtrs(p)

	tests := []struct {
		name     string
		filter   *StepFilter
		expected []string
	}{
		{
			name:     "nil filter returns all steps",
			filter:   nil,
			expected: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"},
		},
		{
			name:     "empty filter returns all steps",
			filter:   &StepFilter{},
			expected: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"},
		},
		{
			name:     "include specific steps",
			filter:   &StepFilter{Include: []string{"plan", "tasks"}},
			expected: []string{"plan", "tasks"},
		},
		{
			name:     "include single step",
			filter:   &StepFilter{Include: []string{"plan"}},
			expected: []string{"plan"},
		},
		{
			name:     "include all steps",
			filter:   &StepFilter{Include: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"}},
			expected: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"},
		},
		{
			name:     "exclude specific steps",
			filter:   &StepFilter{Exclude: []string{"implement", "create-pr"}},
			expected: []string{"specify", "clarify", "plan", "tasks"},
		},
		{
			name:     "exclude single step",
			filter:   &StepFilter{Exclude: []string{"create-pr"}},
			expected: []string{"specify", "clarify", "plan", "tasks", "implement"},
		},
		{
			name:     "exclude all steps returns empty",
			filter:   &StepFilter{Exclude: []string{"specify", "clarify", "plan", "tasks", "implement", "create-pr"}},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Apply(allSteps)
			assert.Equal(t, tt.expected, stepIDs(result))
		})
	}
}

func TestStepFilter_ValidateDependencies(t *testing.T) {
	p := testPipeline()

	tests := []struct {
		name           string
		filter         *StepFilter
		artifactExists func(string) bool
		expectError    bool
		errorContains  string
	}{
		{
			name:   "nil filter passes",
			filter: nil,
		},
		{
			name:   "no dependencies excluded passes",
			filter: &StepFilter{Exclude: []string{"create-pr"}},
		},
		{
			name:           "dependency excluded without prior artifacts fails",
			filter:         &StepFilter{Include: []string{"plan", "implement"}},
			artifactExists: func(stepID string) bool { return false },
			expectError:    true,
			errorContains:  "depends on",
		},
		{
			name:   "dependency excluded but prior artifacts exist passes",
			filter: &StepFilter{Include: []string{"plan", "implement"}},
			artifactExists: func(stepID string) bool {
				return stepID == "clarify" || stepID == "tasks"
			},
		},
		{
			name:   "adjacent steps in filtered set satisfy dependencies",
			filter: &StepFilter{Include: []string{"specify", "clarify", "plan"}},
		},
		{
			name:           "first step with no dependencies always passes",
			filter:         &StepFilter{Include: []string{"specify"}},
			artifactExists: func(stepID string) bool { return false },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tt.filter.Apply(stepPtrs(p))
			err := tt.filter.ValidateDependencies(filtered, p, tt.artifactExists)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepFilter_ShouldRun(t *testing.T) {
	tests := []struct {
		name     string
		filter   *StepFilter
		stepID   string
		expected bool
	}{
		{
			name:     "nil filter always runs",
			filter:   nil,
			stepID:   "plan",
			expected: true,
		},
		{
			name:     "include filter runs included step",
			filter:   &StepFilter{Include: []string{"plan", "tasks"}},
			stepID:   "plan",
			expected: true,
		},
		{
			name:     "include filter skips non-included step",
			filter:   &StepFilter{Include: []string{"plan", "tasks"}},
			stepID:   "implement",
			expected: false,
		},
		{
			name:     "exclude filter runs non-excluded step",
			filter:   &StepFilter{Exclude: []string{"implement"}},
			stepID:   "plan",
			expected: true,
		},
		{
			name:     "exclude filter skips excluded step",
			filter:   &StepFilter{Exclude: []string{"implement"}},
			stepID:   "implement",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.filter.ShouldRun(tt.stepID))
		})
	}
}

func TestStepFilter_IsActive(t *testing.T) {
	assert.False(t, (*StepFilter)(nil).IsActive())
	assert.False(t, (&StepFilter{}).IsActive())
	assert.True(t, (&StepFilter{Include: []string{"plan"}}).IsActive())
	assert.True(t, (&StepFilter{Exclude: []string{"plan"}}).IsActive())
}

func TestParseStepFilter(t *testing.T) {
	tests := []struct {
		name    string
		steps   string
		exclude string
		want    *StepFilter
	}{
		{
			name: "both empty returns nil",
		},
		{
			name:  "steps only",
			steps: "plan,tasks",
			want:  &StepFilter{Include: []string{"plan", "tasks"}},
		},
		{
			name:    "exclude only",
			exclude: "implement,create-pr",
			want:    &StepFilter{Exclude: []string{"implement", "create-pr"}},
		},
		{
			name:  "single step",
			steps: "plan",
			want:  &StepFilter{Include: []string{"plan"}},
		},
		{
			name:  "trims whitespace",
			steps: " plan , tasks ",
			want:  &StepFilter{Include: []string{"plan", "tasks"}},
		},
		{
			name:  "skips empty segments",
			steps: "plan,,tasks,",
			want:  &StepFilter{Include: []string{"plan", "tasks"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStepFilter(tt.steps, tt.exclude)
			assert.Equal(t, tt.want, got)
		})
	}
}
