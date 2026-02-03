package display

import (
	"sync"
	"time"
)

// Spinner provides animated loading indicators.
type Spinner struct {
	mu        sync.Mutex
	frames    []string
	current   int
	animType  AnimationType
	running   bool
	stopChan  chan struct{}
	ticker    *time.Ticker
	charSet   UnicodeCharSet
}

// NewSpinner creates a new spinner with the specified animation type.
func NewSpinner(animType AnimationType) *Spinner {
	charSet := GetUnicodeCharSet()
	frames := getAnimationFrames(animType, charSet)

	return &Spinner{
		frames:   frames,
		current:  0,
		animType: animType,
		running:  false,
		charSet:  charSet,
	}
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
	s.ticker = time.NewTicker(100 * time.Millisecond)

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

// getAnimationFrames returns the animation frames for a given type.
func getAnimationFrames(animType AnimationType, charSet UnicodeCharSet) []string {
	switch animType {
	case AnimationDots:
		return []string{".", "..", "...", "...."}

	case AnimationSpinner:
		// Use Unicode spinner from charset
		if charSet.Spinner[0] != "" {
			return []string{
				charSet.Spinner[0],
				charSet.Spinner[1],
				charSet.Spinner[2],
				charSet.Spinner[3],
			}
		}
		// ASCII fallback
		return []string{"|", "/", "-", "\\"}

	case AnimationLine:
		if DetectUnicodeSupport() {
			return []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
		}
		return []string{"-", "\\", "|", "/"}

	case AnimationBars:
		if DetectUnicodeSupport() {
			return []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
		}
		return []string{"=", "==", "===", "====", "====="}

	case AnimationClock:
		if DetectUnicodeSupport() {
			return []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
		}
		return []string{"|", "/", "-", "\\"}

	case AnimationBouncingBar:
		if DetectUnicodeSupport() {
			return []string{
				"[=   ]",
				"[ =  ]",
				"[  = ]",
				"[   =]",
				"[  = ]",
				"[ =  ]",
			}
		}
		return []string{
			"[>   ]",
			"[ >  ]",
			"[  > ]",
			"[   >]",
			"[  < ]",
			"[ <  ]",
		}

	default:
		return []string{"|", "/", "-", "\\"}
	}
}

// AnimationSet provides a collection of commonly used animations.
type AnimationSet struct {
	Loading    *Spinner
	Processing *Spinner
	Thinking   *Spinner
	Validating *Spinner
}

// NewAnimationSet creates a set of themed spinners.
func NewAnimationSet() *AnimationSet {
	return &AnimationSet{
		Loading:    NewSpinner(AnimationSpinner),
		Processing: NewSpinner(AnimationDots),
		Thinking:   NewSpinner(AnimationLine),
		Validating: NewSpinner(AnimationBars),
	}
}

// StopAll stops all spinners in the set.
func (as *AnimationSet) StopAll() {
	as.Loading.Stop()
	as.Processing.Stop()
	as.Thinking.Stop()
	as.Validating.Stop()
}

// ProgressAnimation combines a progress bar with an animated spinner.
type ProgressAnimation struct {
	mu          sync.Mutex
	progressBar *ProgressBar
	spinner     *Spinner
	label       string
	codec       *ANSICodec
}

// NewProgressAnimation creates an animated progress indicator.
func NewProgressAnimation(label string, total int, width int) *ProgressAnimation {
	return &ProgressAnimation{
		progressBar: NewProgressBar(total, width),
		spinner:     NewSpinner(AnimationSpinner),
		label:       label,
		codec:       NewANSICodec(),
	}
}

// Start begins the animation.
func (pa *ProgressAnimation) Start() {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	pa.spinner.Start()
}

// Stop halts the animation.
func (pa *ProgressAnimation) Stop() {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	pa.spinner.Stop()
}

// SetProgress updates the progress value.
func (pa *ProgressAnimation) SetProgress(current int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	pa.progressBar.SetProgress(current)
}

// Render returns the formatted animation string.
func (pa *ProgressAnimation) Render() string {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	spinnerFrame := pa.spinner.Current()
	prefix := pa.label
	if spinnerFrame != "" {
		prefix = pa.codec.Primary(spinnerFrame) + " " + prefix
	}

	pa.progressBar.SetPrefix(prefix)
	return pa.progressBar.Render()
}

// MultiSpinner manages multiple concurrent spinners.
type MultiSpinner struct {
	mu       sync.Mutex
	spinners map[string]*Spinner
}

// NewMultiSpinner creates a manager for multiple spinners.
func NewMultiSpinner() *MultiSpinner {
	return &MultiSpinner{
		spinners: make(map[string]*Spinner),
	}
}

// Add registers a new spinner with the given ID.
func (ms *MultiSpinner) Add(id string, animType AnimationType) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.spinners[id]; !exists {
		ms.spinners[id] = NewSpinner(animType)
	}
}

// Start starts a specific spinner.
func (ms *MultiSpinner) Start(id string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if spinner, exists := ms.spinners[id]; exists {
		spinner.Start()
	}
}

// Stop stops a specific spinner.
func (ms *MultiSpinner) Stop(id string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if spinner, exists := ms.spinners[id]; exists {
		spinner.Stop()
	}
}

// StopAll stops all registered spinners.
func (ms *MultiSpinner) StopAll() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, spinner := range ms.spinners {
		spinner.Stop()
	}
}

// Get retrieves a spinner by ID.
func (ms *MultiSpinner) Get(id string) *Spinner {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.spinners[id]
}

// Current returns the current frame for a specific spinner.
func (ms *MultiSpinner) Current(id string) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if spinner, exists := ms.spinners[id]; exists {
		return spinner.Current()
	}
	return ""
}

// LoadingIndicator provides a simple loading indicator with message.
type LoadingIndicator struct {
	mu      sync.Mutex
	spinner *Spinner
	message string
	codec   *ANSICodec
}

// NewLoadingIndicator creates a new loading indicator.
func NewLoadingIndicator(message string) *LoadingIndicator {
	return &LoadingIndicator{
		spinner: NewSpinner(AnimationSpinner),
		message: message,
		codec:   NewANSICodec(),
	}
}

// Start begins the loading animation.
func (li *LoadingIndicator) Start() {
	li.mu.Lock()
	defer li.mu.Unlock()
	li.spinner.Start()
}

// Stop halts the loading animation.
func (li *LoadingIndicator) Stop() {
	li.mu.Lock()
	defer li.mu.Unlock()
	li.spinner.Stop()
}

// SetMessage updates the loading message.
func (li *LoadingIndicator) SetMessage(message string) {
	li.mu.Lock()
	defer li.mu.Unlock()
	li.message = message
}

// Render returns the formatted loading indicator string.
func (li *LoadingIndicator) Render() string {
	li.mu.Lock()
	defer li.mu.Unlock()

	spinnerFrame := li.spinner.Current()
	return li.codec.Primary(spinnerFrame) + " " + li.message
}
