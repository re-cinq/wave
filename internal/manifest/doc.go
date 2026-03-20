// Package manifest loads, parses, and validates Wave project configuration
// files (wave.yaml). It defines typed Go structures for the full manifest
// schema including project metadata, adapter bindings, persona definitions
// with permissions and model selection, skill references, and runtime
// settings such as workspace roots, concurrency limits, and timeouts.
// Validation errors include file paths, line numbers, and actionable
// suggestions for resolution.
package manifest
