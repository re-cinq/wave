package sandbox

// SandboxBackendType enumerates the supported sandbox backends.
type SandboxBackendType string

const (
	SandboxBackendNone       SandboxBackendType = "none"
	SandboxBackendDocker     SandboxBackendType = "docker"
	SandboxBackendBubblewrap SandboxBackendType = "bubblewrap"
)

// Config holds the merged sandbox configuration for a single step execution.
type Config struct {
	Backend        SandboxBackendType
	DockerImage    string
	AllowedDomains []string
	EnvPassthrough []string
	WorkspacePath  string
	ArtifactDir    string
	OutputDir      string
	HostUID        int
	HostGID        int
	AdapterBinary  string
	Debug          bool
}
