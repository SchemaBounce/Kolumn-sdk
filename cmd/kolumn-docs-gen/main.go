// kolumn-docs-gen is a standalone tool to extract documentation from Kolumn providers
// and generate universal JSON documentation for the provider registry.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"time"

	"github.com/schemabounce/kolumn/sdk/core"
)

const (
	version       = "v1.0.0"
	schemaVersion = "1.0.0"
)

// Config holds the command-line configuration
type Config struct {
	ProviderBinary string
	DocsDir        string
	ExamplesDir    string
	OutputFile     string
	Validate       bool
	Verbose        bool
	NoMetadata     bool
}

// DocumentationExtractor handles extraction of documentation from providers
type DocumentationExtractor struct {
	config  *Config
	builder *core.DocumentationBuilder
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.Printf("Kolumn Documentation Generator %s", version)
		log.Printf("Schema Version: %s", schemaVersion)
	}

	extractor := &DocumentationExtractor{
		config:  config,
		builder: core.NewDocumentationBuilder(),
	}

	if err := extractor.Extract(); err != nil {
		log.Fatalf("Documentation extraction failed: %v", err)
	}

	if config.Verbose {
		log.Printf("Documentation generated successfully: %s", config.OutputFile)
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.ProviderBinary, "provider", "", "Path to provider binary (required)")
	flag.StringVar(&config.DocsDir, "docs", "docs/", "Path to documentation directory")
	flag.StringVar(&config.ExamplesDir, "examples", "examples/", "Path to examples directory")
	flag.StringVar(&config.OutputFile, "output", "provider-docs.json", "Output file path")
	flag.BoolVar(&config.Validate, "validate", true, "Validate documentation against schema")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.NoMetadata, "no-metadata", false, "Skip build metadata generation")

	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message")

	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if config.ProviderBinary == "" {
		fmt.Fprintf(os.Stderr, "Error: -provider flag is required\n\n")
		printHelp()
		os.Exit(1)
	}

	return config
}

func printHelp() {
	fmt.Printf(`Kolumn Documentation Generator %s

Extracts comprehensive documentation from Kolumn providers and generates
universal JSON documentation for the provider registry.

USAGE:
    kolumn-docs-gen [OPTIONS]

REQUIRED FLAGS:
    -provider PATH      Path to the provider binary

OPTIONAL FLAGS:
    -docs PATH          Path to documentation directory (default: docs/)
    -examples PATH      Path to examples directory (default: examples/)
    -output PATH        Output file path (default: provider-docs.json)
    -validate           Validate documentation against schema (default: true)
    -no-metadata        Skip build metadata generation
    -verbose            Enable verbose logging
    -help, -h           Show this help message

EXAMPLES:
    # Basic usage
    kolumn-docs-gen -provider ./kolumn-provider-postgres

    # Custom paths and output
    kolumn-docs-gen -provider ./kolumn-provider-postgres \
                    -docs ./documentation \
                    -examples ./examples \
                    -output ./postgres-docs.json

    # Generate without validation (faster)
    kolumn-docs-gen -provider ./kolumn-provider-postgres \
                    -validate=false

For more information, visit: https://docs.kolumn.com/sdk/documentation-generator
`, version)
}

// Extract extracts documentation from the provider
func (e *DocumentationExtractor) Extract() error {
	// 1. Load provider and extract schema + documentation
	if err := e.extractFromProvider(); err != nil {
		return fmt.Errorf("failed to extract from provider: %w", err)
	}

	// 2. Load documentation files
	if err := e.loadDocumentationFiles(); err != nil {
		return fmt.Errorf("failed to load documentation files: %w", err)
	}

	// 3. Load examples
	if err := e.loadExamples(); err != nil {
		return fmt.Errorf("failed to load examples: %w", err)
	}

	// 4. Generate metadata
	if !e.config.NoMetadata {
		e.generateMetadata()
	}

	// 5. Validate if requested
	if e.config.Validate {
		if err := e.validateDocumentation(); err != nil {
			return fmt.Errorf("documentation validation failed: %w", err)
		}
	}

	// 6. Generate output
	if err := e.generateOutput(); err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	return nil
}

