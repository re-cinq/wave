# Data Model: Wave Init Interactive Skill Selection

**Feature Branch**: `384-init-skill-selection`
**Date**: 2026-03-14

## New Types

### EcosystemDef (internal/onboarding/skill_step.go)

Defines an ecosystem available for selection during onboarding.

```go
type EcosystemDef struct {
    Name        string          // Display name (e.g., "tessl", "BMAD")
    Value       string          // Select option value (e.g., "tessl", "bmad")
    Prefix      string          // SourceAdapter prefix for installation routing
    Dep         skill.CLIDependency // CLI binary required + install instructions
    InstallAll  bool            // true = bulk install, false = individual selection
    Description string          // Short description for the selection form
}
```

**Usage**: Static slice `ecosystems` defined at package level. Used to populate the ecosystem `huh.Select` form and to drive the post-selection behavior (multi-select vs. confirm/skip).

### SkillSelectionStep (internal/onboarding/skill_step.go)

New `WizardStep` implementation for the ecosystem/skill selection flow.

```go
type SkillSelectionStep struct {
    LookPath   lookPathFunc    // For testing: override exec.LookPath
    RunCommand commandRunner   // For testing: override subprocess execution
}
```

**Interface**: Implements `WizardStep` — `Name() string` returns `"Skill Selection"`, `Run(cfg *WizardConfig) (*StepResult, error)` orchestrates the full flow.

**Behavior by mode**:
- **Non-interactive** (`!cfg.Interactive`): Return immediately with empty skills (skip)
- **Interactive, tessl**: Check CLI → `tessl search ""` → `huh.MultiSelect` → install each selected skill → return names
- **Interactive, install-all**: Check CLI → `huh.Confirm` → run adapter `Install()` → return names
- **Interactive, skip**: Return immediately with empty skills

## Modified Types

### WizardResult (internal/onboarding/onboarding.go)

```go
type WizardResult struct {
    // ... existing fields ...
    Skills []string // NEW: bare skill names installed during onboarding
}
```

### WizardConfig (internal/onboarding/onboarding.go)

No changes needed. The `Reconfigure` and `Existing` fields already support reconfiguration. `Existing.Skills` (from `Manifest.Skills`) provides previously installed skill names.

## Data Flow

```
┌─────────────────────┐
│ RunWizard()         │
│                     │
│ Steps 1-5 (exist)  │
│         │           │
│ Step 6: SkillSelectionStep.Run()
│         │           │
│         ▼           │
│ ┌─ ecosystem select │
│ │  tessl|bmad|...   │
│ │  │                │
│ │  ├─ tessl:        │
│ │  │  tessl search  │──▶ parseTesslSearchOutput()
│ │  │  MultiSelect   │
│ │  │  for each:     │
│ │  │    router.Install("tessl:<name>", store)
│ │  │                │
│ │  ├─ install-all:  │
│ │  │  Confirm       │
│ │  │  router.Install("<prefix>:", store)
│ │  │                │
│ │  └─ skip:         │
│ │     return []     │
│ │                   │
│ └─▶ StepResult.Data["skills"] = []string{names...}
│         │           │
│         ▼           │
│ result.Skills = ... │
│         │           │
│ writeManifest()     │
│   buildManifest()   │
│     m["skills"] = result.Skills  (when non-empty)
│         │           │
│ MarkOnboarded()     │
└─────────────────────┘
```

## Store Interaction

The `SkillSelectionStep` creates a `skill.DirectoryStore` targeting `.wave/skills/` (project-level, precedence 2) as the installation target. This matches the `newSkillStore()` pattern in `cmd/wave/commands/skills.go`.

A `skill.SourceRouter` (via `skill.NewDefaultRouter(".")`) routes prefixed source strings to the correct adapter. For tessl, individual skills are installed as `"tessl:<name>"`. For install-all ecosystems, the adapter ignores the reference (e.g., `"bmad:"` routes to `BMADAdapter` which runs its bulk install command).

## Manifest Output

When `result.Skills` is non-empty, `buildManifest()` adds:

```yaml
skills:
  - golang
  - spec-kit
  - agentic-coding
```

This matches the `Manifest.Skills []string` field format — bare names, no source prefixes.

## Reconfiguration Context

When `cfg.Reconfigure && cfg.Existing != nil`:
- `cfg.Existing.Skills` contains previously installed skill names
- The ecosystem selection form displays: "Currently installed: golang, spec-kit" as context
- The user can choose a new ecosystem or skip
- Previously installed skills are NOT removed — only new skills are added
- The final `result.Skills` is the union of existing + newly installed

## CLI Dependency Definitions

| Ecosystem | CLI Binary | Install Instructions | Install-All? |
|-----------|-----------|---------------------|-------------|
| tessl     | `tessl`   | `npm i -g @tessl/cli` | No (individual select) |
| BMAD      | `npx`     | `npm i -g npx (comes with npm)` | Yes |
| OpenSpec  | `openspec`| `npm i -g @openspec/cli` | Yes |
| Spec-Kit  | `specify` | `npm i -g @speckit/cli` | Yes |

Source: `internal/skill/source_cli.go` — each adapter's `CLIDependency` field.

## Testing Interfaces

For testability, `SkillSelectionStep` accepts injectable functions:

```go
type lookPathFunc func(string) (string, error)
type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
```

These allow tests to:
- Simulate CLI presence/absence without requiring actual binaries
- Return canned `tessl search` output without network access
- Verify install commands without running adapters
