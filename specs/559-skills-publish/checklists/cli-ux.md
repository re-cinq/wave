# CLI User Experience Checklist

**Feature**: #559 Skills Publish
**Generated**: 2026-03-24
**Focus**: CLI design consistency, error messaging, and user workflow requirements

---

## Command Design

- [ ] CHK201 - Are all new subcommands (audit, publish, verify) consistent with existing `wave skills` subcommand patterns (search, sync, install) in flag naming, output structure, and help text format? [Consistency]
- [ ] CHK202 - Is the command hierarchy specified: `wave skills audit`, `wave skills publish`, `wave skills verify` — are there any ambiguities with existing `wave skills` subcommands? [Clarity]
- [ ] CHK203 - Are short flag aliases defined (e.g., `-f` for `--format`, `-n` for `--dry-run`) or explicitly excluded for consistency? [Completeness]
- [ ] CHK204 - Is the default behavior specified for each subcommand when run with no flags (e.g., default format is table, default registry is tessl)? [Completeness]
- [ ] CHK205 - Are exit code requirements defined: 0 for success, 1 for failure, any special codes for partial success in batch mode? [Completeness]

## Error Messaging

- [ ] CHK206 - Are the three error codes (skill_publish_failed, skill_validation_failed, skill_already_exists) sufficient to distinguish all failure modes, or are additional codes needed (e.g., skill_not_found, registry_unreachable, lockfile_corrupt)? [Coverage]
- [ ] CHK207 - Are error messages required to include actionable remediation guidance (e.g., "Run `wave skills audit` to check skill eligibility" or "Use --force to override")? [Completeness]
- [ ] CHK208 - Is it specified how validation errors are displayed — as a list, as a table, or inline per field? [Clarity]
- [ ] CHK209 - Are requirements defined for error output destination: stderr for errors, stdout for normal output, or mixed? [Completeness]

## Progress & Feedback

- [ ] CHK210 - Are progress indicator requirements specified for long-running operations (batch publish of 12+ skills with network calls)? [Completeness]
- [ ] CHK211 - Is it specified whether the publish command provides real-time per-skill status updates during batch mode, or only a final summary? [Clarity]
- [ ] CHK212 - Are requirements defined for verbose/debug output mode for the new subcommands (consistent with Wave's `--debug` flag)? [Consistency]
