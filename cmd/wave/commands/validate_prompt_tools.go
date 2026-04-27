package commands

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
)

// PromptToolWarnEnv, when set to a truthy value, downgrades prompt/tool
// permission mismatches from hard errors to warnings. This is useful when a
// pipeline legitimately mentions a tool name in prose (e.g., documentation
// generation) without intending the model to invoke the tool.
const PromptToolWarnEnv = "WAVE_PROMPT_TOOLS_WARN"

// promptToolFlagWarn mirrors PromptToolWarnEnv as a CLI flag value (set in
// runValidate). Either source is honoured.
var promptToolFlagWarn bool

// promptToolMentionPatterns maps each canonical Claude/Wave tool name to the
// regular expressions used to detect a likely call-out in prose. The patterns
// favour high-precision matches (tool name followed by tool/the/a/to/for, or a
// parenthesised argument list) to keep the false-positive rate low while still
// catching the bug described in the validator: "use the Write tool", "Write a
// JSON file", "Bash(git push)", "use WebFetch to ...".
//
// Patterns are case-insensitive. The grouping anchors each tool name on a word
// boundary so that "Read" does not also match "Already".
var promptToolMentionPatterns = map[string][]*regexp.Regexp{
	"Write":     compilePromptPatterns("Write"),
	"Edit":      compilePromptPatterns("Edit"),
	"MultiEdit": compilePromptPatterns("MultiEdit"),
	"Bash":      compilePromptPatterns("Bash"),
	"Read":      compilePromptPatterns("Read"),
	"Glob":      compilePromptPatterns("Glob"),
	"Grep":      compilePromptPatterns("Grep"),
	"WebFetch":  compileBareToolPattern("WebFetch"),
	"WebSearch": compileBareToolPattern("WebSearch"),
	"Task":      compileBareToolPattern("Task"),
}

// compilePromptPatterns builds the heuristic regex set for tools that take an
// object/argument (Read, Write, Edit, Bash, Glob, Grep, MultiEdit). We require
// either a parenthesised arg form (`Tool(...)`) or one of a small set of
// follow-words that strongly imply a tool reference rather than incidental
// prose. The follow-word list is intentionally narrow to avoid flagging things
// like "we will read carefully" while still catching "Write a JSON file".
func compilePromptPatterns(tool string) []*regexp.Regexp {
	q := regexp.QuoteMeta(tool)
	return []*regexp.Regexp{
		// Tool(arg) — Bash(git push), Write(/tmp/foo)
		regexp.MustCompile(`(?i)\b` + q + `\s*\([^)]*\)`),
		// "Write tool", "Bash tool"
		regexp.MustCompile(`(?i)\b` + q + `\s+tool\b`),
		// "Write the", "Write a", "Write an", "Write to", "Write back",
		// "Write into", "Write out". Same for Read/Edit/etc.
		regexp.MustCompile(`(?i)\b` + q + `\s+(the|a|an|to|back|into|out|for)\b`),
		// "use Write" / "uses Write" / "using Write"
		regexp.MustCompile(`(?i)\buse[sd]?\s+` + q + `\b`),
		regexp.MustCompile(`(?i)\busing\s+` + q + `\b`),
	}
}

