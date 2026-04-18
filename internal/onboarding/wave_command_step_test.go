package onboarding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWaveCommandStep_Name(t *testing.T) {
	step := &WaveCommandStep{}
	if step.Name() != "Wave Command Registration" {
		t.Errorf("Name() = %q, want %q", step.Name(), "Wave Command Registration")
	}
}

func TestWaveCommandStep_Run(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WizardConfig
		wantErr bool
	}{
		{
			name: "generates command file with default WaveDir",
			cfg: WizardConfig{
				WaveDir: filepath.Join(t.TempDir(), ".agents"),
			},
		},
		{
			name: "generates command file non-interactive",
			cfg: WizardConfig{
				WaveDir:     filepath.Join(t.TempDir(), ".agents"),
				Interactive: false,
			},
		},
		{
			name: "generates command file interactive",
			cfg: WizardConfig{
				WaveDir:     filepath.Join(t.TempDir(), ".agents"),
				Interactive: true,
			},
		},
		{
			name: "generates command file on reconfigure",
			cfg: WizardConfig{
				WaveDir:     filepath.Join(t.TempDir(), ".agents"),
				Reconfigure: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &WaveCommandStep{}
			result, err := step.Run(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Verify result
			if result == nil {
				t.Fatal("Run() returned nil result")
			}
			if v, ok := result.Data["wave_command_generated"].(bool); !ok || !v {
				t.Error("expected wave_command_generated=true in result data")
			}

			// Verify file was created
			baseDir := filepath.Dir(tt.cfg.WaveDir)
			commandFile := filepath.Join(baseDir, ".claude", "commands", "wave.md")
			content, err := os.ReadFile(commandFile)
			if err != nil {
				t.Fatalf("failed to read generated file: %v", err)
			}

			contentStr := string(content)

			// Verify YAML frontmatter
			if !strings.HasPrefix(contentStr, "---\n") {
				t.Error("command file missing YAML frontmatter opening ---")
			}
			if !strings.Contains(contentStr, "description:") {
				t.Error("command file missing description in frontmatter")
			}

			// Verify $ARGUMENTS placeholder
			if !strings.Contains(contentStr, "$ARGUMENTS") {
				t.Error("command file missing $ARGUMENTS placeholder")
			}

			// Verify subcommand references
			for _, cmd := range []string{"wave run", "wave list", "wave logs"} {
				if !strings.Contains(contentStr, cmd) {
					t.Errorf("command file missing reference to %q", cmd)
				}
			}

			// Verify status subcommand
			if !strings.Contains(contentStr, "status") {
				t.Error("command file missing reference to status subcommand")
			}
		})
	}
}

func TestWaveCommandStep_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := WizardConfig{
		WaveDir: filepath.Join(tmpDir, ".agents"),
	}

	step := &WaveCommandStep{}

	// First run
	_, err := step.Run(&cfg)
	if err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	commandFile := filepath.Join(tmpDir, ".claude", "commands", "wave.md")
	first, err := os.ReadFile(commandFile)
	if err != nil {
		t.Fatalf("failed to read after first run: %v", err)
	}

	// Second run (idempotent)
	_, err = step.Run(&cfg)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	second, err := os.ReadFile(commandFile)
	if err != nil {
		t.Fatalf("failed to read after second run: %v", err)
	}

	if string(first) != string(second) {
		t.Error("idempotency failed: file content differs between runs")
	}
}

func TestWaveCommandStep_CustomWaveDir(t *testing.T) {
	tmpDir := t.TempDir()
	customWaveDir := filepath.Join(tmpDir, "custom", "path", ".agents")

	cfg := WizardConfig{
		WaveDir: customWaveDir,
	}

	step := &WaveCommandStep{}
	_, err := step.Run(&cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// The command file should be at custom/path/.claude/commands/wave.md
	expectedDir := filepath.Join(tmpDir, "custom", "path")
	commandFile := filepath.Join(expectedDir, ".claude", "commands", "wave.md")
	if _, err := os.Stat(commandFile); os.IsNotExist(err) {
		t.Errorf("expected command file at %s, but it does not exist", commandFile)
	}
}

func TestWaveCommandStep_EmptyWaveDir(t *testing.T) {
	// When WaveDir is empty, baseDir should be "." (current directory)
	// We need to run in a temp dir to avoid polluting the project
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg := WizardConfig{
		WaveDir: "",
	}

	step := &WaveCommandStep{}
	_, err := step.Run(&cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	commandFile := filepath.Join(tmpDir, ".claude", "commands", "wave.md")
	if _, err := os.Stat(commandFile); os.IsNotExist(err) {
		t.Errorf("expected command file at %s, but it does not exist", commandFile)
	}
}
