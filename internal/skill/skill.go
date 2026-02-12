package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// Provisioner discovers and copies skill command files into a workspace.
type Provisioner struct {
	skills   map[string]manifest.SkillConfig
	repoRoot string // Project root where .claude/commands/ lives
}

// NewProvisioner creates a skill provisioner for the given project.
func NewProvisioner(skills map[string]manifest.SkillConfig, repoRoot string) *Provisioner {
	return &Provisioner{
		skills:   skills,
		repoRoot: repoRoot,
	}
}

// Provision copies skill command files into the workspace's .claude/commands/ directory.
// It matches command files based on the skill's commands_glob or the default pattern.
func (p *Provisioner) Provision(workspacePath string, skillNames []string) error {
	if len(skillNames) == 0 {
		return nil
	}

	targetDir := filepath.Join(workspacePath, ".claude", "commands")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	for _, name := range skillNames {
		cfg, ok := p.skills[name]
		if !ok {
			continue // Skip undeclared skills
		}

		// Determine glob pattern for this skill's command files
		pattern := cfg.CommandsGlob
		if pattern == "" {
			// Default: .claude/commands/<skill-name>.*.md
			pattern = filepath.Join(".claude", "commands", name+".*.md")
		}

		// Resolve pattern relative to repo root
		fullPattern := filepath.Join(p.repoRoot, pattern)

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return fmt.Errorf("failed to glob skill commands for %q: %w", name, err)
		}

		for _, src := range matches {
			fileName := filepath.Base(src)
			dst := filepath.Join(targetDir, fileName)

			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("failed to copy skill command %q: %w", fileName, err)
			}
		}
	}

	return nil
}

// DiscoverCommands returns the list of command files available for the named skills.
func (p *Provisioner) DiscoverCommands(skillNames []string) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, name := range skillNames {
		cfg, ok := p.skills[name]
		if !ok {
			continue
		}

		pattern := cfg.CommandsGlob
		if pattern == "" {
			pattern = filepath.Join(".claude", "commands", name+".*.md")
		}

		fullPattern := filepath.Join(p.repoRoot, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob skill commands for %q: %w", name, err)
		}

		for _, m := range matches {
			// Store relative to repo root
			rel, _ := filepath.Rel(p.repoRoot, m)
			result[name] = append(result[name], rel)
		}
	}

	return result, nil
}

// ProvisionAll copies all declared skill command files into the workspace.
func (p *Provisioner) ProvisionAll(workspacePath string) error {
	names := make([]string, 0, len(p.skills))
	for name := range p.skills {
		names = append(names, name)
	}
	return p.Provision(workspacePath, names)
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Preserve directory structure
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// FormatSkillCommandPrompt generates a prompt fragment that tells the agent
// to use a slash command with the given arguments.
func FormatSkillCommandPrompt(command, args string) string {
	// Normalize command name: "speckit.specify" -> "/speckit.specify"
	if !strings.HasPrefix(command, "/") {
		command = "/" + command
	}
	if args != "" {
		return fmt.Sprintf("Run `%s` with: %s", command, args)
	}
	return fmt.Sprintf("Run `%s`", command)
}
