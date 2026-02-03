# Quickstart: Enhanced Pipeline Progress Visualization

**Created**: 2026-02-03
**Feature**: 018-enhanced-progress
**Purpose**: Quick implementation guide for enhanced progress visualization

## Overview

This guide provides a step-by-step implementation path for enhancing Wave's pipeline progress visualization with modern CLI/TUI features including real-time progress bars, animated indicators, and a dashboard layout with the Wave logo.

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)

**Goal**: Establish foundation for enhanced progress display

**Tasks**:
1. Create `internal/display` package structure
2. Implement terminal detection and capability queries
3. Extend Event struct with optional progress fields
4. Add dual-stream output (stderr for progress, stdout for NDJSON)

**Key Files**:
- `internal/display/terminal.go` - TTY detection and sizing
- `internal/display/capability.go` - ANSI feature detection
- `internal/event/types.go` - Enhanced event schema
- `internal/event/emitter.go` - Dual output support

**Validation**:
```bash
# Test terminal detection
wave run --pipeline=test-suite --progress=enhanced

# Verify NDJSON compatibility
wave run --pipeline=test-suite | jq '.state'

# Test non-TTY fallback
wave run --pipeline=test-suite > output.log 2>&1
```

### Phase 2: Basic Progress Display (Week 2)

**Goal**: Implement fundamental progress visualization

**Tasks**:
1. Create progress bar rendering system
2. Add step status indicators with colors
3. Implement elapsed time and ETA calculations
4. Integrate with existing pipeline execution

**Key Components**:
- Progress bars with completion percentage
- Status icons (✓ ✗ ⏳ 🔄)
- Color-coded state transitions
- Real-time timing display

**Example Output**:
```
[14:30:45] ⏳ step-1 (navigator)  [████████░░] 75% (2.3s elapsed, ~1.2s remaining)
[14:30:47] ✓ step-1 (navigator)  [██████████] 100% (3.5s total, 15.2k tokens)
[14:30:48] ⏳ step-2 (craftsman) [░░░░░░░░░░] 0% (Starting...)
```

### Phase 3: Dashboard Layout (Week 3)

**Goal**: Implement rich dashboard interface with Wave branding

**Tasks**:
1. Create panel-based layout system
2. Integrate Wave ASCII logo display
3. Add project information sidebar
4. Implement responsive layout for different terminal sizes

**Dashboard Layout**:
```
┌─ Wave Pipeline Status ─────────────────┬─ Project Info ──────────┐
│  ╦ ╦╔═╗╦  ╦╔═╗                         │ Manifest: wave.yaml     │
│  ║║║╠═╣╚╗╔╝║╣    Step 3 of 7          │ Pipeline: test-suite    │
│  ╚╩╝╩ ╩ ╚╝ ╚═╝   [████████░░░] 75%    │ Workspace: .wave/work   │
├────────────────────────────────────────┼─────────────────────────┤
│ Current: step-3 (craftsman)            │ Tokens: 15.2k ⟲        │
│ Status:  Contract validation...        │ Files: 23 modified      │
│ Elapsed: 00:02:34 | ETA: 00:01:12      │ Artifacts: 8 generated │
└────────────────────────────────────────┴─────────────────────────┘
```

### Phase 4: Advanced Animation (Week 4)

**Goal**: Add engaging animations and "numbers going up" effects

**Tasks**:
1. Implement loading spinners and progress animations
2. Add incrementing counter animations for metrics
3. Create smooth state transitions
4. Add background activity indicators

**Animation Features**:
- Rotating spinners during processing
- Animated token counters
- Smooth progress bar fills
- Pulsing indicators for active operations

## Quick Implementation Guide

### 1. Terminal Detection Setup

