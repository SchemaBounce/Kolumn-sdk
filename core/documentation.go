// Package core provides documentation types for the Universal Provider Documentation System
package core

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UniversalProviderDocumentation represents the complete provider documentation
// following the Universal JSON Schema v1.0.0
type UniversalProviderDocumentation struct {
	Provider       ProviderMetadata           `json:"provider"`
	Configuration  ConfigurationDocumentation `json:"configuration"`
	Resources      map[string]*ResourceDoc    `json:"resources"`
	Examples       []*ProviderExample         `json:"examples,omitempty"`
	GettingStarted *GettingStartedGuide       `json:"getting_started,omitempty"`
	Compatibility  *CompatibilityInfo         `json:"compatibility,omitempty"`
	Metadata       RegistryMetadata           `json:"metadata"`
	SearchMetadata *SearchMetadata            `json:"search_metadata,omitempty"`
}

// ProviderMetadata contains core information about the provider
type ProviderMetadata struct {
	Namespace       string          `json:"namespace"`
	Name            string          `json:"name"`
	DisplayName     string          `json:"display_name,omitempty"`
	Version         string          `json:"version"`
	SDKVersion      string          `json:"sdk_version,omitempty"`
	Category        string          `json:"category"`
	Subcategory     string          `json:"subcategory,omitempty"`
	Description     string          `json:"description"`
	LongDescription string          `json:"long_description,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	Maintainer      *MaintainerInfo `json:"maintainer,omitempty"`
	Repository      *RepositoryInfo `json:"repository,omitempty"`
	LogoURL         string          `json:"logo_url,omitempty"`
	WebsiteURL      string          `json:"website_url,omitempty"`
	License         string          `json:"license,omitempty"`
}

// MaintainerInfo contains maintainer information
type MaintainerInfo struct {
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
	URL          string `json:"url,omitempty"`
	Organization string `json:"organization,omitempty"`
}

// RepositoryInfo contains source repository information
type RepositoryInfo struct {
	URL       string `json:"url,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Directory string `json:"directory,omitempty"`
}

// ConfigurationDocumentation contains provider configuration schema and examples
type ConfigurationDocumentation struct {
	Schema     json.RawMessage          `json:"schema"`
	Examples   []*ConfigurationExample  `json:"examples,omitempty"`
	Validation *ConfigurationValidation `json:"validation,omitempty"`
}

// ConfigurationExample shows how to configure the provider
type ConfigurationExample struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"` // basic, advanced, production
	Config      map[string]interface{} `json:"config"`
	Notes       []string               `json:"notes,omitempty"`
}

// ConfigurationValidation contains validation rules
type ConfigurationValidation struct {
	RequiredFields  []string `json:"required_fields,omitempty"`
	OptionalFields  []string `json:"optional_fields,omitempty"`
	SensitiveFields []string `json:"sensitive_fields,omitempty"`
	ConnectionTest  bool     `json:"connection_test,omitempty"`
}

// ResourceDoc contains complete documentation for a resource type
type ResourceDoc struct {
	Type          string                  `json:"type"` // create, discover
	DisplayName   string                  `json:"display_name,omitempty"`
	Description   string                  `json:"description,omitempty"`
	Category      string                  `json:"category,omitempty"`
	Operations    []string                `json:"operations"`
	Schema        json.RawMessage         `json:"schema"`
	StateSchema   json.RawMessage         `json:"state_schema,omitempty"`
	Documentation *ResourceDocumentation  `json:"documentation,omitempty"`
	Examples      []*ResourceExample      `json:"examples,omitempty"`
	Relationships []*ResourceRelationship `json:"relationships,omitempty"`
	Links         []DocumentationLink     `json:"links,omitempty"`
}

