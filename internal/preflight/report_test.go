package preflight

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
)

// markOnboardedForTest writes an onboarding state file to the given directory,
// mirroring onboarding.MarkOnboarded without importing the onboarding package
// (which would create an import cycle).
func markOnboardedForTest(t *testing.T, waveDir string) {
	t.Helper()
	state := struct {
		Completed   bool      `json:"completed"`
		CompletedAt time.Time `json:"completed_at"`
		Version     int       `json:"version"`
	}{
		Completed:   true,
		CompletedAt: time.Now(),
		Version:     1,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal onboarding state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(waveDir, ".onboarded"), data, 0644); err != nil {
		t.Fatalf("failed to write onboarding state: %v", err)
	}
}

func TestSystemReadinessReport_AllPass(t *testing.T) {
	c := NewChecker(map[string]manifest.SkillConfig{
		"myskill": {Check: "true"},
	})
	c.lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	c.runCmd = func(name string, args ...string) error {
		return nil
	}

	// Create a temp wave dir with onboarding state
	waveDir := t.TempDir()
	markOnboardedForTest(t, waveDir)

	report, err := c.RunSystemReadiness(SystemReadinessOpts{
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
		RemoteURL: "https://github.com/org/repo.git",
		Skills:    []string{"myskill"},
		Tools:     []string{"sh"},
		WaveDir:   waveDir,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !report.AllPassed {
		t.Error("expected all checks to pass")
		for _, check := range report.Checks {
			if !check.OK {
				t.Errorf("  failed check: %s (%s): %s", check.Name, check.Kind, check.Message)
			}
		}
	}

	if report.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	if len(report.Checks) == 0 {
		t.Error("expected at least one check result")
	}
}

func TestSystemReadinessReport_SomeFail(t *testing.T) {
	c := NewChecker(nil)
	c.lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	c.runCmd = func(name string, args ...string) error {
		return nil
	}

	// No onboarding state = wave not initialized
	waveDir := t.TempDir()

	report, err := c.RunSystemReadiness(SystemReadinessOpts{
		WaveDir: waveDir,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if report.AllPassed {
		t.Error("expected not all checks to pass (wave not initialized)")
	}
}

func TestSystemReadinessReport_Empty(t *testing.T) {
	c := NewChecker(nil)

	report, err := c.RunSystemReadiness(SystemReadinessOpts{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !report.AllPassed {
		t.Error("expected all checks to pass for empty opts")
	}

	if len(report.Checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(report.Checks))
	}
}

func TestSystemReadinessReport_JSON(t *testing.T) {
	report := &SystemReadinessReport{
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		AllPassed: false,
		Checks: []Result{
			{Name: "gh", Kind: "tool", OK: true, Message: `tool "gh" found`},
			{Name: "claude", Kind: "adapter", OK: false, Message: `adapter "claude" not found`, Remediation: "Install adapter binary"},
		},
		Summary: "1/2 checks passed (1 failed)",
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	// Verify it can be unmarshaled back
	var decoded SystemReadinessReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal report: %v", err)
	}

	if decoded.AllPassed != false {
		t.Error("expected AllPassed=false")
	}
	if len(decoded.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(decoded.Checks))
	}
	if decoded.Checks[0].OK != true {
		t.Error("expected first check to pass")
	}
	if decoded.Checks[1].OK != false {
		t.Error("expected second check to fail")
	}
	if decoded.Checks[1].Remediation != "Install adapter binary" {
		t.Errorf("expected remediation, got %q", decoded.Checks[1].Remediation)
	}
	if decoded.Summary != "1/2 checks passed (1 failed)" {
		t.Errorf("unexpected summary: %q", decoded.Summary)
	}
}

func TestSystemReadinessReport_WaveInitialized(t *testing.T) {
	c := NewChecker(nil)

	waveDir := t.TempDir()
	markOnboardedForTest(t, waveDir)

	report, err := c.RunSystemReadiness(SystemReadinessOpts{
		WaveDir: waveDir,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !report.AllPassed {
		t.Error("expected all checks to pass")
	}

	// Find the init check
	var initCheck *Result
	for i := range report.Checks {
		if report.Checks[i].Kind == "init" {
			initCheck = &report.Checks[i]
			break
		}
	}
	if initCheck == nil {
		t.Fatal("expected init check in results")
	}
	if !initCheck.OK {
		t.Error("expected init check to pass")
	}
}

func TestSystemReadinessReport_CorruptStateFile(t *testing.T) {
	c := NewChecker(nil)

	waveDir := t.TempDir()
	// Write corrupt JSON to the state file
	if err := os.WriteFile(filepath.Join(waveDir, ".onboarded"), []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt state: %v", err)
	}

	report, err := c.RunSystemReadiness(SystemReadinessOpts{
		WaveDir: waveDir,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if report.AllPassed {
		t.Error("expected not all checks to pass with corrupt state")
	}
}