```go
// internal/display/terminal.go
package display

import (
    "os"
    "golang.org/x/term"
)

func IsTerminal() bool {
    return term.IsTerminal(int(os.Stderr.Fd()))
}

func GetTerminalSize() (width, height int) {
    if !IsTerminal() {
        return 80, 24 // fallback defaults
    }
    w, h, err := term.GetSize(int(os.Stderr.Fd()))
    if err != nil {
        return 80, 24
    }
    return w, h
}

func SupportsColor() bool {
    // Check TERM and COLORTERM environment variables
    term := os.Getenv("TERM")
    colorterm := os.Getenv("COLORTERM")

    return term != "dumb" &&
           (colorterm == "truecolor" || colorterm == "24bit" ||
            strings.Contains(term, "color"))
}
```

### 2. Enhanced Event Emitter

```go
// internal/event/emitter.go - Add to existing file

func (e *NDJSONEmitter) EmitProgress(stepID string, progress int, message string) {
    event := Event{
        Timestamp:  time.Now(),
        PipelineID: e.currentPipelineID,
        StepID:     stepID,
        State:      "step_progress",
        Message:    message,
        Progress:   &progress,
    }
    e.Emit(event)
}

func (e *NDJSONEmitter) shouldShowEnhancedDisplay() bool {
    return e.humanReadable && IsTerminal()
}
```

### 3. Progress Bar Implementation

```go
// internal/display/progress.go
package display

import (
    "fmt"
    "strings"
)

type ProgressBar struct {
    Width     int
    Completed int
    Total     int
    Style     BarStyle
}

type BarStyle int

const (
    StyleBlocks BarStyle = iota
    StyleBar
    StyleGradient
)

func (pb ProgressBar) Render() string {
    if pb.Total == 0 {
        return "[──────────] 0%"
    }

    percentage := float64(pb.Completed) / float64(pb.Total)
    filled := int(percentage * float64(pb.Width))

    var filledChar, emptyChar string
    switch pb.Style {
    case StyleBlocks:
        filledChar, emptyChar = "█", "░"
    case StyleBar:
        filledChar, emptyChar = "=", "-"
    case StyleGradient:
        filledChar, emptyChar = "▓", "░"
    }

    bar := strings.Repeat(filledChar, filled) +
          strings.Repeat(emptyChar, pb.Width-filled)

    return fmt.Sprintf("[%s] %d%% (%d/%d)",
                       bar, int(percentage*100), pb.Completed, pb.Total)
}
```

### 4. Dashboard Panel System

```go
// internal/display/dashboard.go
package display

import (
    "fmt"
    "strings"
)

const WaveLogo = `  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝`

type Panel struct {
    Title   string
    Content string
    X, Y    int
    Width   int
    Height  int
}

func RenderDashboard(width, height int, pipeline PipelineContext) string {
    if width < 80 || height < 10 {
        return renderMinimalView(pipeline)
    }

    leftWidth := width * 2 / 3
    rightWidth := width - leftWidth - 1

    // Create panels
    logoPanel := Panel{
        Title: "Wave Pipeline Status",
        Content: fmt.Sprintf("%s\n\nStep %d of %d\n%s",
                 WaveLogo,
                 pipeline.CurrentStep,
                 pipeline.TotalSteps,
                 renderProgressBar(pipeline)),
        Width: leftWidth,
        Height: 8,
    }

    infoPanel := Panel{
        Title: "Project Info",
        Content: fmt.Sprintf("Manifest: %s\nPipeline: %s\nWorkspace: %s",
                 pipeline.ManifestPath,
                 pipeline.Name,
                 pipeline.WorkspacePath),
        Width: rightWidth,
        Height: 8,
    }

    return renderPanels(logoPanel, infoPanel)
}
```

### 5. Integration with Run Command

```go
// cmd/wave/commands/run.go - Modify existing runRun function

func runRun(opts RunOptions, debug bool) error {
    // ... existing setup code ...

    // Enhanced emitter with progress support
    var emitter event.EventEmitter
    if display.IsTerminal() && !opts.JSONOutput {
        emitter = event.NewEnhancedEmitter()
    } else {
        emitter = event.NewNDJSONEmitter()
    }

    // ... rest of execution ...
}
```

## Configuration Options

### Manifest Configuration

Add optional progress settings to `wave.yaml`:

