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

## TUI Libraries and Frameworks

### 1. Go TUI Libraries
- **Bubbletea**: Modern, idiomatic Go TUI framework
- **tview**: Rich interactive widgets and flexible layouts
- **tcell**: Low-level terminal manipulation library
- **termui**: Dashboard and monitoring UI components
- **lipgloss**: Styling and colors for terminal applications

### 2. Rust TUI Libraries
- **ratatui**: Modern Rust TUI library (successor to tui-rs)
- **crossterm**: Cross-platform terminal handling
- **tui-rs**: Original terminal UI library
- **iced**: GUI and TUI hybrid framework

### 3. Python TUI Libraries
- **Rich**: Rich text and beautiful formatting
- **Textual**: Modern TUI framework for Python
- **curses**: Traditional terminal interface library
- **urwid**: Flexible console UI library

### 4. Node.js TUI Libraries
- **Inquirer.js**: Interactive command-line prompts
- **Blessed**: Terminal interface library
- **ink**: React for CLIs
- **oclif**: CLI framework with rich output

## Core TUI Concepts

### 1. Terminal Capabilities
- **Screen size detection**: Handle resizing and variable dimensions
- **Color support**: ANSI colors, 256-color, RGB
- **Input handling**: Keyboard, mouse, clipboard events
- **Cross-platform**: Windows (cmd/PowerShell), macOS (Terminal.app), Linux (xterm/gnome-terminal)
- **Performance**: Efficient rendering and event loops

### 2. Layout Systems
- **Grid layouts**: CSS Grid-like arrangements
- **Flexbox**: Flexible box layouts
- **Absolute positioning**: Precise coordinate placement
- **Responsive design**: Adaptive layouts for different terminal sizes
- **Scrolling**: Viewports and content pagination

### 3. Interactive Components
- **Menus and navigation**: Keyboard-driven interfaces
- **Forms and input**: Text fields, checkboxes, radio buttons
- **Tables and lists**: Sortable, filterable data displays
- **Progress indicators**: Bars, spinners, status displays
- **Dialogs**: Modals, confirmations, notifications

## TUI Development Patterns

### Bubbletea (Go) Example
```go
package main

import (
    "fmt"
    "strings"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    choices []string
    cursor  int
    selected string
}

func initialModel() model {
    return model{
        choices: []string{"Option 1", "Option 2", "Option 3"},
        cursor:  0,
    }
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyUp:
            if m.cursor > 0 {
                m.cursor--
            }
        case tea.KeyDown:
            if m.cursor < len(m.choices)-1 {
                m.cursor++
            }
        case tea.KeyEnter:
            m.selected = m.choices[m.cursor]
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    s := strings.Builder{}
    s.WriteString("What should we buy at the market?\n\n")

    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
    }

    s.WriteString("\nPress q to quit.\n")
    return s.String()
}

func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
    }
}
```

### Ratatui (Rust) Example
```rust
use ratatui::{
    backend::CrosstermBackend,
    layout::{Constraint, Direction, Layout},
    style::{Color, Modifier, Style},
    text::Span,
    widgets::{Block, Borders, List, ListItem, Paragraph},
    Terminal,
};

struct App {
    items: Vec<String>,
    selected: usize,
}

impl App {
    fn new() -> Self {
        Self {
            items: vec![
                "Item 1".to_string(),
                "Item 2".to_string(),
                "Item 3".to_string(),
            ],
            selected: 0,
        }
    }

    fn next(&mut self) {
        self.selected = (self.selected + 1) % self.items.len();
    }

    fn previous(&mut self) {
        self.selected = if self.selected > 0 {
            self.selected - 1
        } else {
            self.items.len() - 1
        };
    }
}

fn ui(f: &mut Frame, app: &App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .margin(1)
        .constraints(
            [
                Constraint::Percentage(50),
                Constraint::Percentage(50),
            ]
            .as_ref(),
        )
        .split(f.size());

    let items: Vec<ListItem> = app
        .items
        .iter()
        .enumerate()
        .map(|(i, item)| {
            let style = if i == app.selected {
                Style::default().bg(Color::LightBlue)
            } else {
                Style::default()
            };
            ListItem::new(Span::styled(item.as_str(), style))
        })
        .collect();

    let list = List::new(items)
        .block(Block::default().borders(Borders::ALL).title("List"));
    f.render_widget(list, chunks[0]);

    let paragraph = Paragraph::new(format!("Selected item: {}", app.items[app.selected]))
        .block(Block::default().borders(Borders::ALL).title("Details"));
    f.render_widget(paragraph, chunks[1]);
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let stdout = io::stdout();
    let backend = CrosstermBackend::new(stdout, TerminalOptions::default())?;
    let mut terminal = Terminal::new(backend)?;
    
    let mut app = App::new();
    
    loop {
        terminal.draw(|f| ui(f, &app))?;
        
        if let Event::Key(key) = event::read()? {
            match key {
                KeyEvent::Left => app.previous(),
                KeyEvent::Right => app.next(),
                KeyEvent::Char('q') => break,
                _ => {}
            }
        }
    }
    
    Ok(())
}
```

### Rich (Python) Example
```python
from rich.console import Console
from rich.layout import Layout
from rich.panel import Panel
from rich.table import Table
from rich.progress import Progress, SpinnerColumn, TextColumn

console = Console()

# Create a layout
layout = Layout()
layout.split_column(
    Layout(name="header", size=3),
    Layout(name="main"),
    Layout(name="footer", size=3)
)

# Create a table
table = Table(title="Projects")
table.add_column("ID", style="cyan", no_wrap=True)
table.add_column("Name", style="magenta")
table.add_column("Status", style="green")

table.add_row("1", "Project Alpha", "Active")
table.add_row("2", "Project Beta", "Complete")

# Main loop
with console.screen() as screen:
    while True:
        layout["header"].update(Panel("Dashboard", style="bold blue"))
        layout["main"].update(Panel(table))
        layout["footer"].update(Panel("Press 'q' to quit"))
        
        console.print(layout)
        
        # Handle input (simplified)
        if console.input("Continue? (y/n): ").lower() == 'n':
            break
```

