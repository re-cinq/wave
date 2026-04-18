package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishIntegrationRoundtrip(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")

	// Create 3 test skills: 1 standalone, 1 both, 1 wave-specific
	createTestSkillDir(t, skillsDir, "standalone-tool", "Standalone tool", "Pure Go patterns and best practices.")
	createTestSkillDir(t, skillsDir, "mixed-tool", "Mixed tool", "Use wave and pipeline integration for workflow.")
	createTestSkillDir(t, skillsDir, "wave-tool", "Wave tool", "Use wave run in a pipeline with persona and wave.yaml. Use wave init with .agents/ manifest and worktree and wave again and more pipeline and persona and wave")

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})

	// Test audit: verify correct classifications
	classifications, err := ClassifyAll(store)
	if err != nil {
		t.Fatalf("ClassifyAll failed: %v", err)
	}
	if len(classifications) != 3 {
		t.Fatalf("expected 3 classifications, got %d", len(classifications))
	}

	tagMap := make(map[string]string)
	for _, c := range classifications {
		tagMap[c.Name] = c.Tag
	}
	if tagMap["standalone-tool"] != TagStandalone {
		t.Errorf("standalone-tool should be standalone, got %q", tagMap["standalone-tool"])
	}
	if tagMap["mixed-tool"] != TagBoth {
		t.Errorf("mixed-tool should be both, got %q", tagMap["mixed-tool"])
	}
	if tagMap["wave-tool"] != TagWaveSpecific {
		t.Errorf("wave-tool should be wave-specific, got %q", tagMap["wave-tool"])
	}

	// Test publish with mock tessl
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tesslPath := createMockTessl(t, binDir, "https://tessl.io/published", 0)

	lockPath := filepath.Join(dir, "skills.lock")
	publisher := NewPublisher(store, lockPath, "tessl", func(s string) (string, error) {
		return tesslPath, nil
	})

	// PublishAll: standalone and both should publish, wave-specific should be skipped
	results, err := publisher.PublishAll(context.Background(), PublishOpts{})
	if err != nil {
		t.Fatalf("PublishAll failed: %v", err)
	}

	resultMap := make(map[string]PublishResult)
	for _, r := range results {
		resultMap[r.Name] = r
	}

	if r := resultMap["standalone-tool"]; !r.Success {
		t.Errorf("standalone-tool should publish successfully, got error: %s", r.Error)
	}
	if r := resultMap["mixed-tool"]; !r.Success {
		t.Errorf("mixed-tool should publish successfully, got error: %s", r.Error)
	}
	if r := resultMap["wave-tool"]; !r.Skipped {
		t.Error("wave-tool should be skipped")
	}

	// Verify lockfile has correct records
	lf, err := LoadLockfile(lockPath)
	if err != nil {
		t.Fatalf("LoadLockfile failed: %v", err)
	}

	standaloneRec := lf.FindByName("standalone-tool")
	if standaloneRec == nil {
		t.Fatal("expected lockfile record for standalone-tool")
	}
	if standaloneRec.Digest == "" {
		t.Error("expected non-empty digest")
	}

	// Test idempotent re-publish
	results2, err := publisher.PublishAll(context.Background(), PublishOpts{})
	if err != nil {
		t.Fatalf("second PublishAll failed: %v", err)
	}
	for _, r := range results2 {
		if r.Name == "wave-tool" {
			continue // wave-specific always skipped
		}
		if !r.Skipped || r.SkipReason != "up-to-date" {
			t.Errorf("expected %s to be skipped as up-to-date, got Skipped=%v SkipReason=%q", r.Name, r.Skipped, r.SkipReason)
		}
	}

	// Test modify skill content → different digest
	originalDigest := standaloneRec.Digest
	modifiedContent := "---\nname: standalone-tool\ndescription: Standalone tool\n---\nModified body content.\n"
	if err := os.WriteFile(filepath.Join(skillsDir, "standalone-tool", "SKILL.md"), []byte(modifiedContent), 0644); err != nil {
		t.Fatal(err)
	}

	modifiedSkill, err := store.Read("standalone-tool")
	if err != nil {
		t.Fatal(err)
	}
	newDigest, err := ComputeDigest(modifiedSkill)
	if err != nil {
		t.Fatal(err)
	}
	if newDigest == originalDigest {
		t.Error("modified content should produce different digest")
	}

	// Test verify: modified skill should be detected
	for _, rec := range lf.Published {
		if rec.Name != "standalone-tool" {
			continue
		}
		s, readErr := store.Read(rec.Name)
		if readErr != nil {
			t.Fatalf("failed to read skill for verify: %v", readErr)
		}
		actual, _ := ComputeDigest(s)
		if actual == rec.Digest {
			t.Error("expected verify to detect modification")
		}
	}
}

func TestPublishIntegrationValidationFailure(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	skillDir := filepath.Join(skillsDir, "bad-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create skill with empty description (validation failure)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: bad-skill\ndescription: \"\"\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: skillsDir, Precedence: 1})
	publisher := NewPublisher(store, filepath.Join(dir, "skills.lock"), "tessl", func(s string) (string, error) {
		return "", fmt.Errorf("should not be called")
	})

	result := publisher.PublishOne(context.Background(), "bad-skill", PublishOpts{})
	if result.Success {
		t.Error("expected validation failure")
	}
	if result.Error == "" {
		t.Error("expected error message describing validation failure")
	}
}
