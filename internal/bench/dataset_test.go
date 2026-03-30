package bench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDataset(t *testing.T) {
	t.Run("valid JSONL", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")
		content := `{"instance_id":"task-1","repo":"foo/bar","base_commit":"abc123","version":"v1","problem_statement":"Fix the bug","patch":"diff","test_cmd":"go test"}
{"instance_id":"task-2","repo":"baz/qux","base_commit":"def456","version":"v2","problem_statement":"Add feature","patch":"diff2","test_cmd":"npm test"}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		tasks, err := LoadDataset(path)
		if err != nil {
			t.Fatalf("LoadDataset() error = %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("got %d tasks, want 2", len(tasks))
		}
		if tasks[0].ID != "task-1" {
			t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "task-1")
		}
		if tasks[0].Repo != "foo/bar" {
			t.Errorf("tasks[0].Repo = %q, want %q", tasks[0].Repo, "foo/bar")
		}
		if tasks[0].Problem != "Fix the bug" {
			t.Errorf("tasks[0].Problem = %q, want %q", tasks[0].Problem, "Fix the bug")
		}
		if tasks[1].ID != "task-2" {
			t.Errorf("tasks[1].ID = %q, want %q", tasks[1].ID, "task-2")
		}
	})

	t.Run("skips blank lines", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")
		content := `{"instance_id":"task-1","repo":"foo/bar","problem_statement":"Fix it"}

{"instance_id":"task-2","repo":"baz/qux","problem_statement":"Build it"}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		tasks, err := LoadDataset(path)
		if err != nil {
			t.Fatalf("LoadDataset() error = %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("got %d tasks, want 2", len(tasks))
		}
	})

	t.Run("rejects missing instance_id", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")
		content := `{"repo":"foo/bar","problem_statement":"no id"}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadDataset(path)
		if err == nil {
			t.Fatal("LoadDataset() expected error for missing instance_id")
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.jsonl")
		content := `not json at all
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadDataset(path)
		if err == nil {
			t.Fatal("LoadDataset() expected error for invalid JSON")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadDataset("/nonexistent/path/tasks.jsonl")
		if err == nil {
			t.Fatal("LoadDataset() expected error for missing file")
		}
	})
}

func TestListDatasets(t *testing.T) {
	t.Run("finds JSONL files", func(t *testing.T) {
		dir := t.TempDir()
		// Create some dataset files
		for _, name := range []string{"swe-bench-lite.jsonl", "custom.jsonl"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		// Non-JSONL files should be ignored
		if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
			t.Fatal(err)
		}
		// Directories should be ignored
		if err := os.Mkdir(filepath.Join(dir, "subdir.jsonl"), 0o755); err != nil {
			t.Fatal(err)
		}

		datasets, err := ListDatasets(dir)
		if err != nil {
			t.Fatalf("ListDatasets() error = %v", err)
		}
		if len(datasets) != 2 {
			t.Fatalf("got %d datasets, want 2", len(datasets))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := ListDatasets("/nonexistent/dir")
		if err == nil {
			t.Fatal("ListDatasets() expected error for missing directory")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		datasets, err := ListDatasets(dir)
		if err != nil {
			t.Fatalf("ListDatasets() error = %v", err)
		}
		if len(datasets) != 0 {
			t.Fatalf("got %d datasets, want 0", len(datasets))
		}
	})
}
