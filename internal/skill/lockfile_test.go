package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadLockfileNonexistent(t *testing.T) {
	lf, err := LoadLockfile(filepath.Join(t.TempDir(), "missing.lock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("expected version 1, got %d", lf.Version)
	}
	if len(lf.Published) != 0 {
		t.Errorf("expected empty published list, got %d", len(lf.Published))
	}
}

func TestLoadLockfileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills.lock")

	data := `{
  "version": 1,
  "published": [
    {
      "name": "golang",
      "digest": "sha256:abc123",
      "registry": "tessl",
      "url": "https://tessl.io/skills/golang",
      "published_at": "2026-03-24T12:00:00Z"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	lf, err := LoadLockfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("expected version 1, got %d", lf.Version)
	}
	if len(lf.Published) != 1 {
		t.Fatalf("expected 1 record, got %d", len(lf.Published))
	}
	if lf.Published[0].Name != "golang" {
		t.Errorf("expected name 'golang', got %q", lf.Published[0].Name)
	}
	if lf.Published[0].Digest != "sha256:abc123" {
		t.Errorf("expected digest 'sha256:abc123', got %q", lf.Published[0].Digest)
	}
}

func TestLoadLockfileCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills.lock")

	if err := os.WriteFile(path, []byte("not json{{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadLockfile(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestLockfileSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "skills.lock")

	lf := &Lockfile{
		Version: 1,
		Published: []PublishRecord{
			{
				Name:        "golang",
				Digest:      "sha256:abc123",
				Registry:    "tessl",
				URL:         "https://tessl.io/skills/golang",
				PublishedAt: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := lf.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadLockfile(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Version != lf.Version {
		t.Errorf("version mismatch: %d != %d", loaded.Version, lf.Version)
	}
	if len(loaded.Published) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded.Published))
	}
	if loaded.Published[0].Name != "golang" {
		t.Errorf("name mismatch: %q", loaded.Published[0].Name)
	}

	// Verify it's valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Error("saved lockfile is not valid JSON")
	}

	// Verify temp file cleaned up
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should be cleaned up after successful rename")
	}
}

func TestLockfileFindByName(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Published: []PublishRecord{
			{Name: "golang", Digest: "sha256:abc"},
			{Name: "python", Digest: "sha256:def"},
		},
	}

	found := lf.FindByName("golang")
	if found == nil {
		t.Fatal("expected to find golang")
	}
	if found.Digest != "sha256:abc" {
		t.Errorf("wrong digest: %q", found.Digest)
	}

	missing := lf.FindByName("rust")
	if missing != nil {
		t.Error("expected nil for missing skill")
	}
}

func TestLockfileUpsertInsert(t *testing.T) {
	lf := &Lockfile{Version: 1}
	lf.Upsert(PublishRecord{Name: "golang", Digest: "sha256:abc"})

	if len(lf.Published) != 1 {
		t.Fatalf("expected 1 record, got %d", len(lf.Published))
	}
	if lf.Published[0].Name != "golang" {
		t.Errorf("expected name 'golang', got %q", lf.Published[0].Name)
	}
}

func TestLockfileUpsertReplace(t *testing.T) {
	lf := &Lockfile{
		Version: 1,
		Published: []PublishRecord{
			{Name: "golang", Digest: "sha256:old"},
		},
	}
	lf.Upsert(PublishRecord{Name: "golang", Digest: "sha256:new"})

	if len(lf.Published) != 1 {
		t.Fatalf("expected 1 record after replace, got %d", len(lf.Published))
	}
	if lf.Published[0].Digest != "sha256:new" {
		t.Errorf("expected updated digest, got %q", lf.Published[0].Digest)
	}
}
