package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ParseError represents a SKILL.md parsing or validation failure.
type ParseError struct {
	Field      string
	Constraint string
	Value      string
}

func (e *ParseError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("skill %s: %s (got %q)", e.Field, e.Constraint, e.Value)
	}
	return fmt.Sprintf("skill %s: %s", e.Field, e.Constraint)
}

func (e *ParseError) Unwrap() error { return nil }

// SkillError wraps an error with skill name and path context.
type SkillError struct {
	SkillName string
	Path      string
	Err       error
}

func (e *SkillError) Error() string {
	return fmt.Sprintf("skill %q (%s): %v", e.SkillName, e.Path, e.Err)
}

func (e *SkillError) Unwrap() error { return e.Err }

// DiscoveryError is returned when List encounters per-skill failures.
type DiscoveryError struct {
	Errors []SkillError
}

func (e *DiscoveryError) Error() string {
	var msgs []string
	for _, se := range e.Errors {
		msgs = append(msgs, se.Error())
	}
	return fmt.Sprintf("discovery errors: %s", strings.Join(msgs, "; "))
}

// ErrNotFound is returned when a skill does not exist in any source.
var ErrNotFound = errors.New("skill not found")

// Store defines CRUD operations for skill management.
type Store interface {
	Read(name string) (Skill, error)
	Write(skill Skill) error
	List() ([]Skill, error)
	Delete(name string) error
}

// SkillSource is a directory that may contain skill subdirectories.
type SkillSource struct {
	Root       string
	Precedence int
}

// DefaultSources returns the canonical skill source ordering used by the
// CLI run/resume paths: project-level "skills/" wins over installed
// ".agents/skills/". Centralised so both call sites — and any future
// caller (test fixtures, doctor, doc generation) — agree on the layout.
func DefaultSources() []SkillSource {
	return []SkillSource{
		{Root: "skills", Precedence: 2},
		{Root: ".agents/skills", Precedence: 1},
	}
}

// DirectoryStore implements Store backed by filesystem directories.
type DirectoryStore struct {
	sources []SkillSource
}

// containedPath resolves a path within root, rejects symlinks and path traversal.
// Returns the resolved absolute path or an error.
func containedPath(root, name string) (string, error) {
	dir := filepath.Join(root, name)

	// Reject symlinks via Lstat (does not follow symlinks)
	info, err := os.Lstat(dir)
	if err != nil {
		return dir, err // caller handles os.IsNotExist
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink rejected: %s", dir)
	}

	// Verify resolved path stays within root
	absDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("failed to resolve root: %w", err)
	}
	if !strings.HasPrefix(absDir, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected: %s escapes %s", absDir, absRoot)
	}

	return dir, nil
}

// NewDirectoryStore creates a DirectoryStore with sources sorted by precedence (highest first).
func NewDirectoryStore(sources ...SkillSource) *DirectoryStore {
	sorted := make([]SkillSource, len(sources))
	copy(sorted, sources)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Precedence > sorted[j].Precedence
	})
	return &DirectoryStore{sources: sorted}
}

// Read returns a fully-loaded skill by name.
func (ds *DirectoryStore) Read(name string) (Skill, error) {
	if err := ValidateName(name); err != nil {
		return Skill{}, err
	}

	for _, source := range ds.sources {
		skillDir, err := containedPath(source.Root, name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Skill{}, &SkillError{SkillName: name, Path: skillDir, Err: err}
		}

		skillFile := filepath.Join(skillDir, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Skill{}, &SkillError{SkillName: name, Path: skillFile, Err: err}
		}

		skill, err := Parse(data)
		if err != nil {
			return Skill{}, &SkillError{SkillName: name, Path: skillFile, Err: err}
		}

		// FR-004: validate name matches directory name
		if skill.Name != name {
			return Skill{}, &ParseError{
				Field:      "name",
				Constraint: "must match directory name",
				Value:      fmt.Sprintf("frontmatter %q != directory %q", skill.Name, name),
			}
		}

		skill.SourcePath = skillDir
		skill.ResourcePaths = discoverResources(skillDir)
		return skill, nil
	}

	return Skill{}, fmt.Errorf("%w: %s", ErrNotFound, name)
}

// Write persists a skill to the highest-precedence source directory.
func (ds *DirectoryStore) Write(skill Skill) error {
	if len(ds.sources) == 0 {
		return fmt.Errorf("no skill sources configured")
	}
	if err := ValidateName(skill.Name); err != nil {
		return err
	}
	if skill.Description == "" {
		return &ParseError{Field: "description", Constraint: "required"}
	}

	data, err := Serialize(skill)
	if err != nil {
		return err
	}

	root := ds.sources[0].Root
	dir := filepath.Join(root, skill.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Defense-in-depth: verify resolved path stays within source root after creation
	if _, err := containedPath(root, skill.Name); err != nil {
		// Clean up the directory we just created
		_ = os.RemoveAll(dir)
		return fmt.Errorf("path containment check failed: %w", err)
	}

	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	return nil
}

// List returns all discoverable skills with metadata-only loading.
func (ds *DirectoryStore) List() ([]Skill, error) {
	seen := make(map[string]bool)
	var skills []Skill
	var errs []SkillError

	for _, source := range ds.sources {
		if _, err := os.Stat(source.Root); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(source.Root)
		if err != nil {
			errs = append(errs, SkillError{SkillName: "<root>", Path: source.Root, Err: err})
			continue
		}

		for _, entry := range entries {
			// Skip non-directories and symlinks
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			name := entry.Name()
			if seen[name] {
				continue
			}

			// Validate path containment (consistent with Read)
			skillDir, err := containedPath(source.Root, name)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				errs = append(errs, SkillError{SkillName: name, Path: filepath.Join(source.Root, name), Err: err})
				continue
			}

			skillFile := filepath.Join(skillDir, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				errs = append(errs, SkillError{SkillName: name, Path: skillFile, Err: err})
				continue
			}

			skill, err := ParseMetadata(data)
			if err != nil {
				errs = append(errs, SkillError{SkillName: name, Path: skillFile, Err: err})
				continue
			}

			// Validate name-directory consistency (consistent with Read)
			if skill.Name != name {
				errs = append(errs, SkillError{
					SkillName: name,
					Path:      skillFile,
					Err: &ParseError{
						Field:      "name",
						Constraint: "must match directory name",
						Value:      fmt.Sprintf("frontmatter %q != directory %q", skill.Name, name),
					},
				})
				continue
			}

			skill.SourcePath = skillDir
			seen[name] = true
			skills = append(skills, skill)
		}
	}

	if len(errs) > 0 {
		return skills, &DiscoveryError{Errors: errs}
	}
	return skills, nil
}

// Delete removes a skill directory by name from the first source that contains it.
func (ds *DirectoryStore) Delete(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	for _, source := range ds.sources {
		dir, err := containedPath(source.Root, name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("path containment check failed: %w", err)
		}

		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to delete skill directory: %w", err)
		}
		return nil
	}

	return fmt.Errorf("%w: %s", ErrNotFound, name)
}

// discoverResources scans for resource files in known subdirectories of a skill directory.
// Symlinked files and subdirectories are skipped.
func discoverResources(skillDir string) []string {
	resourceDirs := []string{"scripts", "references", "assets"}
	var paths []string

	for _, dir := range resourceDirs {
		full := filepath.Join(skillDir, dir)

		// Skip symlinked resource directories
		if info, err := os.Lstat(full); err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// Skip directories and symlinks
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}

	return paths
}
