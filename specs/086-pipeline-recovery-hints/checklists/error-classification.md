# Error Classification & Recovery UX Checklist

**Feature**: Pipeline Recovery Hints on Failure
**Spec**: `specs/086-pipeline-recovery-hints/spec.md`
**Date**: 2026-02-13

This checklist validates the quality of requirements specific to error classification logic
and the user experience of recovery hints in terminal output.

## Error Classification Requirements

- [ ] CHK101 - Is the precedence defined when an error satisfies multiple classification criteria (e.g., a `contract.ValidationError` that is also wrapped in a `security.SecurityValidationError`)? [Completeness]
- [ ] CHK102 - Does the spec define what `ClassUnknown` means precisely — is it only errors with truly empty messages, or does it include single-word messages like "failed"? [Clarity]
- [ ] CHK103 - Is the boundary between `ClassRuntimeError` and `ClassUnknown` testable — is there a concrete rule for what constitutes an "empty or generic" message? [Clarity]
- [ ] CHK104 - Does the spec account for errors that implement custom `Is()` or `As()` methods that might affect classification? [Coverage]
- [ ] CHK105 - Is the `errors.As()` unwrapping behavior specified for deeply nested error chains (e.g., 3+ levels of wrapping)? [Completeness]
- [ ] CHK106 - Does the refactoring of `isContractValidationError()` (from string-matching to `errors.As()`) have backward-compatibility requirements or migration notes? [Completeness]

## Recovery Hint UX Requirements

- [ ] CHK107 - Is the visual separation between the error message and the recovery block defined (e.g., blank line, horizontal rule, specific character sequence)? [Clarity]
- [ ] CHK108 - Does FR-007 ("clearly labeled") specify the exact header text, or is it flexible (e.g., "Recovery options:" vs "Next steps:" vs "To recover:")? [Clarity]
- [ ] CHK109 - Are the hint labels for each hint type (resume, force, workspace, debug) specified as exact strings or as guidelines? [Clarity]
- [ ] CHK110 - Does the spec define whether the workspace `ls` command should be a plain path or a full command (e.g., `ls <path>` vs just `<path>`)? [Clarity]
- [ ] CHK111 - Is the ordering of hints within the block specified explicitly (resume first, then force, then workspace, then debug), or is it left to implementation? [Completeness]
- [ ] CHK112 - Does the spec address how the recovery block renders when the terminal width is narrow (e.g., long commands that wrap)? [Coverage]

## JSON Output Mode Requirements

- [ ] CHK113 - Is the JSON schema for `RecoveryHintJSON` (Label, Command, Type fields) fully specified with types and constraints? [Completeness]
- [ ] CHK114 - Does the spec define whether the `recovery_hints` field appears on step-level failure events, pipeline-level failure events, or both? [Clarity]
- [ ] CHK115 - Is there a requirement for consumers of JSON output to be able to distinguish between "no hints available" (empty array) and "feature not present" (field absent)? [Completeness]
- [ ] CHK116 - Does the `omitempty` behavior align with the intent — should the field be omitted when there are zero hints, or always present on failure events? [Consistency]

## Shell Escaping Requirements

- [ ] CHK117 - Does the spec enumerate the specific special characters that must be correctly escaped (beyond the examples of quotes, ampersands)? [Completeness]
- [ ] CHK118 - Is the escaping behavior defined for multi-byte UTF-8 characters in the input string? [Coverage]
- [ ] CHK119 - Does the spec address Windows compatibility of the POSIX single-quote escaping strategy, or is Windows explicitly out of scope? [Coverage]
- [ ] CHK120 - Is the empty-string handling specified (should `ShellEscape("")` produce `''` or be omitted from the command entirely)? [Clarity]