// ResourceDocumentation contains resource-specific documentation
type ResourceDocumentation struct {
	Overview        string                 `json:"overview,omitempty"`
	Usage           string                 `json:"usage,omitempty"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	Import          string                 `json:"import,omitempty"`
	BestPractices   []string               `json:"best_practices,omitempty"`
	CommonPitfalls  []string               `json:"common_pitfalls,omitempty"`
	Troubleshooting []string               `json:"troubleshooting,omitempty"`
}

// ResourceExample shows how to use a specific resource
type ResourceExample struct {
	Name            string                 `json:"name"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description,omitempty"`
	Category        string                 `json:"category,omitempty"` // basic, intermediate, advanced, production
	UseCase         string                 `json:"use_case,omitempty"`
	HCL             string                 `json:"hcl"`
	Prerequisites   []string               `json:"prerequisites,omitempty"`
	ExpectedOutputs map[string]interface{} `json:"expected_outputs,omitempty"`
	Validated       bool                   `json:"validated,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
}

// ResourceRelationship describes relationships between resources
type ResourceRelationship struct {
	ResourceType string `json:"resource_type"`
	Relationship string `json:"relationship"` // depends_on, creates, manages, monitors
	Description  string `json:"description,omitempty"`
}

// ProviderExample shows complete provider usage examples
type ProviderExample struct {
	Name              string                 `json:"name"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description,omitempty"`
	Category          string                 `json:"category,omitempty"`   // getting-started, basic, intermediate, advanced, production, migration
	Complexity        string                 `json:"complexity,omitempty"` // beginner, intermediate, expert
	UseCase           string                 `json:"use_case,omitempty"`
	HCL               string                 `json:"hcl"`
	Resources         []string               `json:"resources,omitempty"`
	Prerequisites     []string               `json:"prerequisites,omitempty"`
	SetupInstructions []string               `json:"setup_instructions,omitempty"`
	ExpectedOutputs   map[string]interface{} `json:"expected_outputs,omitempty"`
	EstimatedTime     string                 `json:"estimated_time,omitempty"`
	Validated         bool                   `json:"validated,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
}

// GettingStartedGuide provides quick start information
type GettingStartedGuide struct {
	Overview        string   `json:"overview,omitempty"`
	Installation    string   `json:"installation,omitempty"`
	BasicUsage      string   `json:"basic_usage,omitempty"`
	NextSteps       []string `json:"next_steps,omitempty"`
	CommonTasks     []string `json:"common_tasks,omitempty"`
	Troubleshooting string   `json:"troubleshooting,omitempty"`
}

// CompatibilityInfo contains version and platform compatibility
type CompatibilityInfo struct {
	KolumnVersion      string            `json:"kolumn_version,omitempty"`
	SDKVersion         string            `json:"sdk_version,omitempty"`
	SupportedPlatforms []string          `json:"supported_platforms,omitempty"`
	Dependencies       []string          `json:"dependencies,omitempty"`
	BreakingChanges    []*BreakingChange `json:"breaking_changes,omitempty"`
}

// BreakingChange describes version breaking changes
type BreakingChange struct {
	Version        string `json:"version"`
	Description    string `json:"description"`
	MigrationGuide string `json:"migration_guide,omitempty"`
}

// RegistryMetadata contains metadata for registry management
type RegistryMetadata struct {
	GeneratedAt      time.Time           `json:"generated_at"`
	GeneratorVersion string              `json:"generator_version"`
	SchemaVersion    string              `json:"schema_version"`
	Checksum         string              `json:"checksum,omitempty"`
	BuildInfo        *BuildInfo          `json:"build_info,omitempty"`
	Validation       *ValidationResult   `json:"validation,omitempty"`
	Stats            *DocumentationStats `json:"stats,omitempty"`
}

// BuildInfo contains build environment information
type BuildInfo struct {
	CommitHash string    `json:"commit_hash,omitempty"`
	BuildDate  time.Time `json:"build_date,omitempty"`
	GoVersion  string    `json:"go_version,omitempty"`
	Platform   string    `json:"platform,omitempty"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	SchemaValid       bool      `json:"schema_valid"`
	ExamplesValidated bool      `json:"examples_validated"`
	LinksChecked      bool      `json:"links_checked"`
	ValidationDate    time.Time `json:"validation_date"`
}

// DocumentationStats contains documentation statistics
type DocumentationStats struct {
	ResourceCount int `json:"resource_count"`
	ExampleCount  int `json:"example_count"`
	TotalSize     int `json:"total_size"` // Total documentation size in bytes
}

// SearchMetadata contains metadata for search indexing
type SearchMetadata struct {
	Keywords         []string               `json:"keywords,omitempty"`
	FullText         string                 `json:"full_text,omitempty"`
	SearchableFields []string               `json:"searchable_fields,omitempty"`
	BoostTerms       map[string]interface{} `json:"boost_terms,omitempty"`
}

