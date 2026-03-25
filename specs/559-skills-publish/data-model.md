# Data Model: Publish Wave Skills

**Feature**: #559 Skills Publish
**Date**: 2026-03-24

## Entities

### SkillClassification

Represents the audit result for a single skill.

```go
// SkillClassification holds the audit result for a skill.
type SkillClassification struct {
    Name           string   // Skill name (matches directory name)
    Tag            string   // "standalone" | "wave-specific" | "both"
    WaveRefCount   int      // Number of Wave-specific references found
    Warnings       []string // Compliance warnings (e.g., missing optional fields)
    SourcePath     string   // Filesystem path where skill was discovered
}
```

**Classification rules**:
- `standalone`: WaveRefCount == 0
- `both`: 1 <= WaveRefCount <= 10
- `wave-specific`: WaveRefCount > 10

**Package location**: `internal/skill/classify.go`

---

### PublishRecord

Represents a single published skill entry in the lockfile.

```go
// PublishRecord represents a published skill entry in the lockfile.
type PublishRecord struct {
    Name        string    `json:"name"`
    Digest      string    `json:"digest"`       // "sha256:<hex>"
    Registry    string    `json:"registry"`      // Registry name (e.g., "tessl")
    URL         string    `json:"url"`           // Published URL on registry
    PublishedAt time.Time `json:"published_at"`
}
```

**Package location**: `internal/skill/lockfile.go`

---

### Lockfile

JSON file at `.wave/skills.lock` containing all publish records.

```go
// Lockfile represents the skill publish lockfile.
type Lockfile struct {
    Version   int             `json:"version"`
    Published []PublishRecord `json:"published"`
}
```

**Operations**:
- `LoadLockfile(path string) (*Lockfile, error)` — read and parse lockfile
- `(*Lockfile).Save(path string) error` — atomic write-to-temp-then-rename
- `(*Lockfile).FindByName(name string) *PublishRecord` — lookup by skill name
- `(*Lockfile).Upsert(record PublishRecord)` — insert or update by name

**Package location**: `internal/skill/lockfile.go`

---

### PublishResult

Outcome of a single publish operation.

```go
// PublishResult represents the outcome of publishing one skill.
type PublishResult struct {
    Name     string   // Skill name
    Success  bool     // Whether publish succeeded
    URL      string   // Published URL (empty on failure)
    Digest   string   // Content digest (always computed)
    Warnings []string // Non-blocking warnings
    Error    string   // Error message if Success == false
}
```

**Package location**: `internal/skill/publish.go`

---

### ValidationReport

Result of agentskills.io spec validation.

```go
// ValidationReport contains validation results for a SKILL.md file.
type ValidationReport struct {
    Errors   []ValidationIssue // Blocking issues
    Warnings []ValidationIssue // Non-blocking issues
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
    Field   string // Frontmatter field name
    Message string // Human-readable description
}

// Valid returns true if there are no blocking errors.
func (r *ValidationReport) Valid() bool {
    return len(r.Errors) == 0
}
```

**Package location**: `internal/skill/validate.go` (extend existing file)

---

### AuditOutput (CLI)

CLI output struct for `wave skills audit`.

```go
// SkillAuditItem represents one skill in audit output.
type SkillAuditItem struct {
    Name           string   `json:"name"`
    Classification string   `json:"classification"`
    WaveRefCount   int      `json:"wave_ref_count"`
    Warnings       []string `json:"warnings,omitempty"`
    Source         string   `json:"source"`
}

// SkillAuditOutput is the top-level output for wave skills audit.
type SkillAuditOutput struct {
    Skills   []SkillAuditItem `json:"skills"`
    Summary  AuditSummary     `json:"summary"`
}

// AuditSummary provides aggregate counts.
type AuditSummary struct {
    Total        int `json:"total"`
    Standalone   int `json:"standalone"`
    WaveSpecific int `json:"wave_specific"`
    Both         int `json:"both"`
}
```

**Package location**: `cmd/wave/commands/skills.go`

---

### PublishOutput (CLI)

CLI output struct for `wave skills publish`.

```go
// SkillPublishOutput is the top-level output for wave skills publish.
type SkillPublishOutput struct {
    Results  []PublishResultItem `json:"results"`
    Lockfile string              `json:"lockfile"`
}

// PublishResultItem represents one publish result in CLI output.
type PublishResultItem struct {
    Name     string   `json:"name"`
    Status   string   `json:"status"` // "published" | "skipped" | "failed"
    URL      string   `json:"url,omitempty"`
    Digest   string   `json:"digest,omitempty"`
    Reason   string   `json:"reason,omitempty"` // Skip/failure reason
    Warnings []string `json:"warnings,omitempty"`
}
```

**Package location**: `cmd/wave/commands/skills.go`

---

### VerifyOutput (CLI)

CLI output struct for `wave skills verify`.

```go
// SkillVerifyItem represents one verify result.
type SkillVerifyItem struct {
    Name           string `json:"name"`
    Status         string `json:"status"` // "ok" | "modified" | "missing"
    ExpectedDigest string `json:"expected_digest,omitempty"`
    ActualDigest   string `json:"actual_digest,omitempty"`
}

// SkillVerifyOutput is the top-level output for wave skills verify.
type SkillVerifyOutput struct {
    Results []SkillVerifyItem `json:"results"`
    Summary VerifySummary     `json:"summary"`
}

// VerifySummary provides aggregate verify counts.
type VerifySummary struct {
    Total    int `json:"total"`
    OK       int `json:"ok"`
    Modified int `json:"modified"`
    Missing  int `json:"missing"`
}
```

**Package location**: `cmd/wave/commands/skills.go`

---

## Relationships

```
DirectoryStore ──List()──> []Skill ──classify──> []SkillClassification
                                                        │
                                                        ▼
                                              PublishResult ←── tessl publish
                                                        │
                                                        ▼
                                              Lockfile.Upsert(PublishRecord)
                                                        │
                                                        ▼
                                              .wave/skills.lock (atomic write)
```

## File Layout

```
internal/skill/
├── classify.go          # NEW: SkillClassification, ClassifySkill(), ClassifyAll()
├── classify_test.go     # NEW: classification tests
├── digest.go            # NEW: ComputeDigest() SHA-256 content addressing
├── digest_test.go       # NEW: digest computation tests
├── lockfile.go          # NEW: Lockfile, PublishRecord, Load/Save/Upsert
├── lockfile_test.go     # NEW: lockfile CRUD tests
├── publish.go           # NEW: Publisher, PublishResult, PublishOne(), PublishAll()
├── publish_test.go      # NEW: publish workflow tests
├── parse.go             # EXISTING: add ValidationReport, ValidateForPublish()
├── validate.go          # EXISTING: extend with publish-specific validation
├── store.go             # EXISTING: unchanged
├── source_cli.go        # EXISTING: TesslAdapter (used for tessl publish)
└── ...

cmd/wave/commands/
├── skills.go            # EXISTING: add audit, publish, verify subcommands
└── skills_test.go       # EXISTING: extend with new subcommand tests
```
