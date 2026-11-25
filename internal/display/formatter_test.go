package display

import (
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

func TestFormatter_ColorizeAndError(t *testing.T) {
	colored := &Formatter{codec: &ANSICodec{colors: DefaultColorScheme, ansi: true}}
	plain := plainFormatter()

	if got := colored.Colorize("text", "\033[34m"); got != "\033[34mtext"+DefaultColorScheme.Reset {
		t.Errorf("Colorize = %q, want text wrapped in the color code and reset", got)
	}
	if got := colored.Colorize("text", ""); got != "text" {
		t.Errorf("Colorize with empty code = %q, want unchanged text", got)
	}
	if got := colored.Error("boom"); got != DefaultColorScheme.Error+"boom"+DefaultColorScheme.Reset {
		t.Errorf("Error = %q, want text in the error color", got)
	}
	if got := plain.Colorize("text", "\033[34m"); got != "text" {
		t.Errorf("Colorize without ANSI = %q, want unchanged text", got)
	}
	if got := plain.Error("boom"); got != "boom" {
		t.Errorf("Error without ANSI = %q, want unchanged text", got)
	}
}
