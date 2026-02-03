package github

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter manages GitHub API rate limiting
type RateLimiter struct {
	mu        sync.RWMutex
	limit     int
	remaining int
	reset     time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limit:     5000, // Default GitHub rate limit
		remaining: 5000,
		reset:     time.Now().Add(time.Hour),
	}
}

// Update updates the rate limiter from response headers
func (rl *RateLimiter) Update(headers http.Header) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limit := headers.Get("X-RateLimit-Limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			rl.limit = val
		}
	}

	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			rl.remaining = val
		}
	}

	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			rl.reset = time.Unix(val, 0)
		}
	}
}

// Wait blocks until a request can be made without exceeding rate limits
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.RLock()
	remaining := rl.remaining
	reset := rl.reset
	rl.mu.RUnlock()

	// If we have remaining requests, proceed immediately
	if remaining > 0 {
		return nil
	}

	// Calculate wait time
	waitTime := time.Until(reset)
	if waitTime <= 0 {
		return nil
	}

	// Wait for either context cancellation or rate limit reset
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

// Status returns the current rate limit status
func (rl *RateLimiter) Status() (limit, remaining int, reset time.Time) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.limit, rl.remaining, rl.reset
}

// ResetTime returns when the rate limit will reset
func (rl *RateLimiter) ResetTime() time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.reset
}
