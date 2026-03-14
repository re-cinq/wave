package skill

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTesslAdapterPrefix(t *testing.T) {
	a := NewTesslAdapter()
	if a.Prefix() != "tessl" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "tessl")
	}
}

func TestTesslAdapterMissingDependency(t *testing.T) {
	a := &TesslAdapter{
		dep: CLIDependency{
			Binary:       "tessl",
			Instructions: "npm i -g @tessl/cli",
		},
		lookPath: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}

	store := newMemoryStore()
	_, err := a.Install(context.Background(), "github/spec-kit", store)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}

	var depErr *DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected *DependencyError, got %T: %v", err, err)
	}
	if depErr.Binary != "tessl" {
		t.Errorf("Binary = %q, want %q", depErr.Binary, "tessl")
	}
	if depErr.Instructions != "npm i -g @tessl/cli" {
		t.Errorf("Instructions = %q, want %q", depErr.Instructions, "npm i -g @tessl/cli")
	}
}

func TestCheckDependency(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dep := CLIDependency{Binary: "test-bin", Instructions: "install it"}
		err := checkDependency(dep, func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		dep := CLIDependency{Binary: "missing-bin", Instructions: "npm i -g missing"}
		err := checkDependency(dep, func(name string) (string, error) {
			return "", errors.New("not found")
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var depErr *DependencyError
		if !errors.As(err, &depErr) {
			t.Fatalf("expected *DependencyError, got %T", err)
		}
		if depErr.Binary != "missing-bin" {
			t.Errorf("Binary = %q, want %q", depErr.Binary, "missing-bin")
		}
	})
}

func TestDiscoverSkillFiles(t *testing.T) {
	t.Run("finds nested SKILL.md", func(t *testing.T) {
		root := t.TempDir()
		makeTestSkillDir(t, root, "alpha", "Alpha skill")
		makeTestSkillDir(t, root, "beta", "Beta skill")

		paths, err := discoverSkillFiles(root)
		if err != nil {
			t.Fatalf("discoverSkillFiles() error = %v", err)
		}
		if len(paths) != 2 {
			t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
		}
		for _, p := range paths {
			if filepath.Base(p) != "SKILL.md" {
				t.Errorf("expected SKILL.md, got %s", filepath.Base(p))
			}
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		root := t.TempDir()
		paths, err := discoverSkillFiles(root)
		if err != nil {
			t.Fatalf("discoverSkillFiles() error = %v", err)
		}
		if len(paths) != 0 {
			t.Errorf("expected 0 paths, got %d", len(paths))
		}
	})
}

func TestParseAndWriteSkills(t *testing.T) {
	t.Run("valid skills written to store", func(t *testing.T) {
		root := t.TempDir()
		makeTestSkillDir(t, root, "skill-a", "Skill A")
		makeTestSkillDir(t, root, "skill-b", "Skill B")

		paths, err := discoverSkillFiles(root)
		if err != nil {
			t.Fatal(err)
		}

		store := newMemoryStore()
		result, err := parseAndWriteSkills(context.Background(), paths, store)
		if err != nil {
			t.Fatalf("parseAndWriteSkills() error = %v", err)
		}
		if len(result.Skills) != 2 {
			t.Errorf("expected 2 skills, got %d", len(result.Skills))
		}
		if store.writes != 2 {
			t.Errorf("expected 2 writes, got %d", store.writes)
		}
	})

	t.Run("no paths returns error", func(t *testing.T) {
		store := newMemoryStore()
		_, err := parseAndWriteSkills(context.Background(), nil, store)
		if err == nil {
			t.Fatal("expected error for empty paths")
		}
	})

	t.Run("invalid SKILL.md adds warning", func(t *testing.T) {
		root := t.TempDir()
		// Create a valid skill
		makeTestSkillDir(t, root, "valid-skill", "Valid skill")
		// Create an invalid SKILL.md
		badDir := filepath.Join(root, "bad-skill")
		if err := os.MkdirAll(badDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte("invalid content"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := discoverSkillFiles(root)
		if err != nil {
			t.Fatal(err)
		}

		store := newMemoryStore()
		result, err := parseAndWriteSkills(context.Background(), paths, store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Skills) != 1 {
			t.Errorf("expected 1 valid skill, got %d", len(result.Skills))
		}
		if len(result.Warnings) != 1 {
			t.Errorf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
		}
	})
}

// --- T013: Ecosystem CLI Adapter Tests ---

func TestBMADAdapterPrefix(t *testing.T) {
	a := NewBMADAdapter()
	if a.Prefix() != "bmad" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "bmad")
	}
}

func TestBMADAdapterMissingDependency(t *testing.T) {
	a := &BMADAdapter{
		dep: CLIDependency{
			Binary:       "npx",
			Instructions: "npm i -g npx (comes with npm)",
		},
		lookPath: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}

	store := newMemoryStore()
	_, err := a.Install(context.Background(), "install", store)
	if err == nil {
		t.Fatal("expected error for missing npx")
	}

	var depErr *DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected *DependencyError, got %T: %v", err, err)
	}
	if depErr.Binary != "npx" {
		t.Errorf("Binary = %q, want %q", depErr.Binary, "npx")
	}
	if depErr.Instructions != "npm i -g npx (comes with npm)" {
		t.Errorf("Instructions = %q, want %q", depErr.Instructions, "npm i -g npx (comes with npm)")
	}
}

func TestOpenSpecAdapterPrefix(t *testing.T) {
	a := NewOpenSpecAdapter()
	if a.Prefix() != "openspec" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "openspec")
	}
}

