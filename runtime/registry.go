package runtime

import (
	"context"
	"fmt"
	"sync"
)

// Factory creates a runtime implementation.
type Factory func() (Runtime, error)

var (
	registryMu sync.RWMutex
	runtimeRegistry = make(map[string]Factory)
)

// Register adds a provider factory to the runtime registry.
func Register(provider string, factory Factory) error {
	if provider == "" {
		return fmt.Errorf("provider id cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("factory for provider %q cannot be nil", provider)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	_, exists := runtimeRegistry[provider]
	if exists {
		return fmt.Errorf("provider %q already registered", provider)
	}

	runtimeRegistry[provider] = factory
	return nil
}

// MustRegister panics on registration error.
func MustRegister(provider string, factory Factory) {
	if err := Register(provider, factory); err != nil {
		panic(err)
	}
}

// Lookup retrieves a runtime instance for the given provider.
func Lookup(ctx context.Context, provider string) (Runtime, error) {
	registryMu.RLock()
	factory, exists := runtimeRegistry[provider]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %q not found", provider)
	}

	runtime, err := factory()
	if err != nil {
		return nil, fmt.Errorf("provider %q factory failed: %w", provider, err)
	}

	return runtime, nil
}

// List returns all registered provider identifiers.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	providers := make([]string, 0, len(runtimeRegistry))
	for id := range runtimeRegistry {
		providers = append(providers, id)
	}
	return providers
}

// Clear removes all registered providers (primarily for testing).
func Clear() {
	registryMu.Lock()
	defer registryMu.Unlock()

	runtimeRegistry = make(map[string]Factory)
}
