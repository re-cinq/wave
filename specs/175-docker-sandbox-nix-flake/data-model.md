# Data Model: Nix Flake Packaging and Docker-Based Sandbox

**Branch**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16

## Entity: SandboxBackend (Enumeration)

Represents the sandbox execution strategy.

```go
// SandboxBackendType enumerates the supported sandbox backends.
type SandboxBackendType string

const (
    SandboxBackendNone       SandboxBackendType = "none"
    SandboxBackendDocker     SandboxBackendType = "docker"
    SandboxBackendBubblewrap SandboxBackendType = "bubblewrap"
)
```

**Source**: `internal/sandbox/types.go` (new file)

## Entity: RuntimeSandbox (Modified)

Extends the existing manifest type with Docker-specific fields.

```go
// RuntimeSandbox in internal/manifest/types.go
type RuntimeSandbox struct {
    Enabled               bool               `yaml:"enabled"`
    Backend               string             `yaml:"backend,omitempty"`      // NEW: "docker", "bubblewrap", "none"
    DockerImage           string             `yaml:"docker_image,omitempty"` // NEW: default "ubuntu:24.04"
    DefaultAllowedDomains []string           `yaml:"default_allowed_domains,omitempty"`
    EnvPassthrough        []string           `yaml:"env_passthrough,omitempty"`
}

// ResolveBackend returns the effective sandbox backend, considering
// both the new Backend field and the legacy Enabled boolean.
func (s *RuntimeSandbox) ResolveBackend() SandboxBackendType {
    if s.Backend != "" {
        return SandboxBackendType(s.Backend)
    }
    if s.Enabled {
        return SandboxBackendBubblewrap
    }
    return SandboxBackendNone
}

// GetDockerImage returns the Docker image to use, defaulting to ubuntu:24.04.
func (s *RuntimeSandbox) GetDockerImage() string {
    if s.DockerImage != "" {
        return s.DockerImage
    }
    return "ubuntu:24.04"
}
```

## Entity: SandboxConfig (New)

Merged configuration from manifest runtime and persona overrides, passed to the sandbox runner.

```go
// SandboxConfig in internal/sandbox/config.go
type SandboxConfig struct {
    Backend        SandboxBackendType
    DockerImage    string
    AllowedDomains []string
    EnvPassthrough []string
    WorkspacePath  string
    ArtifactDir    string   // e.g., ".wave/artifacts"
    OutputDir      string   // e.g., ".wave/output"
    HostUID        int
    HostGID        int
    AdapterBinary  string   // Resolved path to adapter binary on host
    Debug          bool
}
```

## Entity: Sandbox (Interface)

Abstraction over sandbox implementations.

```go
// Sandbox in internal/sandbox/sandbox.go
type Sandbox interface {
    // Wrap transforms an exec.Cmd to run inside the sandbox.
    // For Docker: replaces the command with `docker run ... <original cmd>`.
    // For None: returns the command unchanged.
    Wrap(ctx context.Context, cmd *exec.Cmd, cfg SandboxConfig) (*exec.Cmd, error)

    // Validate checks that the sandbox backend is available and functional.
    // Called during preflight.
    Validate() error

    // Cleanup performs any post-execution cleanup (e.g., remove Docker network).
    Cleanup(ctx context.Context) error
}
```

## Entity: DockerSandbox (New)

Docker-based sandbox implementation.

```go
// DockerSandbox in internal/sandbox/docker.go
type DockerSandbox struct {
    dockerPath string // Resolved path to docker binary
}

// NewDockerSandbox creates a Docker sandbox, resolving the docker binary path.
func NewDockerSandbox() (*DockerSandbox, error) {
    path, err := exec.LookPath("docker")
    if err != nil {
        return nil, fmt.Errorf("docker binary not found on PATH: %w", err)
    }
    return &DockerSandbox{dockerPath: path}, nil
}
```

## Entity: NoneSandbox (New)

No-op sandbox implementation (passthrough).

```go
// NoneSandbox in internal/sandbox/none.go
type NoneSandbox struct{}

func (n *NoneSandbox) Wrap(ctx context.Context, cmd *exec.Cmd, cfg SandboxConfig) (*exec.Cmd, error) {
    return cmd, nil // passthrough
}

func (n *NoneSandbox) Validate() error { return nil }
func (n *NoneSandbox) Cleanup(ctx context.Context) error { return nil }
```

## Entity: AdapterRunConfig (Modified)

Extends existing config to carry sandbox backend type.

```go
// In internal/adapter/adapter.go — add to AdapterRunConfig:
SandboxBackend string // "docker", "bubblewrap", "none" — resolved from manifest
DockerImage    string // Docker image to use when SandboxBackend == "docker"
```

## Entity: PreflightCheck for Docker (Extension)

```go
// In internal/preflight/preflight.go — new method:
func (c *Checker) CheckDockerDaemon() Result {
    // 1. Check docker binary on PATH
    // 2. Run `docker info` to verify daemon
    // 3. Return Result with platform-specific recovery hints
}
```

## Relationship Diagram

```
wave.yaml
  └── runtime.sandbox
        ├── backend: "docker" | "bubblewrap" | "none"
        ├── docker_image: "ubuntu:24.04"
        ├── default_allowed_domains: [...]
        └── env_passthrough: [...]

Persona
  └── sandbox
        └── allowed_domains: [...]  (overrides runtime defaults)

Pipeline Executor
  └── buildAdapterConfig()
        ├── resolves SandboxBackend from RuntimeSandbox
        ├── merges persona + runtime sandbox config
        └── passes to AdapterRunConfig

AdapterRunner.Run()
  └── Sandbox.Wrap()
        ├── DockerSandbox: docker run --rm --read-only ... <cmd>
        ├── BubblewrapSandbox: (existing flake.nix behavior, future)
        └── NoneSandbox: passthrough
```

## Integration Points

| Component | File | Change Type |
|-----------|------|-------------|
| Manifest types | `internal/manifest/types.go` | Modify `RuntimeSandbox` |
| Manifest parser tests | `internal/manifest/parser_test.go` | Add tests for new fields |
| Sandbox package | `internal/sandbox/` | New package |
| Adapter config | `internal/adapter/adapter.go` | Add fields to `AdapterRunConfig` |
| Pipeline executor | `internal/pipeline/executor.go` | Wire sandbox backend resolution |
| Preflight checker | `internal/preflight/preflight.go` | Add Docker daemon check |
| Flake.nix | `flake.nix` | Fix vendorHash, verify `nix flake check` |
