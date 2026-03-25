# Research: Publish Wave Skills as Standalone SKILL.md Artifacts

**Feature**: #559 Skills Publish
**Date**: 2026-03-24

## R1: Skill Classification Strategy

**Decision**: Content-based heuristic classification using Wave-reference detection.

**Rationale**: Automated classification by scanning SKILL.md body content for Wave-specific references (keywords: `wave`, `.wave/`, `pipeline`, `persona`, `wave.yaml`, `manifest`, `worktree`, `wave run`, `wave init`). Skills with zero Wave references are `standalone`, skills above a high threshold are `wave-specific`, and those in between are `both`.

**Alternatives Rejected**:
- Manual annotation in frontmatter: Requires upfront author effort, fragile, not scalable
- AI-based semantic analysis: Over-engineered for 13 skills, adds latency and non-determinism

**Evidence from codebase scan** (Wave-reference counts per skill):
| Skill | Wave refs | Classification |
|-------|-----------|---------------|
| agentic-coding | 2 | both |
| bmad | 5 | both |
| cli | 0 | standalone |
| ddd | 0 | standalone |
| gh-cli | 0 | standalone |
| golang | 0 | standalone |
| opsx | 0 | standalone |
| software-architecture | 0 | standalone |
| software-design | 0 | standalone |
| spec-driven-development | 0 | standalone |
| speckit | 1 | both |
| tui | 0 | standalone |
| wave | 139 | wave-specific |

Thresholds: 0 refs → `standalone`, 1-10 refs → `both`, >10 refs → `wave-specific`.

## R2: Registry Integration via Tessl CLI

**Decision**: Delegate publishing to the `tessl` CLI binary, consistent with existing `search` and `sync` commands.

**Rationale**: The codebase already wraps `tessl search` and `tessl install` via subprocess execution (see `source_cli.go:TesslAdapter` and `skills.go:runSkillsSearch`). Publishing follows the same pattern: `tessl publish <path>`. This avoids reimplementing the Tessl HTTP API and stays consistent with the existing adapter architecture.

**Alternatives Rejected**:
- Direct HTTP API calls to Tessl: Would require API key management, HTTP client code, and duplicating what `tessl publish` already does. Violates DRY.
- GitHub Releases as registry: Not aligned with the existing `tessl:`-prefixed ecosystem. Could be supported later via `--registry github`.

**Key Tessl CLI commands**:
- `tessl publish <path>` — publish a skill directory (SKILL.md + resources)
- `tessl search <query>` — already used in `wave skills search`
- `tessl install <ref>` — already used in `wave skills sync`

## R3: Content Digest Computation

**Decision**: SHA-256 hash computed over SKILL.md raw bytes concatenated with sorted resource file bytes, with path separators as delimiters.

**Rationale**: SHA-256 is the industry standard for content addressing (used by npm, Docker, Go modules). Computing over raw file bytes (not parsed content) ensures bitwise reproducibility. Including resource files prevents content-only digest bypass.

**Algorithm**:
```
h = sha256.New()
h.Write(skillMdBytes)
for _, resource := range sortedResourcePaths {
    h.Write([]byte("\n---resource:" + resourceRelPath + "---\n"))
    h.Write(resourceBytes)
}
digest = hex.EncodeToString(h.Sum(nil))
```

**Alternatives Rejected**:
- Merkle tree: Over-engineered for <50 files per skill. Simple concatenated hash is sufficient and debuggable.
- Hash only SKILL.md: Misses resource file changes (scripts, references, assets).

## R4: Lockfile Format

**Decision**: JSON file at `.wave/skills.lock` with an array of publish records.

**Rationale**: JSON is machine-parseable, human-readable, and consistent with Go's `encoding/json`. The `.wave/` directory is the established location for Wave metadata. The lockfile should be committed to version control for team integrity verification.

**Schema**:
```json
{
  "version": 1,
  "published": [
    {
      "name": "golang",
      "digest": "sha256:abc123...",
      "registry": "tessl",
      "url": "https://tessl.io/skills/golang",
      "published_at": "2026-03-24T12:00:00Z"
    }
  ]
}
```

**Alternatives Rejected**:
- YAML lockfile: Go's YAML marshaling is less predictable for round-trips than JSON
- Separate file per skill: Would clutter `.wave/` directory

## R5: Atomic Lockfile Updates

**Decision**: Write-to-temp-then-rename pattern for atomic lockfile updates.

**Rationale**: If a publish operation fails mid-batch, the lockfile must not reflect partial state. The standard pattern is: write to a `.lock.tmp` file, then `os.Rename()` to `.lock`. On POSIX filesystems, rename is atomic within the same directory.

**Implementation**:
```go
tmpPath := lockfilePath + ".tmp"
os.WriteFile(tmpPath, data, 0644)
os.Rename(tmpPath, lockfilePath)
```

## R6: SKILL.md Spec Validation

**Decision**: Reuse existing `parse.go:ValidateFields()` for required-field validation, extend with optional-field warnings.

**Rationale**: The `internal/skill/parse.go` already validates `name`, `description`, and `compatibility` fields via `ValidateFields()`. The agentskills.io spec compliance check extends this with warnings for missing optional fields (`license`, `compatibility`) without blocking the publish.

**Validation levels**:
- **Error (blocks publish)**: Missing `name` or `description`, invalid name format
- **Warning (does not block)**: Missing `license`, missing `compatibility`, missing `allowed-tools`

## R7: Distributable Skills Directory

**Decision**: Create a `skills/` directory at repo root with standalone skills organized as `skills/<name>/SKILL.md`, suitable for independent consumption.

**Rationale**: FR-016 requires skills to be consumable independently of the Wave binary. A top-level `skills/` directory with each skill as a subdirectory mirrors the `.claude/skills/` convention and can be published as a separate git repository or npm package.

**Alternatives Rejected**:
- Flatten into single directory: Loses resource file organization
- Use `.claude/skills/` directly: Contains wave-specific skills mixed in, not suitable for public distribution
