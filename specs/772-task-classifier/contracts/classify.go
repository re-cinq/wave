// Package contracts defines the API surface for internal/classify.
// This file documents the exported functions and types — not compilable code.
// See data-model.md for type definitions.

package contracts

// --- profile.go ---

// TaskProfile: exported struct (see data-model.md for fields)
// Complexity: exported string type + 4 constants (Simple, Medium, Complex, Architectural)
// Domain: exported string type + 7 constants (Security, Performance, Bug, Refactor, Feature, Docs, Research)
// VerificationDepth: exported string type + 3 constants (StructuralOnly, Behavioral, FullSemantic)
// PipelineConfig: exported struct with Name (string) and Reason (string)

// --- analyzer.go ---

// Classify analyzes input text and optional issue body to produce a TaskProfile.
//
// Signature:
//   func Classify(input string, issueBody string) TaskProfile
//
// Behavior:
//   1. Calls suggest.ClassifyInput(input) to determine InputType
//   2. Combines input + issueBody for keyword analysis
//   3. Determines domain via keyword matching with priority ordering (FR-011)
//   4. Determines complexity via keyword matching
//   5. Derives blast_radius from complexity + domain (FR-009)
//   6. Derives verification_depth from complexity (FR-010)
//   7. Returns populated TaskProfile
//
// Edge cases:
//   - Empty/whitespace input: returns default profile (simple/feature/0.1/structural_only)
//   - No keywords matched: returns medium/feature/0.3/behavioral
//   - Mixed domain signals: highest-priority domain wins

// --- selector.go ---

// SelectPipeline maps a TaskProfile to a PipelineConfig.
//
// Signature:
//   func SelectPipeline(profile TaskProfile) PipelineConfig
//
// Routing priority (FR-006):
//   1. profile.InputType == pr_url → PipelineConfig{Name: "ops-pr-review", Reason: "..."}
//   2. profile.Domain == security → PipelineConfig{Name: "audit-security", Reason: "..."}
//   3. profile.Domain == research → PipelineConfig{Name: "impl-research", Reason: "..."}
//   4. profile.Domain == docs → PipelineConfig{Name: "doc-fix", Reason: "..."}
//   5. profile.Domain == refactor && complexity ∈ {complex, architectural} → "impl-speckit"
//   6. profile.Complexity ∈ {simple, medium} → PipelineConfig{Name: "impl-issue", Reason: "..."}
//   7. profile.Complexity ∈ {complex, architectural} → PipelineConfig{Name: "impl-speckit", Reason: "..."}
