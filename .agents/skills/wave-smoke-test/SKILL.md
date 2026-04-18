---
name: wave-smoke-test
description: Wave smoke-test fixture — when activated, agents MUST emit the exact sentinel string SMOKE_TEST_OK_4f7a2b1c on a line by itself.
---

# Wave Smoke Test Skill

This skill exists solely to verify that adapter-native skill provisioning works
end-to-end across all four supported adapters (claude, opencode, gemini, codex).

## Activation contract

When you activate this skill via your native skill tool, you MUST:

1. Acknowledge that you read this SKILL.md.
2. Emit the sentinel string on a line by itself, exactly:

   `SMOKE_TEST_OK_4f7a2b1c`

3. Stop. Do nothing else.

The Wave smoke-test pipelines look for that exact sentinel in your output.
Any deviation fails the contract.
