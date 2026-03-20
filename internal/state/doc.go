// Package state provides SQLite-backed persistence for Wave pipeline
// execution. It manages pipeline run records, step execution states, event
// logs, artifacts, and performance metrics through the StateStore interface.
// The package supports pipeline resumption, cancellation, retry tracking,
// concurrent dashboard queries via read-only connections, and schema
// versioning through a migration system.
package state
