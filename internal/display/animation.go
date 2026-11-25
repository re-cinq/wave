package display

import (
	"sync"
	"time"
)

// Spinner provides animated loading indicators.
type Spinner struct {
	mu       sync.Mutex
	frames   []string
	current  int
	animType AnimationType
	running  bool
	stopChan chan struct{}
	ticker   *time.Ticker
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.current = 0
	s.stopChan = make(chan struct{})
	s.ticker = time.NewTicker(33 * time.Millisecond) // 30 FPS for smooth animation

	go s.animate()
}

// Stop halts the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// animate runs the animation loop.
func (s *Spinner) animate() {
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.mu.Lock()
			s.current = (s.current + 1) % len(s.frames)
			s.mu.Unlock()
		}
	}
}

// Current returns the current frame of the animation.
func (s *Spinner) Current() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.frames) == 0 {
		return ""
	}
	return s.frames[s.current]
}

// IsRunning returns whether the spinner is currently animating.
func (s *Spinner) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