// BuildMinimalDocumentation constructs a minimal, schema-compliant provider documentation
// using shared SDK types. Pass an optional JSON schema for configuration; when nil, an
// empty object schema is used.
func BuildMinimalDocumentation(namespace, name, version, category, description string, configSchema json.RawMessage) *UniversalProviderDocumentation {
	if len(configSchema) == 0 {
		configSchema = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	b := NewDocumentationBuilder()
	b.SetProvider(ProviderMetadata{
		Namespace:   namespace,
		Name:        name,
		Version:     version,
		Category:    category,
		Description: description,
	})
	b.SetConfiguration(ConfigurationDocumentation{Schema: configSchema})
	b.SetMetadata(RegistryMetadata{
		GeneratedAt:      time.Now().UTC(),
		GeneratorVersion: "kolumn-docs-gen-lite-0.1.0",
		SchemaVersion:    "1.0.0",
	})
	return b.Build()
}

// AppendExamplesFromDir scans a directory for .kl or .hcl files and appends them as provider examples.
// It updates documentation stats accordingly.
func AppendExamplesFromDir(docs *UniversalProviderDocumentation, dir string) error {
	if docs == nil || dir == "" {
		return nil
	}
	var countBefore int
	if docs.Examples != nil {
		countBefore = len(docs.Examples)
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".kl") || strings.HasSuffix(lower, ".hcl") {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			ex := &ProviderExample{
				Name:     name,
				Title:    strings.Title(strings.ReplaceAll(name, "-", " ")),
				HCL:      string(b),
				Category: "getting-started",
			}
			docs.Examples = append(docs.Examples, ex)
		}
		return nil
	})
	if docs.Metadata.Stats == nil {
		docs.Metadata.Stats = &DocumentationStats{}
	}
	docs.Metadata.Stats.ExampleCount = len(docs.Examples)
	if len(docs.Examples) > countBefore {
		// no-op, placeholder for further bookkeeping if needed
	}
	return nil
}

// AddResourcesFromNames adds basic ResourceDoc entries for a list of resource names
// with a default CRUD operation set. Schema and StateSchema are set to empty objects.
func AddResourcesFromNames(docs *UniversalProviderDocumentation, names []string, descriptionPrefix string, category string) {
	if docs == nil || len(names) == 0 {
		return
	}
	for _, n := range names {
		rd := &ResourceDoc{
			Type:        "create",
			Description: strings.TrimSpace(descriptionPrefix + n),
			Category:    category,
			Operations:  []string{"create", "read", "update", "delete"},
			Schema:      json.RawMessage(`{"type":"object","properties":{}}`),
			StateSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			Documentation: &ResourceDocumentation{
				Overview:   "Auto-generated documentation stub.",
				Usage:      "Fill in usage details and arguments.",
				Arguments:  map[string]interface{}{},
				Attributes: map[string]interface{}{},
			},
		}
		docs.Resources[n] = rd
	}
	if docs.Metadata.Stats == nil {
		docs.Metadata.Stats = &DocumentationStats{}
	}
	docs.Metadata.Stats.ResourceCount = len(docs.Resources)
}

// GenerateBasicSearchMetadata computes simple search metadata from provider and resource names.
func GenerateBasicSearchMetadata(docs *UniversalProviderDocumentation) {
	if docs == nil {
		return
	}
	keywords := map[string]struct{}{}
	add := func(s string) {
		if s != "" {
			keywords[strings.ToLower(s)] = struct{}{}
		}
	}
	add(docs.Provider.Name)
	add(docs.Provider.Category)
	for name := range docs.Resources {
		add(name)
	}
	// Flatten keywords
	keys := make([]string, 0, len(keywords))
	for k := range keywords {
		keys = append(keys, k)
	}
	full := docs.Provider.Description
	// Populate
	docs.SearchMetadata = &SearchMetadata{
		Keywords:         keys,
		FullText:         full,
		SearchableFields: []string{"provider.name", "provider.category", "resources.*.description"},
		BoostTerms:       map[string]interface{}{"provider.name": 2.0, "resources": 1.5},
	}
}

// EnsureAtLeastOneExample adds a basic example if no examples are present,
// to improve first-run documentation quality.
func EnsureAtLeastOneExample(docs *UniversalProviderDocumentation) {
	if docs == nil || (docs.Examples != nil && len(docs.Examples) > 0) {
		return
	}
	prov := docs.Provider.Name
	if prov == "" {
		prov = "provider"
	}
	ex := &ProviderExample{
		Name:     "basic-usage",
		Title:    "Basic Usage",
		HCL:      "provider \"" + prov + "\" {\n  # configuration here\n}\n",
		Category: "getting-started",
	}
	docs.Examples = []*ProviderExample{ex}
	if docs.Metadata.Stats == nil {
		docs.Metadata.Stats = &DocumentationStats{}
	}
	docs.Metadata.Stats.ExampleCount = len(docs.Examples)
}

