package sandbox

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestDockerSandbox_Wrap_SecurityFlags(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude", "--version")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := result.Args
	expectedFlags := []string{
		"--read-only",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--network=none",
	}
	for _, flag := range expectedFlags {
		if !slices.Contains(args, flag) {
			t.Errorf("missing security flag %q in args: %v", flag, args)
		}
	}
}

func TestDockerSandbox_Wrap_TmpfsMounts(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := strings.Join(result.Args, " ")
	expectedTmpfs := []string{
		"/tmp:rw,nosuid,nodev",
		"/var/run:rw,nosuid,nodev",
		"/home/wave:rw,nosuid,nodev",
	}
	for _, tmpfs := range expectedTmpfs {
		if !strings.Contains(args, tmpfs) {
			t.Errorf("missing tmpfs mount %q in args: %s", tmpfs, args)
		}
	}
}

func TestDockerSandbox_Wrap_UIDGIDFromConfig(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1234,
		HostGID: 5678,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := result.Args
	userIdx := slices.Index(args, "--user")
	if userIdx < 0 || userIdx+1 >= len(args) {
		t.Fatal("--user flag not found in args")
	}
	if args[userIdx+1] != "1234:5678" {
		t.Errorf("expected --user 1234:5678, got %s", args[userIdx+1])
	}
}

func TestDockerSandbox_Wrap_UIDGIDDefaultsToOS(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	expected := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	args := result.Args
	userIdx := slices.Index(args, "--user")
	if userIdx < 0 || userIdx+1 >= len(args) {
		t.Fatal("--user flag not found in args")
	}
	if args[userIdx+1] != expected {
		t.Errorf("expected --user %s, got %s", expected, args[userIdx+1])
	}
}

func TestDockerSandbox_Wrap_EnvPassthrough(t *testing.T) {
	// Set a test env var
	t.Setenv("WAVE_TEST_VAR", "test_value")

	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID:        1000,
		HostGID:        1000,
		EnvPassthrough: []string{"WAVE_TEST_VAR", "NONEXISTENT_VAR"},
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := result.Args
	// WAVE_TEST_VAR should be passed through
	found := false
	for _, arg := range args {
		if arg == "WAVE_TEST_VAR=test_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WAVE_TEST_VAR not found in passthrough env args")
	}

	// NONEXISTENT_VAR should NOT be passed through (not set in env)
	for _, arg := range args {
		if strings.HasPrefix(arg, "NONEXISTENT_VAR=") {
			t.Error("NONEXISTENT_VAR should not be passed through when not set")
		}
	}
}

func TestDockerSandbox_Wrap_StandardEnv(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := result.Args
	expectedEnvs := []string{
		"HOME=/home/wave",
		"TERM=xterm",
		"TMPDIR=/tmp",
	}
	for _, env := range expectedEnvs {
		found := false
		for i, arg := range args {
			if arg == "-e" && i+1 < len(args) && args[i+1] == env {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing standard env %q in args", env)
		}
	}
}

func TestDockerSandbox_Wrap_Mounts(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	cfg := Config{
		HostUID:       1000,
		HostGID:       1000,
		WorkspacePath: "/work/step1",
		ArtifactDir:   "/work/.agents/artifacts",
		OutputDir:     "/work/.agents/output",
		AdapterBinary: "/usr/bin/claude",
	}

	result, err := d.Wrap(context.Background(), cmd, cfg)
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	args := strings.Join(result.Args, " ")

	tests := []struct {
		name     string
		expected string
	}{
		{"workspace rw", "/work/step1:/work/step1:rw"},
		{"artifacts ro", "/work/.agents/artifacts:/work/.agents/artifacts:ro"},
		{"output rw", "/work/.agents/output:/work/.agents/output:rw"},
		{"adapter ro", "/usr/bin/claude:/usr/bin/claude:ro"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(args, tt.expected) {
				t.Errorf("missing mount %q in args: %s", tt.expected, args)
			}
		})
	}

	// Verify -w sets the working directory inside the container
	wIdx := slices.Index(result.Args, "-w")
	if wIdx < 0 || wIdx+1 >= len(result.Args) {
		t.Fatal("-w flag not found in args")
	}
	if result.Args[wIdx+1] != "/work/step1" {
		t.Errorf("expected -w /work/step1, got %s", result.Args[wIdx+1])
	}
}

