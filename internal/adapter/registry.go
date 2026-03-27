package adapter

import "sync"

// AdapterRegistry maps adapter names to their runner implementations.
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
func (r *AdapterRegistry) Register(name string, runner AdapterRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = runner
}

// Resolve looks up the adapter runner for the given name.
// Returns the runner and true if found, or nil and false if not registered.
func (r *AdapterRegistry) Resolve(name string) (AdapterRunner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runner, ok := r.adapters[name]
	return runner, ok
}

// Names returns all registered adapter names.
func (r *AdapterRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
