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
		// Cheapest tier: command step type
		{
			name:        "command step type routes to cheapest",
			step:        &Step{Type: StepTypeCommand},
			persona:     &manifest.Persona{},
			personaName: "anything",
			want:        TierCheapest,
		},
		// Cheapest tier: conditional step type
		{
			name:        "conditional step type routes to cheapest",
			step:        &Step{Type: StepTypeConditional},
			persona:     &manifest.Persona{},
			personaName: "anything",
			want:        TierCheapest,
		},
		// Cheapest tier: navigator persona
		{
			name:        "navigator persona routes to cheapest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "navigator",
			want:        TierCheapest,
		},
		// Cheapest tier: summarizer persona
		{
			name:        "summarizer persona routes to cheapest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "summarizer",
			want:        TierCheapest,
		},
		// Cheapest tier: auditor persona
		{
			name:        "auditor persona routes to cheapest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "auditor",
			want:        TierCheapest,
		},
		// Cheapest tier: planner persona
		{
			name:        "planner persona routes to cheapest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "planner",
			want:        TierCheapest,
		},
		// Cheapest tier: case insensitive persona name
		{
			name:        "persona name matching is case insensitive",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "Senior-Navigator",
			want:        TierCheapest,
		},
		// Strongest tier: craftsman persona
		{
			name:        "craftsman persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "craftsman",
			want:        TierStrongest,
		},
		// Strongest tier: implementer persona
		{
			name:        "implementer persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "implementer",
			want:        TierStrongest,
		},
		// Strongest tier: debugger persona
		{
			name:        "debugger persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "debugger",
			want:        TierStrongest,
		},
		// Strongest tier: researcher persona
		{
			name:        "researcher persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "researcher",
			want:        TierStrongest,
		},
		// Strongest tier: supervisor persona
		{
			name:        "supervisor persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "supervisor",
			want:        TierStrongest,
		},
		// Strongest tier: philosopher persona
		{
			name:        "philosopher persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "philosopher",
			want:        TierStrongest,
		},
		// Strongest tier: provocateur persona
		{
			name:        "provocateur persona routes to strongest",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "provocateur",
			want:        TierStrongest,
		},
		// Strongest tier: sub-pipeline step
		{
			name:        "sub_pipeline step routes to strongest",
			step:        &Step{SubPipeline: "child-pipeline"},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierStrongest,
		},
		// Strongest tier: loop step
		{
			name:        "loop step routes to strongest",
			step:        &Step{Loop: &LoopConfig{MaxIterations: 3}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierStrongest,
		},
		// Strongest tier: branch step
		{
			name:        "branch step routes to strongest",
			step:        &Step{Branch: &BranchConfig{On: "x"}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierStrongest,
		},
		// Strongest tier: aggregate step
		{
			name:        "aggregate step routes to strongest",
			step:        &Step{Aggregate: &AggregateConfig{From: "a", Into: "b", Strategy: "merge_arrays"}},
			persona:     &manifest.Persona{},
			personaName: "generic",
			want:        TierStrongest,
		},
		// Fastest tier: fallthrough
		{
			name:        "generic step and persona routes to balanced",
			step:        &Step{},
			persona:     &manifest.Persona{},
			personaName: "generic-persona",
			want:        TierBalanced,
		},
		// Fastest tier: nil step
		{
			name:        "nil step with generic persona routes to balanced",
			step:        nil,
			persona:     &manifest.Persona{},
			personaName: "generic-persona",
			want:        TierBalanced,
		},
		// Priority: step type beats persona name (command step with strongest persona)
		{
			name:        "command step type overrides strongest persona keyword",
			step:        &Step{Type: StepTypeCommand},
			persona:     &manifest.Persona{},
			personaName: "craftsman",
			want:        TierCheapest,
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
			name:        "auto-routing selects haiku for navigator persona (cheapest tier)",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "navigator",
			want:        "claude-haiku-4-5",
		},
		{
			name:        "auto-routing selects opus for craftsman persona (strongest tier)",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "craftsman",
			want:        "claude-opus-4",
		},
		{
			name:        "auto-routing returns empty for balanced tier (adapter default)",
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
					"cheapest": "custom-haiku",
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
					"cheapest": "custom-haiku",
				},
			},
			personaName: "craftsman",
			want:        "claude-opus-4",
		},
		{
			name:        "command step with auto-route gets haiku (cheapest)",
			override:    "",
			step:        &Step{Type: StepTypeCommand},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "generic",
			want:        "claude-haiku-4-5",
		},
		{
			name:        "sub-pipeline step with auto-route gets opus (strongest)",
			override:    "",
			step:        &Step{SubPipeline: "child"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "generic",
			want:        "claude-opus-4",
		},
		{
			name:        "step model as tier name 'cheapest' resolves via complexity map",
			override:    "",
			step:        &Step{Model: "cheapest"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "any",
			want:        "claude-haiku-4-5",
		},
		{
			name:        "step model as tier name 'strongest' resolves via complexity map",
			override:    "",
			step:        &Step{Model: "strongest"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "any",
			want:        "claude-opus-4",
		},
		{
			name:        "step model as tier name 'balanced' resolves to empty (adapter default)",
			override:    "",
			step:        &Step{Model: "balanced"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "any",
			want:        "",
		},
		{
			name:        "persona model as tier name 'cheapest' resolves via complexity map",
			override:    "",
			step:        &Step{},
			persona:     &manifest.Persona{Model: "cheapest"},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "any",
			want:        "claude-haiku-4-5",
		},
		{
			name:        "step explicit model (not a tier name) passes through unchanged",
			override:    "",
			step:        &Step{Model: "my-custom-model"},
			persona:     &manifest.Persona{},
			routing:     &manifest.RoutingConfig{AutoRoute: true},
			personaName: "any",
			want:        "my-custom-model",
		},
		{
			name:     "custom complexity map applies to tier name in step model",
			override: "",
			step:     &Step{Model: "cheapest"},
			persona:  &manifest.Persona{},
			routing: &manifest.RoutingConfig{
				AutoRoute: true,
				ComplexityMap: map[string]string{
					"cheapest": "my-budget-model",
				},
			},
			personaName: "any",
			want:        "my-budget-model",
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
			name:    "nil routing uses defaults for cheapest",
			routing: nil,
			tier:    "cheapest",
			want:    "claude-haiku-4-5",
		},
		{
			name:    "nil routing uses defaults for strongest",
			routing: nil,
			tier:    "strongest",
			want:    "claude-opus-4",
		},
		{
			name:    "nil routing uses defaults for balanced (empty = adapter default)",
			routing: nil,
			tier:    "balanced",
			want:    "",
		},
		{
			name:    "empty complexity map uses defaults",
			routing: &manifest.RoutingConfig{},
			tier:    "cheapest",
			want:    "claude-haiku-4-5",
		},
		{
			name: "custom map overrides specific tier",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "gpt-4o-mini",
				},
			},
			tier: "cheapest",
			want: "gpt-4o-mini",
		},
		{
			name: "custom map falls back to default for unmapped tier",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "gpt-4o-mini",
				},
			},
			tier: "strongest",
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
			name:    "nil routing returns balanced",
			routing: nil,
			want:    "balanced",
		},
		{
			name:    "empty default tier returns balanced",
			routing: &manifest.RoutingConfig{},
			want:    "balanced",
		},
		{
			name: "custom default tier is returned",
			routing: &manifest.RoutingConfig{
				DefaultTier: "cheapest",
			},
			want: "cheapest",
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
	assert.Equal(t, "claude-haiku-4-5", m["cheapest"])
	assert.Equal(t, "", m["balanced"])
	assert.Equal(t, "claude-opus-4", m["strongest"])
	assert.Len(t, m, 3)
}
