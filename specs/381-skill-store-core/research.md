# Research: Skill Store Core

**Feature**: #381 — Skill Store Core
**Date**: 2026-03-13

## Research Questions

### RQ-1: YAML Frontmatter Parsing in Go

**Decision**: Use `gopkg.in/yaml.v3` (already a project dependency) with manual frontmatter delimiter splitting.

**Rationale**: The Agent Skills Specification format uses `---` delimited YAML frontmatter followed by a markdown body. Go has no standard frontmatter parsing library that's widely adopted. The most reliable approach is:

1. Split the file content on the `---` delimiter pattern (first `---` opens frontmatter, second `---` closes it)
2. Parse the extracted YAML block with `gopkg.in/yaml.v3`
3. Everything after the closing `---` becomes the markdown body

**Alternatives Rejected**:
- `github.com/adrg/frontmatter` — adds a dependency for trivial functionality (split on `---`, unmarshal YAML)
- `github.com/gohugoio/hugo/parser/pageparser` — massive dependency for a single feature
- Custom regex — fragile; delimiter-based splitting is cleaner and handles edge cases better

**Implementation Note**: The frontmatter parser should operate on `[]byte` input (not file path) so it can be tested without filesystem access. File I/O is the store's responsibility.

### RQ-2: Skill Name Validation Regex

**Decision**: Use `regexp.MustCompile("^[a-z0-9]([a-z0-9-]*[a-z0-9])?$")` with max 64 character length.

**Rationale**: Matches the spec's FR-001 constraint. This pattern:
- Starts and ends with alphanumeric
- Allows hyphens in the middle
- No uppercase, no underscores, no dots
- Compatible with directory names on all platforms
- Compatible with existing skill names in `.claude/skills/` (all 13 match)

**Verification**: All 13 existing skill directory names match this pattern:
`agentic-coding`, `bmad`, `cli`, `ddd`, `gh-cli`, `golang`, `opsx`, `software-architecture`, `software-design`, `spec-driven-development`, `speckit`, `tui`, `wave`

### RQ-3: Path Traversal Prevention Strategy

**Decision**: Validate skill names at the domain layer (before any filesystem operation), rejecting names containing `/`, `\`, `..`, or any path separator character.

**Rationale**: The existing `internal/security/path.go` `PathValidator` is designed for validating full file paths against approved directory lists with logging integration. For skill name validation, a simpler approach is sufficient:
- Skill names are identifiers, not paths — they must match `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
- If the name matches this regex, it cannot contain path traversal sequences
- The regex validation IS the path traversal prevention — no need for a separate check
- Additional defense-in-depth: `filepath.Join(root, name)` + verify result starts with `root`

**Alternatives Rejected**:
- Reusing `security.PathValidator` — too heavyweight; requires `SecurityConfig` and `SecurityLogger` setup, designed for full path validation not identifier validation
- Only checking for `..` — insufficient; the regex approach is both simpler and more comprehensive

### RQ-4: Error Type Naming — ParseError vs ValidationError

**Decision**: Name the error type `ParseError` to avoid collision with existing `manifest.ValidationError` and `contract.ValidationError`.

**Rationale**: The codebase already has:
- `manifest.ValidationError` — manifest parsing/validation errors
- `contract.ValidationError` — contract validation errors
- `security.SecurityValidationError` — security validation errors

Adding `skill.ValidationError` would create confusion. `ParseError` accurately describes the error domain (parsing SKILL.md files and validating their content). Per C-005 in the spec.

### RQ-5: Store Interface Design

**Decision**: Define a `Store` interface with 4 methods: `Read`, `Write`, `List`, `Delete`. Implement `DirectoryStore` as the concrete type.

**Rationale**: Wave uses interfaces extensively for testability:
- `tui.SkillDataProvider` — interface for TUI skill data
- `adapter.Adapter` — interface for subprocess adapters
- Pipeline executor uses dependency injection for all external dependencies

The store will be consumed by:
1. Pipeline executor (to inject skill instructions into agent context)
2. CLI commands (`wave list skills`, future `wave skill add/remove`)
3. TUI (could replace or supplement `DefaultSkillDataProvider`)

Interface enables mock injection in all consumers.

### RQ-6: Multi-Source Resolution Order

**Decision**: Source directories are ordered by precedence (highest first). Default order: `.wave/skills/` (project-local) > `.claude/skills/` (user-level).

**Rationale**: Follows Wave's existing configuration layering:
- Project-local configs override user/system defaults
- `.wave/` is the project-level Wave directory
- `.claude/skills/` is where Claude Code stores user skills

**Implementation**: `DirectoryStore` holds an ordered slice of `SkillSource`. List merges all sources; first occurrence of a name wins. Read checks sources in order, returns first match.

### RQ-7: Coexistence with Legacy Provisioner

**Decision**: New types and functions live alongside existing code in `internal/skill/`. No changes to `SkillConfig`, `Provisioner`, or existing tests.

**Rationale**: Per FR-010 and C-002:
- `SkillConfig` + `Provisioner` = legacy command-file provisioning path
- `Skill` + `Store` = new SKILL.md-based store path
- Both represent different aspects of skill management
- The migration from legacy to new system is a future concern (not #381 scope)
- Same package avoids import cycles if cross-references are needed later

### RQ-8: `allowed-tools` Format

**Decision**: Parse as a single YAML string, split on whitespace into `[]string` on the Go struct.

**Rationale**: Per C-001 in the spec. The Agent Skills Specification defines `allowed-tools` as a space-delimited string: `allowed-tools: "Read Write Edit Bash"`. No existing SKILL.md files use this field yet, but conforming to the spec format ensures interoperability. The parser handles both quoted and unquoted YAML string values.

### RQ-9: Progressive Disclosure Loading

**Decision**: Implement two loading modes in the parser:
1. **Metadata-only**: Parse frontmatter only (stop after closing `---`). Returns `Skill` with name + description but empty body. Used for listing.
2. **Full load**: Parse frontmatter + body. Returns complete `Skill`. Used for activation.

**Rationale**: FR-011 specifies a three-tier progressive disclosure model. The first two tiers (metadata-only and full SKILL.md) are implemented in the parser. The third tier (resource file discovery) is implemented in the store's Read operation by scanning for `scripts/`, `references/`, and `assets/` subdirectories.

For 50+ skills (SC-008), metadata-only loading avoids reading full markdown bodies when only listing names and descriptions.
