// Package ui provides user interface utilities for Kolumn providers
package ui

import "strings"

// PrefixBuilder helps providers create consistent UI prefixes
type PrefixBuilder struct {
	providerName string
}

// NewPrefixBuilder creates a new prefix builder for a provider
func NewPrefixBuilder(providerName string) *PrefixBuilder {
	return &PrefixBuilder{
		providerName: NormalizeProviderName(providerName),
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
