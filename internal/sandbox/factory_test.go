package sandbox

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestNewSandbox(t *testing.T) {
	tests := []struct {
		name       string
		backend    SandboxBackendType
		wantType   string
		wantErr    bool
		skipDocker bool
	}{
		{
			name:     "none returns NoneSandbox",
			backend:  SandboxBackendNone,
			wantType: "*sandbox.NoneSandbox",
		},
		{
			name:     "empty string returns NoneSandbox",
			backend:  "",
			wantType: "*sandbox.NoneSandbox",
		},
		{
			name:     "bubblewrap returns NoneSandbox passthrough",
			backend:  SandboxBackendBubblewrap,
			wantType: "*sandbox.NoneSandbox",
		},
		{
			name:       "docker returns DockerSandbox if available",
			backend:    SandboxBackendDocker,
			wantType:   "*sandbox.DockerSandbox",
			skipDocker: true,
		},
		{
			name:    "invalid backend returns error",
			backend: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipDocker {
				if _, err := exec.LookPath("docker"); err != nil {
					t.Skip("docker not available on PATH")
				}
			}

			s, err := NewSandbox(tt.backend)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotType := typeName(s)
			if gotType != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, gotType)
			}
		})
	}
}

func typeName(v any) string {
	return fmt.Sprintf("%T", v)
}
