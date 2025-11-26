package core

import (
	"fmt"
	"strings"
)

// DocumentationLink represents a documentation link surfaced to provider authors
// and registry consumers.
type DocumentationLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type,omitempty"` // "official", "tutorial", "example"
}

const (
	// DefaultDocsBaseURL is the canonical base used for hosted provider docs.
	DefaultDocsBaseURL = "https://schemabounce.com/docs/providers"
	// defaultDocsAPIBasePath exposes the public API path for provider docs.
	defaultDocsAPIBasePath = "/api/v1/providers/documentation/provider"
)

func normalizedDocsBase(base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		trimmed = DefaultDocsBaseURL
	}
	return strings.TrimSuffix(trimmed, "/")
}

// CanonicalProviderDocsURL returns the hosted documentation URL for a given provider version.
func CanonicalProviderDocsURL(namespace, name, version string, baseOverrides ...string) string {
	base := DefaultDocsBaseURL
	if len(baseOverrides) > 0 {
		base = normalizedDocsBase(baseOverrides[0])
	} else {
		base = normalizedDocsBase(base)
	}
	return fmt.Sprintf("%s/%s/%s/%s", base, namespace, name, version)
}

// CanonicalResourceDocsURL returns the hosted documentation URL for a specific resource.
func CanonicalResourceDocsURL(namespace, name, version, resource string, baseOverrides ...string) string {
	return fmt.Sprintf("%s/%s", CanonicalProviderDocsURL(namespace, name, version, baseOverrides...), resource)
}

// CanonicalProviderDocsAPIPath returns the public API path for a provider's documentation payload.
func CanonicalProviderDocsAPIPath(namespace, name, version string) string {
	return fmt.Sprintf("%s/%s/%s/%s", defaultDocsAPIBasePath, namespace, name, version)
}

// CanonicalResourceDocsAPIPath returns the public API path for an individual resource doc.
func CanonicalResourceDocsAPIPath(namespace, name, version, resource string) string {
	return fmt.Sprintf("%s/%s", CanonicalProviderDocsAPIPath(namespace, name, version), resource)
}
