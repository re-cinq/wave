# Stream Verbosity: Real-time Claude Code Activity

## Problem
Wave uses `--output-format json` which buffers entire Claude Code output. Users see 270+ seconds of silence per step. Claude Code supports `--output-format stream-json` with real-time NDJSON streaming of tool calls, messages, and results.

## Phases

### Phase 1: Stream-JSON Adapter
- `internal/adapter/claude.go`: Replace `io.Copy` buffering with `bufio.Scanner` line-by-line processing
- `internal/adapter/claude.go:buildArgs()`: Change `--output-format json` → `--output-format stream-json`
- `internal/adapter/adapter.go`: Add `OnStreamEvent func(StreamEvent)` to `AdapterRunConfig`
- New `StreamEvent` type: `{Type, ToolName, ToolInput, Content, TokensDelta}`
- Accumulate final result from stream for backward compatibility

### Phase 2: Event Bridge
- `internal/pipeline/executor.go`: Pass stream callback that emits `event.Event` for each tool call
- `internal/event/emitter.go`: Add `ToolName`, `ToolTarget` fields to `Event` struct
- Map: tool_use → "Reading file.go", Write → "Writing output.json", Bash → "Running go test"

### Phase 3: Display Rendering
- `internal/event/emitter.go`: Render tool call events in humanReadable mode
- Format: `[HH:MM:SS] running    step    Read  path/to/file.go`
- Throttle: max 1 tool event per second to avoid flooding

### Phase 4: Polish
- Step start: include model name, adapter type in event
- ETA: display already-calculated EstimatedTimeMs
- `wave init`: include speckit commands and .specify/ scripts in defaults

## Key Files
- `internal/adapter/claude.go` (core change)
- `internal/adapter/adapter.go` (types)
- `internal/pipeline/executor.go` (wiring)
- `internal/event/emitter.go` (display)