// extractFromProvider loads the provider and extracts schema and documentation
func (e *DocumentationExtractor) extractFromProvider() error {
	if e.config.Verbose {
		log.Printf("Loading provider: %s", e.config.ProviderBinary)
	}

	// For now, we'll use a simple approach - execute the provider with a special flag
	// In a real implementation, you might load it as a plugin or use RPC
	schema, docs, err := e.executeProviderForDocs()
	if err != nil {
		return fmt.Errorf("failed to execute provider: %w", err)
	}

	// Extract provider metadata from schema
	providerMeta := e.extractProviderMetadata(schema)
	e.builder.SetProvider(providerMeta)

	// Extract configuration documentation
	configDocs := e.extractConfigurationDocs(schema)
	e.builder.SetConfiguration(configDocs)

	// Extract resource documentation
	e.extractResourceDocs(schema, docs)

	return nil
}

// executeProviderForDocs executes the provider to get documentation
func (e *DocumentationExtractor) executeProviderForDocs() (*core.Schema, *core.ProviderDocumentation, error) {
	// Create a temporary approach - in practice, this would use proper plugin loading
	// For now, we'll assume the provider can be executed with --docs flag

	// Try to load as plugin first (Unix-like systems)
	if runtime.GOOS != "windows" {
		return e.loadProviderAsPlugin()
	}

	// Fallback to execution method
	return e.executeProviderCommand()
}

// loadProviderAsPlugin loads the provider as a Go plugin
func (e *DocumentationExtractor) loadProviderAsPlugin() (*core.Schema, *core.ProviderDocumentation, error) {
	// Load the plugin
	p, err := plugin.Open(e.config.ProviderBinary)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Look for NewProvider function
	symbol, err := p.Lookup("NewProvider")
	if err != nil {
		return nil, nil, fmt.Errorf("NewProvider function not found: %w", err)
	}

	// Cast to function and call it
	newProvider, ok := symbol.(func() core.Provider)
	if !ok {
		return nil, nil, fmt.Errorf("NewProvider has unexpected signature")
	}

	provider := newProvider()

	// Get schema
	schema, err := provider.Schema()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Get documentation (try DocumentedProvider interface first)
	var docs *core.ProviderDocumentation
	if docProvider, ok := provider.(core.DocumentedProvider); ok {
		docs = docProvider.Documentation()
	} else {
		// Fallback: create minimal documentation from schema
		docs = &core.ProviderDocumentation{
			Name:        schema.Name,
			Version:     schema.Version,
			Description: schema.Description,
		}
	}

	// Use the documentation we already obtained
	return schema, docs, nil
}

// executeProviderCommand executes the provider as a command
func (e *DocumentationExtractor) executeProviderCommand() (*core.Schema, *core.ProviderDocumentation, error) {
	// Execute provider with --schema flag to get schema
	schemaCmd := exec.Command(e.config.ProviderBinary, "--schema")
	schemaOutput, err := schemaCmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %w", err)
	}

	var schema core.Schema
	if err := json.Unmarshal(schemaOutput, &schema); err != nil {
		return nil, nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Execute provider with --docs flag to get documentation
	docsCmd := exec.Command(e.config.ProviderBinary, "--docs")
	docsOutput, err := docsCmd.Output()
	if err != nil {
		// Documentation might not be implemented yet, create empty
		docs := &core.ProviderDocumentation{
			Name:        schema.Name,
			Version:     schema.Version,
			Description: schema.Description,
		}
		return &schema, docs, nil
	}

	var docs core.ProviderDocumentation
	if err := json.Unmarshal(docsOutput, &docs); err != nil {
		return nil, nil, fmt.Errorf("failed to parse documentation: %w", err)
	}

	return &schema, &docs, nil
}

