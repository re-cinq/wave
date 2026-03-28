package pipeline

import (
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
)

func TestClassifyStepComplexity(t *testing.T) {
	tests := []struct {
		name        string
		step        *Step
		persona     *manifest.Persona
		personaName string
		want        string
	}{
		// Simple tier: command step type
		{
			name:        "command step type routes to simple",
			step:        &Step{Type: StepTypeCommand},
			persona:     &manifest.Persona{},
			personaName: "anything",
			want:        TierSimple,
		},
		// Simple tier: conditional step type
		{
			name:        "conditional step type routes to simple",
			step:        &Step{Type: StepTypeConditional},
			persona:     &manifest.Persona{},
			personaName: "anything",
			want:        TierSimple,
		},
		// Simple tier: navigator persona
		{
			name:        "navigator persona routes to simple",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "navigator",
			want:        TierSimple,
		},
		// Simple tier: summarizer persona
		{
			name:        "summarizer persona routes to simple",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "summarizer",
			want:        TierSimple,
		},
		// Simple tier: auditor persona
		{
			name:        "auditor persona routes to simple",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "auditor",
			want:        TierSimple,
		},
		// Simple tier: planner persona
		{
			name:        "planner persona routes to simple",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "planner",
			want:        TierSimple,
		},
		// Simple tier: case insensitive persona name
		{
			name:        "persona name matching is case insensitive",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "Senior-Navigator",
			want:        TierSimple,
		},
		// Complex tier: craftsman persona
		{
			name:        "craftsman persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "craftsman",
			want:        TierComplex,
		},
		// Complex tier: implementer persona
		{
			name:        "implementer persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "implementer",
			want:        TierComplex,
		},
		// Complex tier: debugger persona
		{
			name:        "debugger persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "debugger",
			want:        TierComplex,
		},
		// Complex tier: researcher persona
		{
			name:        "researcher persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "researcher",
			want:        TierComplex,
		},
		// Complex tier: supervisor persona
		{
			name:        "supervisor persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "supervisor",
			want:        TierComplex,
		},
		// Complex tier: philosopher persona
		{
			name:        "philosopher persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "philosopher",
			want:        TierComplex,
		},
		// Complex tier: provocateur persona
		{
			name:        "provocateur persona routes to complex",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "provocateur",
			want:        TierComplex,
		},
		// Complex tier: sub-pipeline step
		{
			name:        "sub_pipeline step routes to complex",
			step:        &Step{SubPipeline: "child-pipeline"},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierComplex,
		},
		// Complex tier: loop step
		{
			name:        "loop step routes to complex",
			step:        &Step{Loop: &LoopConfig{MaxIterations: 3}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierComplex,
		},
		// Complex tier: branch step
		{
			name:        "branch step routes to complex",
			step:        &Step{Branch: &BranchConfig{On: "x"}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierComplex,
		},
		// Complex tier: aggregate step
		{
			name:        "aggregate step routes to complex",
			step:        &Step{Aggregate: &AggregateConfig{From: "a", Into: "b", Strategy: "merge_arrays"}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierComplex,
		},
		// Standard tier: fallthrough
		{
			name:        "generic step and persona routes to standard",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "generic-persona",
			want:        TierStandard,
		},
		// Standard tier: nil step
		{
			name:        "nil step with generic persona routes to standard",
			step:        nil,
			persona:     &manifest.Persona{},
			personaName: "generic-persona",
			want:        TierStandard,
		},
		// Priority: step type beats persona name (command step with complex persona)
		{
			name:        "command step type overrides complex persona keyword",
			step:        &Step{Type: StepTypeCommand},
			persona:     &manifest.Persona{},
			personaName: "craftsman",
			want:        TierSimple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyStepComplexity(tt.step, tt.persona, tt.personaName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveModelWithAutoRouting(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		step        *Step
		persona     *manifest.Persona
		routing     *manifest.RoutingConfig
		personaName string
		want        string
	}{
		{
			name:        "CLI override wins over auto-routing",
			override:    "cli-model",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "navigator",
			want:        "cli-model",
		},
		{
			name:        "step model wins over auto-routing",
			override:    "",
			step:        &Step{Model: "step-model"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "navigator",
			want:        "step-model",
		},
		{
			name:        "persona model wins over auto-routing",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{Model: "persona-model"},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "navigator",
			want:        "persona-model",
		},
		{
			name:        "auto-routing selects haiku for navigator persona",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "navigator",
			want:        "claude-haiku-4-5",
		},
		{
			name:        "auto-routing selects opus for craftsman persona",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "craftsman",
			want:        "claude-opus-4",
		},
		{
			name:        "auto-routing returns empty for standard tier (adapter default)",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "generic-persona",
			want:        "",
		},
		{
			name:     "auto-routing disabled returns empty",
			override: "",
			step:     &Step{},
			persona:  &manifest.Persona{},
			routing: &manifest.RoutingConfig{
				AutoRoute: false,
			},
			personaName: "navigator",
			want:        "",
		},
		{
			name:        "nil routing returns empty",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     nil,
			personaName: "navigator",
			want:        "",
		},
		{
			name:     "custom complexity map overrides defaults",
			override: "",
			step:     &Step{},
			persona:  &manifest.Persona{},
			routing: &manifest.RoutingConfig{
				AutoRoute: true,
				ComplexityMap: map[string]string{
					"simple": "custom-haiku",
				},
			},
			personaName: "navigator",
			want:        "custom-haiku",
		},
		{
			name:     "custom complexity map falls back to defaults for unmapped tiers",
			override: "",
			step:     &Step{},
			persona:  &manifest.Persona{},
			routing: &manifest.RoutingConfig{
				AutoRoute: true,
				ComplexityMap: map[string]string{
					"simple": "custom-haiku",
				},
			},
			personaName: "craftsman",
			want:        "claude-opus-4",
		},
		{
			name:     "command step with auto-route gets haiku",
			override: "",
			step:     &Step{Type: StepTypeCommand},
			persona:  &manifest.Persona{},
			routing:  &manifest.RoutingConfig{AutoRoute: true},
			personaName: "generic",
			want:        "claude-haiku-4-5",
		},
		{
			name:     "sub-pipeline step with auto-route gets opus",
			override: "",
			step:     &Step{SubPipeline: "child"},
			persona:  &manifest.Persona{},
			routing:  &manifest.RoutingConfig{AutoRoute: true},
			personaName: "generic",
			want:        "claude-opus-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &DefaultPipelineExecutor{modelOverride: tt.override}
			got := executor.resolveModel(tt.step, tt.persona, tt.routing, tt.personaName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoutingConfigResolveComplexityModel(t *testing.T) {
	tests := []struct {
		name    string
		routing *manifest.RoutingConfig
		tier    string
		want    string
	}{
		{
			name:    "nil routing uses defaults for simple",
			routing: nil,
			tier:    "simple",
			want:    "claude-haiku-4-5",
		},
		{
			name:    "nil routing uses defaults for complex",
			routing: nil,
			tier:    "complex",
			want:    "claude-opus-4",
		},
		{
			name:    "nil routing uses defaults for standard (empty)",
			routing: nil,
			tier:    "standard",
			want:    "",
		},
		{
			name:    "empty complexity map uses defaults",
			routing: &manifest.RoutingConfig{},
			tier:    "simple",
			want:    "claude-haiku-4-5",
		},
		{
			name: "custom map overrides specific tier",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"simple": "gpt-4o-mini",
				},
			},
			tier: "simple",
			want: "gpt-4o-mini",
		},
		{
			name: "custom map falls back to default for unmapped tier",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"simple": "gpt-4o-mini",
				},
			},
			tier: "complex",
			want: "claude-opus-4",
		},
		{
			name:    "unknown tier returns empty",
			routing: nil,
			tier:    "unknown-tier",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.routing.ResolveComplexityModel(tt.tier)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoutingConfigEffectiveDefaultTier(t *testing.T) {
	tests := []struct {
		name    string
		routing *manifest.RoutingConfig
		want    string
	}{
		{
			name:    "nil routing returns standard",
			routing: nil,
			want:    "standard",
		},
		{
			name:    "empty default tier returns standard",
			routing: &manifest.RoutingConfig{},
			want:    "standard",
		},
		{
			name: "custom default tier is returned",
			routing: &manifest.RoutingConfig{
				DefaultTier: "simple",
			},
			want: "simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.routing.EffectiveDefaultTier()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultComplexityMap(t *testing.T) {
	m := manifest.DefaultComplexityMap()
	assert.Equal(t, "claude-haiku-4-5", m["simple"])
	assert.Equal(t, "", m["standard"])
	assert.Equal(t, "claude-opus-4", m["complex"])
	assert.Len(t, m, 3)
}
