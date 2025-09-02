package backends

import (
	"context"
	"fmt"
	"strings"

	"github.com/schemabounce/kolumn/sdk/state"
)

// BackendType represents the type of state backend
type BackendType string

const (
	BackendTypeMemory   BackendType = "memory"
	BackendTypeLocal    BackendType = "local"
	BackendTypePostgres BackendType = "postgres"
	BackendTypeS3       BackendType = "s3"
)

// BackendFactory provides methods for creating and configuring state backends
type BackendFactory struct {
	// Available backend creators
	creators map[BackendType]BackendCreator
}

// BackendCreator is a function that creates a new backend instance
type BackendCreator func() state.StateBackend

// NewBackendFactory creates a new backend factory with default backends registered
func NewBackendFactory() *BackendFactory {
	factory := &BackendFactory{
		creators: make(map[BackendType]BackendCreator),
	}

	// Register default backends
	factory.RegisterBackend(BackendTypeMemory, func() state.StateBackend {
		return NewMemoryBackend()
	})

	factory.RegisterBackend(BackendTypeLocal, func() state.StateBackend {
		return NewLocalBackend()
	})

	factory.RegisterBackend(BackendTypePostgres, func() state.StateBackend {
		return NewPostgresBackend()
	})

	factory.RegisterBackend(BackendTypeS3, func() state.StateBackend {
		return NewS3Backend()
	})

	return factory
}

// RegisterBackend registers a new backend type with its creator function
func (f *BackendFactory) RegisterBackend(backendType BackendType, creator BackendCreator) {
	if f.creators == nil {
		f.creators = make(map[BackendType]BackendCreator)
	}
	f.creators[backendType] = creator
}

// CreateBackend creates a new backend instance of the specified type
func (f *BackendFactory) CreateBackend(backendType BackendType) (state.StateBackend, error) {
	creator, exists := f.creators[backendType]
	if !exists {
		return nil, fmt.Errorf("unknown backend type: %s", backendType)
	}

	backend := creator()
	if backend == nil {
		return nil, fmt.Errorf("backend creator returned nil for type: %s", backendType)
	}

	return backend, nil
}

// CreateAndConfigureBackend creates a new backend and configures it
func (f *BackendFactory) CreateAndConfigureBackend(ctx context.Context, backendType BackendType, config map[string]interface{}) (state.StateBackend, error) {
	backend, err := f.CreateBackend(backendType)
	if err != nil {
		return nil, err
	}

	// Configure the backend if it has a Configure method
	if configurableBackend, ok := backend.(ConfigurableBackend); ok {
		if err := configurableBackend.Configure(ctx, config); err != nil {
			return nil, fmt.Errorf("failed to configure %s backend: %w", backendType, err)
		}
	}

	return backend, nil
}

// ListAvailableBackends returns a list of all registered backend types
func (f *BackendFactory) ListAvailableBackends() []BackendType {
	var types []BackendType
	for backendType := range f.creators {
		types = append(types, backendType)
	}
	return types
}

// IsBackendAvailable checks if a backend type is available
func (f *BackendFactory) IsBackendAvailable(backendType BackendType) bool {
	_, exists := f.creators[backendType]
	return exists
}

// ConfigurableBackend represents a backend that can be configured
type ConfigurableBackend interface {
	Configure(ctx context.Context, config map[string]interface{}) error
}

// ParseBackendType parses a string into a BackendType
func ParseBackendType(s string) (BackendType, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "memory":
		return BackendTypeMemory, nil
	case "local", "file", "filesystem":
		return BackendTypeLocal, nil
	case "postgres", "postgresql", "pg":
		return BackendTypePostgres, nil
	case "s3", "aws", "amazon":
		return BackendTypeS3, nil
	default:
		return "", fmt.Errorf("unknown backend type: %s", s)
	}
}

// String returns the string representation of a BackendType
func (bt BackendType) String() string {
	return string(bt)
}

// Validate validates a BackendType
func (bt BackendType) Validate() error {
	switch bt {
	case BackendTypeMemory, BackendTypeLocal, BackendTypePostgres, BackendTypeS3:
		return nil
	default:
		return fmt.Errorf("invalid backend type: %s", bt)
	}
}

// DefaultBackendFactory returns a default factory with all standard backends registered
var DefaultBackendFactory = NewBackendFactory()

// CreateBackend creates a backend using the default factory
func CreateBackend(backendType BackendType) (state.StateBackend, error) {
	return DefaultBackendFactory.CreateBackend(backendType)
}

// CreateAndConfigureBackend creates and configures a backend using the default factory
func CreateAndConfigureBackend(ctx context.Context, backendType BackendType, config map[string]interface{}) (state.StateBackend, error) {
	return DefaultBackendFactory.CreateAndConfigureBackend(ctx, backendType, config)
}

// BackendConfig represents a backend configuration
type BackendConfig struct {
	Type   BackendType            `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// Validate validates a backend configuration
func (bc *BackendConfig) Validate() error {
	if err := bc.Type.Validate(); err != nil {
		return fmt.Errorf("invalid backend type: %w", err)
	}

	if bc.Config == nil {
		bc.Config = make(map[string]interface{})
	}

	return nil
}

// CreateBackendFromConfig creates a backend from a configuration
func CreateBackendFromConfig(ctx context.Context, config *BackendConfig) (state.StateBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("backend config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid backend config: %w", err)
	}

	return CreateAndConfigureBackend(ctx, config.Type, config.Config)
}

// GetDefaultConfig returns default configuration for a backend type
func GetDefaultConfig(backendType BackendType) map[string]interface{} {
	switch backendType {
	case BackendTypeMemory:
		return map[string]interface{}{
			// Memory backend needs no configuration
		}

	case BackendTypeLocal:
		return map[string]interface{}{
			"path":         "kolumn.klstate",
			"backup_count": 10,
			"permissions":  0644,
		}

	case BackendTypePostgres:
		return map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"schema":   "public",
			"ssl_mode": "prefer",
		}

	case BackendTypeS3:
		return map[string]interface{}{
			"region":        "us-east-1",
			"max_retries":   3,
			"storage_class": "STANDARD",
			"encrypt":       true,
		}

	default:
		return map[string]interface{}{}
	}
}

// GetRequiredFields returns the required configuration fields for a backend type
func GetRequiredFields(backendType BackendType) []string {
	switch backendType {
	case BackendTypeMemory:
		return []string{} // No required fields

	case BackendTypeLocal:
		return []string{"path"}

	case BackendTypePostgres:
		return []string{"database", "username"}

	case BackendTypeS3:
		return []string{"bucket"}

	default:
		return []string{}
	}
}

// ValidateConfig validates configuration for a specific backend type
func ValidateConfig(backendType BackendType, config map[string]interface{}) error {
	requiredFields := GetRequiredFields(backendType)

	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("required field '%s' missing for %s backend", field, backendType)
		}
	}

	// Additional validation per backend type
	switch backendType {
	case BackendTypePostgres:
		if port, exists := config["port"]; exists {
			switch v := port.(type) {
			case float64:
				if v <= 0 || v > 65535 {
					return fmt.Errorf("invalid port number: %v", v)
				}
			case int:
				if v <= 0 || v > 65535 {
					return fmt.Errorf("invalid port number: %d", v)
				}
			default:
				return fmt.Errorf("port must be a number")
			}
		}

	case BackendTypeS3:
		if region, exists := config["region"]; exists {
			if regionStr, ok := region.(string); ok && regionStr == "" {
				return fmt.Errorf("region cannot be empty")
			}
		}
	}

	return nil
}
