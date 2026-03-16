package sandbox

import (
	"context"
	"os/exec"
)

// NoneSandbox is a passthrough sandbox that does not isolate execution.
type NoneSandbox struct{}

func (n *NoneSandbox) Wrap(_ context.Context, cmd *exec.Cmd, _ Config) (*exec.Cmd, error) {
	return cmd, nil
}

func (n *NoneSandbox) Validate() error { return nil }

func (n *NoneSandbox) Cleanup(_ context.Context) error { return nil }
