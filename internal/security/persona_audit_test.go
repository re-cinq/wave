package security_test

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
)

// unsafeBodyPattern matches --body followed by a double-quoted string that
// starts with < or $ (indicating interpolation), but NOT the safe heredoc
// pattern --body "$(cat <<'EOF' ..." which uses single-quoted delimiters.
var unsafeBodyPattern = regexp.MustCompile(`--body\s+"[<$]`)

// unsafeTitlePattern matches --title followed by a double-quoted string that
// starts with < or $ (indicating interpolation), but NOT the safe heredoc
// pattern --title "$(cat <<'EOF' ...".
var unsafeTitlePattern = regexp.MustCompile(`--title\s+"[<$]`)

// safeBodyHeredocPattern matches the safe body heredoc form: --body "$(cat <<'EOF'
var safeBodyHeredocPattern = regexp.MustCompile(`--body\s+"\$\(cat\s+<<'`)

// safeTitleHeredocPattern matches the safe title heredoc form: --title "$(cat <<'EOF'
var safeTitleHeredocPattern = regexp.MustCompile(`--title\s+"\$\(cat\s+<<'`)

// unsafeDescriptionPattern matches --description followed by a double-quoted
// string starting with < or $ (GitLab/Gitea use --description instead of --body).
var unsafeDescriptionPattern = regexp.MustCompile(`--description\s+"[<$]`)

// safeDescriptionHeredocPattern matches the safe description heredoc form:
// --description "$(cat <<'EOF'
var safeDescriptionHeredocPattern = regexp.MustCompile(`--description\s+"\$\(cat\s+<<'`)

// TestPersonaAudit_NoUnsafeInlineBody validates that ALL embedded persona
// markdown files do not contain unsafe inline --body or --title patterns with
// double-quoted interpolation.
//
// Safe patterns (allowed):
//   - --body-file /tmp/somefile.md
//   - --body "$(cat <<'EOF' ... EOF )"
//   - --title '...'  (single-quoted, no interpolation)
//
// Unsafe patterns (flagged):
//   - --body "<html>..."        (double-quoted with interpolation)
//   - --body "$variable"        (double-quoted variable expansion)
//   - --title "<title>"         (double-quoted with interpolation)
//   - --title "$variable"       (double-quoted variable expansion)
func TestPersonaAudit_NoUnsafeInlineBody(t *testing.T) {
	personas, err := defaults.GetPersonas()
	if err != nil {
		t.Fatalf("failed to load embedded personas: %v", err)
	}

	if len(personas) == 0 {
		t.Fatal("no embedded personas found; expected at least one")
	}

	// Sort persona names for deterministic output
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	sort.Strings(names)

	type violation struct {
		persona string
		line    int
		text    string
		pattern string
	}

	var violations []violation

	for _, name := range names {
		content := personas[name]
		lines := strings.Split(content, "\n")

		for lineNum, line := range lines {
			// Check --body "< and --body "$
			if unsafeBodyPattern.MatchString(line) {
				// Exclude safe heredoc patterns: --body "$(cat <<'EOF'
				if safeBodyHeredocPattern.MatchString(line) {
					continue
				}
				violations = append(violations, violation{
					persona: name,
					line:    lineNum + 1,
					text:    strings.TrimSpace(line),
					pattern: `--body "< or --body "$`,
				})
			}

			// Check --title "< and --title "$
			if unsafeTitlePattern.MatchString(line) {
				// Exclude safe heredoc patterns: --title "$(cat <<'EOF'
				if safeTitleHeredocPattern.MatchString(line) {
					continue
				}
				violations = append(violations, violation{
					persona: name,
					line:    lineNum + 1,
					text:    strings.TrimSpace(line),
					pattern: `--title "< or --title "$`,
				})
			}

			// Check --description "< and --description "$  (GitLab/Gitea)
			if unsafeDescriptionPattern.MatchString(line) {
				// Exclude safe heredoc patterns: --description "$(cat <<'EOF'
				if safeDescriptionHeredocPattern.MatchString(line) {
					continue
				}
				violations = append(violations, violation{
					persona: name,
					line:    lineNum + 1,
					text:    strings.TrimSpace(line),
					pattern: `--description "< or --description "$`,
				})
			}
		}
	}

	for _, v := range violations {
		t.Errorf("UNSAFE pattern in persona %q (line %d): matched %s\n  content: %s",
			v.persona, v.line, v.pattern, v.text)
	}

	if len(violations) > 0 {
		t.Logf("\n%d unsafe pattern(s) found across embedded personas.\n"+
			"Fix: use --body-file <path> or --body \"$(cat <<'EOF'\\n...\\nEOF\\n)\" "+
			"instead of inline --body \"<...>\" with double-quoted interpolation.",
			len(violations))
	}
}