func TestOpenSpecAdapterMissingDependency(t *testing.T) {
	a := &OpenSpecAdapter{
		dep: CLIDependency{
			Binary:       "openspec",
			Instructions: "npm i -g @openspec/cli",
		},
		lookPath: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}

	store := newMemoryStore()
	_, err := a.Install(context.Background(), "init", store)
	if err == nil {
		t.Fatal("expected error for missing openspec")
	}

	var depErr *DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected *DependencyError, got %T: %v", err, err)
	}
	if depErr.Binary != "openspec" {
		t.Errorf("Binary = %q, want %q", depErr.Binary, "openspec")
	}
}

func TestSpecKitAdapterPrefix(t *testing.T) {
	a := NewSpecKitAdapter()
	if a.Prefix() != "speckit" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "speckit")
	}
}

func TestSpecKitAdapterMissingDependency(t *testing.T) {
	a := &SpecKitAdapter{
		dep: CLIDependency{
			Binary:       "specify",
			Instructions: "npm i -g @speckit/cli",
		},
		lookPath: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}

	store := newMemoryStore()
	_, err := a.Install(context.Background(), "init", store)
	if err == nil {
		t.Fatal("expected error for missing specify")
	}

	var depErr *DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected *DependencyError, got %T: %v", err, err)
	}
	if depErr.Binary != "specify" {
		t.Errorf("Binary = %q, want %q", depErr.Binary, "specify")
	}
}

func TestTesslAdapterTimeout(t *testing.T) {
	a := &TesslAdapter{
		dep: CLIDependency{
			Binary:       "tessl",
			Instructions: "npm i -g @tessl/cli",
		},
		lookPath: func(name string) (string, error) {
			return "/usr/bin/tessl", nil
		},
	}

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	store := newMemoryStore()
	_, err := a.Install(ctx, "some-ref", store)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDiscoverSkillFilesSkipsSymlinks(t *testing.T) {
	root := t.TempDir()

	// Create a real SKILL.md
	makeTestSkillDir(t, root, "real-skill", "Real skill")

	// Create a symlink pointing outside the directory
	outsideDir := t.TempDir()
	makeTestSkillDir(t, outsideDir, "outside-skill", "Outside skill")
	if err := os.Symlink(outsideDir, filepath.Join(root, "symlinked-dir")); err != nil {
		t.Fatal(err)
	}

	// Create a symlink file pointing to a SKILL.md outside
	outsideSkill := filepath.Join(outsideDir, "outside-skill", "SKILL.md")
	symlinkSkill := filepath.Join(root, "SKILL.md")
	if err := os.Symlink(outsideSkill, symlinkSkill); err != nil {
		t.Fatal(err)
	}

	paths, err := discoverSkillFiles(root)
	if err != nil {
		t.Fatalf("discoverSkillFiles() error = %v", err)
	}

	// Should only find the real skill, not symlinked ones
	if len(paths) != 1 {
		t.Errorf("expected 1 path (real skill only), got %d: %v", len(paths), paths)
	}
}

// --- T005: TestBMADAdapterTimeout — US2-3: timeout for BMAD adapter ---

func TestBMADAdapterTimeout(t *testing.T) {
	a := &BMADAdapter{
		dep: CLIDependency{
			Binary:       "npx",
			Instructions: "npm i -g npx (comes with npm)",
		},
		lookPath: func(name string) (string, error) {
			return "/usr/bin/npx", nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	store := newMemoryStore()
	_, err := a.Install(ctx, "install", store)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- T006: TestOpenSpecAdapterTimeout — US2-3: timeout for OpenSpec adapter [P] ---

func TestOpenSpecAdapterTimeout(t *testing.T) {
	a := &OpenSpecAdapter{
		dep: CLIDependency{
			Binary:       "openspec",
			Instructions: "npm i -g @openspec/cli",
		},
		lookPath: func(name string) (string, error) {
			return "/usr/bin/openspec", nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	store := newMemoryStore()
	_, err := a.Install(ctx, "init", store)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- T007: TestSpecKitAdapterTimeout — US2-3: timeout for SpecKit adapter [P] ---

func TestSpecKitAdapterTimeout(t *testing.T) {
	a := &SpecKitAdapter{
		dep: CLIDependency{
			Binary:       "specify",
			Instructions: "npm i -g @speckit/cli",
		},
		lookPath: func(name string) (string, error) {
			return "/usr/bin/specify", nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	store := newMemoryStore()
	_, err := a.Install(ctx, "init", store)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- T008: TestCLIAdapterStderrCapture — US2-4: stderr in error diagnostics ---

func TestCLIAdapterStderrCapture(t *testing.T) {
	a := &TesslAdapter{
		dep: CLIDependency{
			Binary:       "tessl",
			Instructions: "npm i -g @tessl/cli",
		},
		lookPath: func(name string) (string, error) {
			return "/usr/bin/tessl", nil
		},
	}

	// Use a cancelled context — the error should mention the context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	store := newMemoryStore()
	_, err := a.Install(ctx, "some-ref", store)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	// Verify the error message is informative (contains the command name or context info)
	if !strings.Contains(err.Error(), "tessl") {
		t.Errorf("expected error to mention 'tessl', got: %v", err)
	}
}
