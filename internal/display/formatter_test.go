package display

import (
	"strings"
	"testing"
)

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Fatal("NewFormatter returned nil")
	}
	if f.codec == nil {
		t.Error("Expected codec to be initialized")
	}
}

func TestNewFormatterWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		asciiOnly bool
	}{
		{"auto mode", "auto", false},
		{"color on", "on", false},
		{"color off", "off", false},
		{"ascii only", "auto", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatterWithConfig(tt.colorMode, tt.asciiOnly)
			if f == nil {
				t.Fatal("NewFormatterWithConfig returned nil")
			}
		})
	}
}

func TestFormatter_CursorMovement(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	tests := []struct {
		name     string
		fn       func(int) string
		input    int
		wantCode string
	}{
		{"MoveCursorUp_valid", f.MoveCursorUp, 5, "\033[5A"},
		{"MoveCursorUp_zero", f.MoveCursorUp, 0, ""},
		{"MoveCursorUp_negative", f.MoveCursorUp, -1, ""},
		{"MoveCursorDown_valid", f.MoveCursorDown, 3, "\033[3B"},
		{"MoveCursorDown_zero", f.MoveCursorDown, 0, ""},
		{"MoveCursorForward_valid", f.MoveCursorForward, 10, "\033[10C"},
		{"MoveCursorForward_zero", f.MoveCursorForward, 0, ""},
		{"MoveCursorBack_valid", f.MoveCursorBack, 2, "\033[2D"},
		{"MoveCursorBack_zero", f.MoveCursorBack, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.input)
			if f.enabled && got != tt.wantCode {
				t.Errorf("got %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestFormatter_MoveCursorToColumn(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	tests := []struct {
		name   string
		column int
		want   string
	}{
		{"valid column", 10, "\033[10G"},
		{"first column", 1, "\033[1G"},
		{"zero column", 0, "\033[0G"},
		{"negative column", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MoveCursorToColumn(tt.column)
			if f.enabled && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_MoveCursorToPosition(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	tests := []struct {
		name    string
		row     int
		col     int
		want    string
		isEmpty bool
	}{
		{"valid position", 5, 10, "\033[5;10H", false},
		{"origin", 1, 1, "\033[1;1H", false},
		{"zero position", 0, 0, "\033[0;0H", false},
		{"negative row", -1, 5, "", true},
		{"negative col", 5, -1, "", true},
		{"both negative", -1, -1, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MoveCursorToPosition(tt.row, tt.col)
			if tt.isEmpty && got != "" && f.enabled {
				t.Errorf("expected empty for invalid input, got %q", got)
			}
			if !tt.isEmpty && f.enabled && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_ClearFunctions(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	t.Run("ClearLine", func(t *testing.T) {
		got := f.ClearLine()
		// Should return ANSI code if enabled
		if f.enabled && got == "" {
			t.Error("Expected non-empty clear line code when enabled")
		}
	})

	t.Run("ClearScreen", func(t *testing.T) {
		got := f.ClearScreen()
		if f.enabled && got != "\033[2J" {
			t.Errorf("got %q, want %q", got, "\033[2J")
		}
	})

	t.Run("ClearToEndOfLine", func(t *testing.T) {
		got := f.ClearToEndOfLine()
		if f.enabled && got != "\033[K" {
			t.Errorf("got %q, want %q", got, "\033[K")
		}
	})

	t.Run("ClearToStartOfLine", func(t *testing.T) {
		got := f.ClearToStartOfLine()
		if f.enabled && got != "\033[1K" {
			t.Errorf("got %q, want %q", got, "\033[1K")
		}
	})
}

func TestFormatter_TextFormatting(t *testing.T) {
	f := NewFormatterWithConfig("on", false)
	text := "test text"

	tests := []struct {
		name    string
		fn      func(string) string
		wantSeq string
	}{
		{"Bold", f.Bold, "\033[1m"},
		{"Dim", f.Dim, "\033[2m"},
		{"Italic", f.Italic, "\033[3m"},
		{"Underline", f.Underline, "\033[4m"},
		{"Strikethrough", f.Strikethrough, "\033[9m"},
		{"Inverse", f.Inverse, "\033[7m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(text)
			if f.enabled {
				if !strings.Contains(result, text) {
					t.Errorf("Result should contain original text")
				}
				if !strings.Contains(result, tt.wantSeq) {
					t.Errorf("Result should contain formatting sequence %s", tt.wantSeq)
				}
			} else {
				if result != text {
					t.Errorf("When disabled, should return original text")
				}
			}
		})
	}
}

func TestFormatter_ColorFormatting(t *testing.T) {
	f := NewFormatterWithConfig("on", false)
	text := "colored text"

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Primary", f.Primary},
		{"Success", f.Success},
		{"Error", f.Error},
		{"Warning", f.Warning},
		{"Muted", f.Muted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(text)
			if !strings.Contains(result, text) {
				t.Errorf("Result should contain original text")
			}
		})
	}
}

func TestFormatter_Colorize(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	tests := []struct {
		name      string
		text      string
		colorCode string
		wantCode  bool
	}{
		{"with color code", "text", "\033[34m", true},
		{"empty color code", "text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Colorize(tt.text, tt.colorCode)
			if !strings.Contains(result, tt.text) {
				t.Errorf("Result should contain original text")
			}
			if tt.wantCode && f.enabled && !strings.Contains(result, tt.colorCode) {
				t.Errorf("Result should contain color code")
			}
		})
	}
}

func TestFormatter_ProgressBar(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		percent int
		width   int
	}{
		{"zero percent", 0, 20},
		{"half percent", 50, 20},
		{"full percent", 100, 20},
		{"negative percent", -10, 20},
		{"over 100 percent", 150, 20},
		{"small width", 25, 5},
		{"large width", 75, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.ProgressBar(tt.percent, tt.width)
			if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
				t.Error("Progress bar should contain brackets")
			}
			if !strings.Contains(result, "%") {
				t.Error("Progress bar should contain percentage")
			}
		})
	}
}

func TestFormatter_ProgressBarWithLabel(t *testing.T) {
	f := NewFormatter()

	label := "Loading"
	result := f.ProgressBarWithLabel(label, 50, 20)

	if !strings.Contains(result, label) {
		t.Errorf("Result should contain label %q", label)
	}
	if !strings.Contains(result, "[") {
		t.Error("Result should contain progress bar bracket")
	}
}

func TestFormatter_Spinner(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		tick int
	}{
		{0},
		{1},
		{2},
		{3},
		{4}, // Should wrap around
		{100},
	}

	for _, tt := range tests {
		frame := f.Spinner(tt.tick)
		if frame == "" {
			t.Errorf("Spinner frame should not be empty for tick %d", tt.tick)
		}
	}
}

