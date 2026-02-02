package manifest

import (
	"path/filepath"
	"time"
)

type Manifest struct {
	APIVersion  string       `yaml:"apiVersion"`
	Kind        string       `yaml:"kind"`
	Metadata    Metadata     `yaml:"metadata"`
	Adapters    []Adapter    `yaml:"adapters,omitempty"`
	Personas    []Persona    `yaml:"personas,omitempty"`
	Runtime     Runtime      `yaml:"runtime"`
	SkillMounts []SkillMount `yaml:"skillMounts,omitempty"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
}

type Adapter struct {
	Binary             string      `yaml:"binary"`
	Mode               string      `yaml:"mode"`
	OutputFormat       string      `yaml:"outputFormat,omitempty"`
	ProjectFiles       []string    `yaml:"projectFiles,omitempty"`
	DefaultPermissions Permissions `yaml:"defaultPermissions,omitempty"`
	HooksTemplate      []HookRule  `yaml:"hooksTemplate,omitempty"`
}

type Persona struct {
	Adapter          string      `yaml:"adapter"`
	Description      string      `yaml:"description,omitempty"`
	SystemPromptFile string      `yaml:"systemPromptFile"`
	Temperature      float64     `yaml:"temperature,omitempty"`
	Permissions      Permissions `yaml:"permissions,omitempty"`
	Hooks            HookConfig  `yaml:"hooks,omitempty"`
}

type Permissions struct {
	AllowedTools []string `yaml:"allowedTools,omitempty"`
	Deny         []string `yaml:"deny,omitempty"`
}

type HookConfig struct {
	PreToolUse  []HookRule `yaml:"preToolUse,omitempty"`
	PostToolUse []HookRule `yaml:"postToolUse,omitempty"`
}

type HookRule struct {
	Matcher string   `yaml:"matcher"`
	Command []string `yaml:"command"`
}

type Runtime struct {
	WorkspaceRoot        string      `yaml:"workspaceRoot"`
	MaxConcurrentWorkers int         `yaml:"maxConcurrentWorkers,omitempty"`
	DefaultTimeoutMin    int         `yaml:"defaultTimeoutMin,omitempty"`
	Relay                RelayConfig `yaml:"relay,omitempty"`
	Audit                AuditConfig `yaml:"audit,omitempty"`
	MetaPipeline         MetaConfig  `yaml:"metaPipeline,omitempty"`
}

type RelayConfig struct {
	TokenThresholdPercent int    `yaml:"tokenThresholdPercent,omitempty"`
	Strategy              string `yaml:"strategy,omitempty"`
}

type AuditConfig struct {
	LogDir               string `yaml:"logDir,omitempty"`
	LogAllToolCalls      bool   `yaml:"logAllToolCalls,omitempty"`
	LogAllFileOperations bool   `yaml:"logAllFileOperations,omitempty"`
}

type MetaConfig struct {
	MaxDepth       int `yaml:"maxDepth,omitempty"`
	MaxTotalSteps  int `yaml:"maxTotalSteps,omitempty"`
	MaxTotalTokens int `yaml:"maxTotalTokens,omitempty"`
	TimeoutMin     int `yaml:"timeoutMin,omitempty"`
}

type SkillMount struct {
	Path string `yaml:"path"`
}

func (m *Manifest) GetAdapter(name string) *Adapter {
	for i := range m.Adapters {
		if m.Adapters[i].Binary == name {
			return &m.Adapters[i]
		}
	}
	return nil
}

func (m *Manifest) GetPersona(name string) *Persona {
	for i := range m.Personas {
		if m.Personas[i].Adapter == name {
			return &m.Personas[i]
		}
	}
	return nil
}

func (p *Persona) GetSystemPromptPath(root string) string {
	if filepath.IsAbs(p.SystemPromptFile) {
		return p.SystemPromptFile
	}
	return filepath.Join(root, p.SystemPromptFile)
}

func (r *Runtime) GetDefaultTimeout() time.Duration {
	if r.DefaultTimeoutMin > 0 {
		return time.Duration(r.DefaultTimeoutMin) * time.Minute
	}
	return 5 * time.Minute
}
