// Package ui provides user interface utilities for Kolumn providers
package ui

import (
	"strings"
)

// PrefixBuilder helps providers create consistent UI prefixes
type PrefixBuilder struct {
	providerName string
}

// NewPrefixBuilder creates a new prefix builder for a provider
func NewPrefixBuilder(providerName string) *PrefixBuilder {
	return &PrefixBuilder{
		providerName: normalizeProviderName(providerName),
	}
}

// Operation creates a provider-specific operation prefix
// Examples: "POSTGRES-INIT", "MYSQL-PLAN", "KOLUMN-APPLY"
func (pb *PrefixBuilder) Operation(operation string) string {
	return pb.providerName + "-" + strings.ToUpper(operation)
}

// Resource creates a provider-specific resource prefix
// Examples: "POSTGRES-TABLE", "KAFKA-TOPIC", "S3-BUCKET"
func (pb *PrefixBuilder) Resource(resourceType string) string {
	return pb.providerName + "-" + strings.ToUpper(resourceType)
}

// Custom creates a custom provider-specific prefix
func (pb *PrefixBuilder) Custom(suffix string) string {
	return pb.providerName + "-" + strings.ToUpper(suffix)
}

// ProviderName returns the normalized provider name
func (pb *PrefixBuilder) ProviderName() string {
	return pb.providerName
}

// normalizeProviderName converts provider names to standard format
func normalizeProviderName(provider string) string {
	if provider == "" {
		return "UNKNOWN"
	}
	// Simply uppercase the provider name - let providers define their own display names
	return strings.ToUpper(provider)
}

// Common operation types that providers can use
const (
	OpInit     = "INIT"
	OpPlan     = "PLAN"
	OpApply    = "APPLY"
	OpDestroy  = "DESTROY"
	OpValidate = "VALIDATE"
	OpFormat   = "FORMAT"
	OpDownload = "DOWNLOAD"
	OpUpload   = "UPLOAD"
	OpPackage  = "PACKAGE"
	OpProvider = "PROVIDER"
	OpDatabase = "DATABASE"
	OpCloud    = "CLOUD"
	OpSecurity = "SECURITY"
	OpConfig   = "CONFIG"
	OpState    = "STATE"
	OpCheck    = "CHECK"
	OpFail     = "FAIL"
	OpNext     = "NEXT"
	OpItem     = "ITEM"
	OpSuccess  = "SUCCESS"
	OpError    = "ERROR"
	OpWarning  = "WARNING"
	OpInfo     = "INFO"
	OpDebug    = "DEBUG"
	OpQuery    = "QUERY"
)

// Common resource types that providers can use
const (
	ResTable          = "TABLE"
	ResView           = "VIEW"
	ResFunction       = "FUNCTION"
	ResIndex          = "INDEX"
	ResTrigger        = "TRIGGER"
	ResObject         = "OBJECT"
	ResUser           = "USER"
	ResRole           = "ROLE"
	ResPermission     = "PERMISSION"
	ResPolicy         = "POLICY"
	ResClassification = "CLASSIFICATION"
	ResResource       = "RESOURCE"
	ResTopic          = "TOPIC"
	ResBucket         = "BUCKET"
	ResStream         = "STREAM"
	ResQueue          = "QUEUE"
	ResCluster        = "CLUSTER"
)
