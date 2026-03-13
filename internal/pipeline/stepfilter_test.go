package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepFilter_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		filter *StepFilter
		want   bool
	}{
		{"nil filter", nil, false},
		{"empty filter", &StepFilter{}, false},
		{"include set", &StepFilter{Include: []string{"a"}}, true},
		{"exclude set", &StepFilter{Exclude: []string{"a"}}, true},
		{"both set", &StepFilter{Include: []string{"a"}, Exclude: []string{"b"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.filter.IsActive())
		})
	}
}

func TestStepFilter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  *StepFilter
		wantErr string
	}{
		{"nil filter", nil, ""},
		{"empty filter", &StepFilter{}, ""},
		{"include only", &StepFilter{Include: []string{"a"}}, ""},
		{"exclude only", &StepFilter{Exclude: []string{"a"}}, ""},
		{"both set", &StepFilter{Include: []string{"a"}, Exclude: []string{"b"}}, "--steps and --exclude are mutually exclusive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestStepFilter_ValidateWithFromStep(t *testing.T) {
	tests := []struct {
		name     string
		filter   *StepFilter
		fromStep string
		wantErr  string
	}{
		{"nil filter", nil, "plan", ""},
		{"empty filter", &StepFilter{}, "plan", ""},
		{"no from-step", &StepFilter{Include: []string{"a"}}, "", ""},
		{"include + from-step", &StepFilter{Include: []string{"a"}}, "plan", "--from-step and --steps are mutually exclusive"},
		{"exclude + from-step", &StepFilter{Exclude: []string{"a"}}, "plan", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateWithFromStep(tt.fromStep)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestStepFilter_ValidateStepNames(t *testing.T) {
	steps := []Step{
		{ID: "fetch"},
		{ID: "plan"},
		{ID: "implement"},
		{ID: "create-pr"},
	}

	tests := []struct {
		name    string
		filter  *StepFilter
		wantErr string
	}{
		{"nil filter", nil, ""},
		{"valid include names", &StepFilter{Include: []string{"fetch", "plan"}}, ""},
		{"valid exclude names", &StepFilter{Exclude: []string{"implement", "create-pr"}}, ""},
		{
			"invalid include name",
			&StepFilter{Include: []string{"fetch", "nonexistent"}},
			"unknown step(s): nonexistent",
		},
		{
			"invalid exclude name",
			&StepFilter{Exclude: []string{"bogus"}},
			"unknown step(s): bogus",
		},
		{
			"multiple invalid names",
			&StepFilter{Include: []string{"foo", "bar"}},
			"unknown step(s): foo, bar",
		},
		{
			"error lists available steps",
			&StepFilter{Include: []string{"foo"}},
			"available: fetch, plan, implement, create-pr",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.ValidateStepNames(steps)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestStepFilter_Apply(t *testing.T) {
	makeSteps := func(ids ...string) []*Step {
		steps := make([]*Step, len(ids))
		for i, id := range ids {
			steps[i] = &Step{ID: id}
		}
		return steps
	}

	stepIDs := func(steps []*Step) []string {
		ids := make([]string, len(steps))
		for i, s := range steps {
			ids[i] = s.ID
		}
		return ids
	}

	sorted := makeSteps("fetch", "plan", "implement", "create-pr")

	tests := []struct {
		name   string
		filter *StepFilter
		want   []string
	}{
		{"nil filter passes all", nil, []string{"fetch", "plan", "implement", "create-pr"}},
		{"empty filter passes all", &StepFilter{}, []string{"fetch", "plan", "implement", "create-pr"}},
		{"include single", &StepFilter{Include: []string{"plan"}}, []string{"plan"}},
		{"include multiple", &StepFilter{Include: []string{"fetch", "implement"}}, []string{"fetch", "implement"}},
		{"include all", &StepFilter{Include: []string{"fetch", "plan", "implement", "create-pr"}}, []string{"fetch", "plan", "implement", "create-pr"}},
		{"exclude single", &StepFilter{Exclude: []string{"create-pr"}}, []string{"fetch", "plan", "implement"}},
		{"exclude multiple", &StepFilter{Exclude: []string{"implement", "create-pr"}}, []string{"fetch", "plan"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Apply(sorted)
			assert.Equal(t, tt.want, stepIDs(result))
		})
	}
}

func TestStepFilter_Apply_PreservesTopologicalOrder(t *testing.T) {
	sorted := []*Step{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
		{ID: "e"},
	}

	// Include in reverse order — result should still follow original sort
	filter := &StepFilter{Include: []string{"e", "c", "a"}}
	result := filter.Apply(sorted)

	ids := make([]string, len(result))
	for i, s := range result {
		ids[i] = s.ID
	}
	assert.Equal(t, []string{"a", "c", "e"}, ids)
}
