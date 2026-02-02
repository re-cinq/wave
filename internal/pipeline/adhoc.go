package pipeline

import (
	"fmt"

	"github.com/recinq/wave/internal/manifest"
)

const (
	DefaultNavigatorPersona = "navigator"
)

type AdHocOptions struct {
	Input            string
	NavigatorPersona string
	ExecutePersona   string
	Manifest         *manifest.Manifest
}

func GenerateAdHocPipeline(opts AdHocOptions) (*Pipeline, error) {
	if opts.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}

	navigatorPersona := opts.NavigatorPersona
	if navigatorPersona == "" {
		navigatorPersona = DefaultNavigatorPersona
	}

	executePersona := opts.ExecutePersona
	if executePersona == "" {
		return nil, fmt.Errorf("executePersona is required")
	}

	if opts.Manifest.GetPersona(navigatorPersona) == nil {
		return nil, fmt.Errorf("navigator persona %q not found in manifest", navigatorPersona)
	}

	if opts.Manifest.GetPersona(executePersona) == nil {
		return nil, fmt.Errorf("execute persona %q not found in manifest", executePersona)
	}

	pipeline := &Pipeline{
		Kind: "WavePipeline",
		Metadata: PipelineMetadata{
			Name:        "adhoc",
			Description: "Ad-hoc generated pipeline",
		},
		Input: InputConfig{
			Source: "cli",
		},
		Steps: []Step{
			generateNavigateStep(navigatorPersona, opts.Input),
			generateExecuteStep(executePersona, opts.Input),
		},
	}

	injectArtifacts(pipeline)

	return pipeline, nil
}

func generateNavigateStep(persona, input string) Step {
	return Step{
		ID:      "navigate",
		Persona: persona,
		Memory: MemoryConfig{
			Strategy: "fresh",
		},
		Workspace: WorkspaceConfig{
			Root: "./",
			Mount: []Mount{
				{Source: "./", Target: "/src", Mode: "readonly"},
			},
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: fmt.Sprintf("Analyze the codebase for: {{ input }}\nInput: %s\n\nProvide a structured analysis including relevant files, patterns, and dependencies.", input),
		},
		OutputArtifacts: []ArtifactDef{
			{Name: "analysis", Path: "output/analysis.json", Type: "json"},
		},
		Handover: HandoverConfig{
			Contract: ContractConfig{
				Type:       "json_schema",
				SchemaPath: ".wave/contracts/navigation.schema.json",
				Source:     "output/analysis.json",
				OnFailure:  "retry",
				MaxRetries: 2,
			},
		},
	}
}

func generateExecuteStep(persona, input string) Step {
	return Step{
		ID:           "execute",
		Persona:      persona,
		Dependencies: []string{"navigate"},
		Memory: MemoryConfig{
			Strategy: "fresh",
			InjectArtifacts: []ArtifactRef{
				{Step: "navigate", Artifact: "analysis", As: "navigation_report"},
			},
		},
		Workspace: WorkspaceConfig{
			Root: "./",
			Mount: []Mount{
				{Source: "./", Target: "/src", Mode: "readwrite"},
			},
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: fmt.Sprintf("Execute the task: %s\n\nUse the navigation analysis to inform your approach.", input),
		},
		Handover: HandoverConfig{
			Contract: ContractConfig{
				Type:       "test_suite",
				Command:    "go test ./...",
				MustPass:   true,
				OnFailure:  "retry",
				MaxRetries: 3,
			},
			Compaction: CompactionConfig{
				Trigger: "token_limit_80%",
				Persona: "summarizer",
			},
		},
	}
}

func injectArtifacts(pipeline *Pipeline) {
	if len(pipeline.Steps) < 2 {
		return
	}

	for i := range pipeline.Steps {
		if pipeline.Steps[i].ID == "execute" {
			pipeline.Steps[i].Memory.InjectArtifacts = []ArtifactRef{
				{Step: "navigate", Artifact: "analysis", As: "navigation_report"},
			}
		}
	}
}
