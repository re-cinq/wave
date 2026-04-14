# Contract: Proof-of-Concept Implementation

**Deliverable**: `specs/1106-webui-framework-research/poc/<candidate>/`  
**Type**: Behavioral (working implementation)  
**Validates**: FR-003, FR-004, FR-005, SC-002, SC-007

## Acceptance Criteria

### Minimum Delivery (1 PoC required, 2 target)

1. **At least one PoC directory** exists under `poc/` named after the candidate framework
2. **Each PoC contains a README** documenting: build instructions, what is demonstrated, known limitations

### go:embed Compatibility (FR-004, SC-002)

3. **Build output is static files** (HTML, JS, CSS) — no server-side rendering runtime
4. **Files can be embedded** via `//go:embed` directive into a Go binary
5. **A Go integration test or script** demonstrates serving the PoC from an embedded binary
6. **No runtime file dependencies** — the page works from the single binary only

### SSE Integration (FR-005)

7. **PoC connects to the existing SSE endpoint** (`/api/runs/{id}/events`) — no backend changes
8. **Real-time event streaming works**: step state transitions appear live in the UI
9. **Last-Event-ID reconnection**: on disconnect/reconnect, missed events are backfilled
10. **Connection status indication**: visual indicator when SSE is connected/disconnected

### Core Features (SC-002)

11. **DAG visualization renders**: nodes represent pipeline steps with status-colored indicators
12. **Log streaming displays real-time output**: new log lines appear without page reload
13. **Step cards reflect live status**: cards update as steps transition through states
14. **Artifact viewing works**: artifacts can be viewed inline

### Constraints (SC-007)

15. **No existing Go code modified**: the PoC only adds new files, does not touch `internal/webui/`
16. **Existing test suite unaffected**: `go test ./...` still passes

## Validation Method

```bash
# 1. Build the PoC
cd specs/1106-webui-framework-research/poc/<candidate>
# Follow README build instructions

# 2. Verify go:embed compatibility
# Build output should be plain static files (ls build/ or dist/)

# 3. Verify no backend changes
git diff --name-only internal/ | wc -l  # expect: 0

# 4. Manual verification
# Start wave, navigate to run detail, confirm:
# - DAG renders with colored nodes
# - Logs stream in real time
# - Step cards update status
# - SSE connection indicator visible
```