func TestFormatter_FormatDuration(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name       string
		durationMs int64
		want       string
	}{
		{"negative", -100, "0s"},
		{"zero", 0, "0ms"},
		{"milliseconds", 500, "500ms"},
		{"one second", 1000, "1s"},
		{"seconds", 5000, "5s"},
		{"one minute", 60000, "1m 0s"},
		{"minutes and seconds", 90000, "1m 30s"},
		{"one hour", 3600000, "1h 0m"},
		{"hours and minutes", 5400000, "1h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatDuration(tt.durationMs)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.durationMs, got, tt.want)
			}
		})
	}
}

func TestFormatter_FormatETA(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		etaMs int64
		want  string
	}{
		{"negative", -100, "calculating..."},
		{"zero", 0, "calculating..."},
		{"valid", 5000, "ETA: 5s"},
		{"minutes", 90000, "ETA: 1m 30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatETA(tt.etaMs)
			if got != tt.want {
				t.Errorf("FormatETA(%d) = %q, want %q", tt.etaMs, got, tt.want)
			}
		})
	}
}

func TestFormatter_FormatTimestamp(t *testing.T) {
	f := NewFormatter()

	// Test a known timestamp
	result := f.FormatTimestamp(0) // Unix epoch
	// Should return some time format string
	if result == "" {
		t.Error("FormatTimestamp should return non-empty string")
	}
}

func TestFormatter_FormatState(t *testing.T) {
	f := NewFormatter()

	states := []ProgressState{
		StateCompleted,
		StateFailed,
		StateRunning,
		StateSkipped,
		StateCancelled,
		StateNotStarted,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			result := f.FormatState(state)
			if result == "" {
				t.Errorf("FormatState(%s) should not return empty string", state)
			}
		})
	}
}

func TestFormatter_GetStateIcon(t *testing.T) {
	f := NewFormatter()

	states := []ProgressState{
		StateCompleted,
		StateFailed,
		StateRunning,
		StateSkipped,
		StateCancelled,
		StateNotStarted,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			icon := f.GetStateIcon(state)
			if icon == "" {
				t.Errorf("GetStateIcon(%s) should not return empty string", state)
			}
		})
	}
}

