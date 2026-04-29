package sandbox

import (
	"context"
	"fmt"
	"log"
	"os/exec"
)

// RunShell executes a shell command line through the configured sandbox
// backend so that ad-hoc invocations from non-pipeline surfaces (the webui
// dashboard, future admin tooling) inherit the same isolation policy as
// pipeline-driven execution.
//
// The command is invoked as `sh -c <cmdLine>`. The chosen backend (None,
// Docker, or Bubblewrap) wraps the underlying exec.Cmd; the wrapper may add
// a Docker run prefix, FS bind mounts, or a passthrough as appropriate. A
// short audit log line is emitted regardless of backend so operators can
// correlate the invocation with state changes on disk.
//
// Returns combined stdout+stderr and any execution error. Backend
// initialization failures are returned as errors with no output.
func RunShell(ctx context.Context, cmdLine string, cfg Config) ([]byte, error) {
	if cmdLine == "" {
		return nil, fmt.Errorf("sandbox: empty command line")
	}

	sb, err := NewSandbox(cfg.Backend)
	if err != nil {
		return nil, fmt.Errorf("sandbox: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdLine)
	wrapped, err := sb.Wrap(ctx, cmd, cfg)
	if err != nil {
		return nil, fmt.Errorf("sandbox: wrap: %w", err)
	}

	// Audit trail: a single line is sufficient — credentials in cmdLine
	// would be a configuration smell, since RunShell is invoked from
	// trusted code paths (webui handlers, etc.) with operator-defined
	// commands stored in pipeline metadata.
	log.Printf("[sandbox.run] backend=%s cmd=%q", cfg.Backend, cmdLine)

	return wrapped.CombinedOutput()
}
