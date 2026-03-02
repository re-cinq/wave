// Package forge provides automatic forge type detection and pipeline filtering
// based on git remote URLs and configurable domain mappings.
package forge

// ForgeType identifies the type of source forge a repository is hosted on.
type ForgeType string

const (
	GitHub    ForgeType = "github"
	GitLab    ForgeType = "gitlab"
	Bitbucket ForgeType = "bitbucket"
	Gitea     ForgeType = "gitea"
	Unknown   ForgeType = "unknown"
)

// ForgeDetection holds the result of detecting the forge type from a git remote.
type ForgeDetection struct {
	Type     ForgeType `json:"type"`
	Remote   string    `json:"remote"`
	Hostname string    `json:"hostname"`
	CLITool  string    `json:"cli_tool"`
}

// ForgeConfig holds user-configurable domain-to-forge mappings from wave.yaml.
type ForgeConfig struct {
	Domains map[string]string `yaml:"domains,omitempty" json:"domains,omitempty"`
}

// Prefix returns the pipeline name prefix for this forge type.
func (ft ForgeType) Prefix() string {
	switch ft {
	case GitHub:
		return "gh-"
	case GitLab:
		return "gl-"
	case Bitbucket:
		return "bb-"
	case Gitea:
		return "gt-"
	default:
		return ""
	}
}

// CLITool returns the expected CLI tool name for this forge type.
func (ft ForgeType) CLITool() string {
	switch ft {
	case GitHub:
		return "gh"
	case GitLab:
		return "glab"
	case Bitbucket:
		return "bb"
	case Gitea:
		return "tea"
	default:
		return ""
	}
}

// ParseForgeType converts a string to a ForgeType, returning Unknown for unrecognized values.
func ParseForgeType(s string) ForgeType {
	switch ForgeType(s) {
	case GitHub, GitLab, Bitbucket, Gitea:
		return ForgeType(s)
	default:
		return Unknown
	}
}
