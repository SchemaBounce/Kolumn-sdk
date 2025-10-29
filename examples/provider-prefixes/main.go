// Example showing how to use provider-specific prefixes in Kolumn providers
package main

import (
	"fmt"

	"github.com/schemabounce/kolumn/sdk/core"
	"github.com/schemabounce/kolumn/sdk/helpers/ui"
)

// ExampleProvider shows how to integrate provider-specific prefixes
type ExampleProvider struct {
	prefix *ui.PrefixBuilder
}

// NewExampleProvider creates a new example provider with prefix support
func NewExampleProvider(displayName string) *ExampleProvider {
	return &ExampleProvider{
		prefix: ui.NewPrefixBuilder(displayName),
	}
}

// Schema returns the provider schema with display name for UI prefixes
func (p *ExampleProvider) Schema() (*core.Schema, error) {
	return &core.Schema{
		Name:        "example",
		Version:     "1.0.0",
		Protocol:    "1.0",
		Type:        "database",
		Description: "Example provider showing prefix integration",
		DisplayName: p.prefix.ProviderName(), // This enables provider-specific prefixes

		SupportedFunctions: []string{"CreateResource", "ReadResource"},
		ResourceTypes: []core.ResourceTypeDefinition{
			{
				Name:        "table",
				Description: "Database table",
				Operations:  []string{"create", "read", "update", "delete"},
			},
		},
	}, nil
}

// demonstrateProviderPrefixes shows various prefix examples
func (p *ExampleProvider) demonstrateProviderPrefixes() {
	fmt.Println("Provider-specific prefix examples:")
	fmt.Printf("Operation prefixes:\n")
	fmt.Printf("  - Init: [%s]\n", p.prefix.Operation(ui.OpInit))
	fmt.Printf("  - Plan: [%s]\n", p.prefix.Operation(ui.OpPlan))
	fmt.Printf("  - Apply: [%s]\n", p.prefix.Operation(ui.OpApply))

	fmt.Printf("Resource prefixes:\n")
	fmt.Printf("  - Table: [%s]\n", p.prefix.Resource(ui.ResTable))
	fmt.Printf("  - Topic: [%s]\n", p.prefix.Resource(ui.ResTopic))
	fmt.Printf("  - Bucket: [%s]\n", p.prefix.Resource(ui.ResBucket))

	fmt.Printf("Custom prefixes:\n")
	fmt.Printf("  - Connection: [%s]\n", p.prefix.Custom("CONNECTION"))
	fmt.Printf("  - Migration: [%s]\n", p.prefix.Custom("MIGRATION"))
}

func main() {
	fmt.Println("Kolumn Provider Prefix Integration Example")
	fmt.Println("==========================================")

	// Example for PostgreSQL provider
	fmt.Println("\nPostgreSQL Provider:")
	postgresProvider := NewExampleProvider("POSTGRES")
	postgresProvider.demonstrateProviderPrefixes()

	// Example for MySQL provider
	fmt.Println("\nMySQL Provider:")
	mysqlProvider := NewExampleProvider("MYSQL")
	mysqlProvider.demonstrateProviderPrefixes()

	// Example for Kafka provider
	fmt.Println("\nKafka Provider:")
	kafkaProvider := NewExampleProvider("KAFKA")
	kafkaProvider.demonstrateProviderPrefixes()

	fmt.Println("\nIntegration Notes:")
	fmt.Println("- Add DisplayName to your provider's Schema()")
	fmt.Println("- Use ui.PrefixBuilder for consistent prefix formatting")
	fmt.Println("- Kolumn Core will automatically use the DisplayName for UI prefixes")
	fmt.Println("- Providers can define their own display names without core changes")
}
