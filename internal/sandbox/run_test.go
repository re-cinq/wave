package sandbox

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunShellNoneBackend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := RunShell(ctx, "echo hello-sandbox", Config{Backend: SandboxBackendNone})
	if err != nil {
		t.Fatalf("RunShell: %v", err)
	}
	if !strings.Contains(string(out), "hello-sandbox") {
		t.Errorf("expected output to contain hello-sandbox, got %q", out)
	}
}

func TestRunShellDefaultBackend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty backend should fall through to NoneSandbox.
	out, err := RunShell(ctx, "echo default-fallback", Config{})
	if err != nil {
		t.Fatalf("RunShell: %v", err)
	}
	if !strings.Contains(string(out), "default-fallback") {
		t.Errorf("expected default-fallback, got %q", out)
	}
}

func TestRunShellEmptyCommand(t *testing.T) {
	_, err := RunShell(context.Background(), "", Config{Backend: SandboxBackendNone})
	if err == nil {
		t.Fatal("expected error on empty command")
	}
}

func TestRunShellPropagatesExitError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := RunShell(ctx, "exit 7", Config{Backend: SandboxBackendNone})
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}
	// Output is allowed to be empty for `exit`.
	_ = out
}

func TestRunShellUnknownBackend(t *testing.T) {
	_, err := RunShell(context.Background(), "echo x", Config{Backend: SandboxBackendType("nonsense")})
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
