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

// safeBodyCatPattern matches the safe body form using cat to read content:
// --body "$(cat <<'EOF' ...)" or --body "$(cat /tmp/file.md)"
var safeBodyCatPattern = regexp.MustCompile(`--body\s+"\$\(cat\s`)

// safeTitleCatPattern matches the safe title form using cat:
// --title "$(cat <<'EOF' ...)" or --title "$(cat /tmp/file.md)"
var safeTitleCatPattern = regexp.MustCompile(`--title\s+"\$\(cat\s`)

// unsafeDescriptionPattern matches --description followed by a double-quoted
// string starting with < or $ (GitLab/Gitea use --description instead of --body).
var unsafeDescriptionPattern = regexp.MustCompile(`--description\s+"[<$]`)

// safeDescriptionCatPattern matches the safe description form using cat:
// --description "$(cat <<'EOF' ...)" or --description "$(cat /tmp/file.md)"
var safeDescriptionCatPattern = regexp.MustCompile(`--description\s+"\$\(cat\s`)

// unsafeMessagePattern matches --message followed by a double-quoted string
// starting with < or $ (GitLab uses --message for issue/MR notes).
var unsafeMessagePattern = regexp.MustCompile(`--message\s+"[<$]`)

// safeMessageCatPattern matches the safe message form:
// --message "$(cat /tmp/file.md)" or --message "$(cat <<'EOF'
var safeMessageCatPattern = regexp.MustCompile(`--message\s+"\$\(cat\s`)

// violation represents a single unsafe pattern found during audit.
type violation struct {
	source  string
	line    int
	text    string
	pattern string
}

// scanContent checks a single content string for unsafe CLI patterns
// and returns any violations found.
func scanContent(source, content string) []violation {
	var violations []violation
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Check --body "< and --body "$
		if unsafeBodyPattern.MatchString(line) {
			if safeBodyCatPattern.MatchString(line) {
				continue
			}
			violations = append(violations, violation{
				source:  source,
				line:    lineNum + 1,
				text:    strings.TrimSpace(line),
				pattern: `--body "< or --body "$`,
			})
		}

		// Check --title "< and --title "$
		if unsafeTitlePattern.MatchString(line) {
			if safeTitleCatPattern.MatchString(line) {
				continue
			}
			violations = append(violations, violation{
				source:  source,
				line:    lineNum + 1,
				text:    strings.TrimSpace(line),
				pattern: `--title "< or --title "$`,
			})
		}

		// Check --description "< and --description "$  (GitLab/Gitea)
		if unsafeDescriptionPattern.MatchString(line) {
			if safeDescriptionCatPattern.MatchString(line) {
				continue
			}
			violations = append(violations, violation{
				source:  source,
				line:    lineNum + 1,
				text:    strings.TrimSpace(line),
				pattern: `--description "< or --description "$`,
			})
		}

		// Check --message "< and --message "$  (GitLab notes)
		if unsafeMessagePattern.MatchString(line) {
			if safeMessageCatPattern.MatchString(line) {
				continue
			}
			violations = append(violations, violation{
				source:  source,
				line:    lineNum + 1,
				text:    strings.TrimSpace(line),
				pattern: `--message "< or --message "$`,
			})
		}
	}

	return violations
}

// TestPersonaAudit_NoUnsafeInlineBody validates that ALL embedded persona
// markdown files do not contain unsafe inline --body, --title, --description,
// or --message patterns with double-quoted interpolation.
//
// Safe patterns (allowed):
//   - --body-file /tmp/somefile.md
//   - --body "$(cat <<'EOF' ... EOF )"
//   - --title '...'  (single-quoted, no interpolation)
//   - --message "$(cat /tmp/file.md)"
//
// Unsafe patterns (flagged):
//   - --body "<html>..."        (double-quoted with interpolation)
//   - --body "$variable"        (double-quoted variable expansion)
//   - --title "<title>"         (double-quoted with interpolation)
//   - --title "$variable"       (double-quoted variable expansion)
//   - --message "<content>"     (double-quoted with interpolation)
//   - --message "$variable"     (double-quoted variable expansion)
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

	var violations []violation

	for _, name := range names {
		violations = append(violations, scanContent("persona:"+name, personas[name])...)
	}

	for _, v := range violations {
		t.Errorf("UNSAFE pattern in %q (line %d): matched %s\n  content: %s",
			v.source, v.line, v.pattern, v.text)
	}

	if len(violations) > 0 {
		t.Logf("\n%d unsafe pattern(s) found across embedded personas.\n"+
			"Fix: use --body-file <path> or --body \"$(cat <<'EOF'\\n...\\nEOF\\n)\" "+
			"instead of inline --body \"<...>\" with double-quoted interpolation.",
			len(violations))
	}
}

