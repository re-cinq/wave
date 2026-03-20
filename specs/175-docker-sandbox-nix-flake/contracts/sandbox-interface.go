//go:build ignore

// Package sandbox defines the contract interface for sandbox backends.
// This file serves as the API contract — implementations must satisfy this interface.
package sandbox

import (
	"context"
	"os/exec"
)

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

// Sandbox abstracts sandbox implementations.
type Sandbox interface {
	// Wrap transforms an exec.Cmd to run inside the sandbox.
	Wrap(ctx context.Context, cmd *exec.Cmd, cfg Config) (*exec.Cmd, error)

	// Validate checks that the sandbox backend is available.
	Validate() error

	// Cleanup performs post-execution cleanup.
	Cleanup(ctx context.Context) error
}
