# Inline Logs Proposal

## Current vs Enhanced Display

### Current Display:
```
[████████▓▓▒░░░░░░░░░] 40% Step 2/4

✓ reproduce (2.3s)
✓ hypothesize (5.1s)
~ investigate                    🕐 Running for 12.7s
```

### Enhanced with Inline Logs:
```
[████████▓▓▒░░░░░░░░░] 40% Step 2/4

✓ reproduce (2.3s)                   │ Created reproduction case in CLAUDE.md
✓ hypothesize (5.1s)                 │ Generated 3 hypotheses for root cause
~ investigate                        │ › Analyzing codebase structure...
                      🕐 12.7s       │ › Found 127 files in internal/ directory
                                     │ › Examining display/bubbletea_model.go
```

## Design Options

### Option 1: Side-by-side Layout
```
╭─ Steps ──────────────────────────────┬─ Live Logs ──────────────────────────╮
│ [████████▓▓▒░░░░░░░░░] 40% Step 2/4  │                                      │
│                                      │                                      │
│ ✓ reproduce (2.3s)                   │ • Created reproduction.json          │
│ ✓ hypothesize (5.1s)                 │ • Found 3 potential root causes     │
│ ~ investigate           🕐 12.7s     │ › Analyzing internal/display/...     │
│                                      │ › Reading bubbletea_model.go         │
│                                      │ › Searching for event handlers       │
╰──────────────────────────────────────┴──────────────────────────────────────╯
```

### Option 2: Minimal Inline (Recommended)
```
[████████▓▓▒░░░░░░░░░] 40% Step 2/4

✓ reproduce (2.3s)                   Created reproduction case
✓ hypothesize (5.1s)                 Generated 3 hypotheses
~ investigate           🕐 12.7s     › Analyzing codebase structure...
```

### Option 3: Scrolling Log Window
```
[████████▓▓▒░░░░░░░░░] 40% Step 2/4

✓ reproduce (2.3s)
✓ hypothesize (5.1s)
~ investigate                        🕐 12.7s

┌─ Recent Activity ─────────────────────────────────────────┐
│ › Analyzing codebase structure...                         │
│ › Found 127 files in internal/ directory                  │
│ › Examining display/bubbletea_model.go                    │
│ › Searching for event handlers in Update() method         │
│ › Found 3 potential issues with state management          │
└────────────────────────────────────────────────────────────┘
```

## Log Filtering Strategy

### What to Show:
- ✅ **Tool calls**: "Reading file.go", "Running tests", "Executing command"
- ✅ **Progress milestones**: "Found 3 issues", "Generated 5 solutions"
- ✅ **Errors/warnings**: "Warning: deprecated API", "Error: file not found"
- ✅ **Key decisions**: "Choosing approach A over B", "Applying fix to line 42"

### What to Filter Out:
- ❌ Debug spam: "Checking...", "Processing...", "Loading..."
- ❌ Repetitive actions: Token counting, routine operations
- ❌ Internal system messages: Memory allocation, garbage collection

## Truncation Rules

### Smart Truncation:
```bash
# Original log line:
"Reading file /very/long/path/to/internal/display/bubbletea_model.go and analyzing structure"

# Truncated for 50-char limit:
"Reading ...bubbletea_model.go and analyzing..."

# File path compression:
"Reading internal/*/bubbletea_model.go"
```

### Terminal Width Adaptation:
- **Wide terminals (>120 cols)**: Full side-by-side layout
- **Medium terminals (80-120)**: Inline logs, truncated
- **Narrow terminals (<80)**: Logs below steps, minimal

## Implementation Considerations

### Data Source:
- Capture from Claude Code adapter stdout/stderr
- Parse for tool calls, file operations, key phrases
- Filter by importance level (error > warning > info)

### Performance:
- Buffer last 10-20 log lines per step
- Update display max 5 times per second
- Avoid excessive re-rendering

### Configuration:
```yaml
display:
  inline_logs:
    enabled: true
    max_length: 50        # Characters per log line
    show_completed: true  # Show logs for completed steps
    filter_level: "info"  # error|warning|info|debug
```

## Color Scheme

```
✓ reproduce (2.3s)                   Created reproduction case
  ↑ Green step                        ↑ Muted gray log

~ investigate           🕐 12.7s     › Analyzing codebase structure...
  ↑ Cyan running                      ↑ White/bright active log

✗ fix (failed)                       Error: syntax error on line 42
  ↑ Red failed                        ↑ Red error log
```

## Example Terminal Output

### During Active Step:
```bash
$ wave run debug

[████████▓▓▒░░░░░░░░░] 60% Step 3/4

✓ reproduce (2.3s)                   Created reproduction case
✓ hypothesize (5.1s)                 Generated 3 hypotheses
~ investigate           🕐 18.2s     › Found issue in Update() method
```

### With Error:
```bash
[██████░░░░░░░░░░░░░░] 30% Step 2/4

✓ reproduce (2.3s)                   Created reproduction case
✗ hypothesize (failed)               Error: model context exceeded
```

### Completed Pipeline:
```bash
✓ Pipeline 'debug' completed successfully (47.3s)

  Deliverables (4):
     • reproduction.json              Reproduced the issue
     • hypotheses.json               3 potential root causes
     • investigation.md              Found event handling bug
     • fix-summary.md                Applied 2-line fix
```