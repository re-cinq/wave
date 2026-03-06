# Output Stream Discipline Quality Checklist

**Feature**: #260 — CLI Compliance Polish (clig.dev)  
**Date**: 2026-03-07  
**Scope**: Quality of stdout/stderr separation and output format specifications

## Stream Assignment Rules

- [ ] CHK301 - Is there a complete taxonomy of output categories (data, progress, warnings, errors, summaries, debug) with their stream assignments (stdout vs stderr) defined in one place? [Completeness]
- [ ] CHK302 - Does the spec define stream assignment for output produced by adapter subprocesses — is adapter stdout forwarded to Wave's stdout, or captured and re-emitted? [Completeness]
- [ ] CHK303 - Is the "summary at END of stderr" requirement (FR-012) precisely defined — what constitutes "end"? After all NDJSON events? After all progress? [Clarity]
- [ ] CHK304 - Does the stream discipline apply uniformly to all output modes (auto, text, json, quiet), or only to json mode? [Completeness]

## JSON Output Guarantees

- [ ] CHK305 - Is there a guarantee that stdout in `--json` mode contains ONLY valid JSON with no interleaved whitespace, prompts, or library output (e.g., cobra usage strings)? [Coverage]
- [ ] CHK306 - Does the spec define whether cobra's default error/usage output (which goes to stderr) is also JSON-formatted in `--json` mode? [Coverage]
- [ ] CHK307 - Is the NDJSON format for streaming commands fully specified — is there a defined event schema, or just "one JSON object per line"? [Clarity]
- [ ] CHK308 - Does the spec address buffering behavior — are JSON lines flushed immediately, or could they be buffered and arrive in bursts? [Coverage]

## Quiet Mode Output

- [ ] CHK309 - Is the quiet mode output for non-streaming commands specified per-subcommand — what is the "essential" output for `wave status -q`, `wave list -q`, `wave artifacts -q`? [Completeness]
- [ ] CHK310 - Does the spec define whether quiet mode affects exit codes — should a quiet success be distinguishable from a quiet failure by exit code alone? [Completeness]
- [ ] CHK311 - Is the "final summary line" format for `wave run -q` specified — what fields does it contain (status, duration, step count)? [Clarity]
- [ ] CHK312 - Does the spec address `--quiet` behavior for commands that only produce stderr output (e.g., `wave clean`) — is there ANY output at all? [Completeness]
