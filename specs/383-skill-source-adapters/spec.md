# Feature Specification: Ecosystem Adapters for Skill Sources

**Feature Branch**: `383-skill-source-adapters`
**Created**: 2026-03-14
**Status**: Clarified
**Input**: [Issue #383](https://github.com/re-cinq/wave/issues/383) — Ecosystem adapters for skill sources. Source prefix routing (tessl:, bmad:, openspec:, speckit:, github:, file:, https://). Soft dependencies. Subsumes #77.
**Parent**: [Issue #239](https://github.com/re-cinq/wave/issues/239) — Skill management system

## User Scenarios & Testing _(mandatory)_

### User Story 1 — Install a Skill from Tessl Registry (Priority: P1)

A pipeline author wants to declare a skill dependency with a `tessl:` source prefix so that Wave automatically delegates installation to the Tessl CLI (`tessl install <tile>`), discovers the resulting SKILL.md, and copies it into `.wave/skills/`.

**Why this priority**: Tessl is the primary external skill registry. Supporting it unblocks the most common skill distribution use case and directly fulfills issue #77.

**Independent Test**: Can be tested by declaring a `tessl:` source in a skill reference, invoking the adapter, and verifying that the skill store contains a valid SKILL.md afterward.

**Acceptance Scenarios**:

1. **Given** a skill reference with source `tessl:github/spec-kit`, **When** the adapter is invoked, **Then** Wave delegates to `tessl install github/spec-kit`, locates the resulting SKILL.md, validates it, and writes it to the skill store.
2. **Given** the `tessl` CLI is not installed, **When** the adapter is invoked, **Then** Wave returns a clear error message naming the missing tool and providing install instructions (`npm i -g @tessl/cli`).
3. **Given** the Tessl registry does not contain the requested tile, **When** the adapter is invoked, **Then** Wave returns an error including the tile reference and "not found" reason.

---

### User Story 2 — Install a Skill from a GitHub Repository (Priority: P1)

A user wants to install a skill directly from a GitHub repository by specifying `github:owner/repo` (or `github:owner/repo/path/to/skill`), so Wave can clone the repo, locate SKILL.md files, and copy them into `.wave/skills/`.

**Why this priority**: GitHub is the most common code hosting platform. Many skills will be shared via repositories before they appear in any registry.

**Independent Test**: Can be tested by pointing the adapter at a public GitHub repo containing a SKILL.md, invoking it, and verifying the skill appears in the store.

**Acceptance Scenarios**:

1. **Given** a skill reference `github:re-cinq/wave-skills/golang`, **When** the adapter is invoked, **Then** Wave clones the repository, navigates to the `golang` subdirectory, validates the SKILL.md, and writes the skill to the store.
2. **Given** a skill reference `github:owner/repo` (no path suffix), **When** the repository root contains a SKILL.md, **Then** Wave installs that single skill.
3. **Given** a skill reference `github:owner/repo` (no path suffix), **When** the repository root contains no SKILL.md but has multiple skill subdirectories, **Then** Wave discovers and installs all valid skills found.
4. **Given** `git` is not available on PATH, **When** the adapter is invoked, **Then** Wave returns a clear error message.

---

### User Story 3 — Install a Skill from a Local Path (Priority: P2)

A skill developer wants to install a skill from a local directory (`file:./my-skills/custom-skill`) during development, so they can iterate without publishing to any registry.

**Why this priority**: Essential for the skill development inner loop — authors need to test skills locally before distributing them.

**Independent Test**: Can be tested by creating a local directory with a valid SKILL.md and invoking the adapter with a `file:` reference.

**Acceptance Scenarios**:

1. **Given** a skill reference `file:./my-skills/custom-skill`, **When** the directory contains a valid SKILL.md, **Then** Wave copies the skill directory into the store.
2. **Given** a skill reference `file:/absolute/path/to/skill`, **When** the directory exists, **Then** Wave resolves the absolute path, validates, and copies to the store.
3. **Given** the referenced path does not exist, **When** the adapter is invoked, **Then** Wave returns a "path not found" error with the resolved absolute path.
4. **Given** the referenced path is a symlink pointing outside the project, **When** the adapter is invoked, **Then** Wave rejects it with a path containment error.

---

### User Story 4 — Install Skills from Ecosystem CLIs: BMAD, OpenSpec, Spec-Kit (Priority: P2)

A user wants to install skills from the BMAD, OpenSpec, or Spec-Kit ecosystems using their respective prefixes (`bmad:`, `openspec:`, `speckit:`), so Wave delegates to the native CLI of each ecosystem.

**Why this priority**: These are Wave's core partner ecosystems. Supporting them via dedicated adapters provides a unified installation experience.

**Independent Test**: Can be tested per ecosystem by invoking the adapter with the ecosystem prefix and verifying correct CLI delegation and SKILL.md extraction.

**Acceptance Scenarios**:

1. **Given** a skill reference `bmad:install`, **When** the adapter is invoked, **Then** Wave delegates to `npx bmad-method install --tools claude-code --yes`, locates the resulting SKILL.md files, and writes them to the store.
2. **Given** a skill reference `openspec:init`, **When** the adapter is invoked, **Then** Wave delegates to `openspec init`, extracts workflow skill files, and writes them to the store.
3. **Given** a skill reference `speckit:init`, **When** the adapter is invoked, **Then** Wave delegates to `specify init`, extracts skill files, and writes them to the store.
4. **Given** the required CLI tool for any ecosystem is not installed, **When** the adapter is invoked, **Then** Wave returns a clear error naming the missing tool and providing install instructions.

---

### User Story 5 — Install a Skill from a URL Archive (Priority: P3)

A user wants to install a skill from a remote archive (`https://example.com/skills/my-skill.tar.gz`) so they can distribute skills via static hosting or release assets.

**Why this priority**: Provides a universal fallback for skill distribution that doesn't depend on any specific registry or CLI tool.

**Independent Test**: Can be tested by hosting a tar.gz archive containing a valid SKILL.md and invoking the adapter.

**Acceptance Scenarios**:

1. **Given** a skill reference `https://example.com/skill.tar.gz`, **When** the adapter downloads and extracts the archive, **Then** Wave validates SKILL.md contents and writes the skill to the store.
2. **Given** a skill reference `https://example.com/skill.zip`, **When** the adapter downloads and extracts the archive, **Then** Wave validates and writes the skill to the store.
3. **Given** a URL returns a non-archive response (e.g., HTML), **When** the adapter is invoked, **Then** Wave returns an error indicating the response is not a recognized archive format.
4. **Given** a URL is unreachable or returns an HTTP error, **When** the adapter is invoked, **Then** Wave returns a descriptive network error.

---

### User Story 6 — Source Prefix Routing (Priority: P1)

A pipeline author or CLI user provides a source string (e.g., `tessl:github/spec-kit`). The system parses the prefix, selects the correct adapter, and delegates installation.

**Why this priority**: This is the dispatch mechanism that enables all other stories. Without prefix routing, no adapter can be reached.

**Independent Test**: Can be tested by passing various source strings and verifying the correct adapter is selected (or an error for unknown prefixes).

**Acceptance Scenarios**:

1. **Given** a source string `tessl:github/spec-kit`, **When** parsed, **Then** the system selects the Tessl adapter with reference `github/spec-kit`.
2. **Given** a source string `github:owner/repo`, **When** parsed, **Then** the system selects the GitHub adapter with reference `owner/repo`.
3. **Given** a source string `file:./local/path`, **When** parsed, **Then** the system selects the local file adapter with reference `./local/path`.
4. **Given** a source string `https://example.com/skill.tar.gz`, **When** parsed, **Then** the system selects the URL adapter with the full URL as reference.
5. **Given** a source string with an unknown prefix `foobar:something`, **When** parsed, **Then** the system returns an error listing recognized prefixes.
6. **Given** a source string with no prefix (bare name like `golang`), **When** parsed, **Then** the system treats it as a local store lookup (not an adapter invocation).

---

### Edge Cases

- What happens when a `tessl:` source resolves to multiple skills? The adapter installs all discovered skills and returns a list.
- What happens when the extracted archive contains no valid SKILL.md? The adapter returns an error indicating no valid skill was found.
- What happens when a skill with the same name already exists in the store? The adapter overwrites it (latest install wins), consistent with `DirectoryStore.Write` behavior.
- What happens when two adapters are invoked concurrently for the same skill name? The last write wins; no locking is required since the store uses atomic file writes.
- What happens when the `file:` reference contains path traversal (`file:../../etc/passwd`)? The adapter rejects it via path containment validation.
- What happens when a `github:` reference points to a private repository? The adapter uses the system's git credentials; if authentication fails, it returns a clear error.
- What happens when the network is unavailable for `tessl:`, `github:`, or `https:` adapters? Each returns a descriptive network error without hanging (timeout enforced).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST define a `SourceAdapter` interface with `Install(ctx context.Context, ref string, store Store) (*InstallResult, error)` and `Prefix() string` methods that all source adapters implement (see Clarification C-005).
- **FR-002**: System MUST implement a source prefix router that parses source strings and dispatches to the correct adapter. The router MUST first check for URL-scheme prefixes (`https://`, `http://`) before applying the generic `prefix:reference` split on the first colon (see Clarification C-001).
- **FR-003**: System MUST support the following source prefixes: `tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`.
- **FR-004**: The `tessl:` adapter MUST delegate installation to the `tessl` CLI tool (`tessl install <reference>`).
- **FR-005**: The `bmad:` adapter MUST delegate installation to `npx bmad-method install --tools claude-code --yes`.
- **FR-006**: The `openspec:` adapter MUST delegate installation to `openspec init`.
- **FR-007**: The `speckit:` adapter MUST delegate installation to `specify init`.
- **FR-008**: The `github:` adapter MUST clone the specified repository and discover SKILL.md files within it.
- **FR-009**: The `file:` adapter MUST copy skill directories from local filesystem paths, enforcing path containment (no symlink traversal, no escape beyond project root).
- **FR-010**: The `https://` adapter MUST download and extract archive files (tar.gz, zip), then discover SKILL.md files within.
- **FR-011**: All ecosystem CLIs (`tessl`, `npx`, `openspec`, `specify`, `git`) MUST be treated as soft dependencies — when unavailable, the system MUST return an actionable error message naming the missing tool and providing install instructions.
- **FR-012**: Every adapter MUST validate extracted content against the existing skill parser before writing to the skill store.
- **FR-013**: Adapters MUST write installed skills to the skill store via its existing write operation.
- **FR-014**: Source strings with no recognized prefix (bare names) MUST be treated as local store lookups, not adapter invocations.
- **FR-015**: The system MUST return a descriptive error for unknown source prefixes, listing all recognized prefixes.
- **FR-016**: Adapters MUST enforce timeouts on network and subprocess operations to prevent indefinite hangs. Default: 2 minutes for subprocess/network operations, 30 seconds for HTTP response headers. Timeouts MUST be enforced via `context.WithTimeout` (see Clarification C-004).
- **FR-017**: The `github:` adapter MUST support optional path suffixes (`github:owner/repo/path/to/skill`) to install a specific skill from a multi-skill repository.

### Key Entities

- **SourceAdapter**: Interface with `Install(ctx, ref, store)` and `Prefix()` methods. Each adapter handles one source prefix. Receives `context.Context` for timeout enforcement and `Store` for dependency injection. Manages its own temporary directories for fetch operations (see C-003, C-005).
- **SourceRouter**: Registry of adapters keyed by prefix string. Parses source strings, checks URL-scheme prefixes first, then splits on first colon (see C-001). Dispatches to the matched adapter or returns an error listing known prefixes.
- **SourceReference**: A parsed source string consisting of a prefix (identifying the adapter) and a reference (the adapter-specific locator, e.g., `github/spec-kit` for Tessl, `owner/repo` for GitHub).
- **InstallResult**: The outcome of an adapter invocation — `Skills []Skill` (successfully installed) and `Warnings []string`.
- **SoftDependency**: An external CLI tool required by an adapter that may or may not be installed. Checked via `exec.LookPath`. When absent, produces a structured error with install instructions rather than a cryptic failure.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All 7 source prefixes (`tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`) route to the correct adapter without error when the prefix is parsed.
- **SC-002**: Each adapter produces at least one valid skill (passing parser validation) when given a well-formed reference pointing to valid content.
- **SC-003**: Every adapter returns an actionable error (naming the missing tool and install instructions) when its required CLI dependency is absent — verified by unit tests with mocked command lookup.
- **SC-004**: The `file:` adapter rejects all path traversal attempts (symlinks, `..` escape) — verified by unit tests exercising path containment constraints.
- **SC-005**: Unknown source prefixes produce an error listing all recognized prefixes — verified by unit test.
- **SC-006**: All adapter tests pass with `go test -race ./internal/skill/...` — no data races.
- **SC-007**: The adapter interface is extensible — a new adapter can be added by implementing the interface and registering its prefix, without modifying existing adapter code.

## Clarifications

The following ambiguities were identified during spec review and resolved based on codebase patterns and established conventions.

### C-001: `https://` Prefix Parsing Strategy

**Ambiguity**: The `https://` prefix doesn't follow the `prefix:reference` split-on-first-colon pattern used by other prefixes. Splitting `https://example.com/skill.tar.gz` on `:` yields prefix `https` and reference `//example.com/skill.tar.gz`, which loses the scheme semantics.

**Resolution**: The router MUST check for `https://` (and `http://`) as a prefix match _before_ applying the generic `prefix:reference` split. When a source string starts with `https://`, the entire URL is passed as the reference to the URL adapter. The prefix for registration purposes is `https://` (not `https`). This is a special case — all other prefixes use the single-colon split.

**Rationale**: This matches how URLs are universally parsed. The alternative (splitting on first colon) would require the URL adapter to reconstruct the scheme, adding unnecessary complexity. User Story 6 scenario 4 already implies this behavior: "the full URL as reference."

### C-002: Relationship Between `SourceAdapter` and Existing `SkillConfig`

**Ambiguity**: The existing `SkillConfig` type (in `internal/skill/types.go`) has `Install`, `Init`, and `Check` commands for manifest-declared skills. The spec introduces `SourceAdapter` as a new installation mechanism. It's unclear whether these are parallel systems or whether source adapters replace `SkillConfig.Install`.

**Resolution**: Source adapters and `SkillConfig` operate at different levels and coexist:
- **`SkillConfig`** is for manifest-level skill declarations (`wave.yaml` skills section) with custom shell commands for install/check. It is consumed by the preflight `Checker`.
- **`SourceAdapter`** is for resolving source-prefixed references (e.g., `tessl:github/spec-kit`) to installed skills in the store. It is a new subsystem in `internal/skill/`.
- A future manifest enhancement may allow `source: tessl:github/spec-kit` as an alternative to `install: tessl install github/spec-kit`, but that is out of scope for this feature. This feature introduces the adapter infrastructure only.

**Rationale**: The existing preflight system works and is deployed. Replacing it would be a breaking change with no benefit. The source adapter system adds new capability (prefix-based routing, structured install results) without disrupting existing manifest configurations.

### C-003: Temporary Directory for Adapter Work

**Ambiguity**: Adapters that clone repositories (`github:`) or download archives (`https://`) need a temporary scratch directory. The spec doesn't specify where this goes or how cleanup happens.

**Resolution**: Each adapter invocation MUST create a temporary directory via `os.MkdirTemp("", "wave-skill-*")` and clean it up with `defer os.RemoveAll(tmpDir)` before returning. The adapter's `Install` method is responsible for:
1. Creating the temp directory
2. Performing the fetch/clone/download into it
3. Discovering and parsing SKILL.md files within it
4. Writing validated skills to the store via `Store.Write`
5. Cleaning up the temp directory on return (success or error)

This means the temp directory is ephemeral and scoped to the adapter call — no persistent staging area is needed.

**Rationale**: This follows Go's standard temp directory pattern and matches Wave's ephemeral workspace philosophy. Using `os.TempDir()` as the parent avoids polluting the project tree. The `defer` cleanup pattern ensures no leaked directories even on error paths.

### C-004: Default Timeout Values for FR-016

**Ambiguity**: FR-016 requires timeouts on network and subprocess operations but specifies no default values.

**Resolution**: The following default timeouts apply:
- **Subprocess operations** (CLI adapters: tessl, bmad, openspec, speckit): **2 minutes** — consistent with `Runtime.GetDefaultTimeout()` which returns 5 minutes for full pipeline steps; individual CLI invocations should complete faster.
- **Network operations** (git clone, HTTP download): **2 minutes** — sufficient for reasonable repository sizes and archive downloads.
- **HTTP response header timeout**: **30 seconds** — fail fast if the server doesn't respond.
- All timeouts MUST be enforced via `context.WithTimeout` passed to `exec.CommandContext` (for subprocesses) or `http.Client.Timeout` / request context (for HTTP).
- Timeouts are NOT configurable in this initial implementation; they can be made configurable via manifest settings in a future enhancement if needed.

**Rationale**: 2 minutes aligns with Wave's existing adapter timeout patterns (the adapter package uses `exec.Command` with similar timeouts). 30 seconds for HTTP headers prevents hanging on unresponsive servers while allowing slow-but-progressing downloads to complete.

### C-005: `SourceAdapter` Interface Method Signature

**Ambiguity**: FR-001 specifies "a common adapter interface with at least an install operation" but doesn't define the Go method signature, context propagation, or return type.

**Resolution**: The `SourceAdapter` interface MUST have the following signature:

```go
type SourceAdapter interface {
    Install(ctx context.Context, ref string, store Store) (*InstallResult, error)
    Prefix() string
}
```

Where:
- `ctx` carries the timeout deadline and cancellation signal
- `ref` is the adapter-specific reference (everything after the prefix, e.g., `github/spec-kit` for `tessl:github/spec-kit`)
- `store` is the skill store to write installed skills into (dependency injection for testability)
- `InstallResult` contains `Skills []Skill` (successfully installed) and `Warnings []string`
- `Prefix()` returns the string prefix this adapter handles (e.g., `"tessl"`, `"github"`, `"file"`, `"https://"`), used for registration

The router maintains a `map[string]SourceAdapter` keyed by prefix. Registration is via `RegisterAdapter(adapter SourceAdapter)` which calls `adapter.Prefix()` to determine the key.

**Rationale**: Passing `context.Context` as the first parameter follows Go convention and enables timeout enforcement (C-004). Passing `Store` as a parameter rather than embedding it in the adapter enables unit testing with mock stores. The `Prefix()` method enables self-registration (SC-007) — new adapters declare their own prefix rather than the router hardcoding it.
