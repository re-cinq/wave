# Research: Enhanced Pipeline Progress Visualization

**Created**: 2026-02-03
**Feature**: 018-enhanced-progress
**Purpose**: Research technical approaches for implementing enhanced pipeline progress visualization in Wave

## Executive Summary

Based on comprehensive analysis of Wave's current architecture and modern CLI progress patterns, this research recommends implementing enhanced progress visualization using:

1. **ANSI escape sequences with Zero Dependencies**: Extend existing ANSI color system rather than adding TUI libraries
2. **Dual-Stream Architecture**: Progress on stderr, NDJSON on stdout for compatibility
3. **Dashboard Layout**: ASCII logo panel with live metrics and project information
4. **Animation Techniques**: Spinners, progress bars, and incrementing counters for continuous feedback

## Current State Analysis

### Wave's Existing Progress System

Wave currently implements a robust event-driven progress system with:

**Event Structure** (`internal/event/emitter.go`):
```go
type Event struct {
    Timestamp  time.Time `json:"timestamp"`
    PipelineID string    `json:"pipeline_id"`
    StepID     string    `json:"step_id,omitempty"`
    State      string    `json:"state"`
    DurationMs int64     `json:"duration_ms"`
    Message    string    `json:"message,omitempty"`
    Persona    string    `json:"persona,omitempty"`
    Artifacts  []string  `json:"artifacts,omitempty"`
    TokensUsed int       `json:"tokens_used,omitempty"`
}
```

**Current Output Modes**:
1. **NDJSON**: Machine-readable events to stdout
2. **Human-readable**: Colored text with timestamps

**Existing ANSI Colors**:
- `started`: cyan (`\033[36m`)
- `running`: yellow (`\033[33m`)
- `completed`: green (`\033[32m`)
- `failed`: red (`\033[31m`)
- `retrying`: magenta (`\033[35m`)

**Wave ASCII Logo** (from `cmd/wave/main.go`):
```
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator
```

### Architecture Constraints

**Constitutional Requirements**:
- Single static binary (no runtime dependencies)
- Backward compatible NDJSON output
- Observer-only progress (no execution modification)
- <5% performance overhead target

**Available Infrastructure**:
- `golang.org/x/term`: TTY detection and terminal utilities
- SQLite state store: Historical data for performance metrics
- Event emission points: Pipeline start/end, step transitions, contract validation

## Research Findings

### 1. Modern CLI Progress Patterns

**Best Practices from Industry Tools**:

**Docker CLI Pattern**:
- Multi-line progress for parallel operations
- Real-time byte/percentage updates
- Clear status indicators (✓, ✗, ⏳)

**kubectl Pattern**:
- Resource status tables with consistent formatting
- Progress percentages for deployments
- Color-coded status indicators

**Claude Code Pattern** (referenced by user):
- Smooth animations and loading indicators
- Professional aesthetic with consistent branding
- "Numbers going up" for continuous engagement

### 2. ANSI Escape Sequence Techniques

**Terminal Capabilities**:

**Cursor Control**:
```
\033[H        # Move cursor to home position
\033[2J       # Clear entire screen
\033[K        # Clear line from cursor to end
\033[s        # Save cursor position
\033[u        # Restore cursor position
\033[nA       # Move cursor up n lines
```

**Progress Bar Implementation**:
```go
func drawProgressBar(current, total int, width int) string {
    filled := int(float64(current) / float64(total) * float64(width))
    bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
    percent := int(float64(current) / float64(total) * 100)
    return fmt.Sprintf("[%s] %d%% (%d/%d)", bar, percent, current, total)
}
```

**Animation Sequences**:
```go
var spinners = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func getSpinnerFrame(step int) string {
    return spinners[step%len(spinners)]
}
```

### 3. Dashboard Layout Design

