package webui

import (
	"log"
	"strings"
	"sync"
	"time"
)

// cacheEntry holds a cached value and its expiration time.
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// apiCache is a thread-safe in-memory TTL cache for API responses.
type apiCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// newAPICache creates a new cache with the given default TTL.
func newAPICache(ttl time.Duration) *apiCache {
	return &apiCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached value by key. Returns (nil, false) on miss or expiry.
// Safe to call on a nil receiver (always returns miss).
func (c *apiCache) Get(key string) (interface{}, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		log.Printf("[webui] cache miss: %s", key)
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		// Expired — clean up lazily
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		log.Printf("[webui] cache miss (expired): %s", key)
		return nil, false
	}

	log.Printf("[webui] cache hit: %s", key)
	return entry.data, true
}

// Set stores a value in the cache with the default TTL.
// Safe to call on a nil receiver (no-op).
func (c *apiCache) Set(key string, value interface{}) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// SetWithTTL stores a value in the cache with a custom TTL.
// Safe to call on a nil receiver (no-op).
func (c *apiCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes specific keys from the cache.
// Safe to call on a nil receiver (no-op).
func (c *apiCache) Invalidate(keys ...string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	for _, key := range keys {
		delete(c.entries, key)
	}
	c.mu.Unlock()
}

// InvalidatePrefix removes all cache entries whose key starts with the given prefix.
// Safe to call on a nil receiver (no-op).
func (c *apiCache) InvalidatePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}

// Clear removes all entries from the cache.
// Safe to call on a nil receiver (no-op).
func (c *apiCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}
