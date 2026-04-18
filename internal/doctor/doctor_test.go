package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/forge"
)

func TestRunChecks_AllHealthy(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
adapters:
  claude:
    binary: echo
    mode: headless
runtime:
  workspace_root: .agents/workspaces
ontology:
  telos: "Test project purpose"
  contexts:
    - name: core
      description: Core functionality
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	_ = os.MkdirAll(pipelinesDir, 0755)

	// Create context skill so ontology check passes
	skillDir := filepath.Join(tmp, "skills", "wave-ctx-core")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Core Context\n"), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: pipelinesDir,
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{
				Type:           forge.ForgeGitHub,
				Host:           "github.com",
				Owner:          "test",
				Repo:           "repo",
				CLITool:        "gh",
				PipelinePrefix: "gh",
			}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	if report.Summary != StatusOK {
		t.Errorf("expected summary OK, got %v", report.Summary)
		for _, r := range report.Results {
			if r.Status != StatusOK {
				t.Logf("  %s: %s (%s)", r.Name, r.Message, r.Status)
			}
		}
	}

	if report.ForgeInfo == nil {
		t.Error("expected ForgeInfo to be set")
	} else if report.ForgeInfo.Type != forge.ForgeGitHub {
		t.Errorf("expected GitHub forge, got %v", report.ForgeInfo.Type)
	}
}

func TestRunChecks_MissingManifest(t *testing.T) {
	tmp := t.TempDir()

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: filepath.Join(tmp, "nonexistent.yaml"),
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return false
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	if report.Summary != StatusErr {
		t.Errorf("expected summary Err, got %v", report.Summary)
	}

	// Should have errors for onboarding and manifest
	var foundManifestErr, foundOnboardingErr bool
	for _, r := range report.Results {
		if r.Name == "Manifest Valid" && r.Status == StatusErr {
			foundManifestErr = true
		}
		if r.Name == "Wave Initialized" && r.Status == StatusErr {
			foundOnboardingErr = true
		}
	}
	if !foundManifestErr {
		t.Error("expected manifest error in results")
	}
	if !foundOnboardingErr {
		t.Error("expected onboarding error in results")
	}
}

func TestRunChecks_NoGit(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			if name == "git" {
				return fmt.Errorf("not a git repo")
			}
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	var foundGitErr bool
	for _, r := range report.Results {
		if r.Name == "Git Repository" && r.Status == StatusErr {
			foundGitErr = true
		}
	}
	if !foundGitErr {
		t.Error("expected git error in results")
	}
}

func TestRunChecks_MissingAdapter(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
adapters:
  claude:
    binary: nonexistent-binary
    mode: headless
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			if file == "nonexistent-binary" {
				return "", fmt.Errorf("not found")
			}
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	var foundAdapterErr bool
	for _, r := range report.Results {
		if r.Name == "Adapter: claude" && r.Status == StatusErr {
			foundAdapterErr = true
		}
	}
	if !foundAdapterErr {
		t.Error("expected adapter error in results")
	}
}

func TestRunChecks_ForgeWithMissingCLI(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			if file == "gh" {
				return "", fmt.Errorf("not found")
			}
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{
				Type:           forge.ForgeGitHub,
				Host:           "github.com",
				Owner:          "test",
				Repo:           "repo",
				CLITool:        "gh",
				PipelinePrefix: "gh",
			}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	var foundCLIWarn bool
	for _, r := range report.Results {
		if r.Name == "Forge CLI: gh" && r.Status == StatusWarn {
			foundCLIWarn = true
		}
	}
	if !foundCLIWarn {
		t.Error("expected forge CLI warning in results")
	}
}

