package pipeline

import "github.com/recinq/wave/internal/manifest"

// ResolveStepPermissions merges tool permissions from three precedence layers
// into the effective permission set for a single step invocation:
//
//  1. step.Permissions   — per-step overlay declared in the pipeline YAML
//  2. persona.Permissions — the agent persona configured in wave.yaml
//  3. adapterDef.DefaultPermissions — the adapter-wide baseline
//
// The merge is additive on AllowedTools: a step may ADD tools the persona does
// not normally grant (e.g. a navigator step that needs Write for a specific
// artifact path). The merge is also additive on Deny — deny rules from any
// layer compose, and the underlying PermissionChecker enforces deny-first
// precedence so persona-level prohibitions cannot be removed by a step
// declaration. Order within each slice is preserved (highest precedence
// first) and duplicate patterns are collapsed.
//
// Either of step or adapterDef may be nil; persona must be non-nil, since
// invoking a step without a persona has no defined permission semantics.
func ResolveStepPermissions(step *Step, persona *manifest.Persona, adapterDef *manifest.Adapter) manifest.Permissions {
	var stepPerms manifest.Permissions
	if step != nil {
		stepPerms = step.Permissions
	}
	var adapterPerms manifest.Permissions
	if adapterDef != nil {
		adapterPerms = adapterDef.DefaultPermissions
	}
	var personaPerms manifest.Permissions
	if persona != nil {
		personaPerms = persona.Permissions
	}

	return manifest.Permissions{
		AllowedTools: mergePatterns(stepPerms.AllowedTools, personaPerms.AllowedTools, adapterPerms.AllowedTools),
		Deny:         mergePatterns(stepPerms.Deny, personaPerms.Deny, adapterPerms.Deny),
	}
}

// mergePatterns concatenates pattern slices in precedence order, dropping
// empties and collapsing exact-string duplicates. The first occurrence of
// each pattern wins, so highest-precedence entries appear earliest in the
// resulting slice. Returns nil when no input contains entries, matching the
// zero value of Permissions.AllowedTools / Permissions.Deny so downstream
// code sees the same shape it would have without an override.
func mergePatterns(layers ...[]string) []string {
	var total int
	for _, l := range layers {
		total += len(l)
	}
	if total == 0 {
		return nil
	}
	seen := make(map[string]struct{}, total)
	out := make([]string, 0, total)
	for _, layer := range layers {
		for _, pat := range layer {
			if pat == "" {
				continue
			}
			if _, dup := seen[pat]; dup {
				continue
			}
			seen[pat] = struct{}{}
			out = append(out, pat)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
