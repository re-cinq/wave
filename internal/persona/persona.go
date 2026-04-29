// Package persona defines the neutral persona type used by the agent compiler.
//
// It exists in its own package so that both internal/manifest (which loads
// persona definitions from wave.yaml) and internal/adapter (which compiles a
// persona into an agent .md file) can reference a shared type without forming
// an import cycle. Prior to this package, internal/adapter defined a private
// PersonaSpec mirror of the manifest persona to dodge the cycle.
package persona

// Persona holds the subset of persona configuration needed by the agent
// compiler. It is intentionally narrower than manifest.Persona: only the
// fields written into the generated agent .md frontmatter are present.
//
// Callers that hold a manifest.Persona should construct this struct from
// the fields they need (Model, Permissions.AllowedTools, Permissions.Deny).
type Persona struct {
	// Model is the Claude model identifier (e.g. "claude-opus-4") or tier alias
	// (cheapest, balanced, strongest, resolved before this point by the executor).
	// Leave empty to omit the frontmatter field and inherit the CLI default.
	Model string

	// AllowedTools is the list of tool names the agent may use.
	AllowedTools []string

	// DenyTools is the list of tool patterns the agent must not use.
	DenyTools []string
}
