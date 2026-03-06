# Event Streaming Quality Checklist: TUI Live Output Streaming

**Feature**: #257 | **Date**: 2026-03-06
**Focus**: Event delivery, buffer management, and display flag filtering requirements quality.

## Event Delivery Requirements

- [ ] CHK101 - Is the event delivery latency requirement specified — FR-002 says "in real time" but does it define an acceptable upper bound for delivery delay? [Clarity]
- [ ] CHK102 - Are ordering guarantees specified for events delivered via `program.Send()` — is it documented that events from a single pipeline arrive in emission order? [Completeness]
- [ ] CHK103 - Is the behavior defined for when `program.Send()` is called from the executor goroutine while the TUI is in the middle of a render cycle — is backpressure or queuing behavior specified? [Completeness]
- [ ] CHK104 - Does FR-018 (run ID on every event message) specify how run IDs are generated and whether they're guaranteed unique across TUI sessions? [Clarity]
- [ ] CHK105 - Is there a requirement for event batching during high-frequency emission periods (edge case 2 mentions batching but no FR covers it)? [Completeness]

## Buffer Management Requirements

- [ ] CHK106 - Is the 1000-line buffer capacity justified with data — the spec says "200-500 visible events typical" but are there profiled numbers for verbose+debug mode? [Completeness]
- [ ] CHK107 - Does the ring buffer specification address multi-line events (e.g., error blocks span 6+ lines) — are they stored as multiple buffer entries or a single entry? [Clarity]
- [ ] CHK108 - Is the buffer cleanup trigger precisely defined — C6 says "cleaned up when pipeline transitions to Finished" but does this mean on the TransitionTimerMsg or on the state store update? [Clarity]
- [ ] CHK109 - Are thread-safety requirements for the EventBuffer stated — the executor goroutine appends while the UI goroutine reads for rendering? [Completeness]
- [ ] CHK110 - Does the spec define behavior when `Lines()` is called on an empty buffer — is an empty slice returned, or a "Waiting for events..." placeholder? [Consistency]

## Display Flag Requirements

- [ ] CHK111 - Is the interaction between all flag combinations specified — what does verbose+debug+output-only show? Does output-only truly override both? [Clarity]
- [ ] CHK112 - Are the exact event state strings used in `shouldFormat()` defined as constants or enums, or could they drift from the executor's event states? [Consistency]
- [ ] CHK113 - Is the initial flag state (all off) explicitly stated as a requirement, or only implied by the "default mode" description? [Completeness]
- [ ] CHK114 - Does the spec define whether flag state persists when navigating away from a pipeline and back, or is it reset to defaults? [Completeness]
- [ ] CHK115 - Are the flag toggle keys (`v`, `d`, `o`) specified as case-sensitive? What happens with `V`, `D`, `O`? [Clarity]
