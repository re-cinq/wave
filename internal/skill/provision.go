package skill

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// SkillInfo holds metadata about a provisioned skill for passing to the adapter.
type SkillInfo struct {
	Name        string
	Description string
	SourcePath  string
}

// ProvisionFromStore reads skills from the store and provisions them into the
// workspace at .agents/skills/<name>/SKILL.md. Resource files (scripts/, references/,
// assets/) are also copied with path containment checks.
//
// Returns metadata for each successfully provisioned skill. Returns a hard error
// if any named skill cannot be read from the store.
func ProvisionFromStore(store Store, workspacePath string, skillNames []string) ([]SkillInfo, error) {
	if len(skillNames) == 0 {
		return nil, nil
	}

	var infos []SkillInfo
	for _, name := range skillNames {
		s, err := store.Read(name)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				log.Printf("[WARN] skill %q not found in store, skipping", name)
				continue
			}
			return nil, fmt.Errorf("skill %q: %w", name, err)
		}

		skillDir := filepath.Join(workspacePath, ".agents", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return nil, fmt.Errorf("skill %q: failed to create directory: %w", name, err)
		}

		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(s.Body), 0o644); err != nil {
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
		})
	}

	return infos, nil
}