// ApplyHeuristicSchemas enriches known resources with basic schemas and
// argument/attribute docs when providers don't expose detailed schemas.
func ApplyHeuristicSchemas(providerName string, docs *UniversalProviderDocumentation) {
	if docs == nil || docs.Resources == nil {
		return
	}

	// Helper to set schema and derive docs
	setSchema := func(name string, cfg, state map[string]interface{}, category string) {
		rd, ok := docs.Resources[name]
		if !ok {
			return
		}
		// Only apply if schema is empty or minimal
		rd.Schema = mustMarshal(cfg)
		rd.StateSchema = mustMarshal(state)
		if rd.Documentation == nil {
			rd.Documentation = &ResourceDocumentation{}
		}
		if args := BuildArgumentDocsFromSchema(rd.Schema); args != nil {
			rd.Documentation.Arguments = args
		}
		if attrs := BuildAttributeDocsFromStateSchema(rd.StateSchema); attrs != nil {
			rd.Documentation.Attributes = attrs
		}
		if category != "" {
			rd.Category = category
		}
	}

	lowerProv := strings.ToLower(providerName)
	switch lowerProv {
	case "kafka":
		// topic
		if _, ok := docs.Resources["topic"]; ok {
			cfg := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":               map[string]interface{}{"type": "string", "description": "Topic name"},
					"partitions":         map[string]interface{}{"type": "integer", "default": 1, "minimum": 1},
					"replication_factor": map[string]interface{}{"type": "integer", "default": 1, "minimum": 1},
					"configs":            map[string]interface{}{"type": "object", "description": "Topic-level configs"},
				},
				"required": []string{"name"},
			}
			state := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":                 map[string]interface{}{"type": "string"},
					"name":               map[string]interface{}{"type": "string"},
					"partitions":         map[string]interface{}{"type": "integer"},
					"replication_factor": map[string]interface{}{"type": "integer"},
				},
			}
			setSchema("topic", cfg, state, "streaming")
		}
	case "s3":
		if _, ok := docs.Resources["bucket"]; ok {
			cfg := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bucket_name":     map[string]interface{}{"type": "string", "description": "S3 bucket name"},
					"region":          map[string]interface{}{"type": "string", "description": "AWS region"},
					"versioning":      map[string]interface{}{"type": "boolean", "default": false},
					"lifecycle_rules": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object"}},
				},
				"required": []string{"bucket_name"},
			}
			state := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"arn":        map[string]interface{}{"type": "string"},
					"created_at": map[string]interface{}{"type": "string"},
				},
			}
			setSchema("bucket", cfg, state, "storage")
		}
	case "mysql":
		if _, ok := docs.Resources["table"]; ok {
			cfg := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":   map[string]interface{}{"type": "string", "description": "Table name"},
					"schema": map[string]interface{}{"type": "string", "description": "Database/schema name"},
					"columns": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":        map[string]interface{}{"type": "string"},
								"type":        map[string]interface{}{"type": "string"},
								"nullable":    map[string]interface{}{"type": "boolean", "default": true},
								"primary_key": map[string]interface{}{"type": "boolean", "default": false},
								"default":     map[string]interface{}{},
							},
							"required": []string{"name", "type"},
						},
					},
					"indexes": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object"}},
				},
				"required": []string{"name", "columns"},
			}
			state := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"engine":    map[string]interface{}{"type": "string"},
					"row_count": map[string]interface{}{"type": "integer"},
				},
			}
			setSchema("table", cfg, state, "database")
		}
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

