package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSteps(ids ...string) []*Step {
	steps := make([]*Step, len(ids))
	for i, id := range ids {
		steps[i] = &Step{ID: id}
	}
	return steps
}

func makeStepsWithArtifacts(defs ...struct {
	id        string
	artifacts []ArtifactDef
	injects   []ArtifactRef
}) []*Step {
	steps := make([]*Step, len(defs))
	for i, d := range defs {
		steps[i] = &Step{
			ID:              d.id,
			OutputArtifacts: d.artifacts,
			Memory:          MemoryConfig{InjectArtifacts: d.injects},
		}
	}
	return steps
}

func TestStepFilter_Mode(t *testing.T) {
	tests := []struct {
		name     string
		filter   StepFilter
		expected string
	}{
		{"empty filter", StepFilter{}, "none"},
		{"include mode", StepFilter{Include: []string{"a"}}, "include"},
		{"exclude mode", StepFilter{Exclude: []string{"a"}}, "exclude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.filter.Mode())
		})
	}
}

func TestStepFilter_IsActive(t *testing.T) {
	assert.False(t, (&StepFilter{}).IsActive())
	assert.True(t, (&StepFilter{Include: []string{"a"}}).IsActive())
	assert.True(t, (&StepFilter{Exclude: []string{"a"}}).IsActive())
}

