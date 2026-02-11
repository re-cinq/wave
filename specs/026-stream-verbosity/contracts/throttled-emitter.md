# Contract: ThrottledProgressEmitter

## Interface
Implements `ProgressEmitter` interface from `internal/display/types.go`.

## Behavioral Contract

### Input: event.Event
### Output: error (nil on success)

### Rules
1. Events with State != "stream_activity" MUST be passed through immediately to the inner emitter
2. Events with State == "stream_activity" MUST be coalesced: at most 1 emission per throttleInterval (1 second default)
3. When multiple stream_activity events arrive within the same throttle window, only the most recent event is emitted (most-recent-wins)
4. Thread-safe: concurrent calls to EmitProgress MUST NOT race
5. The first stream_activity event in any window MUST be emitted immediately (no initial delay)

### Invariants
- NDJSONEmitter is NOT wrapped by this — it bypasses throttling entirely
- The inner ProgressEmitter receives all non-stream-activity events unchanged
- The throttle interval is configurable but defaults to 1 second
