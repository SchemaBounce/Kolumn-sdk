package backends_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/schemabounce/kolumn/sdk/state"
	"github.com/schemabounce/kolumn/sdk/state/backends"
	"github.com/schemabounce/kolumn/sdk/types"
)

// ExampleMemoryBackend demonstrates basic usage of the memory backend
func ExampleMemoryBackend() {
	// Create a memory backend (useful for testing)
	backend := backends.NewMemoryBackend()
	ctx := context.Background()

	// Create a sample state
	state := &types.UniversalState{
		Version:          1,
		TerraformVersion: "1.0.0",
		Serial:           1,
		Lineage:          "example-lineage-123",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Resources:        []types.UniversalResource{},
		Providers:        make(map[string]types.ProviderState),
		Dependencies:     []types.Dependency{},
		Checksums:        make(map[string]string),
	}

	// Store the state
	err := backend.PutState(ctx, "my-project", state)
	if err != nil {
		log.Printf("Error storing state: %v", err)
		return
	}

	// Retrieve the state
	retrievedState, err := backend.GetState(ctx, "my-project")
	if err != nil {
		log.Printf("Error retrieving state: %v", err)
		return
	}

	// List all states
	states, err := backend.ListStates(ctx)
	if err != nil {
		log.Printf("Error listing states: %v", err)
		return
	}

	fmt.Printf("Stored lineage: %s\n", retrievedState.Lineage)
	fmt.Printf("Available states: %v\n", states)

	// Output:
	// Stored lineage: example-lineage-123
	// Available states: [my-project]
}

// ExampleBackendFactory demonstrates using the factory to create backends
func ExampleBackendFactory() {
	factory := backends.NewBackendFactory()
	ctx := context.Background()

	// List available backend types
	availableTypes := factory.ListAvailableBackends()
	fmt.Printf("Available backend types: %v\n", availableTypes)

	// Create a memory backend using the factory
	memoryBackend, err := factory.CreateBackend(backends.BackendTypeMemory)
	if err != nil {
		log.Printf("Error creating backend: %v", err)
		return
	}

	// Use the backend
	state := &types.UniversalState{
		Lineage: "factory-example",
		Serial:  1,
	}

	err = memoryBackend.PutState(ctx, "test", state)
	if err != nil {
		log.Printf("Error storing state: %v", err)
		return
	}

	fmt.Println("Successfully created and used backend from factory")

	// Output:
	// Available backend types: [memory local postgres s3]
	// Successfully created and used backend from factory
}

// Example_localBackendConfiguration demonstrates configuring a local backend
func Example_localBackendConfiguration() {
	factory := backends.NewBackendFactory()
	ctx := context.Background()

	// Configure local backend
	config := map[string]interface{}{
		"path":         "/tmp/my-project.klstate",
		"backup_dir":   "/tmp/backups",
		"backup_count": 5,
		"permissions":  0644,
	}

	backend, err := factory.CreateAndConfigureBackend(ctx, backends.BackendTypeLocal, config)
	if err != nil {
		log.Printf("Error creating local backend: %v", err)
		return
	}

	fmt.Printf("Successfully configured local backend with path: %s\n", config["path"])

	// The backend is now ready to use
	_ = backend // Use the backend as needed

	// Output:
	// Successfully configured local backend with path: /tmp/my-project.klstate
}

// Example_postgresBackendConfiguration demonstrates configuring a PostgreSQL backend
func Example_postgresBackendConfiguration() {
	// This example shows configuration - actual connection would require a running PostgreSQL instance
	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "kolumn_state",
		"username": "kolumn_user",
		"password": "secure_password",
		"schema":   "public",
		"ssl_mode": "prefer",
	}

	// Validate the configuration
	err := backends.ValidateConfig(backends.BackendTypePostgres, config)
	if err != nil {
		log.Printf("Invalid configuration: %v", err)
		return
	}

	fmt.Println("PostgreSQL backend configuration is valid")
	fmt.Printf("Required fields: %v\n", backends.GetRequiredFields(backends.BackendTypePostgres))

	// Output:
	// PostgreSQL backend configuration is valid
	// Required fields: [database username]
}

// Example_s3BackendConfiguration demonstrates configuring an S3 backend
func Example_s3BackendConfiguration() {
	// This example shows configuration - actual usage would require AWS credentials
	config := map[string]interface{}{
		"bucket":     "my-terraform-state",
		"key_prefix": "projects/my-project",
		"region":     "us-west-2",
		"encrypt":    true,
	}

	// Validate the configuration
	err := backends.ValidateConfig(backends.BackendTypeS3, config)
	if err != nil {
		log.Printf("Invalid configuration: %v", err)
		return
	}

	// Get default configuration
	defaultConfig := backends.GetDefaultConfig(backends.BackendTypeS3)
	fmt.Printf("Default S3 config: %+v\n", defaultConfig)

	fmt.Println("S3 backend configuration is valid")

	// Output:
	// Default S3 config: map[encrypt:true max_retries:3 region:us-east-1 storage_class:STANDARD]
	// S3 backend configuration is valid
}

// Example_backendWithLocking demonstrates state locking functionality
func Example_backendWithLocking() {
	backend := backends.NewMemoryBackend()
	ctx := context.Background()

	// Create lock info
	lockInfo := &state.LockInfo{
		ID:        "lock-12345",
		Path:      "my-project",
		Who:       "user@example.com",
		Version:   "1.0.0",
		Created:   time.Now().Format(time.RFC3339),
		Reason:    "Running terraform apply",
		Operation: "apply",
	}

	// Acquire lock
	lockID, err := backend.Lock(ctx, lockInfo)
	if err != nil {
		log.Printf("Error acquiring lock: %v", err)
		return
	}

	fmt.Printf("Acquired lock with ID: %s\n", lockID)

	// Try to acquire again (should fail)
	_, err = backend.Lock(ctx, lockInfo)
	if err != nil {
		fmt.Printf("Second lock attempt failed as expected: %v\n", err)
	}

	// Release lock
	err = backend.Unlock(ctx, lockID, lockInfo)
	if err != nil {
		log.Printf("Error releasing lock: %v", err)
		return
	}

	fmt.Println("Successfully released lock")

	// Output:
	// Acquired lock with ID: lock-12345
	// Second lock attempt failed as expected: state is already locked by user@example.com (ID: lock-12345)
	// Successfully released lock
}

// Example_backendTypeValidation demonstrates backend type parsing and validation
func Example_backendTypeValidation() {
	// Parse various backend type strings
	validInputs := []string{"memory", "local", "postgres", "s3", "postgresql", "aws"}

	for _, input := range validInputs {
		backendType, err := backends.ParseBackendType(input)
		if err != nil {
			log.Printf("Error parsing '%s': %v", input, err)
			continue
		}

		fmt.Printf("'%s' -> %s\n", input, backendType)
	}

	// Test validation
	for _, bt := range []backends.BackendType{"memory", "local", "postgres", "s3"} {
		err := bt.Validate()
		if err != nil {
			log.Printf("Invalid backend type %s: %v", bt, err)
		} else {
			fmt.Printf("Backend type %s is valid\n", bt)
		}
	}

	// Output:
	// 'memory' -> memory
	// 'local' -> local
	// 'postgres' -> postgres
	// 's3' -> s3
	// 'postgresql' -> postgres
	// 'aws' -> s3
	// Backend type memory is valid
	// Backend type local is valid
	// Backend type postgres is valid
	// Backend type s3 is valid
}