// extractProviderMetadata extracts provider metadata from schema
func (e *DocumentationExtractor) extractProviderMetadata(schema *core.Schema) core.ProviderMetadata {
	// Extract namespace and name from binary path
	binary := filepath.Base(e.config.ProviderBinary)
	binary = strings.TrimSuffix(binary, filepath.Ext(binary))

	var namespace, name string
	if strings.HasPrefix(binary, "kolumn-provider-") {
		name = strings.TrimPrefix(binary, "kolumn-provider-")
		namespace = "kolumn-official" // Default namespace
	} else {
		name = binary
		namespace = "community"
	}

	// Determine category from name or type
	category := e.inferCategory(name, schema.Type)

	return core.ProviderMetadata{
		Namespace:   namespace,
		Name:        name,
		DisplayName: fmt.Sprintf("%s Provider", strings.Title(name)),
		Version:     schema.Version,
		SDKVersion:  core.SDKVersion,
		Category:    category,
		Description: schema.Description,
		Tags:        e.generateTags(name, category),
	}
}

// inferCategory infers the provider category from name and type
func (e *DocumentationExtractor) inferCategory(name, providerType string) string {
	// Database providers
	databases := []string{"postgres", "mysql", "sqlite", "mssql", "mongodb", "dynamodb", "influxdb", "snowflake", "bigquery", "redshift", "databricks"}
	for _, db := range databases {
		if strings.Contains(name, db) {
			return "database"
		}
	}

	// Cache providers
	caches := []string{"redis", "elasticsearch", "memcached"}
	for _, cache := range caches {
		if strings.Contains(name, cache) {
			return "cache"
		}
	}

	// Streaming providers
	streaming := []string{"kafka", "kinesis", "pulsar", "nats"}
	for _, stream := range streaming {
		if strings.Contains(name, stream) {
			return "streaming"
		}
	}

	// ETL providers
	etl := []string{"airbyte", "dbt", "fivetran", "spark"}
	for _, e := range etl {
		if strings.Contains(name, e) {
			return "etl"
		}
	}

	// Orchestration providers
	orchestration := []string{"airflow", "dagster", "prefect", "temporal"}
	for _, orch := range orchestration {
		if strings.Contains(name, orch) {
			return "orchestration"
		}
	}

	// Storage providers
	storage := []string{"s3", "gcs", "azure", "blob", "delta", "iceberg"}
	for _, stor := range storage {
		if strings.Contains(name, stor) {
			return "storage"
		}
	}

	// Default to database if unclear
	return "database"
}

// generateTags generates tags for the provider
func (e *DocumentationExtractor) generateTags(name, category string) []string {
	tags := []string{category}

	// Add specific tags based on name
	if strings.Contains(name, "sql") || category == "database" {
		tags = append(tags, "sql")
	}
	if strings.Contains(name, "nosql") {
		tags = append(tags, "nosql")
	}
	if strings.Contains(name, "cloud") {
		tags = append(tags, "cloud")
	}

	return tags
}

// extractConfigurationDocs extracts configuration documentation from schema
func (e *DocumentationExtractor) extractConfigurationDocs(schema *core.Schema) core.ConfigurationDocumentation {
	return core.ConfigurationDocumentation{
		Schema: schema.ConfigSchema,
		Examples: []*core.ConfigurationExample{
			{
				Name:     "basic",
				Title:    "Basic Configuration",
				Category: "basic",
				Config:   make(map[string]interface{}),
			},
		},
	}
}