// TestPersonaAudit_SafePatternsAllowed verifies that the audit regex correctly
// allows patterns that are known to be safe.
func TestPersonaAudit_SafePatternsAllowed(t *testing.T) {
	safeLines := []struct {
		name string
		line string
	}{
		{
			name: "body-file pattern",
			line: `gh pr create --title 'Fix bug' --body-file /tmp/pr-body.md`,
		},
		{
			name: "body with single-quoted heredoc cat",
			line: `gh issue create --body "$(cat <<'EOF'`,
		},
		{
			name: "title with single-quoted heredoc cat",
			line: `gh issue create --title "$(cat <<'EOF'`,
		},
		{
			name: "single-quoted title",
			line: `gh issue create --title 'Fix the bug'`,
		},
		{
			name: "body-file with variable path",
			line: `gh pr create --body-file "$TMPDIR/body.md"`,
		},
		{
			name: "description with single-quoted heredoc cat",
			line: `glab issue create --description "$(cat <<'EOF'`,
		},
		{
			name: "no body or title flags at all",
			line: `gh issue list --repo owner/repo`,
		},
	}

	for _, tt := range safeLines {
		t.Run(tt.name, func(t *testing.T) {
			hasUnsafeBody := unsafeBodyPattern.MatchString(tt.line) &&
				!safeBodyHeredocPattern.MatchString(tt.line)
			hasUnsafeTitle := unsafeTitlePattern.MatchString(tt.line) &&
				!safeTitleHeredocPattern.MatchString(tt.line)
			hasUnsafeDesc := unsafeDescriptionPattern.MatchString(tt.line) &&
				!safeDescriptionHeredocPattern.MatchString(tt.line)

			if hasUnsafeBody || hasUnsafeTitle || hasUnsafeDesc {
				t.Errorf("safe line incorrectly flagged as unsafe: %s", tt.line)
			}
		})
	}
}

// TestPersonaAudit_UnsafePatternsDetected verifies that the audit regex
// correctly catches patterns that are known to be unsafe.
func TestPersonaAudit_UnsafePatternsDetected(t *testing.T) {
	unsafeLines := []struct {
		name    string
		line    string
		pattern string
	}{
		{
			name:    "body with double-quoted interpolation variable",
			line:    `gh issue create --body "$UNTRUSTED_CONTENT"`,
			pattern: "body",
		},
		{
			name:    "body with double-quoted HTML tag",
			line:    `gh pr create --body "<h1>Title</h1>"`,
			pattern: "body",
		},
		{
			name:    "title with double-quoted interpolation variable",
			line:    `gh issue create --title "$TITLE_VAR"`,
			pattern: "title",
		},
		{
			name:    "title with double-quoted HTML-like content",
			line:    `gh issue create --title "<fix>: something"`,
			pattern: "title",
		},
		{
			name:    "description with double-quoted interpolation",
			line:    `glab issue create --description "$BODY"`,
			pattern: "description",
		},
		{
			name:    "description with double-quoted HTML tag",
			line:    `glab issue create --description "<p>content</p>"`,
			pattern: "description",
		},
	}

	for _, tt := range unsafeLines {
		t.Run(tt.name, func(t *testing.T) {
			detected := false
			switch tt.pattern {
			case "body":
				detected = unsafeBodyPattern.MatchString(tt.line) &&
					!safeBodyHeredocPattern.MatchString(tt.line)
			case "title":
				detected = unsafeTitlePattern.MatchString(tt.line) &&
					!safeTitleHeredocPattern.MatchString(tt.line)
			case "description":
				detected = unsafeDescriptionPattern.MatchString(tt.line) &&
					!safeDescriptionHeredocPattern.MatchString(tt.line)
			}

			if !detected {
				t.Errorf("unsafe line NOT detected: %s", tt.line)
			}
		})
	}
}

// TestPersonaAudit_AllPersonasLoaded ensures the audit test covers all personas.
// This prevents silent regression if new personas are added without the audit
// covering them.
func TestPersonaAudit_AllPersonasLoaded(t *testing.T) {
	personas, err := defaults.GetPersonas()
	if err != nil {
		t.Fatalf("failed to load embedded personas: %v", err)
	}

	personaNames := defaults.PersonaNames()

	if len(personas) != len(personaNames) {
		t.Errorf("GetPersonas() returned %d personas but PersonaNames() returned %d",
			len(personas), len(personaNames))
	}

	// Verify each persona has non-empty content
	for name, content := range personas {
		if strings.TrimSpace(content) == "" {
			t.Errorf("persona %q has empty content", name)
		}
	}

	t.Logf("audited %d embedded personas", len(personas))
	for _, name := range sorted(personas) {
		t.Logf("  - %s", name)
	}
}

// sorted returns the sorted keys of a map.
func sorted(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestPersonaAudit_DescriptionPatternsCoveredForForges ensures that the audit
// also covers forge-specific argument patterns (--description for GitLab/Gitea).
func TestPersonaAudit_DescriptionPatternsCoveredForForges(t *testing.T) {
	personas, err := defaults.GetPersonas()
	if err != nil {
		t.Fatalf("failed to load embedded personas: %v", err)
	}

	// Collect all personas that mention forge CLIs
	forgeCLIs := map[string]string{
		"glab": "GitLab",
		"tea":  "Gitea",
	}

	for name, content := range personas {
		for cli, forge := range forgeCLIs {
			if strings.Contains(content, cli) {
				t.Logf("persona %q references %s CLI (%s) - checked for --description patterns",
					name, cli, forge)
			}
		}
	}

	// Verify that --description patterns are covered by the main audit
	// (this is a meta-test to ensure completeness)
	testLine := `glab issue create --description "<body>"`
	if !unsafeDescriptionPattern.MatchString(testLine) {
		t.Error("audit regex should detect --description with unsafe interpolation")
	}
}