// TestPipelineAudit_NoUnsafeInlinePatterns validates that ALL embedded pipeline
// YAML files do not contain unsafe inline --body, --title, --description,
// or --message patterns with double-quoted interpolation.
func TestPipelineAudit_NoUnsafeInlinePatterns(t *testing.T) {
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		t.Fatalf("failed to load embedded pipelines: %v", err)
	}

	if len(pipelines) == 0 {
		t.Fatal("no embedded pipelines found; expected at least one")
	}

	names := make([]string, 0, len(pipelines))
	for name := range pipelines {
		names = append(names, name)
	}
	sort.Strings(names)

	var violations []violation

	for _, name := range names {
		violations = append(violations, scanContent("pipeline:"+name, pipelines[name])...)
	}

	for _, v := range violations {
		t.Errorf("UNSAFE pattern in %q (line %d): matched %s\n  content: %s",
			v.source, v.line, v.pattern, v.text)
	}

	if len(violations) > 0 {
		t.Logf("\n%d unsafe pattern(s) found across embedded pipelines.\n"+
			"Fix: capture command output to a variable first, or use --body-file.",
			len(violations))
	}

	t.Logf("audited %d embedded pipelines", len(pipelines))
}

// TestPromptAudit_NoUnsafeInlinePatterns validates that ALL embedded prompt
// files do not contain unsafe inline --body, --title, --description,
// or --message patterns with double-quoted interpolation.
func TestPromptAudit_NoUnsafeInlinePatterns(t *testing.T) {
	prompts, err := defaults.GetPrompts()
	if err != nil {
		t.Fatalf("failed to load embedded prompts: %v", err)
	}

	if len(prompts) == 0 {
		t.Fatal("no embedded prompts found; expected at least one")
	}

	names := make([]string, 0, len(prompts))
	for name := range prompts {
		names = append(names, name)
	}
	sort.Strings(names)

	var violations []violation

	for _, name := range names {
		violations = append(violations, scanContent("prompt:"+name, prompts[name])...)
	}

	for _, v := range violations {
		t.Errorf("UNSAFE pattern in %q (line %d): matched %s\n  content: %s",
			v.source, v.line, v.pattern, v.text)
	}

	if len(violations) > 0 {
		t.Logf("\n%d unsafe pattern(s) found across embedded prompts.\n"+
			"Fix: use --body-file <path> or --body \"$(cat <<'EOF'\\n...\\nEOF\\n)\" "+
			"instead of inline --body \"<...>\" with double-quoted interpolation.",
			len(violations))
	}

	t.Logf("audited %d embedded prompts", len(prompts))
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
		{
			name: "message with cat from file",
			line: `glab issue note 42 --message "$(cat /tmp/glab-comment.md)"`,
		},
		{
			name: "message with cat heredoc",
			line: `glab issue note 42 --message "$(cat <<'EOF'`,
		},
	}

	for _, tt := range safeLines {
		t.Run(tt.name, func(t *testing.T) {
			violations := scanContent("test", tt.line)
			if len(violations) > 0 {
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
		{
			name:    "message with double-quoted interpolation",
			line:    `glab issue note 42 --message "$UNTRUSTED"`,
			pattern: "message",
		},
		{
			name:    "message with double-quoted HTML content",
			line:    `glab mr note 5 --message "<h1>injected</h1>"`,
			pattern: "message",
		},
	}

	for _, tt := range unsafeLines {
		t.Run(tt.name, func(t *testing.T) {
			violations := scanContent("test", tt.line)
			if len(violations) == 0 {
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
				t.Logf("persona %q references %s CLI (%s) - checked for --description and --message patterns",
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

	// Verify that --message patterns are covered by the main audit
	testLine = `glab issue note 42 --message "<content>"`
	if !unsafeMessagePattern.MatchString(testLine) {
		t.Error("audit regex should detect --message with unsafe interpolation")
	}
}
