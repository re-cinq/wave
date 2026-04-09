package skill

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// lookPathFunc is a function type for looking up executables on PATH.
// Defaults to exec.LookPath but can be overridden for testing.
type lookPathFunc func(string) (string, error)

// checkDependency verifies that a CLI tool is available on PATH.
// Returns a *DependencyError if the binary is not found.
func checkDependency(dep CLIDependency, lookPath lookPathFunc) error {
	_, err := lookPath(dep.Binary)
	if err != nil {
		return &DependencyError{
			Binary:       dep.Binary,
			Instructions: dep.Instructions,
		}
	}
	return nil
}

// discoverSkillFiles walks a directory tree finding all SKILL.md files.
// Returns absolute paths to each discovered SKILL.md.
// Symlinks are skipped to prevent directory escape attacks.
func discoverSkillFiles(dir string) ([]string, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Use Lstat to detect symlinks — skip them to prevent escape attacks
		linfo, lerr := os.Lstat(path)
		if lerr != nil {
			return lerr
		}
		if linfo.Mode()&os.ModeSymlink != 0 {
			if linfo.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() && info.Name() == "SKILL.md" {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			paths = append(paths, abs)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover skill files: %w", err)
	}
	return paths, nil
}

// parseAndWriteSkills reads each SKILL.md path, parses it, and writes to the store.
// Returns an InstallResult with all successfully installed skills and any warnings.
func parseAndWriteSkills(_ context.Context, paths []string, store Store) (*InstallResult, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no SKILL.md files found")
	}

	result := &InstallResult{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to read %s: %v", path, err))
			continue
		}

		skill, err := Parse(data)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to parse %s: %v", path, err))
			continue
		}

		if err := store.Write(skill); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to write skill %q: %v", skill.Name, err))
			continue
		}

		result.Skills = append(result.Skills, skill)
	}

	if len(result.Skills) == 0 {
		return nil, fmt.Errorf("no valid skills found in %d SKILL.md files", len(paths))
	}

	return result, nil
}

// TesslAdapter installs skills from the Tessl registry via the tessl CLI.
type TesslAdapter struct {
	dep      CLIDependency
	lookPath lookPathFunc
}

// NewTesslAdapter creates a TesslAdapter with default exec.LookPath.
func NewTesslAdapter() *TesslAdapter {
	return &TesslAdapter{
		dep: CLIDependency{
			Binary:       "tessl",
			Instructions: "npm i -g @tessl/cli",
		},
		lookPath: exec.LookPath,
	}
}

// Prefix returns "tessl".
func (a *TesslAdapter) Prefix() string { return "tessl" }

// Install runs `tessl install <ref>` and discovers resulting SKILL.md files.
func (a *TesslAdapter) Install(ctx context.Context, ref string, store Store) (*InstallResult, error) {
	if err := checkDependency(a.dep, a.lookPath); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-tessl-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ctx, cancel := context.WithTimeout(ctx, CLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tessl", "install", ref)
	cmd.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tessl install %s failed: %v\nstderr: %s", ref, err, stderr.String())
	}

	paths, err := discoverSkillFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}

// BMADAdapter installs skills from the BMAD ecosystem via npx.
type BMADAdapter struct {
	dep      CLIDependency
	lookPath lookPathFunc
}

// NewBMADAdapter creates a BMADAdapter with default exec.LookPath.
func NewBMADAdapter() *BMADAdapter {
	return &BMADAdapter{
		dep: CLIDependency{
			Binary:       "npx",
			Instructions: "npm i -g npx (comes with npm)",
		},
		lookPath: exec.LookPath,
	}
}

// Prefix returns "bmad".
func (a *BMADAdapter) Prefix() string { return "bmad" }

// Install runs `npx bmad-method install --tools claude-code --yes` and discovers skills.
func (a *BMADAdapter) Install(ctx context.Context, _ string, store Store) (*InstallResult, error) {
	if err := checkDependency(a.dep, a.lookPath); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-bmad-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ctx, cancel := context.WithTimeout(ctx, CLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npx", "bmad-method", "install", "--tools", "claude-code", "--yes")
	cmd.Dir = tmpDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("npx bmad-method install failed: %v\nstderr: %s", err, stderr.String())
	}

	paths, err := discoverSkillFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}

// OpenSpecAdapter installs skills from the OpenSpec ecosystem.
type OpenSpecAdapter struct {
	dep      CLIDependency
	lookPath lookPathFunc
}

// NewOpenSpecAdapter creates an OpenSpecAdapter with default exec.LookPath.
func NewOpenSpecAdapter() *OpenSpecAdapter {
	return &OpenSpecAdapter{
		dep: CLIDependency{
			Binary:       "openspec",
			Instructions: "npm i -g @openspec/cli",
		},
		lookPath: exec.LookPath,
	}
}

// Prefix returns "openspec".
func (a *OpenSpecAdapter) Prefix() string { return "openspec" }

// Install runs `openspec init` and discovers resulting skill files.
func (a *OpenSpecAdapter) Install(ctx context.Context, _ string, store Store) (*InstallResult, error) {
	if err := checkDependency(a.dep, a.lookPath); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-openspec-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ctx, cancel := context.WithTimeout(ctx, CLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "openspec", "init")
	cmd.Dir = tmpDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("openspec init failed: %v\nstderr: %s", err, stderr.String())
	}

	paths, err := discoverSkillFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}

// SpecKitAdapter installs skills from the SpecKit ecosystem.
type SpecKitAdapter struct {
	dep      CLIDependency
	lookPath lookPathFunc
}

// NewSpecKitAdapter creates a SpecKitAdapter with default exec.LookPath.
func NewSpecKitAdapter() *SpecKitAdapter {
	return &SpecKitAdapter{
		dep: CLIDependency{
			Binary:       "specify",
			Instructions: "npm i -g @speckit/cli",
		},
		lookPath: exec.LookPath,
	}
}

// Prefix returns "speckit".
func (a *SpecKitAdapter) Prefix() string { return "speckit" }

// Install runs `specify init` and discovers resulting skill files.
func (a *SpecKitAdapter) Install(ctx context.Context, _ string, store Store) (*InstallResult, error) {
	if err := checkDependency(a.dep, a.lookPath); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-speckit-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ctx, cancel := context.WithTimeout(ctx, CLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "specify", "init")
	cmd.Dir = tmpDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("specify init failed: %v\nstderr: %s", err, stderr.String())
	}

	paths, err := discoverSkillFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}
