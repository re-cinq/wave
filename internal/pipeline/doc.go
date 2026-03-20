// Package pipeline implements DAG-based pipeline orchestration for Wave.
// It resolves step dependencies via topological sorting, executes steps
// in isolated workspaces with artifact injection and adapter invocation,
// validates outputs against contracts, and persists state to SQLite for
// resumption from checkpoints. The package supports advanced composition
// primitives (iterate, branch, gate, loop, aggregate, sub-pipeline),
// concurrent step execution, and real-time progress tracking with ETA
// calculation.
package pipeline