func TestFormatter_FormatBytes(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatter_FormatTokens(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{"small", 500, "500 tokens"},
		{"exactly 1k", 1000, "1.0k tokens"},
		{"large", 5000, "5.0k tokens"},
		{"very large", 15500, "15.5k tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatTokens(tt.tokens)
			if got != tt.want {
				t.Errorf("FormatTokens(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestFormatter_TableRow(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		columns []string
		widths  []int
		want    string
	}{
		{
			name:    "equal width",
			columns: []string{"col1", "col2"},
			widths:  []int{10, 10},
			want:    "col1       col2      ",
		},
		{
			name:    "truncate long",
			columns: []string{"very long column", "short"},
			widths:  []int{10, 10},
			want:    "very lo... short     ",
		},
		{
			name:    "mismatched lengths",
			columns: []string{"a", "b", "c"},
			widths:  []int{5, 5},
			want:    "a b c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.TableRow(tt.columns, tt.widths)
			if got != tt.want {
				t.Errorf("TableRow() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_Truncate(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{"no truncation needed", "short", 10, "short"},
		{"exact fit", "exact", 5, "exact"},
		{"truncate with ellipsis", "very long text", 10, "very lo" + f.charSet.Ellipsis},
		{"very small width", "text", 3, "tex"},
		{"width of 2", "text", 2, "te"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.Truncate(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}

func TestFormatter_Wrap(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"empty text", "", 10, []string{""}},
		{"no wrap needed", "short", 20, []string{"short"}},
		{"wrap at word boundary", "this is a longer text", 10, []string{"this is a", "longer", "text"}},
		{"zero width", "text", 0, []string{"text"}},
		{"negative width", "text", -5, []string{"text"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.Wrap(tt.text, tt.width)
			if len(got) != len(tt.want) {
				t.Errorf("Wrap() returned %d lines, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Wrap() line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFormatter_Box(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		content string
		title   string
		width   int
	}{
		{"with title", "content", "Title", 30},
		{"without title", "content", "", 30},
		{"multiline content", "line1\nline2", "Title", 30},
		{"small width with title", "content", "Very Long Title", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Box(tt.content, tt.title, tt.width)
			if !strings.Contains(result, "┌") {
				t.Error("Box should contain top-left corner")
			}
			if !strings.Contains(result, "└") {
				t.Error("Box should contain bottom-left corner")
			}
			if !strings.Contains(result, "│") {
				t.Error("Box should contain side borders")
			}
		})
	}
}

func TestFormatter_BulletList(t *testing.T) {
	f := NewFormatter()

	items := []string{"item1", "item2", "item3"}
	result := f.BulletList(items)

	for _, item := range items {
		if !strings.Contains(result, item) {
			t.Errorf("BulletList should contain item %q", item)
		}
	}
	// Check for bullet character
	if !strings.Contains(result, f.charSet.Bullet) {
		t.Error("BulletList should contain bullet character")
	}
}

func TestFormatter_NumberedList(t *testing.T) {
	f := NewFormatter()

	items := []string{"first", "second", "third"}
	result := f.NumberedList(items)

	for _, item := range items {
		if !strings.Contains(result, item) {
			t.Errorf("NumberedList should contain item %q", item)
		}
	}
	// Check for numbers
	if !strings.Contains(result, "1.") {
		t.Error("NumberedList should contain '1.'")
	}
	if !strings.Contains(result, "2.") {
		t.Error("NumberedList should contain '2.'")
	}
}

func TestFormatter_HorizontalRule(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		width int
	}{
		{"default width", 0},
		{"custom width", 40},
		{"negative width", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.HorizontalRule(tt.width)
			if result == "" {
				t.Error("HorizontalRule should not be empty")
			}
			if !strings.Contains(result, "─") {
				t.Error("HorizontalRule should contain horizontal line character")
			}
		})
	}
}

func TestFormatter_Reset(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	result := f.Reset()
	if f.enabled && result != "\033[0m" {
		t.Errorf("Reset() = %q, want %q", result, "\033[0m")
	}
}

func TestFormatter_CursorVisibility(t *testing.T) {
	f := NewFormatterWithConfig("on", false)

	t.Run("HideCursor", func(t *testing.T) {
		result := f.HideCursor()
		if f.enabled && result == "" {
			t.Error("HideCursor should return non-empty when enabled")
		}
	})

	t.Run("ShowCursor", func(t *testing.T) {
		result := f.ShowCursor()
		if f.enabled && result == "" {
			t.Error("ShowCursor should return non-empty when enabled")
		}
	})

	t.Run("SaveCursorPosition", func(t *testing.T) {
		result := f.SaveCursorPosition()
		if f.enabled && result == "" {
			t.Error("SaveCursorPosition should return non-empty when enabled")
		}
	})

	t.Run("RestoreCursorPosition", func(t *testing.T) {
		result := f.RestoreCursorPosition()
		if f.enabled && result == "" {
			t.Error("RestoreCursorPosition should return non-empty when enabled")
		}
	})
}
