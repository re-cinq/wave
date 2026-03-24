package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ComputeDigest computes a SHA-256 content digest for a skill.
// The digest covers the raw SKILL.md bytes followed by sorted resource files.
func ComputeDigest(s Skill) (string, error) {
	skillMdPath := filepath.Join(s.SourcePath, "SKILL.md")
	skillMdBytes, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", fmt.Errorf("cannot read SKILL.md: %w", err)
	}

	h := sha256.New()
	h.Write(skillMdBytes)

	// Sort resource paths for deterministic ordering
	sorted := make([]string, len(s.ResourcePaths))
	copy(sorted, s.ResourcePaths)
	sort.Strings(sorted)

	for _, relpath := range sorted {
		absPath := filepath.Join(s.SourcePath, relpath)
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			// Skip unreadable resource files
			continue
		}
		fmt.Fprintf(h, "\n---resource:%s---\n", relpath)
		h.Write(data)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
