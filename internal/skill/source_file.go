package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileAdapter installs skills from local filesystem paths.
type FileAdapter struct {
	projectRoot string
}

// NewFileAdapter creates a FileAdapter with the given project root for path containment.
func NewFileAdapter(projectRoot string) *FileAdapter {
	return &FileAdapter{projectRoot: projectRoot}
}

// Prefix returns "file".
func (a *FileAdapter) Prefix() string { return "file" }

// Install copies a skill from a local path into the store.
func (a *FileAdapter) Install(_ context.Context, ref string, store Store) (*InstallResult, error) {
	// Resolve the path
	var resolved string
	if filepath.IsAbs(ref) {
		resolved = ref
	} else {
		resolved = filepath.Join(a.projectRoot, ref)
	}
	resolved = filepath.Clean(resolved)

	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Path containment check: verify resolved path is within project root
	absRoot, err := filepath.Abs(a.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}

	// Check for symlinks via Lstat before EvalSymlinks
	info, err := os.Lstat(absResolved)
	if err != nil {
		return nil, fmt.Errorf("path not found: %s", absResolved)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlink rejected: %s", absResolved)
	}

	// Verify resolved path stays within project root after symlink evaluation
	evalResolved, err := filepath.EvalSymlinks(absResolved)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}
	evalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}
	if !strings.HasPrefix(evalResolved, evalRoot+string(filepath.Separator)) && evalResolved != evalRoot {
		return nil, fmt.Errorf("path traversal detected: %s escapes project root %s", evalResolved, evalRoot)
	}

	// Verify it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absResolved)
	}

	// Check for SKILL.md
	skillFile := filepath.Join(absResolved, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no SKILL.md found in %s", absResolved)
		}
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	skill, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md in %s: %w", absResolved, err)
	}

	if err := store.Write(skill); err != nil {
		return nil, fmt.Errorf("failed to write skill %q: %w", skill.Name, err)
	}

	return &InstallResult{
		Skills: []Skill{skill},
	}, nil
}
