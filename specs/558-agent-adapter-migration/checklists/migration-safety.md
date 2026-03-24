# Migration Safety Checklist

**Feature**: #558 — Migrate Adapter to Agent-Based Execution
**Date**: 2026-03-23

This checklist validates that migration-specific requirements are adequately specified
to prevent regressions, data loss, and runtime failures during the transition.

---

## Deletion Safety

- [ ] CHK101 - Are all references to `ClaudeSettings` enumerated across the codebase, not just `internal/adapter/`? [Deletion Safety]
- [ ] CHK102 - Are all references to `ClaudePermissions` enumerated across the codebase, not just `internal/adapter/`? [Deletion Safety]
- [ ] CHK103 - Is there a requirement to verify `normalizeAllowedTools` is not called indirectly through any wrapper or alias? [Deletion Safety]
- [ ] CHK104 - Does the spec require verifying that no other package imports or references `UseAgentFlag` beyond `internal/adapter/` and `internal/pipeline/`? [Deletion Safety]

## Behavioral Equivalence

- [ ] CHK105 - Is the set of tools available to Claude Code identical before and after migration for every persona configuration? [Behavioral Equivalence]
- [ ] CHK106 - Is the deny/disallow behavior identical between `settings.json` deny rules and agent frontmatter `disallowedTools`? [Behavioral Equivalence]
- [ ] CHK107 - Does `permissionMode: dontAsk` produce the same runtime behavior as `--dangerously-skip-permissions`? [Behavioral Equivalence]
- [ ] CHK108 - Is there a requirement to verify that the agent file body content (base protocol + persona + contract + restrictions) produces identical Claude Code behavior as the old CLAUDE.md? [Behavioral Equivalence]
- [ ] CHK109 - Are there requirements confirming that sandbox behavior (bubblewrap, network isolation) is unaffected by the settings.json format change? [Behavioral Equivalence]

## Version Compatibility

- [ ] CHK110 - Is the minimum Claude Code version that supports `--agent` specified or referenced from research spec #557? [Version Compatibility]
- [ ] CHK111 - Is there a preflight check requirement to detect Claude Code versions that don't support `--agent`? [Version Compatibility]
- [ ] CHK112 - Are the agent frontmatter field names (`tools`, `disallowedTools`, `permissionMode`) validated against Claude Code's documented schema? [Version Compatibility]

## Atomicity

- [ ] CHK113 - Is the order of file writes specified (agent file before or after sandbox settings.json)? [Atomicity]
- [ ] CHK114 - Is behavior defined for partial write failure (agent file written but settings.json write fails, or vice versa)? [Atomicity]
- [ ] CHK115 - Are there requirements for the agent file to be written atomically (write-then-rename) to prevent Claude Code from reading a partial file? [Atomicity]
