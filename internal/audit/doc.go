// Package audit provides execution trace logging with automatic credential
// scrubbing for Wave pipeline runs. It records tool calls, file operations,
// step lifecycle events, and contract validation results to structured
// NDJSON trace files under .agents/traces/. All logged output is sanitized
// to remove API keys, tokens, secrets, and private keys before being
// written to disk.
package audit