func TestStepFilter_Validate(t *testing.T) {
	steps := makeSteps("a", "b", "c", "d")

	tests := []struct {
		name      string
		filter    StepFilter
		wantErr   bool
		errSubstr string
	}{
		{
			name:   "empty filter is valid",
			filter: StepFilter{},
		},
		{
			name:   "valid include steps",
			filter: StepFilter{Include: []string{"a", "b"}},
		},
		{
			name:   "valid exclude steps",
			filter: StepFilter{Exclude: []string{"c", "d"}},
		},
		{
			name:   "single valid include step",
			filter: StepFilter{Include: []string{"a"}},
		},
		{
			name:      "invalid include step",
			filter:    StepFilter{Include: []string{"nonexistent"}},
			wantErr:   true,
			errSubstr: "unknown step(s) in --steps: nonexistent",
		},
		{
			name:      "invalid exclude step",
			filter:    StepFilter{Exclude: []string{"nonexistent"}},
			wantErr:   true,
			errSubstr: "unknown step(s) in --exclude: nonexistent",
		},
		{
			name:      "mix of valid and invalid steps",
			filter:    StepFilter{Include: []string{"a", "nonexistent"}},
			wantErr:   true,
			errSubstr: "unknown step(s) in --steps: nonexistent",
		},
		{
			name:      "multiple invalid steps",
			filter:    StepFilter{Include: []string{"x", "y"}},
			wantErr:   true,
			errSubstr: "unknown step(s) in --steps: x, y",
		},
		{
			name:      "error lists available steps",
			filter:    StepFilter{Include: []string{"nonexistent"}},
			wantErr:   true,
			errSubstr: "available: a, b, c, d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate(steps)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepFilter_MutualExclusivity(t *testing.T) {
	steps := makeSteps("a", "b", "c")

	filter := StepFilter{
		Include: []string{"a"},
		Exclude: []string{"b"},
	}
	err := filter.Validate(steps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--steps and --exclude are mutually exclusive")
}

func TestStepFilter_Apply_Include(t *testing.T) {
	steps := makeSteps("a", "b", "c", "d")

	tests := []struct {
		name     string
		include  []string
		expected []string
	}{
		{"single step", []string{"b"}, []string{"b"}},
		{"multiple steps", []string{"a", "c"}, []string{"a", "c"}},
		{"all steps", []string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d"}},
		{"preserves order", []string{"d", "a"}, []string{"a", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := StepFilter{Include: tt.include}
			result, err := filter.Apply(steps)
			require.NoError(t, err)

			ids := make([]string, len(result))
			for i, s := range result {
				ids[i] = s.ID
			}
			assert.Equal(t, tt.expected, ids)
		})
	}
}

func TestStepFilter_Apply_Exclude(t *testing.T) {
	steps := makeSteps("a", "b", "c", "d")

	tests := []struct {
		name     string
		exclude  []string
		expected []string
	}{
		{"single step", []string{"c"}, []string{"a", "b", "d"}},
		{"multiple steps", []string{"b", "d"}, []string{"a", "c"}},
		{"first and last", []string{"a", "d"}, []string{"b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := StepFilter{Exclude: tt.exclude}
			result, err := filter.Apply(steps)
			require.NoError(t, err)

			ids := make([]string, len(result))
			for i, s := range result {
				ids[i] = s.ID
			}
			assert.Equal(t, tt.expected, ids)
		})
	}
}

func TestStepFilter_Apply_NoFilter(t *testing.T) {
	steps := makeSteps("a", "b", "c")
	filter := StepFilter{}
	result, err := filter.Apply(steps)
	require.NoError(t, err)
	assert.Equal(t, steps, result)
}

func TestStepFilter_EmptyResult(t *testing.T) {
	steps := makeSteps("a", "b", "c")

	filter := StepFilter{Exclude: []string{"a", "b", "c"}}
	_, err := filter.Apply(steps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "excluded all steps")
}

func TestStepFilter_ValidateDependencies(t *testing.T) {
	type stepDef struct {
		id        string
		artifacts []ArtifactDef
		injects   []ArtifactRef
	}

	t.Run("dependencies satisfied by filtered set", func(t *testing.T) {
		allSteps := makeStepsWithArtifacts(
			stepDef{id: "a", artifacts: []ArtifactDef{{Name: "output1", Path: ".wave/output/out1"}}},
			stepDef{id: "b", injects: []ArtifactRef{{Step: "a", Artifact: "output1", As: "input1"}}},
		)
		filter := StepFilter{Include: []string{"a", "b"}}
		filtered, _ := filter.Apply(allSteps)
		err := filter.ValidateDependencies(filtered, allSteps, nil)
		require.NoError(t, err)
	})

	t.Run("dependencies satisfied by workspace artifacts", func(t *testing.T) {
		allSteps := makeStepsWithArtifacts(
			stepDef{id: "a", artifacts: []ArtifactDef{{Name: "output1", Path: ".wave/output/out1"}}},
			stepDef{id: "b", injects: []ArtifactRef{{Step: "a", Artifact: "output1", As: "input1"}}},
		)
		filter := StepFilter{Include: []string{"b"}}
		filtered, _ := filter.Apply(allSteps)
		artifactPaths := map[string]string{
			"a:output1": "/some/workspace/path/.wave/output/out1",
		}
		err := filter.ValidateDependencies(filtered, allSteps, artifactPaths)
		require.NoError(t, err)
	})

	t.Run("missing dependency artifact", func(t *testing.T) {
		allSteps := makeStepsWithArtifacts(
			stepDef{id: "a", artifacts: []ArtifactDef{{Name: "output1", Path: ".wave/output/out1"}}},
			stepDef{id: "b", injects: []ArtifactRef{{Step: "a", Artifact: "output1", As: "input1"}}},
		)
		filter := StepFilter{Include: []string{"b"}}
		filtered, _ := filter.Apply(allSteps)
		err := filter.ValidateDependencies(filtered, allSteps, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing artifacts")
		assert.Contains(t, err.Error(), "output1")
		assert.Contains(t, err.Error(), "skipped step 'a'")
	})

	t.Run("no filter is always valid", func(t *testing.T) {
		filter := StepFilter{}
		err := filter.ValidateDependencies(nil, nil, nil)
		require.NoError(t, err)
	})

	t.Run("no inject artifacts is valid", func(t *testing.T) {
		allSteps := makeStepsWithArtifacts(
			stepDef{id: "a"},
			stepDef{id: "b"},
		)
		filter := StepFilter{Include: []string{"b"}}
		filtered, _ := filter.Apply(allSteps)
		err := filter.ValidateDependencies(filtered, allSteps, nil)
		require.NoError(t, err)
	})
}