```yaml
# wave.yaml
runtime:
  progress:
    style: "enhanced"           # enhanced, basic, minimal
    animation: true            # enable animations
    refresh_rate: 60          # milliseconds between updates
    show_logo: true           # display Wave logo
    show_metrics: true        # display token/file counts
    color_theme: "default"    # default, dark, light, high_contrast
```

### Environment Variables

Support environment-based configuration:

```bash
# Disable enhanced progress (fall back to basic)
WAVE_PROGRESS=basic wave run --pipeline=test

# Force color output even when piped
FORCE_COLOR=1 wave run --pipeline=test

# Disable animations for slower terminals
WAVE_NO_ANIMATION=1 wave run --pipeline=test
```

## Testing Strategy

### Unit Tests

```go
// internal/display/progress_test.go
func TestProgressBarRendering(t *testing.T) {
    tests := []struct {
        name      string
        completed int
        total     int
        expected  string
    }{
        {"zero progress", 0, 10, "[░░░░░░░░░░] 0% (0/10)"},
        {"half complete", 5, 10, "[█████░░░░░] 50% (5/10)"},
        {"full complete", 10, 10, "[██████████] 100% (10/10)"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pb := ProgressBar{
                Width: 10,
                Completed: tt.completed,
                Total: tt.total,
                Style: StyleBlocks,
            }
            result := pb.Render()
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```bash
#!/bin/bash
# Test enhanced progress with real pipeline

# Test 1: Basic progress display
echo "Testing basic progress display..."
wave run --pipeline=hello-world --progress=enhanced

# Test 2: Dashboard layout
echo "Testing dashboard layout..."
wave run --pipeline=test-suite --progress=enhanced --dashboard=true

# Test 3: Non-TTY fallback
echo "Testing non-TTY fallback..."
wave run --pipeline=hello-world --progress=enhanced > output.log 2>&1
grep -q "completed" output.log || echo "FAIL: No completion message"

# Test 4: NDJSON compatibility
echo "Testing NDJSON compatibility..."
wave run --pipeline=hello-world | jq '.state' > /dev/null || echo "FAIL: Invalid JSON"
```

## Performance Considerations

### Optimization Guidelines

1. **Efficient Rendering**: Use double-buffering to prevent flicker
2. **Smart Updates**: Only redraw when data actually changes
3. **Background Processing**: Calculate ETA and metrics in separate goroutines
4. **Memory Management**: Limit history retention and cleanup old progress data

### Monitoring

```go
// Add performance metrics to progress system
type ProgressMetrics struct {
    RenderTime    time.Duration
    UpdateCount   int64
    MemoryUsage   int64
    OverheadRatio float64 // Progress overhead vs total execution time
}
```

## Troubleshooting

### Common Issues

**Issue**: Progress not showing
- **Check**: Terminal detection with `wave --debug run`
- **Fix**: Set `TERM=xterm-256color` if using basic terminal

**Issue**: Corrupted display output
- **Check**: Terminal size with `tput cols` and `tput lines`
- **Fix**: Increase terminal size or use `--progress=minimal`

**Issue**: Performance degradation
- **Check**: Refresh rate with `WAVE_DEBUG_PROGRESS=1`
- **Fix**: Increase refresh rate or disable animations

### Debug Commands

```bash
# Debug progress system
WAVE_DEBUG_PROGRESS=1 wave run --pipeline=test

# Test terminal capabilities
wave debug terminal

# Validate progress events
wave run --pipeline=test | jq 'select(.state == "step_progress")'
```

## Next Steps

After implementing the basic enhanced progress system:

1. **User Feedback**: Gather feedback on visual design and performance
2. **Additional Animations**: Add more sophisticated progress effects
3. **Customization**: Allow user-defined color schemes and layouts
4. **Integration**: Connect with monitoring tools and CI/CD systems
5. **Documentation**: Create comprehensive user guide and examples

## Success Metrics

- **User Satisfaction**: 95% of users can identify current pipeline status within 2 seconds
- **Performance**: Progress overhead remains under 5% of total execution time
- **Compatibility**: All existing NDJSON tools continue working without modification
- **Adoption**: Enhanced progress becomes the default experience for interactive use