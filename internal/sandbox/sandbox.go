package sandbox

import (
	"context"
	"os/exec"
)

// Sandbox abstracts sandbox implementations.
type Sandbox interface {
	// Wrap transforms an exec.Cmd to run inside the sandbox.
	Wrap(ctx context.Context, cmd *exec.Cmd, cfg Config) (*exec.Cmd, error)

	// Validate checks that the sandbox backend is available.
	Validate() error

	// Cleanup performs post-execution cleanup.
	Cleanup(ctx context.Context) error
}