## Input Handling Patterns

### Cross-Platform Input Events
```go
// Go with Bubbletea - platform-abstracted
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyCtrlC:
            return m, tea.Quit
        case tea.KeyUp, tea.KeyCtrlP:
            // Up arrow or Ctrl+P
            if m.cursor > 0 {
                m.cursor--
            }
        case tea.KeyDown, tea.KeyCtrlN:
            // Down arrow or Ctrl+N
            if m.cursor < len(m.items)-1 {
                m.cursor++
            }
        case tea.KeyEnter:
            m.selected = m.items[m.cursor]
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}
```

### Complex Input Handling
```rust
// Rust with crossterm
use crossterm::{
    event::{self, Event, KeyCode, KeyEvent},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode},
};

fn handle_input() -> Result<(), Box<dyn std::error::Error>> {
    enable_raw_mode()?;
    
    loop {
        match event::read()? {
            Event::Key(KeyEvent { code, .. }) => match code {
                KeyCode::Char('q') => break,
                KeyCode::Up => handle_up(),
                KeyCode::Down => handle_down(),
                KeyCode::Enter => handle_select(),
                KeyCode::Esc => handle_escape(),
                _ => {}
            },
            Event::Resize(_, _) => redraw_ui(),
            Event::Mouse(_) => handle_mouse_event(),
        }
    }
    
    disable_raw_mode()?;
    Ok(())
}
```

## Layout and Responsive Design

### Responsive Layout Algorithm
```go
type LayoutConstraints struct {
    MinWidth  int
    MaxWidth  int
    MinHeight int
    MaxHeight int
}

func calculateLayout(termWidth, termHeight int, items []Widget) []Rect {
    var layout []Rect
    
    // Simple responsive grid
    cols := max(1, termWidth/40) // Minimum 40 chars per column
    rows := (len(items) + cols - 1) / cols
    
    itemWidth := termWidth / cols
    itemHeight := termHeight / rows
    
    for i, item := range items {
        row := i / cols
        col := i % cols
        
        x := col * itemWidth
        y := row * itemHeight
        
        layout = append(layout, Rect{
            X: x, Y: y,
            Width: itemWidth, Height: itemHeight,
        })
    }
    
    return layout
}
```

### Adaptive Component Layout
```rust
struct ResponsiveLayout {
    layouts: HashMap<TerminalSize, Layout>,
    current: Layout,
}

impl ResponsiveLayout {
    fn update_for_size(&mut self, size: TerminalSize) {
        self.current = self.layouts
            .get(&size)
            .unwrap_or_else(|| self.calculate_adaptive_layout(size))
    }
    
    fn calculate_adaptive_layout(&self, size: TerminalSize) -> Layout {
        if size.width < 80 {
            // Mobile-style vertical layout
            self.vertical_layout()
        } else if size.width < 120 {
            // Tablet-style mixed layout
            self.mixed_layout()
        } else {
            // Desktop-style horizontal layout
            self.horizontal_layout()
        }
    }
}
```

## When to Use This Skill

Use this skill when you need to:
- Create interactive terminal applications
- Build command-line tools with rich user interfaces
- Design terminal dashboards and monitoring tools
- Implement cross-platform console applications
- Handle complex user input in terminals
- Create responsive terminal layouts
- Build interactive system administration tools
- Develop terminal-based productivity applications

## Best Practices

### 1. Performance
- Use efficient rendering (double buffering, differential updates)
- Minimize redraws and optimize event loops
- Handle large datasets with virtual scrolling

### 2. Accessibility
- Provide keyboard navigation for all interactions
- Support high contrast and color-blind friendly themes
- Include clear visual indicators and status messages

### 3. Cross-Platform Compatibility
- Test on Windows, macOS, and Linux terminals
- Handle different terminal capabilities gracefully
- Provide fallbacks for limited terminal features

### 4. User Experience
- Include help text and keyboard shortcuts
- Provide progress indicators for long operations
- Implement undo/redo where appropriate
- Save and restore application state

## Testing TUI Applications

### Unit Testing Components
```go
func TestModelUpdate(t *testing.T) {
    tests := []struct {
        name     string
        model    model
        msg       tea.Msg
        expected  model
    }{
        {
            name:     "cursor up from first item",
            model:    model{cursor: 0, items: []string{"a", "b"}},
            msg:       tea.KeyMsg{Type: tea.KeyUp},
            expected:  model{cursor: 0, items: []string{"a", "b"}}, // Can't go up from first
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            updated, _ := tt.model.Update(tt.msg)
            assert.Equal(t, tt.expected, updated)
        })
    }
}
```

### Integration Testing
```python
def test_full_workflow(capsys):
    """Test complete TUI workflow"""
    # Simulate user input
    with patch('builtins.input', return_value='test\n'):
        app.run()
    
    # Check output
    captured = capsys.readouterr()
    assert 'Welcome' in captured.out
    assert 'Goodbye' in captured.out
```

Always prioritize:
- Responsive design for different terminal sizes
- Intuitive keyboard navigation
- Clear visual hierarchy and feedback
- Cross-platform compatibility
- Performance and efficiency