// compileBareToolPattern handles tools whose names are themselves unambiguous
// in English text (WebFetch, WebSearch, Task). Any case-insensitive mention is
// suspicious for these.
func compileBareToolPattern(tool string) []*regexp.Regexp {
	q := regexp.QuoteMeta(tool)
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b` + q + `\b`),
	}
}

// detectPromptToolMentions scans the prompt body for tool-name mentions and
// returns the set of canonical tool names mentioned. The returned slice is
// sorted for deterministic output.
func detectPromptToolMentions(prompt string) []string {
	if prompt == "" {
		return nil
	}
	mentioned := make(map[string]bool)
	for tool, patterns := range promptToolMentionPatterns {
		for _, p := range patterns {
			if p.MatchString(prompt) {
				mentioned[tool] = true
				break
			}
		}
	}
	if len(mentioned) == 0 {
		return nil
	}
	out := make([]string, 0, len(mentioned))
	for t := range mentioned {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// extractAllowedToolNames returns the canonical tool names a persona is
// granted, stripping any parenthesised argument scope. For example, given
// `["Read", "Write(.agents/artifact.json)", "Bash(git log*)"]` it returns
// `["Bash", "Read", "Write"]`.
func extractAllowedToolNames(perms manifest.Permissions) []string {
	if len(perms.AllowedTools) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(perms.AllowedTools))
	for _, t := range perms.AllowedTools {
		base := strings.TrimSpace(t)
		if i := strings.Index(base, "("); i > 0 {
			base = base[:i]
		}
		base = strings.TrimSpace(base)
		if base != "" {
			seen[base] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// readStepPrompt resolves the prompt body for a step. Inline `source` wins;
// otherwise the contents of `source_path` are read (templated paths containing
// `{{` are skipped). Returns ("", nil) when neither source nor a readable path
// exists — the caller treats this as "nothing to scan".
func readStepPrompt(step pipeline.Step) (string, error) {
	if step.Exec.Source != "" {
		return step.Exec.Source, nil
	}
	if step.Exec.SourcePath == "" {
		return "", nil
	}
	if strings.Contains(step.Exec.SourcePath, "{{") {
		return "", nil
	}
	data, err := os.ReadFile(step.Exec.SourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// promptToolFinding describes a single mismatch between the tools mentioned
// in a step's prompt and the tools the resolved persona is granted.
type promptToolFinding struct {
	Pipeline string
	StepID   string
	Persona  string
	Tool     string
	Allowed  []string
}

func (f promptToolFinding) String() string {
	return fmt.Sprintf(
		"%s:%s asks for %q but persona %q only grants %v",
		f.Pipeline, f.StepID, f.Tool, f.Persona, f.Allowed,
	)
}

// validatePromptToolPermissions walks every prompt-type step in the parsed
// pipeline, resolves the persona via the manifest, and reports any tool
// mentioned in the prompt that the persona does not grant. Composition steps
// (gates, branches, loops, sub-pipelines, command/conditional steps) are
// skipped — they do not run a model under a persona's permissions.
//
// Findings are returned even when a step has no persona resolved (the upstream
// validator already reports that as a separate error).
func validatePromptToolPermissions(pipelineName string, p *pipeline.Pipeline, m *manifest.Manifest) []promptToolFinding {
	if p == nil || m == nil {
		return nil
	}
	var findings []promptToolFinding
	for _, step := range p.Steps {
		if isCompositionStep(step) {
			continue
		}
		if step.Persona == "" {
			continue
		}
		// Skip non-prompt execs (slash_command, command). Empty Type is
		// historically treated as a prompt by the executor.
		if step.Exec.Type != "" && step.Exec.Type != "prompt" {
			continue
		}

		prompt, err := readStepPrompt(step)
		if err != nil {
			continue
		}
		mentioned := detectPromptToolMentions(prompt)
		// The executor auto-injects "Write valid <type> to <path> using the
		// Write tool" into the prompt for any step declaring file-based
		// output_artifacts (see executor.go buildContractPrompt). The YAML
		// itself often does not mention Write — so without this implicit
		// pass we miss the most common navigator+output_artifacts mismatch.
		if hasFileOutputArtifact(step) {
			mentioned = appendUnique(mentioned, "Write")
		}
		if len(mentioned) == 0 {
			continue
		}

		persona := resolvePersonaForStep(step, m)
		if persona == nil {
			// Persona missing from manifest — already reported elsewhere.
			continue
		}
		// Honor the per-step Permissions overlay so a pipeline can grant a tool
		// to a single step without changing its persona. The adapter argument is
		// nil here because the validator runs against the manifest before any
		// adapter is selected; adapter-level grants don't apply at validate time.
		effective := pipeline.ResolveStepPermissions(&step, persona, nil)
		allowed := extractAllowedToolNames(effective)
		allowedSet := make(map[string]bool, len(allowed))
		for _, a := range allowed {
			allowedSet[a] = true
		}

		// Wildcard "*" in the persona's allowed list grants every tool.
		if allowedSet["*"] {
			continue
		}
		for _, tool := range mentioned {
			if allowedSet[tool] {
				continue
			}
			findings = append(findings, promptToolFinding{
				Pipeline: pipelineName,
				StepID:   step.ID,
				Persona:  step.Persona,
				Tool:     tool,
				Allowed:  allowed,
			})
		}
	}
	return findings
}

// hasFileOutputArtifact reports whether the step declares at least one
// file-based output_artifact. The executor injects "Write valid <type> ...
// using the Write tool" into the prompt for any such step, so the persona
// must grant Write even when the YAML prompt itself never says it.
//
// stdout artifacts (path == "stdout") are excluded — the executor captures
// stdout instead of asking the model to Write a file.
func hasFileOutputArtifact(step pipeline.Step) bool {
	for _, art := range step.OutputArtifacts {
		if !art.IsStdoutArtifact() {
			return true
		}
	}
	return false
}

// appendUnique appends s to slice if not already present, preserving order.
func appendUnique(slice []string, s string) []string {
	for _, x := range slice {
		if x == s {
			return slice
		}
	}
	return append(slice, s)
}

// resolvePersonaForStep returns the persona referenced by a step, expanding
// `{{ forge.type }}` placeholders by trying each known forge expansion in turn
// (github, gitlab, gitea, bitbucket). The first persona that resolves wins;
// if none do we return nil so the caller can skip silently.
func resolvePersonaForStep(step pipeline.Step, m *manifest.Manifest) *manifest.Persona {
	name := step.Persona
	if !strings.Contains(name, "{{") {
		return m.GetPersona(name)
	}
	for _, ft := range []string{"github", "gitlab", "gitea", "bitbucket"} {
		expanded := strings.ReplaceAll(name, "{{ forge.type }}", ft)
		expanded = strings.ReplaceAll(expanded, "{{forge.type}}", ft)
		if p := m.GetPersona(expanded); p != nil {
			return p
		}
	}
	return nil
}

// promptToolWarnEnabled reports whether prompt/tool mismatches should be
// downgraded to warnings. Honoured sources: --prompt-tools-warn flag (set on
// runValidate options) or the WAVE_PROMPT_TOOLS_WARN env var (truthy values:
// "1", "true", "yes", "on").
func promptToolWarnEnabled(opts ValidateOptions) bool {
	if opts.PromptToolsWarn || promptToolFlagWarn {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv(PromptToolWarnEnv)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
