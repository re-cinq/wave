package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// createTestSkillDir creates a skill directory with SKILL.md for testing.
func createTestSkillDir(t *testing.T, dir, name, description, body string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s\n", name, description, body)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// createMockTessl writes a shell script acting as a fake tessl binary.
func createMockTessl(t *testing.T, dir, stdout string, exitCode int) string {
	t.Helper()
	path := filepath.Join(dir, "tessl")
	script := fmt.Sprintf("#!/bin/sh\necho '%s'\nexit %d\n", stdout, exitCode)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPublishOneSuccess(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	lockPath := filepath.Join(dir, "skills.lock")

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "https://tessl.io/skills/golang", 0)

	p := NewPublisher(store, lockPath, "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	result := p.PublishOne(context.Background(), "golang", PublishOpts{})
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.URL != "https://tessl.io/skills/golang" {
		t.Errorf("unexpected URL: %q", result.URL)
	}
	if result.Digest == "" {
		t.Error("expected digest to be set")
	}

	// Verify lockfile was updated
	lf, err := LoadLockfile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	rec := lf.FindByName("golang")
	if rec == nil {
		t.Fatal("expected lockfile record for golang")
	}
	if rec.Digest != result.Digest {
		t.Errorf("lockfile digest mismatch: %q != %q", rec.Digest, result.Digest)
	}
}

func TestPublishOneValidationFailure(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	skillDir := filepath.Join(skillsDir, "bad")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Missing description
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: bad\ndescription: \"\"\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", nil)

	result := p.PublishOne(context.Background(), "bad", PublishOpts{})
	if result.Success {
		t.Fatal("expected failure for invalid skill")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestPublishOneWaveSpecificWarning(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "wave-tool", "Wave tool", "use wave run in a pipeline with persona and wave.yaml. Use wave init with .agents/ manifest and worktree and wave again and more pipeline and persona and wave")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", nil)

	result := p.PublishOne(context.Background(), "wave-tool", PublishOpts{})
	if result.Success {
		t.Error("expected wave-specific skill to be skipped")
	}
	if !result.Skipped {
		t.Error("expected Skipped == true")
	}
}

func TestPublishOneWaveSpecificForce(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "wave-tool", "Wave tool", "use wave run in a pipeline with persona and wave.yaml. Use wave init with .agents/ manifest and worktree and wave again and more pipeline and persona and wave")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "https://tessl.io/skills/wave-tool", 0)

	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	result := p.PublishOne(context.Background(), "wave-tool", PublishOpts{Force: true})
	if !result.Success {
		t.Fatalf("expected success with --force, got error: %s", result.Error)
	}
}

func TestPublishOneDryRun(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	lockPath := filepath.Join(dir, "skills.lock")
	p := NewPublisher(store, lockPath, "tessl", nil)

	result := p.PublishOne(context.Background(), "golang", PublishOpts{DryRun: true})
	if !result.Success {
		t.Fatalf("expected success for dry-run, got error: %s", result.Error)
	}
	if result.URL != "[dry-run]" {
		t.Errorf("expected [dry-run] URL, got %q", result.URL)
	}
	if result.Digest == "" {
		t.Error("expected digest to be computed in dry-run")
	}

	// Lockfile should NOT be updated
	lf, err := LoadLockfile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Published) != 0 {
		t.Error("lockfile should not be updated in dry-run mode")
	}
}

func TestPublishOneIdempotent(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	lockPath := filepath.Join(dir, "skills.lock")

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "https://tessl.io/skills/golang", 0)

	p := NewPublisher(store, lockPath, "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	// First publish
	r1 := p.PublishOne(context.Background(), "golang", PublishOpts{})
	if !r1.Success {
		t.Fatalf("first publish failed: %s", r1.Error)
	}

	// Second publish — should be skipped as up-to-date
	r2 := p.PublishOne(context.Background(), "golang", PublishOpts{})
	if !r2.Skipped {
		t.Error("expected second publish to be skipped")
	}
	if r2.SkipReason != "up-to-date" {
		t.Errorf("expected skip reason 'up-to-date', got %q", r2.SkipReason)
	}
}

func TestPublishOneTesslNotFound(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", func(s string) (string, error) {
		return "", fmt.Errorf("not found")
	})

	result := p.PublishOne(context.Background(), "golang", PublishOpts{})
	if result.Success {
		t.Error("expected failure when tessl not found")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestPublishOneTesslFailure(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "error output", 1)

	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	result := p.PublishOne(context.Background(), "golang", PublishOpts{})
	if result.Success {
		t.Error("expected failure when tessl exits non-zero")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestPublishOneSkillNotFound(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", nil)

	result := p.PublishOne(context.Background(), "nonexistent", PublishOpts{})
	if result.Success {
		t.Error("expected failure for missing skill")
	}
}

func TestPublishAllMixed(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	createTestSkillDir(t, skillsDir, "golang", "Go development", "pure go patterns")
	createTestSkillDir(t, skillsDir, "wave-tool", "Wave tool", "use wave run in a pipeline with persona and wave.yaml. Use wave init with .agents/ manifest and worktree and wave again and more pipeline and persona and wave")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "https://tessl.io/published", 0)

	p := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	results, err := p.PublishAll(context.Background(), PublishOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	resultMap := make(map[string]PublishResult)
	for _, r := range results {
		resultMap[r.Name] = r
	}

	if r, ok := resultMap["golang"]; ok {
		if !r.Success {
			t.Errorf("golang should succeed, got error: %s", r.Error)
		}
	}
	if r, ok := resultMap["wave-tool"]; ok {
		if !r.Skipped {
			t.Error("wave-tool should be skipped")
		}
	}
}
