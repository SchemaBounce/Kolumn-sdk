// Package types provides shared type definitions for the Kolumn SDK
package types

import (
	"encoding/json"
	"time"
)

// UniversalState represents the universal state format for all providers
type UniversalState struct {
	// Core State Information
	Version       int64  `json:"version"`
	KolumnVersion string `json:"kolumn_version"`
	Serial        int64  `json:"serial"`
	Lineage       string `json:"lineage"`

	// Universal Resources
	Resources []UniversalResource `json:"resources"`

	// Provider Configurations
	Providers map[string]ProviderState `json:"providers"`

	// Cross-Provider Dependencies
	Dependencies []Dependency `json:"dependencies"`

	// Metadata and Governance
	Metadata   StateMetadata   `json:"metadata"`
	Governance GovernanceState `json:"governance"`

	// Checksums for Integrity
	Checksums map[string]string `json:"checksums"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UniversalResource represents a resource in universal state format
type UniversalResource struct {
	// Resource Identity
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider"`

	// Resource State
	Mode      ResourceMode       `json:"mode"`
	Instances []ResourceInstance `json:"instances"`

	// Dependencies and References
	DependsOn  []string            `json:"depends_on,omitempty"`
	References []ResourceReference `json:"references,omitempty"`

	// Provider-Specific State
	ProviderState json.RawMessage `json:"provider_state"`

	// Enhanced Metadata
	Metadata ResourceMetadata `json:"metadata"`

	// Governance Information
	Classifications []string         `json:"classifications,omitempty"`
	Compliance      ComplianceStatus `json:"compliance,omitempty"`
}

// ResourceMode represents the resource mode
type ResourceMode string

const (
	ResourceModeManaged ResourceMode = "managed"
	ResourceModeData    ResourceMode = "data"
)

// ResourceInstance represents a resource instance
type ResourceInstance struct {
	IndexKey   interface{}            `json:"index_key,omitempty"`
	Status     ResourceStatus         `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	Private    json.RawMessage        `json:"private,omitempty"`
	Metadata   ResourceMetadata       `json:"metadata"`

	// State-specific fields
	Tainted             bool   `json:"tainted,omitempty"`
	Deposed             string `json:"deposed,omitempty"`
	CreateBeforeDestroy bool   `json:"create_before_destroy,omitempty"`
}

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	StatusUnknown  ResourceStatus = "unknown"
	StatusCreating ResourceStatus = "creating"
	StatusReady    ResourceStatus = "ready"
	StatusUpdating ResourceStatus = "updating"
	StatusDeleting ResourceStatus = "deleting"
	StatusError    ResourceStatus = "error"
	StatusTainted  ResourceStatus = "tainted"
)

// ResourceReference represents a reference between resources
type ResourceReference struct {
	SourcePath      string `json:"source_path"`
	TargetResource  string `json:"target_resource"`
	TargetAttribute string `json:"target_attribute"`
	ReferenceType   string `json:"reference_type"`
}

// ResourceMetadata contains additional resource metadata
type ResourceMetadata map[string]interface{}

// ProviderState contains provider-specific state information
type ProviderState struct {
	Config   map[string]interface{} `json:"config"`
	Version  string                 `json:"version"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Dependency represents a cross-provider dependency
type Dependency struct {
	ID             string         `json:"id"`
	ResourceID     string         `json:"resource_id"`
	DependsOnID    string         `json:"depends_on_id"`
	DependencyType DependencyType `json:"dependency_type"`
	Constraint     string         `json:"constraint,omitempty"`
	Optional       bool           `json:"optional"`
}

// DependencyType represents the type of dependency
type DependencyType string

const (
	DependencyTypeExplicit DependencyType = "explicit"
	DependencyTypeImplicit DependencyType = "implicit"
	DependencyTypeData     DependencyType = "data"
	DependencyTypeOutput   DependencyType = "output"
)

// StateMetadata contains metadata about the state
type StateMetadata struct {
	Format           string                 `json:"format"`
	FormatVersion    string                 `json:"format_version"`
	Generator        string                 `json:"generator"`
	GeneratorVersion string                 `json:"generator_version"`
	CreatedBy        string                 `json:"created_by,omitempty"`
	Environment      string                 `json:"environment,omitempty"`
	Workspace        string                 `json:"workspace,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	Custom           map[string]interface{} `json:"custom,omitempty"`
}

// GovernanceState contains governance-related state information
type GovernanceState struct {
	Classifications  []ClassificationState `json:"classifications,omitempty"`
	Policies         []PolicyState         `json:"policies,omitempty"`
	ComplianceStatus ComplianceStatus      `json:"compliance_status,omitempty"`
	DataLineage      DataLineageState      `json:"data_lineage,omitempty"`
}

// ClassificationState represents the state of data classifications
type ClassificationState struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Level       string                 `json:"level"`
	Resources   []string               `json:"resources"`
	Policies    []string               `json:"policies"`
	LastUpdated time.Time              `json:"last_updated"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyState represents the state of governance policies
type PolicyState struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Status        string            `json:"status"`
	Resources     []string          `json:"resources"`
	LastEvaluated time.Time         `json:"last_evaluated"`
	Violations    []PolicyViolation `json:"violations,omitempty"`
}

// ComplianceStatus represents compliance status
type ComplianceStatus struct {
	Framework   string                 `json:"framework"`
	Status      string                 `json:"status"`
	LastChecked time.Time              `json:"last_checked"`
	Issues      []ComplianceIssue      `json:"issues,omitempty"`
	Score       float64                `json:"score,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataLineageState represents data lineage information
type DataLineageState struct {
	Graph       map[string][]string    `json:"graph"`
	LastUpdated time.Time              `json:"last_updated"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Resource    string    `json:"resource"`
	Policy      string    `json:"policy"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	Resource    string    `json:"resource"`
	Rule        string    `json:"rule"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
}
