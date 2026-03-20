// Package workspace manages ephemeral isolated execution environments for
// Wave pipeline steps. It creates sandboxed workspace directories, injects
// artifacts from prior steps, and handles cleanup. The package supports
// configurable mounts with readonly and readwrite modes, recursive file
// copying with intelligent skipping of large files and system directories,
// symbolic link resolution with path traversal prevention, and workspace
// discovery utilities for retention policies.
package workspace
