package skill

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ProvisionLevel controls how much skill content is written during provisioning.
type ProvisionLevel int

const (
	// Level1Metadata writes a stub SKILL.md with only frontmatter and a pointer
	// to request full content via the Skill tool.
	Level1Metadata ProvisionLevel = 1
	// Level2Instructions writes the full SKILL.md body (current behavior).
	Level2Instructions ProvisionLevel = 2
)

// SkillInfo holds metadata about a provisioned skill for passing to the adapter.
type SkillInfo struct {
	Name        string
	Description string
	SourcePath  string
	Level       ProvisionLevel
}

// ProvisionFromStore reads skills from the store and provisions them into the
// workspace at .wave/skills/<name>/SKILL.md. Resource files (scripts/, references/,
// assets/) are also copied with path containment checks.
//
// Returns metadata for each successfully provisioned skill. Returns a hard error
// if any named skill cannot be read from the store.
//
// This is equivalent to calling ProvisionFromStoreWithLevel with Level2Instructions.
func ProvisionFromStore(store Store, workspacePath string, skillNames []string) ([]SkillInfo, error) {
	return ProvisionFromStoreWithLevel(store, workspacePath, skillNames, Level2Instructions)
}

// ProvisionFromStoreWithLevel reads skills from the store and provisions them into
// the workspace at .wave/skills/<name>/SKILL.md with the specified level of content.
//
// Level1Metadata writes a stub SKILL.md with only a brief description and a pointer
// to request full content via the Skill tool. The original body is not written.
//
// Level2Instructions writes the full SKILL.md body (same as ProvisionFromStore).
//
// In both cases, resource files (scripts/, references/, assets/) are copied with
// path containment checks. Returns metadata for each successfully provisioned skill.
// Returns a hard error if any named skill cannot be read from the store.
func ProvisionFromStoreWithLevel(store Store, workspacePath string, skillNames []string, level ProvisionLevel) ([]SkillInfo, error) {
	if len(skillNames) == 0 {
		return nil, nil
	}

	var infos []SkillInfo
	for _, name := range skillNames {
		var s Skill
		var err error

		switch level {
		case Level1Metadata:
			s, err = store.ReadMetadata(name)
		default:
			s, err = store.Read(name)
		}

		if err != nil {
			if errors.Is(err, ErrNotFound) {
				log.Printf("[WARN] skill %q not found in store, skipping", name)
				continue
			}
			return nil, fmt.Errorf("skill %q: %w", name, err)
		}

		skillDir := filepath.Join(workspacePath, ".wave", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return nil, fmt.Errorf("skill %q: failed to create directory: %w", name, err)
		}

		var skillMDContent string
		if level == Level1Metadata {
			skillMDContent = fmt.Sprintf("# %s\n\n%s\n\nThis skill's full instructions are available on-demand. Use the Read tool to read\nthe reference files in this directory, or invoke the Skill tool to load full content.\n", s.Name, s.Description)
		} else {
			skillMDContent = s.Body
		}

		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMDContent), 0o644); err != nil {
			return nil, fmt.Errorf("skill %q: failed to write SKILL.md: %w", name, err)
		}

		// Copy resource files with path containment checks
		absSkillDir, err := filepath.Abs(skillDir)
		if err != nil {
			return nil, fmt.Errorf("skill %q: failed to resolve skill directory: %w", name, err)
		}

		for _, rp := range s.ResourcePaths {
			dstPath := filepath.Join(skillDir, rp)
			absDst, err := filepath.Abs(dstPath)
			if err != nil || !strings.HasPrefix(absDst, absSkillDir+string(filepath.Separator)) {
				return nil, fmt.Errorf("skill %q: resource %q path traversal blocked", name, rp)
			}

			srcPath := filepath.Join(s.SourcePath, rp)
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return nil, fmt.Errorf("skill %q: resource %q read failed: %w", name, rp, err)
			}

			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return nil, fmt.Errorf("skill %q: resource %q mkdir failed: %w", name, rp, err)
			}

			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return nil, fmt.Errorf("skill %q: resource %q write failed: %w", name, rp, err)
			}
		}

		infos = append(infos, SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			SourcePath:  s.SourcePath,
			Level:       level,
		})
	}

	return infos, nil
}
