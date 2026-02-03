package display

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// AnimatedCounter provides "numbers going up" animations for metrics.
// It smoothly animates from a start value to a target value over a specified duration.
type AnimatedCounter struct {
	mu         sync.RWMutex
	current    float64
	target     float64
	start      float64
	duration   time.Duration
	startTime  time.Time
	running    bool
	stopChan   chan struct{}
	updateFunc func(float64) // Callback for each update
	ticker     *time.Ticker
	precision  int    // Decimal precision for display
	suffix     string // Optional suffix (e.g., "k", "M", "tokens")
	easing     EasingFunction
	onComplete func() // Callback when animation completes
}

// EasingFunction defines a function that maps progress (0-1) to eased progress (0-1).
type EasingFunction func(t float64) float64

// Easing functions for smooth animations

// EaseLinear provides linear interpolation (no easing).
func EaseLinear(t float64) float64 {
	return t
}

// EaseOutQuad provides quadratic ease-out (fast start, slow end).
func EaseOutQuad(t float64) float64 {
	return -t * (t - 2)
}

// EaseInOutCubic provides cubic ease-in-out (smooth acceleration and deceleration).
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}

// EaseOutExpo provides exponential ease-out (very fast start, very slow end).
func EaseOutExpo(t float64) float64 {
	if t >= 1.0 {
		return 1.0
	}
	return 1 - math.Pow(2, -10*t)
}

// NewAnimatedCounter creates a new animated counter.
func NewAnimatedCounter(initialValue float64, duration time.Duration) *AnimatedCounter {
	return &AnimatedCounter{
		current:   initialValue,
		target:    initialValue,
		start:     initialValue,
		duration:  duration,
		running:   false,
		precision: 0,
		suffix:    "",
		easing:    EaseOutQuad, // Default to ease-out-quad for smooth deceleration
	}
}

// SetTarget sets a new target value and starts the animation.
func (ac *AnimatedCounter) SetTarget(target float64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.start = ac.current
	ac.target = target
	ac.startTime = time.Now()

	if !ac.running {
		ac.startAnimation()
	}
}

// SetTargetWithDuration sets a new target value with custom duration.
func (ac *AnimatedCounter) SetTargetWithDuration(target float64, duration time.Duration) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.start = ac.current
	ac.target = target
	ac.duration = duration
	ac.startTime = time.Now()

	if !ac.running {
		ac.startAnimation()
	}
}

// SetPrecision sets the number of decimal places to display.
func (ac *AnimatedCounter) SetPrecision(precision int) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.precision = precision
}

// SetSuffix sets the suffix to append to the value (e.g., "k", "M", " tokens").
func (ac *AnimatedCounter) SetSuffix(suffix string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.suffix = suffix
}

// SetEasing sets the easing function for the animation.
func (ac *AnimatedCounter) SetEasing(easing EasingFunction) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.easing = easing
}

// SetUpdateCallback sets a callback function that is called on each update.
func (ac *AnimatedCounter) SetUpdateCallback(fn func(float64)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.updateFunc = fn
}

// SetOnComplete sets a callback function that is called when animation completes.
func (ac *AnimatedCounter) SetOnComplete(fn func()) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.onComplete = fn
}

// startAnimation begins the animation loop (must be called with lock held).
func (ac *AnimatedCounter) startAnimation() {
	ac.running = true
	ac.stopChan = make(chan struct{})
	ac.ticker = time.NewTicker(16 * time.Millisecond) // ~60 FPS

	go ac.animate()
}

// Stop halts the animation and sets the counter to the target value.
func (ac *AnimatedCounter) Stop() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.running {
		return
	}

	ac.running = false
	close(ac.stopChan)
	if ac.ticker != nil {
		ac.ticker.Stop()
	}

	ac.current = ac.target
}

// animate runs the animation loop.
func (ac *AnimatedCounter) animate() {
	for {
		select {
		case <-ac.stopChan:
			return
		case <-ac.ticker.C:
			ac.mu.Lock()
			elapsed := time.Since(ac.startTime)

			// Calculate progress (0-1)
			progress := float64(elapsed) / float64(ac.duration)
			if progress >= 1.0 {
				// Animation complete
				ac.current = ac.target
				ac.running = false

				// Call update callback
				if ac.updateFunc != nil {
					ac.updateFunc(ac.current)
				}

				// Call completion callback
				onComplete := ac.onComplete
				ac.mu.Unlock()

				if ac.ticker != nil {
					ac.ticker.Stop()
				}

				if onComplete != nil {
					onComplete()
				}
				return
			}

			// Apply easing function
			easedProgress := ac.easing(progress)

			// Interpolate between start and target
			ac.current = ac.start + (ac.target-ac.start)*easedProgress

			// Call update callback
			if ac.updateFunc != nil {
				ac.updateFunc(ac.current)
			}

			ac.mu.Unlock()
		}
	}
}

