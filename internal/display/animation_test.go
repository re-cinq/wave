package display

import (
	"sync"
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	tests := []struct {
		name     string
		animType AnimationType
	}{
		{"dots", AnimationDots},
		{"spinner", AnimationSpinner},
		{"line", AnimationLine},
		{"bars", AnimationBars},
		{"clock", AnimationClock},
		{"bouncing bar", AnimationBouncingBar},
		{"unknown", AnimationType("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewSpinner(tt.animType)
			if spinner == nil {
				t.Fatal("NewSpinner returned nil")
			}
			if len(spinner.frames) == 0 {
				t.Error("Spinner should have frames")
			}
		})
	}
}

func TestSpinner_StartStop(t *testing.T) {
	spinner := NewSpinner(AnimationSpinner)

	// Initially not running
	if spinner.IsRunning() {
		t.Error("Spinner should not be running initially")
	}

	// Start
	spinner.Start()
	if !spinner.IsRunning() {
		t.Error("Spinner should be running after Start()")
	}

	// Start again (should be idempotent)
	spinner.Start()
	if !spinner.IsRunning() {
		t.Error("Spinner should still be running after second Start()")
	}

	// Stop
	spinner.Stop()
	if spinner.IsRunning() {
		t.Error("Spinner should not be running after Stop()")
	}

	// Stop again (should be idempotent)
	spinner.Stop()
	if spinner.IsRunning() {
		t.Error("Spinner should still not be running after second Stop()")
	}
}

func TestSpinner_Current(t *testing.T) {
	spinner := NewSpinner(AnimationSpinner)

	// Get current frame before starting
	frame := spinner.Current()
	if frame == "" {
		t.Error("Current() should return a frame even when not running")
	}

	// Start and verify frames are accessible
	spinner.Start()
	defer spinner.Stop()

	frame = spinner.Current()
	if frame == "" {
		t.Error("Current() should return a frame when running")
	}
}

func TestSpinner_FrameAdvancement(t *testing.T) {
	spinner := NewSpinner(AnimationDots)
	spinner.Start()
	defer spinner.Stop()

	// Wait for animation to potentially advance
	time.Sleep(100 * time.Millisecond)

	frame := spinner.Current()
	if frame == "" {
		t.Error("Should have a current frame")
	}
}

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

func TestSpinner_ConcurrentAccess(t *testing.T) {
	spinner := NewSpinner(AnimationSpinner)
	spinner.Start()
	defer spinner.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = spinner.Current()
				_ = spinner.IsRunning()
			}
		}()
	}
	wg.Wait()
}

func TestGetAnimationFrames(t *testing.T) {
	charSet := GetUnicodeCharSet()

	tests := []struct {
		animType AnimationType
		minLen   int
	}{
		{AnimationDots, 4},
		{AnimationSpinner, 4},
		{AnimationLine, 4},
		{AnimationBars, 5},
		{AnimationClock, 4},
		{AnimationBouncingBar, 6},
		{AnimationType("unknown"), 4},
	}

	for _, tt := range tests {
		t.Run(string(tt.animType), func(t *testing.T) {
			frames := getAnimationFrames(tt.animType, charSet)
			if len(frames) < tt.minLen {
				t.Errorf("Expected at least %d frames, got %d", tt.minLen, len(frames))
			}
			for i, frame := range frames {
				if frame == "" {
					t.Errorf("Frame %d should not be empty", i)
				}
			}
		})
	}
}
