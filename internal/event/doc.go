// Package event provides structured progress event emission for real-time
// monitoring of Wave pipeline execution. Events are emitted as
// newline-delimited JSON (NDJSON) to stdout, capturing step state
// transitions, token usage, artifact production, errors, and recovery
// hints. The package supports dual-stream output for simultaneous
// machine-readable logging and enhanced terminal progress visualization.
package event