// Current returns the current animated value.
func (ac *AnimatedCounter) Current() float64 {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.current
}

// Target returns the target value.
func (ac *AnimatedCounter) Target() float64 {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.target
}

// IsAnimating returns whether the counter is currently animating.
func (ac *AnimatedCounter) IsAnimating() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.running
}

// Format returns the formatted string representation of the current value.
func (ac *AnimatedCounter) Format() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	formatStr := fmt.Sprintf("%%.%df%%s", ac.precision)
	return fmt.Sprintf(formatStr, ac.current, ac.suffix)
}

// FormatWithCommas returns the formatted string with thousand separators.
func (ac *AnimatedCounter) FormatWithCommas() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	value := int64(ac.current)
	return formatNumberWithCommas(value) + ac.suffix
}

// Jump immediately sets the counter to the target value without animation.
func (ac *AnimatedCounter) Jump(value float64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.running {
		ac.running = false
		if ac.ticker != nil {
			ac.ticker.Stop()
		}
		close(ac.stopChan)
	}

	ac.current = value
	ac.target = value
	ac.start = value
}

// MultiCounter manages multiple animated counters for different metrics.
type MultiCounter struct {
	mu       sync.RWMutex
	counters map[string]*AnimatedCounter
}

// NewMultiCounter creates a new multi-counter manager.
func NewMultiCounter() *MultiCounter {
	return &MultiCounter{
		counters: make(map[string]*AnimatedCounter),
	}
}

// Add registers a new counter with the given ID.
func (mc *MultiCounter) Add(id string, initialValue float64, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.counters[id]; !exists {
		mc.counters[id] = NewAnimatedCounter(initialValue, duration)
	}
}

// SetTarget sets a new target for a specific counter.
func (mc *MultiCounter) SetTarget(id string, target float64) {
	mc.mu.RLock()
	counter, exists := mc.counters[id]
	mc.mu.RUnlock()

	if exists {
		counter.SetTarget(target)
	}
}

// Get retrieves a counter by ID.
func (mc *MultiCounter) Get(id string) *AnimatedCounter {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.counters[id]
}

// GetValue returns the current value of a counter.
func (mc *MultiCounter) GetValue(id string) float64 {
	mc.mu.RLock()
	counter, exists := mc.counters[id]
	mc.mu.RUnlock()

	if exists {
		return counter.Current()
	}
	return 0
}

// Format returns the formatted value of a counter.
func (mc *MultiCounter) Format(id string) string {
	mc.mu.RLock()
	counter, exists := mc.counters[id]
	mc.mu.RUnlock()

	if exists {
		return counter.Format()
	}
	return "0"
}

// StopAll stops all counters.
func (mc *MultiCounter) StopAll() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, counter := range mc.counters {
		counter.Stop()
	}
}

// PerformanceCounter provides animated counters for common performance metrics.
type PerformanceCounter struct {
	Tokens    *AnimatedCounter
	Files     *AnimatedCounter
	Artifacts *AnimatedCounter
	Duration  *AnimatedCounter
	codec     *ANSICodec
}

// NewPerformanceCounter creates a new performance metrics counter.
func NewPerformanceCounter() *PerformanceCounter {
	return &PerformanceCounter{
		Tokens:    NewAnimatedCounter(0, 500*time.Millisecond),
		Files:     NewAnimatedCounter(0, 500*time.Millisecond),
		Artifacts: NewAnimatedCounter(0, 500*time.Millisecond),
		Duration:  NewAnimatedCounter(0, 500*time.Millisecond),
		codec:     NewANSICodec(),
	}
}

// UpdateTokens animates the token count to a new value.
func (pc *PerformanceCounter) UpdateTokens(tokens int) {
	pc.Tokens.SetTarget(float64(tokens))
}

