// Package listing provides domain logic for the `wave list` CLI commands.
//
// It owns pure data collection, filtering, sorting, and the JSON output schemas
// for runs, workspaces, pipelines, personas, adapters, contracts and skills.
// The cmd/wave/commands package consumes this package as a narrow surface and
// only handles cobra wiring, flag parsing and table rendering.
package listing

// PipelineInfo describes a pipeline declared on disk.
type PipelineInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	StepCount   int      `json:"step_count"`
	Steps       []string `json:"steps"`
}

// PersonaInfo describes a persona declared in the manifest.
type PersonaInfo struct {
	Name         string   `json:"name"`
	Adapter      string   `json:"adapter"`
	Description  string   `json:"description"`
	Temperature  float64  `json:"temperature"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	DeniedTools  []string `json:"denied_tools,omitempty"`
}

// AdapterInfo describes an adapter declared in the manifest, including binary
// availability on PATH.
type AdapterInfo struct {
	Name         string `json:"name"`
	Binary       string `json:"binary"`
	Mode         string `json:"mode"`
	OutputFormat string `json:"output_format"`
	Available    bool   `json:"available"`
}

// RunInfo holds information about a pipeline run.
type RunInfo struct {
	RunID      string `json:"run_id"`
	Pipeline   string `json:"pipeline"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	Duration   string `json:"duration"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// ContractInfo holds information about a contract schema.
type ContractInfo struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	UsedBy []ContractUsage `json:"used_by,omitempty"`
}

// ContractUsage shows where a contract is used inside a pipeline step.
type ContractUsage struct {
	Pipeline string `json:"pipeline"`
	Step     string `json:"step"`
	Persona  string `json:"persona"`
}

// SkillInfo holds information about a declared skill.
type SkillInfo struct {
	Name      string   `json:"name"`
	Check     string   `json:"check"`
	Install   string   `json:"install,omitempty"`
	Installed bool     `json:"installed"`
	UsedBy    []string `json:"used_by,omitempty"`
}

// CompositionInfo holds information about a composition pipeline.
type CompositionInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SubPipelines []string `json:"sub_pipelines"`
	StepTypes    []string `json:"step_types"`
}

// Output is the aggregated JSON payload emitted by `wave list --format json`.
type Output struct {
	Adapters  []AdapterInfo  `json:"adapters,omitempty"`
	Runs      []RunInfo      `json:"runs,omitempty"`
	Pipelines []PipelineInfo `json:"pipelines,omitempty"`
	Personas  []PersonaInfo  `json:"personas,omitempty"`
	Contracts []ContractInfo `json:"contracts,omitempty"`
	Skills    []SkillInfo    `json:"skills,omitempty"`
}

// RunsOptions filter the set of runs returned by ListRuns.
type RunsOptions struct {
	Limit    int
	Pipeline string
	Status   string
}

// ManifestPersona mirrors the subset of a persona's manifest entry consumed by
// the listing package.
type ManifestPersona struct {
	Adapter          string  `yaml:"adapter"`
	Description      string  `yaml:"description"`
	SystemPromptFile string  `yaml:"system_prompt_file"`
	Temperature      float64 `yaml:"temperature"`
	Permissions      struct {
		AllowedTools []string `yaml:"allowed_tools"`
		Deny         []string `yaml:"deny"`
	} `yaml:"permissions"`
}

// ManifestAdapter mirrors the subset of an adapter's manifest entry consumed by
// the listing package.
type ManifestAdapter struct {
	Binary       string `yaml:"binary"`
	Mode         string `yaml:"mode"`
	OutputFormat string `yaml:"output_format"`
}

// Manifest mirrors the subset of wave.yaml consumed by the listing package.
type Manifest struct {
	Adapters map[string]ManifestAdapter `yaml:"adapters"`
	Personas map[string]ManifestPersona `yaml:"personas"`
}

// PipelineSkillConfig is the shape of a single skill entry inside a pipeline's
// `requires.skills` block.
type PipelineSkillConfig struct {
	Install      string `yaml:"install,omitempty"`
	Init         string `yaml:"init,omitempty"`
	Check        string `yaml:"check,omitempty"`
	CommandsGlob string `yaml:"commands_glob,omitempty"`
}
