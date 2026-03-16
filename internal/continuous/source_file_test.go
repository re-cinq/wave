package continuous

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "items.txt")

	content := "https://github.com/org/repo/issues/1\nhttps://github.com/org/repo/issues/2\nhttps://github.com/org/repo/issues/3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	src, err := NewFileSource(path)
	if err != nil {
		t.Fatalf("NewFileSource: %v", err)
	}

	if got := src.Name(); got != "file("+path+")" {
		t.Errorf("Name() = %q", got)
	}

	ctx := context.Background()
	var items []*WorkItem
	for {
		item, err := src.Next(ctx)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if item == nil {
			break
		}
		items = append(items, item)
	}

	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[0].Input != "https://github.com/org/repo/issues/1" {
		t.Errorf("items[0].Input = %q", items[0].Input)
	}
}

func TestFileSourceEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	src, err := NewFileSource(path)
	if err != nil {
		t.Fatalf("NewFileSource: %v", err)
	}

	item, err := src.Next(context.Background())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil for empty source, got %+v", item)
	}
}

func TestFileSourceMissing(t *testing.T) {
	_, err := NewFileSource("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileSourceSkipsBlankLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "with-blanks.txt")
	content := "item1\n\n  \nitem2\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	src, err := NewFileSource(path)
	if err != nil {
		t.Fatalf("NewFileSource: %v", err)
	}

	ctx := context.Background()
	var items []*WorkItem
	for {
		item, err := src.Next(ctx)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if item == nil {
			break
		}
		items = append(items, item)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}
