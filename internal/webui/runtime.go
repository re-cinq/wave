package webui

// controlGateRegistry is the narrow gate-registry contract that the HTTP
// handlers in handlers_control.go depend on. Defining the interface here
// (rather than reaching for *GateRegistry directly) keeps the handler file
// free of any internal/pipeline imports: every method on this interface
// trades only in primitive types or webui-local error sentinels.
//
// The concrete *GateRegistry (gate_handler.go) satisfies this interface;
// see the compile-time guard below. Other webui files still hold a
// *GateRegistry via serverRealtime — the interface is intentionally
// scoped to handlers_control.go's surface and not threaded through the
// Server struct. Broader cleanup of the webui→pipeline import seam will
// land in follow-up PRs (issue #1498 sub-tasks).
type controlGateRegistry interface {
	// ResolveChoice validates a human's gate decision and resolves the
	// pending gate for the run. It returns the resolved choice key and
	// label on success, or one of the sentinel errors (ErrGateNotPending,
	// ErrGateStepMismatch, ErrGateInvalidChoice) on failure.
	ResolveChoice(runID, stepID, choiceKey, text string) (key, label string, err error)
}

// Compile-time guard: *GateRegistry must satisfy controlGateRegistry. If a
// future change to GateRegistry breaks this contract, the package will fail
// to build and the handler can be updated alongside.
var _ controlGateRegistry = (*GateRegistry)(nil)
