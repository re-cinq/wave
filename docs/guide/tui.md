# TUI Guide

Wave includes a terminal user interface (TUI) that provides real-time pipeline monitoring with progress bars, spinners, and interactive controls.

## Enabling / Disabling

The TUI is enabled by default when running in an interactive terminal. Wave auto-detects whether to use the TUI based on several factors:

### Detection Logic

1. **`--no-tui` flag** (highest priority) — disables the TUI
2. **`WAVE_FORCE_TTY` env var** — override TTY detection:
   - `"1"` or `"true"` → force TUI on
   - `"0"` or `"false"` → force TUI off
3. **Dumb terminals** — TUI disabled when `TERM=dumb`
4. **TTY check** — TUI enabled only when stdout is a terminal

### Examples

```bash
# Default: TUI auto-detected
wave run speckit-flow "task"

# Force disable TUI
wave run speckit-flow "task" --no-tui

# Force text output
wave run speckit-flow "task" -o text

# Force TUI in non-TTY context
WAVE_FORCE_TTY=1 wave run speckit-flow "task"
```

## CI/CD Environments

Wave automatically detects CI/CD environments (GitHub Actions, GitLab CI, CircleCI, etc.) and disables the TUI in favor of plain text or JSON output. No configuration needed.

For explicit control in CI:

```bash
# JSON output for machine parsing
wave run speckit-flow "task" -o json

# Plain text for CI logs
wave run speckit-flow "task" -o text
```

## Output Modes

| Mode | Flag | Description |
|------|------|-------------|
| `auto` | `-o auto` (default) | TUI if terminal, text otherwise |
| `text` | `-o text` | Plain text progress to stderr |
| `json` | `-o json` | NDJSON events to stdout |
| `quiet` | `-o quiet` | Only final result |

## Related Topics

- [CLI Reference](/reference/cli) — All command flags
- [Event Format](/reference/events) — JSON event schema
- [Environment Variables](/reference/environment) — `WAVE_FORCE_TTY` and display settings