// UpdateFiles animates the file count to a new value.
func (pc *PerformanceCounter) UpdateFiles(files int) {
	pc.Files.SetTarget(float64(files))
}

// UpdateArtifacts animates the artifact count to a new value.
func (pc *PerformanceCounter) UpdateArtifacts(artifacts int) {
	pc.Artifacts.SetTarget(float64(artifacts))
}

// UpdateDuration animates the duration to a new value (in milliseconds).
func (pc *PerformanceCounter) UpdateDuration(durationMs int64) {
	pc.Duration.SetTarget(float64(durationMs))
}

// Render returns a formatted string showing all counters.
func (pc *PerformanceCounter) Render() string {
	tokens := int(pc.Tokens.Current())
	files := int(pc.Files.Current())
	artifacts := int(pc.Artifacts.Current())
	durationMs := int64(pc.Duration.Current())

	var parts []string

	if tokens > 0 {
		tokenStr := FormatTokenCount(tokens)
		parts = append(parts, pc.codec.Primary(fmt.Sprintf("%s tokens", tokenStr)))
	}

	if files > 0 {
		fileStr := FormatFileCount(files)
		parts = append(parts, pc.codec.Muted(fileStr))
	}

	if artifacts > 0 {
		artifactStr := fmt.Sprintf("%d artifact", artifacts)
		if artifacts > 1 {
			artifactStr += "s"
		}
		parts = append(parts, pc.codec.Muted(artifactStr))
	}

	if durationMs > 0 {
		durationStr := FormatDuration(durationMs)
		parts = append(parts, pc.codec.Muted(durationStr))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s", joinParts(parts, " • "))
}

// RenderCompact returns a compact one-line representation.
func (pc *PerformanceCounter) RenderCompact() string {
	tokens := int(pc.Tokens.Current())
	files := int(pc.Files.Current())
	artifacts := int(pc.Artifacts.Current())

	return fmt.Sprintf("%s | %d files | %d artifacts",
		FormatTokenCount(tokens), files, artifacts)
}

// Stop stops all animations.
func (pc *PerformanceCounter) Stop() {
	pc.Tokens.Stop()
	pc.Files.Stop()
	pc.Artifacts.Stop()
	pc.Duration.Stop()
}

// TokenBurnRateCounter provides an animated display of token burn rate.
type TokenBurnRateCounter struct {
	mu          sync.RWMutex
	burnRate    *AnimatedCounter // tokens per second
	totalTokens int
	totalTimeMs int64
	codec       *ANSICodec
}

// NewTokenBurnRateCounter creates a new token burn rate counter.
func NewTokenBurnRateCounter() *TokenBurnRateCounter {
	return &TokenBurnRateCounter{
		burnRate: NewAnimatedCounter(0, 300*time.Millisecond),
		codec:    NewANSICodec(),
	}
}

// Update recalculates and animates the burn rate based on new totals.
func (tbrc *TokenBurnRateCounter) Update(totalTokens int, totalTimeMs int64) {
	tbrc.mu.Lock()
	defer tbrc.mu.Unlock()

	tbrc.totalTokens = totalTokens
	tbrc.totalTimeMs = totalTimeMs

	if totalTimeMs > 0 {
		rate := float64(totalTokens) / (float64(totalTimeMs) / 1000.0)
		tbrc.burnRate.SetTarget(rate)
	}
}

// Current returns the current animated burn rate.
func (tbrc *TokenBurnRateCounter) Current() float64 {
	return tbrc.burnRate.Current()
}

// Render returns a formatted string showing the burn rate.
func (tbrc *TokenBurnRateCounter) Render() string {
	rate := tbrc.burnRate.Current()

	if rate < 1 {
		return tbrc.codec.Muted("< 1 token/s")
	}

	if rate < 1000 {
		return tbrc.codec.Primary(fmt.Sprintf("%.1f tokens/s", rate))
	}

	// Show in k/s for large rates
	return tbrc.codec.Primary(fmt.Sprintf("%.2fk tokens/s", rate/1000.0))
}

// Helper functions

// formatNumberWithCommas formats an integer with thousand separators.
func formatNumberWithCommas(n int64) string {
	if n < 0 {
		return "-" + formatNumberWithCommas(-n)
	}

	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result []byte
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(digit))
	}

	return string(result)
}

// joinParts joins string parts with a separator.
func joinParts(parts []string, separator string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += separator + parts[i]
	}
	return result
}
