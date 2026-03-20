// Package relay implements token usage monitoring and context compaction
// for Wave pipelines. It tracks token consumption across pipeline steps
// and triggers LLM-based summarization when approaching context window
// limits. Compacted checkpoints capture key decisions and summaries,
// enabling downstream steps to operate with fresh context while retaining
// essential information from prior work.
package relay
