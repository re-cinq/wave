# Contract: Compose Sequence Validation

## Input Contract

`ValidateSequence(seq Sequence) CompatibilityResult`

### Sequence
- `Entries` — ordered list of `SequenceEntry` (minimum 0, typically 2+)
- Each `SequenceEntry` has `PipelineName string` and `Pipeline *pipeline.Pipeline`
- Pipeline must be fully loaded (all steps with artifacts resolved)

### Pre-conditions
- Each `Pipeline` in the sequence has at least one step
- Pipeline names are non-empty
- Pipeline definitions are structurally valid (passed `wave validate`)

## Output Contract

### CompatibilityResult
- `Flows []ArtifactFlow` — one per adjacent pair (len = max(0, len(entries) - 1))
- `Status CompatibilityStatus` — overall: Valid, Warning, or Error
- `Diagnostics []string` — human-readable issue descriptions

### Status Rules
| Condition | Status |
|-----------|--------|
| All required inputs matched | `CompatibilityValid` |
| Only optional inputs unmatched | `CompatibilityWarning` |
| Any required input unmatched | `CompatibilityError` |
| Empty or single-entry sequence | `CompatibilityValid` (no boundaries to validate) |

### FlowMatch Rules
| Condition | MatchStatus |
|-----------|-------------|
| `output.Name == input.Artifact` | `MatchCompatible` |
| Input exists but no matching output and `input.Optional == false` | `MatchMissing` |
| Input exists but no matching output and `input.Optional == true` | `MatchMissing` (with Optional flag) |
| Output exists but no input consumes it | `MatchUnmatched` |

## CLI Contract

### `wave compose p1 p2 [p3...]`
- Exit code 0: sequence is valid, execution started (or `--validate-only` passed)
- Exit code 1: sequence has incompatible artifacts (printed to stderr)
- Exit code 1: pipeline not found (printed to stderr)
- Exit code 0: `--validate-only` with valid sequence (compatibility report to stdout)
- Exit code 1: `--validate-only` with invalid sequence (errors to stderr)

### Output Format
Validation report follows the same format as `wave run --dry-run`:
```
Sequence validation: speckit-flow → wave-evolve → wave-review

Boundary 1: speckit-flow → wave-evolve
  ✓ spec-status → spec_info (compatible)
  
Boundary 2: wave-evolve → wave-review  
  ✗ review_input (missing — no matching output from wave-evolve)

Result: 1 error, 0 warnings
```
