package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPath_File(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "nested", "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := CopyPath(src, dst); err != nil {
		t.Fatalf("CopyPath: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content: got %q want %q", got, "hello")
	}
}

func TestCopyPath_Directory(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "srcdir")
	sub := filepath.Join(src, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	dst := filepath.Join(tmp, "dstdir")
	if err := CopyPath(src, dst); err != nil {
		t.Fatalf("CopyPath: %v", err)
	}

	for rel, want := range map[string]string{
		"a.txt":     "a",
		"sub/b.txt": "b",
	} {
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != want {
			t.Errorf("%s: got %q want %q", rel, got, want)
		}
	}
}

func TestCopyPath_MissingSource(t *testing.T) {
	tmp := t.TempDir()
	err := CopyPath(filepath.Join(tmp, "nope"), filepath.Join(tmp, "out"))
	if err == nil {
		t.Errorf("expected error for missing source")
	}
}

func TestCopyPath_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "empty")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(tmp, "out")
	if err := CopyPath(src, dst); err != nil {
		t.Fatalf("CopyPath: %v", err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("dst not a directory")
	}
}
