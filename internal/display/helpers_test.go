package display

import (
	"testing"
)

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{"zero", 0, "0"},
		{"small", 500, "500"},
		{"just under 1k", 999, "999"},
		{"exactly 1k", 1000, "1.0k"},
		{"over 1k", 1500, "1.5k"},
		{"large", 10000, "10.0k"},
		{"very large", 100000, "100.0k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTokenCount(tt.tokens)
			if got != tt.want {
				t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestFormatFileCount(t *testing.T) {
	tests := []struct {
		name  string
		files int
		want  string
	}{
		{"zero", 0, "0 files"},
		{"one", 1, "1 file"},
		{"two", 2, "2 files"},
		{"many", 100, "100 files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileCount(tt.files)
			if got != tt.want {
				t.Errorf("FormatFileCount(%d) = %q, want %q", tt.files, got, tt.want)
			}
		})
	}
}

func TestFormatDuration_PackageLevel(t *testing.T) {
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
			got := FormatDuration(tt.durationMs)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.durationMs, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkFormatTokenCount_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatTokenCount(500)
	}
}

func BenchmarkFormatTokenCount_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatTokenCount(150000)
	}
}

func BenchmarkFormatFileCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatFileCount(42)
	}
}

func BenchmarkFormatDuration_Milliseconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDuration(500)
	}
}

func BenchmarkFormatDuration_Seconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDuration(5000)
	}
}

func BenchmarkFormatDuration_Minutes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDuration(90000)
	}
}

func BenchmarkFormatDuration_Hours(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDuration(5400000)
	}
}

func BenchmarkFormatter_ProgressBar(b *testing.B) {
	f := NewFormatter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ProgressBar(50, 40)
	}
}

func BenchmarkFormatter_Bold(b *testing.B) {
	f := NewFormatter()
	text := "sample text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Bold(text)
	}
}

func BenchmarkFormatter_Colorize(b *testing.B) {
	f := NewFormatter()
	text := "sample text"
	color := "\033[34m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Colorize(text, color)
	}
}

func BenchmarkFormatter_Truncate(b *testing.B) {
	f := NewFormatter()
	text := "this is a very long piece of text that will be truncated"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Truncate(text, 20)
	}
}

func BenchmarkFormatter_Wrap(b *testing.B) {
	f := NewFormatter()
	text := "this is a longer piece of text that needs to be wrapped across multiple lines"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Wrap(text, 20)
	}
}

func BenchmarkFormatter_Box(b *testing.B) {
	f := NewFormatter()
	content := "Box content\nWith multiple lines"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Box(content, "Title", 40)
	}
}

func BenchmarkFormatter_TableRow(b *testing.B) {
	f := NewFormatter()
	columns := []string{"Column1", "Column2", "Column3"}
	widths := []int{15, 15, 15}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.TableRow(columns, widths)
	}
}

func BenchmarkFormatter_FormatBytes(b *testing.B) {
	f := NewFormatter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.FormatBytes(1073741824) // 1 GB
	}
}

func BenchmarkSpinner_Current(b *testing.B) {
	spinner := NewSpinner(AnimationSpinner)
	spinner.Start()
	defer spinner.Stop()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spinner.Current()
	}
}

func BenchmarkProgressBar_Render(b *testing.B) {
	pb := NewProgressBar(100, 40)
	pb.SetProgress(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.Render()
	}
}

func BenchmarkStepStatus_Render(b *testing.B) {
	ss := NewStepStatus("step-1", "Test Step", "developer")
	ss.UpdateState(StateRunning)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ss.Render()
	}
}

func BenchmarkANSICodec_Colorize(b *testing.B) {
	codec := NewANSICodec()
	text := "sample text"
	color := "\033[34m"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Colorize(text, color)
	}
}

func BenchmarkGetUnicodeCharSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetUnicodeCharSet()
	}
}

func BenchmarkTerminalColorContext_FormatState(b *testing.B) {
	tcc := NewTerminalColorContext()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tcc.FormatState(StateRunning)
	}
}

func BenchmarkTerminalColorContext_GetStateIcon(b *testing.B) {
	tcc := NewTerminalColorContext()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tcc.GetStateIcon(StateCompleted)
	}
}

func BenchmarkGetColorSchemeByName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetColorSchemeByName("dark")
	}
}

func BenchmarkDisplayConfig_Validate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := DisplayConfig{
			RefreshRate:      0,
			ColorMode:        "invalid",
			ColorTheme:       "unknown",
			AnimationEnabled: true,
		}
		config.Validate()
	}
}

func BenchmarkProgressAnimation_Render(b *testing.B) {
	pa := NewProgressAnimation("Loading", 100, 40)
	pa.Start()
	defer pa.Stop()
	pa.SetProgress(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pa.Render()
	}
}

func BenchmarkMultiSpinner_Current(b *testing.B) {
	ms := NewMultiSpinner()
	ms.Add("test", AnimationSpinner)
	ms.Start("test")
	defer ms.Stop("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.Current("test")
	}
}

func BenchmarkLoadingIndicator_Render(b *testing.B) {
	li := NewLoadingIndicator("Loading...")
	li.Start()
	defer li.Stop()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		li.Render()
	}
}

func BenchmarkResizeHandler_GetCurrentSize(b *testing.B) {
	rh := NewResizeHandler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rh.GetCurrentSize()
	}
}
