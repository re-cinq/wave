package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// DockerSandbox executes adapter subprocesses inside Docker containers.
type DockerSandbox struct {
	dockerPath string
}

// NewDockerSandbox creates a Docker sandbox, resolving the docker binary path.
func NewDockerSandbox() (*DockerSandbox, error) {
	path, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker binary not found on PATH: %w", err)
	}
	return &DockerSandbox{dockerPath: path}, nil
}

// newTestDockerSandbox creates a DockerSandbox with a fixed binary path for testing
// without requiring docker on the system.
func newTestDockerSandbox() *DockerSandbox {
	return &DockerSandbox{dockerPath: "/usr/bin/docker"}
}

func (d *DockerSandbox) Validate() error {
	cmd := exec.Command(d.dockerPath, "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon not available: %w\n  Hint: Start Docker with: systemctl start docker (Linux) or open Docker Desktop (macOS/Windows)", err)
	}
	return nil
}

func (d *DockerSandbox) Wrap(ctx context.Context, cmd *exec.Cmd, cfg Config) (*exec.Cmd, error) {
	args := []string{
		"run", "--rm",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,nodev",
		"--tmpfs", "/var/run:rw,nosuid,nodev",
		"--tmpfs", "/home/wave:rw,nosuid,nodev",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--network=none",
	}

	// UID/GID mapping
	uid := cfg.HostUID
	gid := cfg.HostGID
	if uid == 0 {
		uid = os.Getuid()
	}
	if gid == 0 {
		gid = os.Getgid()
	}
	args = append(args, "--user", strconv.Itoa(uid)+":"+strconv.Itoa(gid))

	// Environment: HOME and standard vars
	args = append(args, "-e", "HOME=/home/wave")
	args = append(args, "-e", "TERM=xterm")
	args = append(args, "-e", "TMPDIR=/tmp")

	// Passthrough environment variables
	for _, envName := range cfg.EnvPassthrough {
		if val, ok := os.LookupEnv(envName); ok {
			args = append(args, "-e", envName+"="+val)
		}
	}

	// Workspace bind mount
	if cfg.WorkspacePath != "" {
		args = append(args, "-v", cfg.WorkspacePath+":"+cfg.WorkspacePath+":rw")
		args = append(args, "-w", cfg.WorkspacePath)
	}

	// Artifact directories
	if cfg.ArtifactDir != "" {
		args = append(args, "-v", cfg.ArtifactDir+":"+cfg.ArtifactDir+":ro")
	}
	if cfg.OutputDir != "" {
		args = append(args, "-v", cfg.OutputDir+":"+cfg.OutputDir+":rw")
	}

	// Adapter binary bind mount
	if cfg.AdapterBinary != "" {
		args = append(args, "-v", cfg.AdapterBinary+":"+cfg.AdapterBinary+":ro")
	}

	// Image
	image := cfg.DockerImage
	if image == "" {
		image = "ubuntu:24.04"
	}
	args = append(args, image)

	// Original command and arguments
	args = append(args, cmd.Path)
	args = append(args, cmd.Args[1:]...)

	dockerCmd := exec.CommandContext(ctx, d.dockerPath, args...)
	dockerCmd.Dir = cmd.Dir
	dockerCmd.Env = cmd.Env
	dockerCmd.Stdin = cmd.Stdin
	dockerCmd.Stdout = cmd.Stdout
	dockerCmd.Stderr = cmd.Stderr

	return dockerCmd, nil
}

func (d *DockerSandbox) Cleanup(_ context.Context) error {
	// Container is --rm, so no cleanup needed for basic case.
	// Future: clean up Docker networks for proxy sidecar.
	return nil
}
