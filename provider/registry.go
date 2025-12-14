package provider

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]func() (Provider, error))
	mu       sync.RWMutex
)

// Register adds a provider factory to the registry.
// This is typically called from a provider package's init() function.
func Register(name string, factory func() (Provider, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Get retrieves a provider by name.
// Returns an error if the provider is not registered.
func Get(name string) (Provider, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %q (available: %v)", name, Available())
	}

	return factory()
}

// Available returns the names of all registered providers.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[name]
	return ok
}
