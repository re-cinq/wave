# CLI UX Quality Checklist: Wave Skills CLI

**Feature**: `wave skills` CLI — list/search/install/remove/sync
**Focus**: CLI ergonomics, output formatting, and user experience requirements
**Generated**: 2026-03-14

---

## Output Formatting

- [ ] CHK101 - Are column widths, alignment, and truncation rules specified for table output across all subcommands? [Completeness]
- [ ] CHK102 - Is the JSON output structure for error responses specified (matching CLIError format vs subcommand-specific error JSON)? [Completeness]
- [ ] CHK103 - Are color/styling requirements specified for table output (e.g., warning indicators, success/failure highlighting)? [Clarity]
- [ ] CHK104 - Is the "no results" message for search differentiated from "no skills installed" in list? [Clarity]
- [ ] CHK105 - Are the table headers (Name, Description, Source, Used By, Rating) consistently capitalized and ordered across all subcommands? [Consistency]

## Error UX

- [ ] CHK106 - Does each error scenario specify both the error message and actionable suggestion text? [Completeness]
- [ ] CHK107 - Is the recognized prefix list in the "unrecognized prefix" error dynamically generated from the router, or hardcoded? [Clarity]
- [ ] CHK108 - Are install instructions for missing dependencies (tessl, git, npx) specified per-dependency with concrete install commands? [Completeness]
- [ ] CHK109 - Does the "skill not found" error on remove include a list of installed skills as suggestions? [Completeness]
- [ ] CHK110 - Is the exit code behavior specified for each error scenario (non-zero exit on error)? [Completeness]

## Interactive Patterns

- [ ] CHK111 - Is the confirmation prompt behavior specified for non-TTY environments (e.g., piped stdin with no --yes flag)? [Coverage]
- [ ] CHK112 - Does the spec define what input values are accepted for the confirmation prompt (Y/y/yes/n/N/no/empty)? [Clarity]
- [ ] CHK113 - Is the default behavior on empty confirmation input (pressing Enter) explicitly defined? [Clarity]

## Command Discovery

- [ ] CHK114 - Does the parent `wave skills` help text include usage examples demonstrating common workflows? [Completeness]
- [ ] CHK115 - Are subcommand aliases defined (e.g., `rm` for `remove`, `ls` for `list`)? [Coverage]
- [ ] CHK116 - Is shell completion support considered for subcommand names and flag values? [Coverage]
