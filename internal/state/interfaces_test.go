package state

// Compile-time assertions that the concrete *stateStore satisfies every
// domain-scoped interface as well as the aggregate StateStore. Adding or
// removing a method on any interface without updating the concrete type will
// fail the build at this site.
var (
	_ RunStore     = (*stateStore)(nil)
	_ EventStore   = (*stateStore)(nil)
	_ WebhookStore = (*stateStore)(nil)
	_ ChatStore    = (*stateStore)(nil)
	_ StateStore   = (*stateStore)(nil)
)
