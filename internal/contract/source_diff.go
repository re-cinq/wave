package contract

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// sourceDiffValidator checks that the current git diff contains at least MinFiles
// qualifying source files (matched by Glob, not excluded by Exclude patterns).
// This catches the "verified everything as already correct" failure mode where
// no real code changes were made.
type sourceDiffValidator struct{}

func (v *sourceDiffValidator) Validate(cfg ContractConfig, workspacePath string) error {
	minFiles := cfg.MinFiles
	if minFiles <= 0 {
		minFiles = 1
	}

	// git diff --name-only HEAD lists files changed relative to HEAD (staged + unstaged)
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		// If HEAD doesn't exist (initial commit), try against empty tree
		cmd2 := exec.Command("git", "diff", "--name-only", "--cached")
		cmd2.Dir = workspacePath
		out2, err2 := cmd2.Output()
		if err2 != nil {
			return fmt.Errorf("source_diff: could not run git diff: %w", err)
		}
		out = out2
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	qualifying := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Apply glob filter (empty glob matches all files)
		if cfg.Glob != "" {
			matched, err := filepath.Match(cfg.Glob, filepath.Base(line))
			if err != nil {
				return fmt.Errorf("source_diff: invalid glob %q: %w", cfg.Glob, err)
			}
			// Also try matching the full path
			if !matched {
				matched, _ = filepath.Match(cfg.Glob, line)
			}
			if !matched {
				continue
			}
		}

		// Apply exclude patterns
		excluded := false
		for _, pattern := range cfg.Exclude {
			if matchExclude(pattern, line) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		qualifying++
	}

	if qualifying < minFiles {
		return fmt.Errorf("source_diff: found %d qualifying changed source file(s), need at least %d — "+
			"ensure the implementation modifies real source files (not only specs/ or .agents/ files)",
			qualifying, minFiles)
	}

	return nil
}

// matchExclude checks whether filePath should be excluded by the given pattern.
// It handles /** suffix patterns (e.g., "specs/**", ".agents/**") by checking if
// the file lives under the prefix directory. For patterns without **, it falls
// back to filepath.Match against both the full path and the base name.
func matchExclude(pattern, filePath string) bool {
	// Handle dir/** — match anything under that directory prefix
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if strings.HasPrefix(filePath, prefix+"/") || filePath == prefix {
			return true
		}
		return false
	}

	// Handle **/ prefix — match any file with this suffix
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		return strings.HasSuffix(filePath, "/"+suffix) || filePath == suffix
	}

	// Standard filepath.Match against full path
	m, err := filepath.Match(pattern, filePath)
	if err == nil && m {
		return true
	}

	// Also try against base name alone
	m, _ = filepath.Match(pattern, filepath.Base(filePath))
	return m
}
