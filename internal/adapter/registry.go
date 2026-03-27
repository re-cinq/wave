package adapter

import (
	"sort"
	"strings"
	"sync"
)

// AdapterRegistry maps adapter names to their runner implementations.
// Names are normalized to lowercase for case-insensitive lookup.
// It is safe for concurrent use.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]AdapterRunner
}

// NewAdapterRegistry creates an empty adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]AdapterRunner),
	}
}

// Register adds or replaces an adapter runner under the given name.
// The name is normalized to lowercase.
func (r *AdapterRegistry) Register(name string, runner AdapterRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[strings.ToLower(name)] = runner
}

// Resolve looks up the adapter runner for the given name.
// The name is normalized to lowercase for case-insensitive lookup.
// Returns the runner and true if found, or nil and false if not registered.
func (r *AdapterRegistry) Resolve(name string) (AdapterRunner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runner, ok := r.adapters[strings.ToLower(name)]
	return runner, ok
}

// Names returns all registered adapter names in sorted order for deterministic
// iteration.
func (r *AdapterRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
