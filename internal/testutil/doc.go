// Package testutil provides shared test utilities for Wave's test suite.
//
// It extracts commonly duplicated test infrastructure — event collectors,
// state store mocks, and manifest helpers — into a single reusable package.
//
// # EventCollector
//
// Thread-safe event.EventEmitter implementation that collects events for assertions:
//
//	collector := testutil.NewEventCollector()
//	executor := pipeline.NewDefaultPipelineExecutor(mockAdapter, pipeline.WithEmitter(collector))
//	// ... run pipeline ...
//	assert.True(t, collector.HasEventWithState("completed"))
//
// # MockStateStore
//
// Configurable state.StateStore mock using functional options. Default methods
// return zero values. Override specific methods as needed:
//
//	store := testutil.NewMockStateStore(
//	    testutil.WithSavePipelineState(func(id, status, input string) error {
//	        // custom behavior
//	        return nil
//	    }),
//	)
//
// # CreateTestManifest
//
// Creates a standard test manifest with navigator and craftsman personas:
//
//	m := testutil.CreateTestManifest(t.TempDir())
package testutil
