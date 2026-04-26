package checks

import (
	"errors"
	"testing"
)

func TestDockerDaemon_BinaryNotFound(t *testing.T) {
	lookPath := func(name string) (string, error) {
		return "", errors.New("not found")
	}
	runCmd := func(name string, args ...string) error {
		t.Fatalf("runCmd should not be called when binary missing; got %s %v", name, args)
		return nil
	}

	got := DockerDaemon(runCmd, lookPath)
	if got.BinaryFound {
		t.Errorf("BinaryFound = true, want false")
	}
	if got.DaemonUp {
		t.Errorf("DaemonUp = true, want false")
	}
}

func TestDockerDaemon_BinaryFoundDaemonDown(t *testing.T) {
	lookPath := func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	runCmd := func(name string, args ...string) error {
		return errors.New("cannot connect to docker daemon")
	}

	got := DockerDaemon(runCmd, lookPath)
	if !got.BinaryFound {
		t.Errorf("BinaryFound = false, want true")
	}
	if got.DaemonUp {
		t.Errorf("DaemonUp = true, want false")
	}
}

func TestDockerDaemon_DaemonUp(t *testing.T) {
	lookPath := func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	runCmd := func(name string, args ...string) error {
		if name != "docker" || len(args) != 1 || args[0] != "info" {
			t.Errorf("runCmd called with unexpected args: %s %v", name, args)
		}
		return nil
	}

	got := DockerDaemon(runCmd, lookPath)
	if !got.BinaryFound || !got.DaemonUp {
		t.Errorf("got %+v, want BinaryFound=true DaemonUp=true", got)
	}
}

func TestDockerDaemon_NilDefaults(t *testing.T) {
	// nil lookPath/runCmd should not panic; behaviour depends on host.
	// Just assert no panic and field consistency.
	got := DockerDaemon(nil, nil)
	if !got.BinaryFound && got.DaemonUp {
		t.Errorf("invariant violated: DaemonUp=true with BinaryFound=false: %+v", got)
	}
}

func TestSkill_EmptyCheck(t *testing.T) {
	runCmd := func(name string, args ...string) error {
		t.Fatalf("runCmd should not be called for empty check")
		return nil
	}
	got := Skill(runCmd, "")
	if got.HasCheck {
		t.Errorf("HasCheck = true, want false")
	}
	if got.Installed {
		t.Errorf("Installed = true, want false")
	}
}

func TestSkill_CheckPasses(t *testing.T) {
	var called bool
	runCmd := func(name string, args ...string) error {
		called = true
		if name != "sh" || len(args) != 2 || args[0] != "-c" || args[1] != "true" {
			t.Errorf("unexpected runCmd args: %s %v", name, args)
		}
		return nil
	}
	got := Skill(runCmd, "true")
	if !called {
		t.Error("expected runCmd to be called")
	}
	if !got.HasCheck || !got.Installed {
		t.Errorf("got %+v, want HasCheck=true Installed=true", got)
	}
}

func TestSkill_CheckFails(t *testing.T) {
	runCmd := func(name string, args ...string) error {
		return errors.New("not installed")
	}
	got := Skill(runCmd, "false")
	if !got.HasCheck {
		t.Errorf("HasCheck = false, want true")
	}
	if got.Installed {
		t.Errorf("Installed = true, want false")
	}
}

func TestSkillInstalled_DelegatesToSkill(t *testing.T) {
	runCmd := func(name string, args ...string) error { return nil }
	if !SkillInstalled(runCmd, "true") {
		t.Error("expected installed=true for passing check")
	}
	if SkillInstalled(runCmd, "") {
		t.Error("expected installed=false for empty check")
	}
}

func TestSkillInstalledWithToolBin_PrimaryPasses(t *testing.T) {
	runCmd := func(name string, args ...string) error { return nil }
	runCmdEnv := func(env []string, name string, args ...string) error {
		t.Fatal("runCmdEnv should not be called when primary check passes")
		return nil
	}
	if !SkillInstalledWithToolBin(runCmd, runCmdEnv, "true") {
		t.Error("expected true when primary check passes")
	}
}

func TestSkillInstalledWithToolBin_FallbackPasses(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("PATH", "/usr/bin")

	primaryCalls, envCalls := 0, 0
	runCmd := func(name string, args ...string) error {
		primaryCalls++
		return errors.New("not on PATH")
	}
	runCmdEnv := func(env []string, name string, args ...string) error {
		envCalls++
		want := "PATH=/home/testuser/.local/bin:/usr/bin"
		if len(env) != 1 || env[0] != want {
			t.Errorf("env = %v, want [%q]", env, want)
		}
		return nil
	}
	if !SkillInstalledWithToolBin(runCmd, runCmdEnv, "mybin --version") {
		t.Error("expected true when fallback succeeds")
	}
	if primaryCalls != 1 || envCalls != 1 {
		t.Errorf("calls primary=%d env=%d, want 1/1", primaryCalls, envCalls)
	}
}

func TestSkillInstalledWithToolBin_BothFail(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")

	runCmd := func(name string, args ...string) error { return errors.New("fail") }
	runCmdEnv := func(env []string, name string, args ...string) error { return errors.New("fail") }

	if SkillInstalledWithToolBin(runCmd, runCmdEnv, "missing") {
		t.Error("expected false when both attempts fail")
	}
}

func TestSkillInstalledWithToolBin_EmptyHome(t *testing.T) {
	t.Setenv("HOME", "")
	envCalled := false
	runCmd := func(name string, args ...string) error { return errors.New("fail") }
	runCmdEnv := func(env []string, name string, args ...string) error {
		envCalled = true
		return nil
	}
	if SkillInstalledWithToolBin(runCmd, runCmdEnv, "missing") {
		t.Error("expected false when HOME unset")
	}
	if envCalled {
		t.Error("runCmdEnv should not be called when HOME is empty")
	}
}

func TestSkillInstalledWithToolBin_EmptyCheck(t *testing.T) {
	if SkillInstalledWithToolBin(nil, nil, "") {
		t.Error("expected false for empty check")
	}
}

func TestDefaultRunCmd_Smoke(t *testing.T) {
	// `true` is on every POSIX system; verifies the default exec path executes
	// without panicking and returns nil for a successful command.
	if err := DefaultRunCmd("sh", "-c", "true"); err != nil {
		t.Errorf("DefaultRunCmd(sh -c true) returned error: %v", err)
	}
	if err := DefaultRunCmd("sh", "-c", "exit 1"); err == nil {
		t.Error("expected non-nil error for failing command")
	}
}

func TestDefaultRunCmdWithEnv_PassesEnv(t *testing.T) {
	// Asserts that injected env vars reach the subprocess.
	err := DefaultRunCmdWithEnv([]string{"WAVE_CHECKS_TEST=1"}, "sh", "-c", "[ \"$WAVE_CHECKS_TEST\" = 1 ]")
	if err != nil {
		t.Errorf("expected env var visible to subprocess, got error: %v", err)
	}
}