func TestRunChecks_RequiredTools(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	_ = os.MkdirAll(pipelinesDir, 0755)
	_ = os.WriteFile(filepath.Join(pipelinesDir, "test.yaml"), []byte(`kind: Pipeline
metadata:
  name: test
requires:
  tools:
    - jq
    - curl
steps:
  - id: step1
    persona: navigator
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: pipelinesDir,
		LookPath: func(file string) (string, error) {
			if file == "jq" {
				return "", fmt.Errorf("not found")
			}
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	var foundJQErr, foundCurlOK bool
	for _, r := range report.Results {
		if r.Name == "Tool: jq" && r.Status == StatusErr {
			foundJQErr = true
		}
		if r.Name == "Tool: curl" && r.Status == StatusOK {
			foundCurlOK = true
		}
	}
	if !foundJQErr {
		t.Error("expected jq tool error")
	}
	if !foundCurlOK {
		t.Error("expected curl tool OK")
	}
}

func TestCheckAdapterRegistry_WithAdapters(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
  codex:
    binary: codex
    mode: headless
  gemini:
    binary: gemini
    mode: headless
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	var found bool
	for _, r := range report.Results {
		if r.Name == "Adapter Registry" {
			found = true
			if r.Status != StatusOK {
				t.Errorf("expected StatusOK, got %v", r.Status)
			}
			if r.Category != "capabilities" {
				t.Errorf("expected category 'capabilities', got %q", r.Category)
			}
			// All three adapters should be mentioned
			for _, name := range []string{"claude", "codex", "gemini"} {
				if !contains(r.Message, name) {
					t.Errorf("expected message to contain %q, got %q", name, r.Message)
				}
			}
		}
	}
	if !found {
		t.Error("expected Adapter Registry check in results")
	}
}

func TestCheckAdapterRegistry_NoAdapters(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	for _, r := range report.Results {
		if r.Name == "Adapter Registry" {
			if r.Status != StatusOK {
				t.Errorf("expected StatusOK, got %v", r.Status)
			}
			if !contains(r.Message, "No adapters registered") {
				t.Errorf("expected 'No adapters registered', got %q", r.Message)
			}
			return
		}
	}
	t.Error("expected Adapter Registry check in results")
}

func TestCheckRetryPolicies_AllNamed(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	_ = os.MkdirAll(pipelinesDir, 0755)
	_ = os.WriteFile(filepath.Join(pipelinesDir, "test.yaml"), []byte(`kind: Pipeline
metadata:
  name: test
steps:
  - id: step1
    persona: navigator
    retry:
      policy: standard
      max_attempts: 2
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: pipelinesDir,
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	for _, r := range report.Results {
		if r.Name == "Retry Policies" {
			if r.Status != StatusOK {
				t.Errorf("expected StatusOK, got %v", r.Status)
			}
			if !contains(r.Message, "named policies") {
				t.Errorf("expected message about named policies, got %q", r.Message)
			}
			return
		}
	}
	t.Error("expected Retry Policies check in results")
}

func TestCheckRetryPolicies_RawMaxAttempts(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	_ = os.MkdirAll(pipelinesDir, 0755)
	_ = os.WriteFile(filepath.Join(pipelinesDir, "test.yaml"), []byte(`kind: Pipeline
metadata:
  name: test
steps:
  - id: step1
    persona: navigator
    retry:
      max_attempts: 3
  - id: step2
    persona: navigator
    retry:
      policy: standard
      max_attempts: 2
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: pipelinesDir,
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	for _, r := range report.Results {
		if r.Name == "Retry Policies" {
			if r.Status != StatusWarn {
				t.Errorf("expected StatusWarn, got %v", r.Status)
			}
			if !contains(r.Message, "raw max_attempts") {
				t.Errorf("expected warning about raw max_attempts, got %q", r.Message)
			}
			return
		}
	}
	t.Error("expected Retry Policies check in results")
}

func TestCheckEngineCapabilities(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "wave.yaml")
	_ = os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .agents/workspaces
`), 0644)

	report, err := RunChecks(context.Background(), Options{
		ManifestPath: manifestPath,
		WaveDir:      tmp,
		PipelinesDir: filepath.Join(tmp, "pipelines"),
		LookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		RunCmd: func(name string, args ...string) error {
			return nil
		},
		DetectForge: func() (forge.ForgeInfo, error) {
			return forge.ForgeInfo{Type: forge.ForgeUnknown}, nil
		},
		CheckOnboarded: func(waveDir string) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	for _, r := range report.Results {
		if r.Name == "Engine Capabilities" {
			if r.Status != StatusOK {
				t.Errorf("expected StatusOK, got %v", r.Status)
			}
			if r.Category != "capabilities" {
				t.Errorf("expected category 'capabilities', got %q", r.Category)
			}
			// Verify all epic #589 capabilities are listed
			for _, cap := range []string{"graph loops", "gates", "hooks", "retro", "fork/rewind", "llm_judge", "thread continuity", "sub-pipelines"} {
				if !contains(r.Message, cap) {
					t.Errorf("expected message to contain %q, got %q", cap, r.Message)
				}
			}
			return
		}
	}
	t.Error("expected Engine Capabilities check in results")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusOK, "ok"},
		{StatusWarn, "warn"},
		{StatusErr, "error"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}
