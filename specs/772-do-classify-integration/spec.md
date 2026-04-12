# WS1.5: Integrate task classifier into wave do command

**Issue**: [re-cinq/wave#772](https://github.com/re-cinq/wave/issues/772)
**Labels**: epic
**Author**: nextlevelshit
**Complexity**: medium

## Description

Modify `cmd/wave/commands/do.go` to call `classify.Classify()` then `classify.SelectPipeline()` before pipeline generation. Add `--dry-run` flag enhancement showing classification result and selected pipeline. Add `--no-classify` flag to bypass classification and use the existing ad-hoc pipeline behavior. Keep backward compatibility.

## Acceptance Criteria

1. `wave do "fix the login bug"` classifies the input and selects an appropriate pipeline (e.g. `impl-issue`) instead of always generating a navigate-execute ad-hoc pipeline
2. `--dry-run` shows classification output (domain, complexity, blast radius) and the selected pipeline name + reason
3. `--no-classify` bypasses classification entirely and runs the original ad-hoc pipeline behavior
4. Without `--no-classify`, if classification selects a known pipeline from the manifest, that pipeline is executed; if no matching pipeline exists, fallback to the ad-hoc pipeline
5. Backward compatibility: existing flags (`--persona`, `--mock`, `--model`, `--dry-run`) continue to work unchanged
6. All existing tests pass; new tests cover classification integration, `--no-classify`, and dry-run output changes

## Dependencies

- `internal/classify` package (WS1.1-1.4) — already implemented with `Classify()` and `SelectPipeline()` functions
- `internal/suggest` package — used by classify for input type detection

## Constraints

- Single static binary — no new runtime dependencies
- Maintain the existing two-step ad-hoc pipeline as fallback
- Classification is advisory: if the selected pipeline doesn't exist in the manifest, fall back gracefully
