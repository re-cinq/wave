// Package state provides SQLite-backed persistence for Wave pipeline
// execution. It manages pipeline run records, step execution states, event
// logs, artifacts, and performance metrics.
//
// The persistence surface is exposed as five domain-scoped interfaces —
// RunStore (run + step lifecycle), EventStore (events + audit + artifacts),
// OntologyStore (decision lineage), WebhookStore (webhook CRUD + delivery),
// and ChatStore (bidirectional chat sessions) — plus an aggregate StateStore
// that embeds all five and adds Close. Consumers should depend on the
// smallest interface that satisfies their call sites; the aggregate is
// retained for constructors and root-level orchestrators that span domains.
//
// The package supports pipeline resumption, cancellation, retry tracking,
// concurrent dashboard queries via read-only connections, and schema
// versioning through a migration system.
package state
