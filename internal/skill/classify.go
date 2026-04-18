package skill

import (
	"errors"
	"strings"
)

// Classification tag constants.
const (
	TagStandalone   = "standalone"
	TagWaveSpecific = "wave-specific"
	TagBoth         = "both"
)

// waveKeywords are patterns that indicate Wave-specific content.
var waveKeywords = []string{
	"wave run",
	"wave init",
	"wave.yaml",
	".agents/",
	"pipeline",
	"persona",
	"manifest",
	"worktree",
	"wave",
}

// SkillClassification holds the audit result for a skill.
type SkillClassification struct {
	Name         string
	Tag          string
	WaveRefCount int
	Warnings     []string
	SourcePath   string
}

// ClassifySkill classifies a skill based on Wave-specific keyword occurrences in its body.
func ClassifySkill(s Skill) SkillClassification {
	c := SkillClassification{
		Name:       s.Name,
		SourcePath: s.SourcePath,
	}

	body := strings.ToLower(s.Body)
	c.WaveRefCount = countWaveRefs(body)

	switch {
	case c.WaveRefCount == 0:
		c.Tag = TagStandalone
	case c.WaveRefCount > 10:
		c.Tag = TagWaveSpecific
	default:
		c.Tag = TagBoth
	}

	if s.Description == "" {
		c.Warnings = append(c.Warnings, "missing description")
	}
	if s.License == "" {
		c.Warnings = append(c.Warnings, "missing license")
	}

	return c
}

// countWaveRefs counts occurrences of wave-specific keywords in lowercased text.
// Multi-word keywords are matched first to avoid double-counting.
func countWaveRefs(body string) int {
	count := 0
	// Count multi-word keywords first, then replace them to avoid double counting
	remaining := body
	for _, kw := range waveKeywords {
		n := strings.Count(remaining, kw)
		count += n
		if n > 0 {
			remaining = strings.ReplaceAll(remaining, kw, strings.Repeat("_", len(kw)))
		}
	}
	return count
}

// ClassifyAll classifies all skills from a store.
func ClassifyAll(store Store) ([]SkillClassification, error) {
	skills, err := store.List()
	if err != nil {
		var discErr *DiscoveryError
		if !errors.As(err, &discErr) {
			return nil, err
		}
		// Continue with discovered skills despite warnings
	}

	var results []SkillClassification
	for _, s := range skills {
		full, readErr := store.Read(s.Name)
		if readErr != nil {
			continue
		}
		results = append(results, ClassifySkill(full))
	}
	return results, nil
}
