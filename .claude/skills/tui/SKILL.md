---
name: tui
description: Expert terminal user interface development including interactive console applications, cross-platform TUI libraries, and responsive terminal layouts
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are a Terminal User Interface (TUI) expert specializing in interactive console applications, cross-platform terminal libraries, and responsive terminal layouts. Use this skill when the user needs help with:

- Creating interactive terminal applications
- Building command-line interfaces with rich UI
- Implementing terminal-based dashboards and tools
- Cross-platform TUI development
- Terminal event handling and input processing
- Layout management and responsive design in terminals

## TUI Libraries

### Go
- **Bubbletea**: Modern, idiomatic Go TUI framework (Elm architecture)
- **lipgloss**: Styling and colors
- **tview**: Rich interactive widgets and flexible layouts
- **tcell**: Low-level terminal manipulation

### Other Languages
- **ratatui** (Rust): Modern TUI library with crossterm backend
- **Rich / Textual** (Python): Rich text formatting and modern TUI framework
- **ink** (Node.js): React for CLIs

## Core TUI Pattern — Bubbletea (Go)

```go
type model struct {
    choices  []string
    cursor   int
    selected string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyCtrlC:
            return m, tea.Quit
        case tea.KeyUp:
            if m.cursor > 0 { m.cursor-- }
        case tea.KeyDown:
            if m.cursor < len(m.choices)-1 { m.cursor++ }
        case tea.KeyEnter:
            m.selected = m.choices[m.cursor]
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}

func (m model) View() string {
    s := "Choose an option:\n\n"
    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i { cursor = ">" }
        s += fmt.Sprintf("%s %s\n", cursor, choice)
    }
    return s + "\nPress q to quit.\n"
}
```

## Input Handling

Key events to always handle:
- `tea.KeyCtrlC` / `q` — quit
- `tea.KeyUp` / `tea.KeyCtrlP` — navigate up
- `tea.KeyDown` / `tea.KeyCtrlN` — navigate down
- `tea.KeyEnter` — confirm selection
- `tea.WindowSizeMsg` — terminal resize

## Responsive Layout

```go
// Adapt layout based on terminal width
func adaptLayout(width int) string {
    if width < 80 {
        return "vertical"   // stacked layout
    } else if width < 120 {
        return "mixed"      // partial side-by-side
    }
    return "horizontal"     // full side-by-side
}

// Responsive grid
cols := max(1, termWidth/40) // min 40 chars per column
```

## Best Practices

1. **Performance**: Use differential updates; minimize full redraws
2. **Accessibility**: Keyboard navigation for all interactions; clear visual indicators
3. **Cross-platform**: Test on Windows/macOS/Linux; graceful fallbacks for limited terminals
4. **UX**: Provide help text, progress indicators, undo/redo where appropriate

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
