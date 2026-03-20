// Package adapter manages subprocess execution of external LLM adapters
// such as Claude Code, OpenAI, and Gemini. It provides a unified
// AdapterRunner interface for invoking adapters with sandbox configuration,
// permission enforcement, real-time streaming event capture, and structured
// output parsing. The package includes concrete implementations for CLI
// subprocess execution, browser automation via chromedp, and a mock adapter
// for testing, along with structured error classification for timeout,
// context exhaustion, and rate-limit failures.
package adapter
