package display

import (
	"testing"
)

func TestSpinner_EmptyFrames(t *testing.T) {
	// Create spinner with empty frames (edge case)
	spinner := &Spinner{
		frames:   []string{},
		running:  false,
		animType: AnimationDots,
	}

	frame := spinner.Current()
	if frame != "" {
		t.Errorf("Empty frames should return empty string, got %q", frame)
	}
}
