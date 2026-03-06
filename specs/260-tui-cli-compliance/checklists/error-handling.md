# Error Handling Quality Checklist

**Feature**: #260 — CLI Compliance Polish (clig.dev)  
**Date**: 2026-03-07  
**Scope**: Quality of structured error output and actionable error message specifications

## Error Code Taxonomy

- [ ] CHK201 - Is every error code in the data model reachable from at least one concrete code path identified in the plan or tasks? [Coverage]
- [ ] CHK202 - Are the error codes exhaustive — does the taxonomy cover ALL possible failure modes in the current codebase, or just the ones listed? [Completeness]
- [ ] CHK203 - Is the `internal_error` catch-all code documented as a fallback — is there guidance on when a specific code should be added vs. when `internal_error` is acceptable? [Clarity]
- [ ] CHK204 - Are error codes stable identifiers that can be relied upon by external tooling, or are they subject to change? Is this contract documented? [Completeness]

## Actionable Suggestions

- [ ] CHK205 - Does every error code have a corresponding suggestion string defined in the spec or plan? [Completeness]
- [ ] CHK206 - Are suggestions context-aware — does `pipeline_not_found` list available pipelines dynamically, or just suggest a static `wave list pipelines` command? [Clarity]
- [ ] CHK207 - Is the suggestion format specified — plain text sentence, shell command with backticks, or mixed? Is it consistent across all error codes? [Consistency]
- [ ] CHK208 - Does the spec define whether suggestions are shown in BOTH JSON and text error modes, or only in text mode? [Completeness]

## JSON Error Schema

- [ ] CHK209 - Is the JSON error output schema versioned or otherwise forward-compatible — can new fields be added without breaking existing consumers? [Coverage]
- [ ] CHK210 - Does the spec define whether the JSON error object is emitted as a single line (for NDJSON compatibility) or pretty-printed? [Clarity]
- [ ] CHK211 - Is the JSON error output on stderr validated by the same contract mechanism as stdout JSON output, or is it informal? [Consistency]
- [ ] CHK212 - Does the spec define the behavior when JSON marshaling of the error itself fails — is there a fallback plain-text error? [Coverage]

## Debug Context

- [ ] CHK213 - Is the `debug` field content specified — does it contain the Go error chain (`%+v`), stack trace, or structured context? [Clarity]
- [ ] CHK214 - Does the spec define whether debug information appears in BOTH JSON and text error rendering, or only in one mode? [Completeness]
- [ ] CHK215 - Is there a security review requirement for the debug field — could it leak sensitive information (file paths, credentials, env vars)? [Coverage]
