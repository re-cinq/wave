package sandbox

import "fmt"

// NewSandbox creates a Sandbox implementation for the given backend type.
func NewSandbox(backend SandboxBackendType) (Sandbox, error) {
	switch backend {
	case SandboxBackendNone, "":
		return &NoneSandbox{}, nil
	case SandboxBackendDocker:
		return newDockerSandbox()
	case SandboxBackendBubblewrap:
		// Bubblewrap is handled by the Nix flake dev shell, not this package.
		return &NoneSandbox{}, nil
	default:
		return nil, fmt.Errorf("unknown sandbox backend: %q", backend)
	}
}