func TestDockerSandbox_Wrap_DefaultImage(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude", "--version")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	// The default image should appear before the original command
	args := result.Args
	imgIdx := slices.Index(args, "ubuntu:24.04")
	if imgIdx < 0 {
		t.Fatal("default image ubuntu:24.04 not found in args")
	}
	// The original command should follow the image
	if imgIdx+1 >= len(args) || args[imgIdx+1] != "/usr/bin/claude" {
		t.Errorf("expected original command after image, got args: %v", args[imgIdx:])
	}
}

func TestDockerSandbox_Wrap_CustomImage(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID:     1000,
		HostGID:     1000,
		DockerImage: "alpine:3.19",
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	if !slices.Contains(result.Args, "alpine:3.19") {
		t.Error("custom image alpine:3.19 not found in args")
	}
	if slices.Contains(result.Args, "ubuntu:24.04") {
		t.Error("default image should not appear when custom image is set")
	}
}

func TestDockerSandbox_Wrap_PreservesStdio(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader("input")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = "/some/dir"
	cmd.Env = []string{"FOO=bar"}

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	if result.Dir != "/some/dir" {
		t.Errorf("Dir not preserved: got %s", result.Dir)
	}
	if result.Stdout != &stdout {
		t.Error("Stdout not preserved")
	}
	if result.Stderr != &stderr {
		t.Error("Stderr not preserved")
	}
	if result.Stdin == nil {
		t.Error("Stdin not preserved")
	}
	if len(result.Env) != 1 || result.Env[0] != "FOO=bar" {
		t.Errorf("Env not preserved: got %v", result.Env)
	}
}

func TestDockerSandbox_Wrap_OriginalCommandAndArgs(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude", "--model", "opus", "run")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	// Find the image in args, everything after should be the original command
	args := result.Args
	imgIdx := slices.Index(args, "ubuntu:24.04")
	if imgIdx < 0 {
		t.Fatal("image not found in args")
	}

	trailing := args[imgIdx+1:]
	expected := []string{"/usr/bin/claude", "--model", "opus", "run"}
	if len(trailing) != len(expected) {
		t.Fatalf("expected trailing args %v, got %v", expected, trailing)
	}
	for i, exp := range expected {
		if trailing[i] != exp {
			t.Errorf("trailing arg[%d]: expected %q, got %q", i, exp, trailing[i])
		}
	}
}

func TestDockerSandbox_Wrap_ConcurrentInstances(t *testing.T) {
	d := newTestDockerSandbox()
	const n = 10

	var wg sync.WaitGroup
	results := make([]*exec.Cmd, n)
	errors := make([]error, n)

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := exec.Command("/usr/bin/claude")
			cfg := Config{
				HostUID:       1000,
				HostGID:       1000,
				WorkspacePath: "/work/step" + strconv.Itoa(idx),
			}
			results[idx], errors[idx] = d.Wrap(context.Background(), cmd, cfg)
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errors[i] != nil {
			t.Fatalf("concurrent Wrap[%d] returned error: %v", i, errors[i])
		}
		expectedMount := "/work/step" + strconv.Itoa(i) + ":/work/step" + strconv.Itoa(i) + ":rw"
		args := strings.Join(results[i].Args, " ")
		if !strings.Contains(args, expectedMount) {
			t.Errorf("concurrent Wrap[%d] missing expected mount %q", i, expectedMount)
		}
	}

	// Verify all commands are independent (different pointers)
	for i := range n {
		for j := i + 1; j < n; j++ {
			if results[i] == results[j] {
				t.Errorf("concurrent Wrap[%d] and Wrap[%d] returned same cmd pointer", i, j)
			}
		}
	}
}

func TestDockerSandbox_Cleanup(t *testing.T) {
	d := newTestDockerSandbox()
	if err := d.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
}

func TestDockerSandbox_Wrap_NoMountsWhenEmpty(t *testing.T) {
	d := newTestDockerSandbox()
	cmd := exec.Command("/usr/bin/claude")

	result, err := d.Wrap(context.Background(), cmd, Config{
		HostUID: 1000,
		HostGID: 1000,
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}

	// With empty config paths, there should be no -v flags
	args := result.Args
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			t.Errorf("unexpected -v mount: %s", args[i+1])
		}
	}
}
