package defaults

import (
	"regexp"
	"strings"
	"testing"
)

// TestPersonaFilesNoLanguageReferences asserts zero programming language-specific
// keyword matches across all persona files and base-protocol.md (SC-003).
func TestPersonaFilesNoLanguageReferences(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	// Patterns from contracts/persona-validation.md
	langPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bGolang\b`),
		regexp.MustCompile(`\bGo\s+(code|program|module|package|binary|runtime|compiler)\b`),
		regexp.MustCompile(`\bPython\b`),
		regexp.MustCompile(`\bTypeScript\b`),
		regexp.MustCompile(`\bJavaScript\b`),
		regexp.MustCompile(`(?i)\bJava\b`),
		regexp.MustCompile(`\bRust\b`),
		regexp.MustCompile(`\bRuby\b`),
		regexp.MustCompile(`\bSwift\b`),
		regexp.MustCompile(`\bKotlin\b`),
		regexp.MustCompile(`\bC\+\+\b`),
		regexp.MustCompile(`\bC#\b`),
	}

	for name, content := range personas {
		for _, re := range langPatterns {
			if matches := re.FindAllString(content, -1); len(matches) > 0 {
				t.Errorf("persona %q contains language reference: %v (pattern: %s)", name, matches, re.String())
			}
		}
	}
}

// TestPersonaFilesTokenRange verifies all 17 persona files (excluding base-protocol.md)
// are within the 100-400 token range using word count heuristic (SC-001).
func TestPersonaFilesTokenRange(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	for name, content := range personas {
		if name == "base-protocol.md" {
			continue
		}

		words := len(strings.Fields(content))
		tokens := words * 100 / 75

		if tokens < 100 {
			t.Errorf("persona %q has ~%d tokens (%d words) — below minimum of 100", name, tokens, words)
		}
		if tokens > 400 {
			t.Errorf("persona %q has ~%d tokens (%d words) — above maximum of 400", name, tokens, words)
		}
	}
}

// TestPersonaFilesMandatorySections verifies all 17 persona files contain the
// three mandatory structural elements (SC-007):
// 1. H1 identity heading
// 2. Responsibilities section
// 3. Output contract section (heading containing "Output")
func TestPersonaFilesMandatorySections(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	h1Pattern := regexp.MustCompile(`(?m)^# .+`)
	respPattern := regexp.MustCompile(`(?mi)^## (Responsibilities|Step-by-Step)`)
	outputPattern := regexp.MustCompile(`(?mi)^## .*Output`)

	for name, content := range personas {
		if name == "base-protocol.md" {
			continue
		}

		if !h1Pattern.MatchString(content) {
			t.Errorf("persona %q missing H1 identity heading", name)
		}
		if !respPattern.MatchString(content) {
			t.Errorf("persona %q missing responsibilities section", name)
		}
		if !outputPattern.MatchString(content) {
			t.Errorf("persona %q missing output contract section (heading with 'Output')", name)
		}
	}
}

// TestAllPersonasCovered verifies exactly 17 persona files exist (excluding
// base-protocol.md) with the expected names (SC-004).
func TestAllPersonasCovered(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}

	expected := map[string]bool{
		"navigator.md":        true,
		"implementer.md":      true,
		"reviewer.md":         true,
		"planner.md":          true,
		"researcher.md":       true,
		"debugger.md":         true,
		"auditor.md":          true,
		"craftsman.md":        true,
		"summarizer.md":       true,
		"github-analyst.md":   true,
		"github-commenter.md": true,
		"github-enhancer.md":  true,
		"philosopher.md":      true,
		"provocateur.md":      true,
		"validator.md":        true,
		"synthesizer.md":      true,
		"supervisor.md":       true,
	}

	for name := range expected {
		if _, ok := personas[name]; !ok {
			t.Errorf("expected persona %q not found", name)
		}
	}

	// Count persona files (excluding base-protocol.md)
	count := 0
	for name := range personas {
		if name != "base-protocol.md" {
			count++
		}
	}
	if count != 17 {
		t.Errorf("expected 17 persona files, got %d", count)
	}
}