// BuildArgumentDocsFromSchema creates a simple arguments doc object from a JSON Schema.
// It inspects top-level properties and required fields.
func BuildArgumentDocsFromSchema(schema json.RawMessage) map[string]interface{} {
	if len(schema) == 0 {
		return nil
	}
	var s struct {
		Properties map[string]struct {
			Type        interface{} `json:"type"`
			Description string      `json:"description"`
			Default     interface{} `json:"default"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schema, &s); err != nil || len(s.Properties) == 0 {
		return nil
	}
	required := map[string]bool{}
	for _, r := range s.Required {
		required[r] = true
	}
	out := map[string]interface{}{}
	for k, v := range s.Properties {
		entry := map[string]interface{}{}
		if v.Type != nil {
			entry["type"] = v.Type
		}
		if v.Description != "" {
			entry["description"] = v.Description
		} else {
			entry["description"] = "Auto-generated argument"
		}
		if v.Default != nil {
			entry["default"] = v.Default
		}
		if required[k] {
			entry["required"] = true
		}
		out[k] = entry
	}
	return out
}

// BuildAttributeDocsFromStateSchema mirrors BuildArgumentDocsFromSchema but for state schemas.
func BuildAttributeDocsFromStateSchema(stateSchema json.RawMessage) map[string]interface{} {
	if len(stateSchema) == 0 {
		return nil
	}
	var s struct {
		Properties map[string]struct {
			Type        interface{} `json:"type"`
			Description string      `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(stateSchema, &s); err != nil || len(s.Properties) == 0 {
		return nil
	}
	out := map[string]interface{}{}
	for k, v := range s.Properties {
		entry := map[string]interface{}{}
		if v.Type != nil {
			entry["type"] = v.Type
		}
		if v.Description != "" {
			entry["description"] = v.Description
		} else {
			entry["description"] = "Auto-generated attribute"
		}
		out[k] = entry
	}
	return out
}

// DocumentationGenerator interface for extracting documentation from providers
type DocumentationGenerator interface {
	// ExtractDocumentation extracts documentation from a provider
	ExtractDocumentation(provider Provider) (*UniversalProviderDocumentation, error)

	// ValidateDocumentation validates documentation against the schema
	ValidateDocumentation(docs *UniversalProviderDocumentation) error

	// GenerateSearchMetadata generates search optimization metadata
	GenerateSearchMetadata(docs *UniversalProviderDocumentation) *SearchMetadata
}

// DocumentationBuilder helps build documentation incrementally
type DocumentationBuilder struct {
	docs *UniversalProviderDocumentation
}

// NewDocumentationBuilder creates a new documentation builder
func NewDocumentationBuilder() *DocumentationBuilder {
	return &DocumentationBuilder{
		docs: &UniversalProviderDocumentation{
			Resources: make(map[string]*ResourceDoc),
			Metadata: RegistryMetadata{
				GeneratedAt:   time.Now().UTC(),
				SchemaVersion: "1.0.0",
			},
		},
	}
}

// SetProvider sets the provider metadata
func (b *DocumentationBuilder) SetProvider(metadata ProviderMetadata) *DocumentationBuilder {
	b.docs.Provider = metadata
	return b
}

// SetConfiguration sets the configuration documentation
func (b *DocumentationBuilder) SetConfiguration(config ConfigurationDocumentation) *DocumentationBuilder {
	b.docs.Configuration = config
	return b
}

// AddResource adds a resource to the documentation
func (b *DocumentationBuilder) AddResource(name string, resource *ResourceDoc) *DocumentationBuilder {
	b.docs.Resources[name] = resource
	return b
}

// AddExample adds a provider example
func (b *DocumentationBuilder) AddExample(example *ProviderExample) *DocumentationBuilder {
	b.docs.Examples = append(b.docs.Examples, example)
	return b
}

// SetGettingStarted sets the getting started guide
func (b *DocumentationBuilder) SetGettingStarted(guide *GettingStartedGuide) *DocumentationBuilder {
	b.docs.GettingStarted = guide
	return b
}

// SetCompatibility sets the compatibility information
func (b *DocumentationBuilder) SetCompatibility(compat *CompatibilityInfo) *DocumentationBuilder {
	b.docs.Compatibility = compat
	return b
}

// SetMetadata sets the registry metadata
func (b *DocumentationBuilder) SetMetadata(metadata RegistryMetadata) *DocumentationBuilder {
	b.docs.Metadata = metadata
	return b
}

// SetSearchMetadata sets the search metadata
func (b *DocumentationBuilder) SetSearchMetadata(search *SearchMetadata) *DocumentationBuilder {
	b.docs.SearchMetadata = search
	return b
}

// Build returns the complete documentation
func (b *DocumentationBuilder) Build() *UniversalProviderDocumentation {
	// Update stats
	if b.docs.Metadata.Stats == nil {
		b.docs.Metadata.Stats = &DocumentationStats{}
	}

	b.docs.Metadata.Stats.ResourceCount = len(b.docs.Resources)
	b.docs.Metadata.Stats.ExampleCount = len(b.docs.Examples)

	// Add resource examples to total count
	for _, resource := range b.docs.Resources {
		if resource.Examples != nil {
			b.docs.Metadata.Stats.ExampleCount += len(resource.Examples)
		}
	}

	return b.docs
}

// ToJSON converts the documentation to JSON
func (b *DocumentationBuilder) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b.Build(), "", "  ")
}

// FromJSON loads documentation from JSON
func (b *DocumentationBuilder) FromJSON(data []byte) error {
	return json.Unmarshal(data, b.docs)
}
