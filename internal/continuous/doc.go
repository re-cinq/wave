// Package continuous implements continuous pipeline execution mode.
// It provides a loop controller that iterates over work items from
// configurable sources, executing a pipeline for each item with
// deduplication, failure policies, and graceful shutdown support.
package continuous
