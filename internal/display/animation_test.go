package display

import (
	"strings"
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

func TestNewAnimationSet(t *testing.T) {
	as := NewAnimationSet()
	if as == nil {
		t.Fatal("NewAnimationSet returned nil")
	}
	if as.Loading == nil {
		t.Error("Loading spinner should not be nil")
	}
	if as.Processing == nil {
		t.Error("Processing spinner should not be nil")
	}
	if as.Thinking == nil {
		t.Error("Thinking spinner should not be nil")
	}
	if as.Validating == nil {
		t.Error("Validating spinner should not be nil")
	}
}

func TestAnimationSet_StopAll(t *testing.T) {
	as := NewAnimationSet()

	// Start all
	as.Loading.Start()
	as.Processing.Start()
	as.Thinking.Start()
	as.Validating.Start()

	// Verify all running
	if !as.Loading.IsRunning() {
		t.Error("Loading should be running")
	}

	// Stop all
	as.StopAll()

	// Verify all stopped
	if as.Loading.IsRunning() {
		t.Error("Loading should be stopped")
	}
	if as.Processing.IsRunning() {
		t.Error("Processing should be stopped")
	}
	if as.Thinking.IsRunning() {
		t.Error("Thinking should be stopped")
	}
	if as.Validating.IsRunning() {
		t.Error("Validating should be stopped")
	}
}

func TestNewProgressAnimation(t *testing.T) {
	pa := NewProgressAnimation("Loading", 100, 20)
	if pa == nil {
		t.Fatal("NewProgressAnimation returned nil")
	}
	if pa.label != "Loading" {
		t.Errorf("Label = %q, want %q", pa.label, "Loading")
	}
	if pa.progressBar == nil {
		t.Error("ProgressBar should not be nil")
	}
	if pa.spinner == nil {
		t.Error("Spinner should not be nil")
	}
}

func TestProgressAnimation_StartStop(t *testing.T) {
	pa := NewProgressAnimation("Test", 100, 20)

	pa.Start()
	if !pa.spinner.IsRunning() {
		t.Error("Spinner should be running after Start()")
	}

	pa.Stop()
	if pa.spinner.IsRunning() {
		t.Error("Spinner should not be running after Stop()")
	}
}

func TestProgressAnimation_SetProgress(t *testing.T) {
	pa := NewProgressAnimation("Test", 100, 20)

	pa.SetProgress(50)
	if pa.progressBar.current != 50 {
		t.Errorf("Progress = %d, want 50", pa.progressBar.current)
	}

	pa.SetProgress(100)
	if pa.progressBar.current != 100 {
		t.Errorf("Progress = %d, want 100", pa.progressBar.current)
	}
}

func TestProgressAnimation_Render(t *testing.T) {
	pa := NewProgressAnimation("Loading", 100, 20)
	pa.Start()
	defer pa.Stop()

	pa.SetProgress(50)
	result := pa.Render()

	if !strings.Contains(result, "Loading") {
		t.Error("Render should contain label")
	}
	if !strings.Contains(result, "[") {
		t.Error("Render should contain progress bar bracket")
	}
}

func TestProgressAnimation_ConcurrentAccess(t *testing.T) {
	pa := NewProgressAnimation("Test", 100, 20)
	pa.Start()
	defer pa.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				pa.SetProgress(n * 10)
				_ = pa.Render()
			}
		}(i)
	}
	wg.Wait()
}

func TestNewMultiSpinner(t *testing.T) {
	ms := NewMultiSpinner()
	if ms == nil {
		t.Fatal("NewMultiSpinner returned nil")
	}
	if ms.spinners == nil {
		t.Error("spinners map should not be nil")
	}
}

func TestMultiSpinner_Add(t *testing.T) {
	ms := NewMultiSpinner()

	ms.Add("spinner1", AnimationDots)
	ms.Add("spinner2", AnimationSpinner)

	// Add same ID again (should not duplicate)
	ms.Add("spinner1", AnimationBars)

	if len(ms.spinners) != 2 {
		t.Errorf("Expected 2 spinners, got %d", len(ms.spinners))
	}
}

