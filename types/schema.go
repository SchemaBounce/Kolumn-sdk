// Package types provides shared type definitions for the Kolumn SDK
package types

import (
	"encoding/json"
)

// ProviderSchema defines the schema and capabilities of a provider
type ProviderSchema struct {
	// Provider metadata
	Provider ProviderSpec `json:"provider"`

	// Supported functions
	Functions map[string]FunctionSpec `json:"functions"`

	// Resource types this provider manages
	ResourceTypes []string `json:"resource_types,omitempty"`

	// Provider capabilities
	Capabilities []string `json:"capabilities,omitempty"`

	// Configuration schema
	ConfigSchema *ConfigSchema `json:"config_schema,omitempty"`
}

// ProviderSpec contains provider metadata
type ProviderSpec struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
	Website     string `json:"website,omitempty"`
}

// FunctionSpec defines a provider function
type FunctionSpec struct {
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Idempotent   bool            `json:"idempotent"`
	RequiresAuth bool            `json:"requires_auth"`
	Capabilities []string        `json:"capabilities,omitempty"`
}

// ConfigSchema defines the provider configuration schema
type ConfigSchema struct {
	Version    int                 `json:"version"`
	Block      *ConfigBlock        `json:"block"`
	Properties map[string]Property `json:"properties,omitempty"`
}

// ConfigBlock represents a configuration block
type ConfigBlock struct {
	Attributes map[string]*Attribute `json:"attributes,omitempty"`
	BlockTypes map[string]*BlockType `json:"block_types,omitempty"`
}

// Attribute defines a configuration attribute
type Attribute struct {
	Type        AttributeType `json:"type"`
	Description string        `json:"description"`
	Required    bool          `json:"required"`
	Optional    bool          `json:"optional"`
	Computed    bool          `json:"computed"`
	Sensitive   bool          `json:"sensitive"`
	Default     interface{}   `json:"default,omitempty"`
}

// BlockType defines a nested configuration block type
type BlockType struct {
	Block       *ConfigBlock `json:"block"`
	NestingMode NestingMode  `json:"nesting_mode"`
	MinItems    int          `json:"min_items,omitempty"`
	MaxItems    int          `json:"max_items,omitempty"`
}

// AttributeType represents the type of a configuration attribute
type AttributeType string

const (
	TypeString  AttributeType = "string"
	TypeNumber  AttributeType = "number"
	TypeBool    AttributeType = "bool"
	TypeList    AttributeType = "list"
	TypeMap     AttributeType = "map"
	TypeSet     AttributeType = "set"
	TypeObject  AttributeType = "object"
	TypeDynamic AttributeType = "dynamic"
)

// NestingMode defines how blocks are nested
type NestingMode string

const (
	NestingSingle NestingMode = "single"
	NestingList   NestingMode = "list"
	NestingMap    NestingMode = "map"
	NestingSet    NestingMode = "set"
)

// Property represents a JSON schema property
type Property struct {
	Type        string              `json:"type,omitempty"`
	Description string              `json:"description,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Default     interface{}         `json:"default,omitempty"`
	Examples    []interface{}       `json:"examples,omitempty"`
	Enum        []interface{}       `json:"enum,omitempty"`
}