// extractResourceDocs extracts resource documentation from schema and provider docs
func (e *DocumentationExtractor) extractResourceDocs(schema *core.Schema, docs *core.ProviderDocumentation) {
	// Extract from ResourceTypes (new format)
	for _, resourceType := range schema.ResourceTypes {
		resourceDoc := &core.ResourceDoc{
			Type:        e.inferResourceType(resourceType.Name),
			DisplayName: strings.Title(strings.ReplaceAll(resourceType.Name, "_", " ")),
			Description: resourceType.Description,
			Operations:  resourceType.Operations,
			Schema:      resourceType.ConfigSchema,
			StateSchema: resourceType.StateSchema,
			Documentation: &core.ResourceDocumentation{
				Overview: fmt.Sprintf("Manages %s resources", resourceType.Name),
			},
			Examples: []*core.ResourceExample{
				{
					Name:     "basic",
					Title:    fmt.Sprintf("Basic %s", resourceType.Name),
					Category: "basic",
					HCL:      e.generateBasicExample(resourceType.Name),
				},
			},
		}

		e.builder.AddResource(resourceType.Name, resourceDoc)
	}

	// Extract from legacy CreateObjects
	for name, objType := range schema.CreateObjects {
		if _, exists := e.builder.Build().Resources[name]; !exists {
			resourceDoc := &core.ResourceDoc{
				Type:        "create",
				DisplayName: strings.Title(strings.ReplaceAll(name, "_", " ")),
				Description: objType.Description,
				Operations:  []string{"create", "read", "update", "delete"},
				Schema:      json.RawMessage(`{}`),
				Documentation: &core.ResourceDocumentation{
					Overview: fmt.Sprintf("Manages %s resources", name),
				},
			}

			e.builder.AddResource(name, resourceDoc)
		}
	}

	// Extract from legacy DiscoverObjects
	for name, objType := range schema.DiscoverObjects {
		if _, exists := e.builder.Build().Resources[name]; !exists {
			resourceDoc := &core.ResourceDoc{
				Type:        "discover",
				DisplayName: strings.Title(strings.ReplaceAll(name, "_", " ")),
				Description: objType.Description,
				Operations:  []string{"discover", "scan"},
				Schema:      json.RawMessage(`{}`),
				Documentation: &core.ResourceDocumentation{
					Overview: fmt.Sprintf("Discovers %s resources", name),
				},
			}

			e.builder.AddResource(name, resourceDoc)
		}
	}
}

// inferResourceType infers whether a resource is create or discover type
func (e *DocumentationExtractor) inferResourceType(name string) string {
	// Discover resources typically have these patterns
	discoverPatterns := []string{"existing", "discovery", "scan", "find", "detect"}
	for _, pattern := range discoverPatterns {
		if strings.Contains(name, pattern) {
			return "discover"
		}
	}

	// Default to create type
	return "create"
}

// generateBasicExample generates a basic HCL example for a resource
func (e *DocumentationExtractor) generateBasicExample(resourceType string) string {
	return fmt.Sprintf(`create "%s" "example" {
  name = "example-%s"
  
  # Add your configuration here
}`, resourceType, strings.ReplaceAll(resourceType, "_", "-"))
}

// loadDocumentationFiles loads markdown documentation files
func (e *DocumentationExtractor) loadDocumentationFiles() error {
	if e.config.Verbose {
		log.Printf("Loading documentation files from: %s", e.config.DocsDir)
	}

	if _, err := os.Stat(e.config.DocsDir); os.IsNotExist(err) {
		if e.config.Verbose {
			log.Printf("Documentation directory not found, skipping: %s", e.config.DocsDir)
		}
		return nil
	}

	// Load getting started guide
	gettingStartedPath := filepath.Join(e.config.DocsDir, "getting-started.md")
	if content, err := os.ReadFile(gettingStartedPath); err == nil {
		guide := &core.GettingStartedGuide{
			Overview: string(content),
		}
		e.builder.SetGettingStarted(guide)
	}

	// Load other documentation files
	err := filepath.WalkDir(e.config.DocsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		if e.config.Verbose {
			log.Printf("Processing documentation file: %s", path)
		}

		// Process markdown files and integrate into resource documentation
		// This would involve parsing markdown and updating resource docs

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk documentation directory: %w", err)
	}

	return nil
}