func TestMultiSpinner_StartStop(t *testing.T) {
	ms := NewMultiSpinner()
	ms.Add("test", AnimationSpinner)

	ms.Start("test")
	if spinner := ms.Get("test"); spinner != nil && !spinner.IsRunning() {
		t.Error("Spinner should be running after Start()")
	}

	ms.Stop("test")
	if spinner := ms.Get("test"); spinner != nil && spinner.IsRunning() {
		t.Error("Spinner should not be running after Stop()")
	}

	// Start/Stop non-existent spinner (should not panic)
	ms.Start("nonexistent")
	ms.Stop("nonexistent")
}

func TestMultiSpinner_StopAll(t *testing.T) {
	ms := NewMultiSpinner()
	ms.Add("s1", AnimationDots)
	ms.Add("s2", AnimationSpinner)
	ms.Add("s3", AnimationBars)

	ms.Start("s1")
	ms.Start("s2")
	ms.Start("s3")

	ms.StopAll()

	for id := range ms.spinners {
		if spinner := ms.Get(id); spinner != nil && spinner.IsRunning() {
			t.Errorf("Spinner %s should be stopped", id)
		}
	}
}

func TestMultiSpinner_Get(t *testing.T) {
	ms := NewMultiSpinner()
	ms.Add("exists", AnimationSpinner)

	spinner := ms.Get("exists")
	if spinner == nil {
		t.Error("Get should return spinner for existing ID")
	}

	spinner = ms.Get("nonexistent")
	if spinner != nil {
		t.Error("Get should return nil for non-existent ID")
	}
}

func TestMultiSpinner_Current(t *testing.T) {
	ms := NewMultiSpinner()
	ms.Add("test", AnimationSpinner)
	ms.Start("test")
	defer ms.Stop("test")

	frame := ms.Current("test")
	if frame == "" {
		t.Error("Current should return non-empty frame")
	}

	frame = ms.Current("nonexistent")
	if frame != "" {
		t.Error("Current should return empty for non-existent ID")
	}
}

func TestMultiSpinner_ConcurrentAccess(t *testing.T) {
	ms := NewMultiSpinner()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := string(rune('a' + n))
			ms.Add(id, AnimationSpinner)
			ms.Start(id)
			for j := 0; j < 50; j++ {
				_ = ms.Current(id)
				_ = ms.Get(id)
			}
			ms.Stop(id)
		}(i)
	}
	wg.Wait()
}

func TestNewLoadingIndicator(t *testing.T) {
	li := NewLoadingIndicator("Loading...")
	if li == nil {
		t.Fatal("NewLoadingIndicator returned nil")
	}
	if li.message != "Loading..." {
		t.Errorf("message = %q, want %q", li.message, "Loading...")
	}
	if li.spinner == nil {
		t.Error("spinner should not be nil")
	}
}

func TestLoadingIndicator_StartStop(t *testing.T) {
	li := NewLoadingIndicator("Test")

	li.Start()
	if !li.spinner.IsRunning() {
		t.Error("Spinner should be running after Start()")
	}

	li.Stop()
	if li.spinner.IsRunning() {
		t.Error("Spinner should not be running after Stop()")
	}
}

func TestLoadingIndicator_SetMessage(t *testing.T) {
	li := NewLoadingIndicator("Initial")

	li.SetMessage("Updated")
	if li.message != "Updated" {
		t.Errorf("message = %q, want %q", li.message, "Updated")
	}
}

func TestLoadingIndicator_Render(t *testing.T) {
	li := NewLoadingIndicator("Loading data")
	li.Start()
	defer li.Stop()

	result := li.Render()
	if !strings.Contains(result, "Loading data") {
		t.Error("Render should contain message")
	}
}

func TestLoadingIndicator_ConcurrentAccess(t *testing.T) {
	li := NewLoadingIndicator("Test")
	li.Start()
	defer li.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				li.SetMessage("Message " + string(rune('0'+n)))
				_ = li.Render()
			}
		}(i)
	}
	wg.Wait()
}
