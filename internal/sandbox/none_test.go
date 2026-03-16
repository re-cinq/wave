package sandbox

import (
	"context"
	"os/exec"
	"testing"
)

func TestNoneSandbox_Wrap(t *testing.T) {
	s := &NoneSandbox{}
	original := exec.Command("echo", "hello")
	original.Dir = "/tmp"

	result, err := s.Wrap(context.Background(), original, Config{})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}
	if result != original {
		t.Error("Wrap should return the same cmd pointer")
	}
	if result.Dir != "/tmp" {
		t.Error("Wrap should not modify cmd fields")
	}
}

func TestNoneSandbox_Validate(t *testing.T) {
	s := &NoneSandbox{}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestNoneSandbox_Cleanup(t *testing.T) {
	s := &NoneSandbox{}
	if err := s.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
}