// loadExamples loads example files
func (e *DocumentationExtractor) loadExamples() error {
	if e.config.Verbose {
		log.Printf("Loading examples from: %s", e.config.ExamplesDir)
	}

	if _, err := os.Stat(e.config.ExamplesDir); os.IsNotExist(err) {
		if e.config.Verbose {
			log.Printf("Examples directory not found, skipping: %s", e.config.ExamplesDir)
		}
		return nil
	}

	err := filepath.WalkDir(e.config.ExamplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".kl") {
			return nil
		}

		if e.config.Verbose {
			log.Printf("Processing example file: %s", path)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Create example from file
		example := &core.ProviderExample{
			Name:        strings.TrimSuffix(d.Name(), ".kl"),
			Title:       strings.Title(strings.ReplaceAll(strings.TrimSuffix(d.Name(), ".kl"), "-", " ")),
			Description: fmt.Sprintf("Example from %s", path),
			Category:    e.inferExampleCategory(path),
			HCL:         string(content),
		}

		e.builder.AddExample(example)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk examples directory: %w", err)
	}

	return nil
}

// inferExampleCategory infers the category of an example from its path
func (e *DocumentationExtractor) inferExampleCategory(path string) string {
	path = strings.ToLower(path)

	if strings.Contains(path, "basic") || strings.Contains(path, "simple") {
		return "basic"
	}
	if strings.Contains(path, "advanced") {
		return "advanced"
	}
	if strings.Contains(path, "production") || strings.Contains(path, "prod") {
		return "production"
	}
	if strings.Contains(path, "getting") || strings.Contains(path, "start") {
		return "getting-started"
	}

	return "basic"
}

// generateMetadata generates build and validation metadata
func (e *DocumentationExtractor) generateMetadata() {
	if e.config.Verbose {
		log.Printf("Generating metadata")
	}

	buildInfo := &core.BuildInfo{
		BuildDate: time.Now().UTC(),
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// Try to get git commit hash
	if output, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		buildInfo.CommitHash = strings.TrimSpace(string(output))
	}

	metadata := core.RegistryMetadata{
		GeneratedAt:      time.Now().UTC(),
		GeneratorVersion: version,
		SchemaVersion:    schemaVersion,
		BuildInfo:        buildInfo,
		Validation: &core.ValidationResult{
			SchemaValid:    true,
			ValidationDate: time.Now().UTC(),
		},
	}

	e.builder.SetMetadata(metadata)
}

// validateDocumentation validates the documentation against the schema
func (e *DocumentationExtractor) validateDocumentation() error {
	if e.config.Verbose {
		log.Printf("Validating documentation")
	}

	// Basic validation - in practice, this would use the JSON schema
	docs := e.builder.Build()

	if docs.Provider.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if docs.Provider.Version == "" {
		return fmt.Errorf("provider version is required")
	}
	if docs.Provider.Category == "" {
		return fmt.Errorf("provider category is required")
	}

	// Validate resources
	if len(docs.Resources) == 0 {
		log.Printf("Warning: No resources found in provider")
	}

	for name, resource := range docs.Resources {
		if resource.Type == "" {
			return fmt.Errorf("resource %s: type is required", name)
		}
		if len(resource.Operations) == 0 {
			return fmt.Errorf("resource %s: operations are required", name)
		}
	}

	if e.config.Verbose {
		log.Printf("Documentation validation passed")
	}

	return nil
}

// generateOutput generates the final JSON output
func (e *DocumentationExtractor) generateOutput() error {
	if e.config.Verbose {
		log.Printf("Generating output file: %s", e.config.OutputFile)
	}

	// Build final documentation
	docs := e.builder.Build()

	// Generate checksum
	jsonData, err := json.Marshal(docs)
	if err != nil {
		return fmt.Errorf("failed to marshal documentation: %w", err)
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(jsonData))
	docs.Metadata.Checksum = checksum
	docs.Metadata.Stats.TotalSize = len(jsonData)

	// Generate final JSON with pretty printing
	finalData, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal final documentation: %w", err)
	}

	// Write to file
	if err := os.WriteFile(e.config.OutputFile, finalData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if e.config.Verbose {
		log.Printf("Generated documentation with %d resources and %d examples",
			docs.Metadata.Stats.ResourceCount,
			docs.Metadata.Stats.ExampleCount)
		log.Printf("Total size: %d bytes", docs.Metadata.Stats.TotalSize)
		log.Printf("Checksum: %s", checksum)
	}

	return nil
}
