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
	os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
adapters:
  claude:
    binary: echo
    mode: headless
runtime:
  workspace_root: .wave/workspaces
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	os.MkdirAll(pipelinesDir, 0755)

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
	os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .wave/workspaces
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
	os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
adapters:
  claude:
    binary: nonexistent-binary
    mode: headless
runtime:
  workspace_root: .wave/workspaces
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
	os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .wave/workspaces
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
	os.WriteFile(manifestPath, []byte(`apiVersion: wave/v1
kind: Manifest
metadata:
  name: test-project
runtime:
  workspace_root: .wave/workspaces
`), 0644)

	pipelinesDir := filepath.Join(tmp, "pipelines")
	os.MkdirAll(pipelinesDir, 0755)
	os.WriteFile(filepath.Join(pipelinesDir, "test.yaml"), []byte(`kind: Pipeline
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