**Panel-Based Layout**:
```
┌─ Wave Pipeline Status ─────────────────┬─ Project Info ──────────┐
│  ╦ ╦╔═╗╦  ╦╔═╗                         │ Manifest: wave.yaml     │
│  ║║║╠═╣╚╗╔╝║╣    Step 3 of 7          │ Pipeline: test-suite    │
│  ╚╩╝╩ ╩ ╚╝ ╚═╝   [████████░░░] 75%    │ Workspace: .wave/work   │
├────────────────────────────────────────┼─────────────────────────┤
│ ⏳ step-3 (craftsman) Running...       │ Tokens: 15.2k ⟲        │
│    Elapsed: 00:02:34 | ETA: 00:01:12   │ Files: 23 modified      │
│    Contract validation in progress...   │ Artifacts: 8 generated │
└────────────────────────────────────────┴─────────────────────────┘

[14:30:45] running   step-3 (craftsman) Contract validation started...
```

**Key Design Decisions**:

**Decision**: Use stderr for progress display, stdout for NDJSON
**Rationale**: Maintains backward compatibility with existing NDJSON consumers
**Alternatives considered**: Separate progress mode flag (adds complexity)

**Decision**: Box-drawing characters for panel borders
**Rationale**: Professional appearance, widely supported
**Alternatives considered**: ASCII-only borders (less visually appealing)

**Decision**: Logo integration in top-left panel
**Rationale**: Consistent branding without overwhelming output
**Alternatives considered**: Header banner (takes too much vertical space)

**Decision**: Real-time metrics with animation
**Rationale**: Addresses user requirement for "numbers going up"
**Alternatives considered**: Static metrics (less engaging)

### 4. Animation and Engagement Techniques

**Continuous Feedback Strategies**:

1. **Incrementing Counters**: Token usage, files processed, time elapsed
2. **Progress Indicators**: Step completion bars, overall pipeline progress
3. **Activity Spinners**: For operations without measurable progress
4. **Status Transitions**: Smooth color changes for state updates

**Performance Considerations**:
- 60ms refresh rate (16 FPS) for smooth animation
- Buffered output to prevent screen flicker
- Graceful degradation for non-TTY environments

### 5. Terminal Compatibility

**Detection Strategy**:
```go
import "golang.org/x/term"

func isTerminal() bool {
    return term.IsTerminal(int(os.Stderr.Fd()))
}

func getTerminalSize() (width, height int) {
    if !isTerminal() {
        return 80, 24 // fallback defaults
    }
    width, height, _ := term.GetSize(int(os.Stderr.Fd()))
    return width, height
}
```

**Graceful Degradation**:
- Non-TTY: Fall back to simple text progress
- Small terminals: Compress layout, remove decorative elements
- No color support: Use ASCII symbols instead of colors

### 6. Implementation Architecture

**Recommended Package Structure**:
```
internal/display/
├── progress.go       # Core progress bar and status display
├── dashboard.go      # Panel layout and ASCII logo rendering
├── animation.go      # Spinners, counters, and real-time updates
├── formatter.go      # ANSI escape sequence management
└── terminal.go       # TTY detection and capability queries
```

**Integration Points**:
1. **Event Emitter Enhancement**: Add progress events without breaking NDJSON
2. **Command Integration**: Inject progress display into `run`, `logs`, `status`
3. **State Store Queries**: Historical performance data for ETA calculations

## Technical Recommendations

### Phase 1: Core Infrastructure
1. Create `internal/display` package with terminal detection
2. Extend Event struct with optional progress fields
3. Implement dual-stream output (stderr progress, stdout NDJSON)

### Phase 2: Basic Progress Display
1. Progress bars for step execution
2. Status indicators with color coding
3. Elapsed time and ETA calculations

### Phase 3: Dashboard Enhancement
1. Panel-based layout with Wave logo
2. Live metrics display (tokens, files, artifacts)
3. Project information sidebar

### Phase 4: Advanced Animation
1. Smooth progress transitions
2. Incrementing counter animations
3. Loading spinners for active operations

### Immediate Next Steps
1. Implement terminal detection utilities
2. Create basic progress bar rendering
3. Integrate with existing event system
4. Add progress display to `wave run` command

## Validation Criteria

**Success Metrics**:
- Users can identify current step within 2 seconds
- Progress updates occur within 1 second of state changes
- Performance overhead remains under 5%
- Backward compatibility with existing NDJSON tools

**Testing Strategy**:
- Unit tests for progress rendering logic
- Integration tests across terminal types
- Performance benchmarks for animation overhead
- Compatibility tests with CI/CD environments