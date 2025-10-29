// Package core - Governance integration for Kolumn Provider SDK
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// GOVERNANCE INTEGRATION FOR EXTERNAL PROVIDERS
// =============================================================================
//
// This module enables external providers to receive and respect governance
// decisions from the Kolumn provider, creating true cross-provider governance.

// GovernanceAwareProvider extends the base Provider interface with governance capabilities
type GovernanceAwareProvider interface {
	Provider

	// ConfigureGovernance sets up governance context and enforcement
	ConfigureGovernance(ctx context.Context, governanceCtx *GovernanceContext) error

	// ValidateGovernanceCompliance validates that a resource operation complies with governance rules
	ValidateGovernanceCompliance(ctx context.Context, operation string, resource string, config map[string]interface{}) (*GovernanceValidationResult, error)

	// ApplyGovernanceRules applies governance rules to a resource configuration
	ApplyGovernanceRules(ctx context.Context, resourceType string, config map[string]interface{}, governanceCtx *GovernanceContext) (map[string]interface{}, error)

	// GetGovernanceCapabilities returns the governance capabilities supported by this provider
	GetGovernanceCapabilities() *GovernanceCapabilities
}

// GovernanceContext represents governance metadata passed from Kolumn provider
type GovernanceContext struct {
	// Core governance information
	DataObjects     map[string]*DataObjectContext     `json:"data_objects"`
	Classifications map[string]*ClassificationContext `json:"classifications"`
	Roles           map[string]*RoleContext           `json:"roles,omitempty"`
	Permissions     map[string]*PermissionContext     `json:"permissions,omitempty"`

	// Request-specific context
	RequestContext *RequestGovernanceContext `json:"request_context"`

	// Enforcement settings
	EnforcementLevel string            `json:"enforcement_level"` // strict, advisory, disabled
	TierLimitations  map[string]string `json:"tier_limitations"`  // tier-based feature restrictions

	// Audit information
	AuditContext *AuditContext `json:"audit_context"`
}

// DataObjectContext represents governance context for a data object
type DataObjectContext struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Columns         []ColumnContext        `json:"columns"`
	Classifications []string               `json:"classifications"`
	Metadata        map[string]interface{} `json:"metadata"`

	// Governance-specific context
	EncryptionRequired bool             `json:"encryption_required"`
	AccessControls     []AccessControl  `json:"access_controls"`
	ComplianceRules    []ComplianceRule `json:"compliance_rules"`
	DataLineage        *DataLineageInfo `json:"data_lineage,omitempty"`
}

// ColumnContext represents governance context for a column
type ColumnContext struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Classifications []string `json:"classifications"`
	Required        bool     `json:"required"`
	PrimaryKey      bool     `json:"primary_key"`
	Unique          bool     `json:"unique"`

	// Governance-specific context
	EncryptionMethod string            `json:"encryption_method,omitempty"`
	MaskingRule      string            `json:"masking_rule,omitempty"`
	AccessLevel      string            `json:"access_level"`
	ComplianceFlags  []string          `json:"compliance_flags"`
	Transformations  []ColumnTransform `json:"transformations,omitempty"`
}

// ColumnTransform represents a column transformation rule
type ColumnTransform struct {
	Type              string                     `json:"type"`
	Classifications   []string                   `json:"classifications,omitempty"`
	Required          *bool                      `json:"required,omitempty"`
	PrimaryKey        *bool                      `json:"primary_key,omitempty"`
	Unique            *bool                      `json:"unique,omitempty"`
	Nullable          *bool                      `json:"nullable,omitempty"`
	DefaultValue      string                     `json:"default,omitempty"`
	ProviderOverrides map[string]ColumnTransform `json:"provider_overrides,omitempty"`
}

// ClassificationContext represents governance context for a classification
type ClassificationContext struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Level        string                 `json:"level"`
	Requirements map[string]interface{} `json:"requirements"`

	// Provider-specific enforcement
	ProviderEnforcement map[string]*ProviderEnforcementRules `json:"provider_enforcement"`

	// Compliance framework mappings
	ComplianceFrameworks map[string]*ComplianceFrameworkMapping `json:"compliance_frameworks"`
}

// ProviderEnforcementRules defines how a classification should be enforced by a specific provider
type ProviderEnforcementRules struct {
	ProviderType       string            `json:"provider_type"` // postgres, kafka, s3, etc.
	EncryptionRequired bool              `json:"encryption_required"`
	EncryptionConfig   map[string]string `json:"encryption_config"`
	AccessRestrictions []string          `json:"access_restrictions"`
	AuditRequirements  []string          `json:"audit_requirements"`
	CustomRules        map[string]string `json:"custom_rules"`
}

// ComplianceFrameworkMapping maps classifications to compliance framework requirements
type ComplianceFrameworkMapping struct {
	Framework    string            `json:"framework"` // GDPR, SOX, PCI, HIPAA
	Requirements []string          `json:"requirements"`
	Controls     map[string]string `json:"controls"`
	Reporting    []string          `json:"reporting"`
}

// RoleContext represents governance context for a role
type RoleContext struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Permissions  []string               `json:"permissions"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Restrictions map[string]interface{} `json:"restrictions"`

	// Provider-specific role mappings
	ProviderRoles map[string]string `json:"provider_roles"` // provider -> role mapping
}

// PermissionContext represents governance context for a permission
type PermissionContext struct {
	Name                     string                 `json:"name"`
	Description              string                 `json:"description"`
	Actions                  []string               `json:"actions"`
	Resources                []string               `json:"resources"`
	Conditions               map[string]interface{} `json:"conditions"`
	AppliesToClassifications []string               `json:"applies_to_classifications"`
	Transformations          *TransformationConfig  `json:"transformations,omitempty"`

	// Provider-specific permission mappings
	ProviderMappings map[string]*ProviderPermissionMapping `json:"provider_mappings"`
}

// ProviderPermissionMapping defines how permissions map to provider-specific capabilities
type ProviderPermissionMapping struct {
	ProviderType     string            `json:"provider_type"`
	NativeActions    []string          `json:"native_actions"`    // Provider's native permission actions
	PolicyStatements []string          `json:"policy_statements"` // Provider-specific policy language
	Constraints      map[string]string `json:"constraints"`       // Additional constraints
}

// TransformationConfig represents data transformation rules
type TransformationConfig struct {
	Type              string            `json:"type"`
	ProviderFunctions map[string]string `json:"provider_functions,omitempty"`
	DataFilters       []DataFilter      `json:"data_filters,omitempty"`
}

// DataFilter represents a data filtering rule
type DataFilter struct {
	Column    string `json:"column"`
	Operation string `json:"operation"`
	Value     string `json:"value"`
}

// RequestGovernanceContext represents governance context for a specific request
type RequestGovernanceContext struct {
	RequestID       string                 `json:"request_id"`
	Operation       string                 `json:"operation"` // create, read, update, delete
	ResourceType    string                 `json:"resource_type"`
	ResourceName    string                 `json:"resource_name"`
	TargetProvider  string                 `json:"target_provider"`
	UserContext     *UserContext           `json:"user_context,omitempty"`
	RequestMetadata map[string]interface{} `json:"request_metadata"`

	// Governance decisions for this request
	AppliedClassifications []string              `json:"applied_classifications"`
	RequiredPermissions    []string              `json:"required_permissions"`
	EnforcedPolicies       []string              `json:"enforced_policies"`
	SecurityRequirements   *SecurityRequirements `json:"security_requirements"`
}

// UserContext represents the user making the request
type UserContext struct {
	UserID       string   `json:"user_id"`
	Username     string   `json:"username"`
	Groups       []string `json:"groups"`
	Roles        []string `json:"roles"`
	Capabilities []string `json:"capabilities"`
	SessionID    string   `json:"session_id"`
}

// SecurityRequirements represents security requirements for the request
type SecurityRequirements struct {
	EncryptionRequired  bool              `json:"encryption_required"`
	EncryptionConfig    map[string]string `json:"encryption_config"`
	AccessLogging       bool              `json:"access_logging"`
	AuditTrail          bool              `json:"audit_trail"`
	DataMasking         []string          `json:"data_masking"`          // Columns that need masking
	RowLevelSecurity    bool              `json:"row_level_security"`    // RLS required
	ColumnLevelSecurity map[string]string `json:"column_level_security"` // Column -> security level
}

// AccessControl represents an access control rule
type AccessControl struct {
	Principal    string            `json:"principal"`    // User, role, or group
	Actions      []string          `json:"actions"`      // Allowed actions
	Conditions   map[string]string `json:"conditions"`   // Access conditions
	Restrictions map[string]string `json:"restrictions"` // Access restrictions
}

// ComplianceRule represents a compliance requirement
type ComplianceRule struct {
	Framework   string            `json:"framework"`   // GDPR, SOX, PCI, etc.
	Rule        string            `json:"rule"`        // Specific rule identifier
	Description string            `json:"description"` // Human-readable description
	Controls    map[string]string `json:"controls"`    // Implementation controls
	Validation  string            `json:"validation"`  // How to validate compliance
}

// DataLineageInfo represents data lineage information
type DataLineageInfo struct {
	SourceSystems   []string          `json:"source_systems"`
	Dependencies    []string          `json:"dependencies"`
	Transformations []string          `json:"transformations"`
	Consumers       []string          `json:"consumers"`
	Metadata        map[string]string `json:"metadata"`
}

// AuditContext represents audit information for governance decisions
type AuditContext struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Timestamp    time.Time              `json:"timestamp"`
	Actor        string                 `json:"actor"`         // Who initiated the action
	Source       string                 `json:"source"`        // Source system/component
	Target       string                 `json:"target"`        // Target resource
	Action       string                 `json:"action"`        // Action performed
	Outcome      string                 `json:"outcome"`       // Success, failure, etc.
	Details      map[string]interface{} `json:"details"`       // Additional audit details
	RequestTrace string                 `json:"request_trace"` // Trace ID for correlation
}

// =============================================================================
// GOVERNANCE VALIDATION AND ENFORCEMENT
// =============================================================================

// GovernanceValidationResult represents the result of governance validation
type GovernanceValidationResult struct {
	IsCompliant     bool                       `json:"is_compliant"`
	Violations      []GovernanceViolation      `json:"violations,omitempty"`
	Warnings        []GovernanceWarning        `json:"warnings,omitempty"`
	Recommendations []GovernanceRecommendation `json:"recommendations,omitempty"`
	AppliedRules    []string                   `json:"applied_rules"`
	Metadata        map[string]interface{}     `json:"metadata,omitempty"`
}

// GovernanceViolation represents a governance rule violation
type GovernanceViolation struct {
	Rule        string `json:"rule"`
	Level       string `json:"level"` // error, warning
	Message     string `json:"message"`
	Field       string `json:"field,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Framework   string `json:"framework,omitempty"` // GDPR, SOX, etc.
	Remediation string `json:"remediation,omitempty"`
}

// GovernanceWarning represents a governance warning
type GovernanceWarning struct {
	Rule       string `json:"rule"`
	Message    string `json:"message"`
	Field      string `json:"field,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Impact     string `json:"impact,omitempty"`
}

// GovernanceRecommendation represents a governance recommendation
type GovernanceRecommendation struct {
	Type      string `json:"type"` // security, compliance, performance
	Message   string `json:"message"`
	Rationale string `json:"rationale"`
	Priority  string `json:"priority"` // high, medium, low
	Action    string `json:"action"`   // suggested action
	Framework string `json:"framework,omitempty"`
}

// GovernanceCapabilities represents the governance capabilities of a provider
type GovernanceCapabilities struct {
	SupportsEncryption       bool     `json:"supports_encryption"`
	EncryptionMethods        []string `json:"encryption_methods"`
	SupportsAccessControls   bool     `json:"supports_access_controls"`
	SupportsAuditLogging     bool     `json:"supports_audit_logging"`
	SupportsDataMasking      bool     `json:"supports_data_masking"`
	SupportsRowLevelSecurity bool     `json:"supports_row_level_security"`
	SupportedCompliance      []string `json:"supported_compliance"` // GDPR, SOX, PCI, etc.
	CustomGovernanceFeatures []string `json:"custom_governance_features"`
	GovernanceVersion        string   `json:"governance_version"`
}

// =============================================================================
// SDK HELPER FUNCTIONS FOR EXTERNAL PROVIDERS
// =============================================================================

// GovernanceHelper provides utility functions for external providers
type GovernanceHelper struct {
	providerType string
	capabilities *GovernanceCapabilities
}

// NewGovernanceHelper creates a new governance helper for a provider
func NewGovernanceHelper(providerType string, capabilities *GovernanceCapabilities) *GovernanceHelper {
	return &GovernanceHelper{
		providerType: providerType,
		capabilities: capabilities,
	}
}

// ExtractGovernanceRequirements extracts governance requirements for a resource
func (gh *GovernanceHelper) ExtractGovernanceRequirements(
	ctx context.Context,
	resourceType string,
	config map[string]interface{},
	governanceCtx *GovernanceContext,
) (*ResourceGovernanceRequirements, error) {

	requirements := &ResourceGovernanceRequirements{
		ResourceType:       resourceType,
		EncryptionRequired: false,
		EncryptionConfig:   make(map[string]string),
		AccessControls:     []AccessControl{},
		ComplianceRules:    []ComplianceRule{},
		AuditRequirements:  []string{},
	}

	// Extract column-level requirements
	if columns, ok := config["columns"]; ok {
		if err := gh.extractColumnRequirements(columns, governanceCtx, requirements); err != nil {
			return nil, err
		}
	}

	// Extract classification-based requirements
	if classifications, ok := config["classifications"]; ok {
		if err := gh.extractClassificationRequirements(classifications, governanceCtx, requirements); err != nil {
			return nil, err
		}
	}

	// Extract provider-specific enforcement rules
	if err := gh.extractProviderSpecificRules(governanceCtx, requirements); err != nil {
		return nil, err
	}

	return requirements, nil
}

// ResourceGovernanceRequirements represents governance requirements for a specific resource
type ResourceGovernanceRequirements struct {
	ResourceType       string            `json:"resource_type"`
	EncryptionRequired bool              `json:"encryption_required"`
	EncryptionConfig   map[string]string `json:"encryption_config"`
	AccessControls     []AccessControl   `json:"access_controls"`
	ComplianceRules    []ComplianceRule  `json:"compliance_rules"`
	AuditRequirements  []string          `json:"audit_requirements"`
	CustomRules        map[string]string `json:"custom_rules"`

	// Column-specific requirements
	ColumnRequirements map[string]*ColumnGovernanceRequirements `json:"column_requirements"`
}

// ColumnGovernanceRequirements represents governance requirements for a specific column
type ColumnGovernanceRequirements struct {
	Name             string   `json:"name"`
	EncryptionMethod string   `json:"encryption_method,omitempty"`
	AccessLevel      string   `json:"access_level"`
	MaskingRule      string   `json:"masking_rule,omitempty"`
	ComplianceFlags  []string `json:"compliance_flags"`
	AuditRequired    bool     `json:"audit_required"`
}

// extractColumnRequirements extracts governance requirements from column definitions
func (gh *GovernanceHelper) extractColumnRequirements(
	columns interface{},
	governanceCtx *GovernanceContext,
	requirements *ResourceGovernanceRequirements,
) error {

	requirements.ColumnRequirements = make(map[string]*ColumnGovernanceRequirements)

	// Parse columns (this would be more sophisticated in a real implementation)
	if colBytes, err := json.Marshal(columns); err == nil {
		var columnConfigs []ColumnContext
		if err := json.Unmarshal(colBytes, &columnConfigs); err == nil {
			for _, col := range columnConfigs {
				colReq := &ColumnGovernanceRequirements{
					Name:            col.Name,
					AccessLevel:     col.AccessLevel,
					ComplianceFlags: col.ComplianceFlags,
				}

				if col.EncryptionMethod != "" {
					colReq.EncryptionMethod = col.EncryptionMethod
					requirements.EncryptionRequired = true
				}

				if col.MaskingRule != "" {
					colReq.MaskingRule = col.MaskingRule
				}

				// Check if audit is required for this column's classifications
				colReq.AuditRequired = gh.requiresAudit(col.Classifications, governanceCtx)

				requirements.ColumnRequirements[col.Name] = colReq
			}
		}
	}

	return nil
}

// extractClassificationRequirements extracts requirements based on classifications
func (gh *GovernanceHelper) extractClassificationRequirements(
	classifications interface{},
	governanceCtx *GovernanceContext,
	requirements *ResourceGovernanceRequirements,
) error {

	if classArray, ok := classifications.([]interface{}); ok {
		for _, c := range classArray {
			if cStr, ok := c.(string); ok {
				if classCtx, exists := governanceCtx.Classifications[cStr]; exists {
					// Extract provider-specific enforcement rules
					if enforcement, exists := classCtx.ProviderEnforcement[gh.providerType]; exists {
						if enforcement.EncryptionRequired {
							requirements.EncryptionRequired = true
							for k, v := range enforcement.EncryptionConfig {
								requirements.EncryptionConfig[k] = v
							}
						}

						requirements.AuditRequirements = append(requirements.AuditRequirements, enforcement.AuditRequirements...)
					}
				}
			}
		}
	}

	return nil
}

// extractProviderSpecificRules extracts provider-specific governance rules
func (gh *GovernanceHelper) extractProviderSpecificRules(
	governanceCtx *GovernanceContext,
	requirements *ResourceGovernanceRequirements,
) error {

	// Add provider-specific rules based on governance context
	// This would be customized for each provider type

	return nil
}

// requiresAudit checks if classifications require audit logging
func (gh *GovernanceHelper) requiresAudit(classifications []string, governanceCtx *GovernanceContext) bool {
	for _, classification := range classifications {
		if classCtx, exists := governanceCtx.Classifications[classification]; exists {
			if classCtx.Level == "confidential" || classCtx.Level == "restricted" || classCtx.Level == "secret" {
				return true
			}
		}
	}
	return false
}

// ValidateComplianceFramework validates compliance with a specific framework
func (gh *GovernanceHelper) ValidateComplianceFramework(
	framework string,
	config map[string]interface{},
	governanceCtx *GovernanceContext,
) (*ComplianceValidationResult, error) {

	result := &ComplianceValidationResult{
		Framework:   framework,
		IsCompliant: true,
		Violations:  []ComplianceViolation{},
		Controls:    []ComplianceControl{},
	}

	// Validate based on framework requirements
	switch framework {
	case "GDPR":
		if err := gh.validateGDPRCompliance(config, governanceCtx, result); err != nil {
			return nil, err
		}
	case "SOX":
		if err := gh.validateSOXCompliance(config, governanceCtx, result); err != nil {
			return nil, err
		}
	case "PCI":
		if err := gh.validatePCICompliance(config, governanceCtx, result); err != nil {
			return nil, err
		}
	case "HIPAA":
		if err := gh.validateHIPAACompliance(config, governanceCtx, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// ComplianceValidationResult represents the result of compliance validation
type ComplianceValidationResult struct {
	Framework   string                `json:"framework"`
	IsCompliant bool                  `json:"is_compliant"`
	Violations  []ComplianceViolation `json:"violations"`
	Controls    []ComplianceControl   `json:"controls"`
	Score       float64               `json:"score"` // Compliance score 0-100
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // high, medium, low
	Remediation string `json:"remediation"`
}

// ComplianceControl represents an implemented compliance control
type ComplianceControl struct {
	Control     string `json:"control"`
	Description string `json:"description"`
	Status      string `json:"status"` // implemented, not_implemented, partial
}

// Framework-specific validation functions
func (gh *GovernanceHelper) validateGDPRCompliance(
	config map[string]interface{},
	governanceCtx *GovernanceContext,
	result *ComplianceValidationResult,
) error {
	// GDPR-specific validation logic
	return nil
}

func (gh *GovernanceHelper) validateSOXCompliance(
	config map[string]interface{},
	governanceCtx *GovernanceContext,
	result *ComplianceValidationResult,
) error {
	// SOX-specific validation logic
	return nil
}

func (gh *GovernanceHelper) validatePCICompliance(
	config map[string]interface{},
	governanceCtx *GovernanceContext,
	result *ComplianceValidationResult,
) error {
	// PCI-specific validation logic
	return nil
}

func (gh *GovernanceHelper) validateHIPAACompliance(
	config map[string]interface{},
	governanceCtx *GovernanceContext,
	result *ComplianceValidationResult,
) error {
	// HIPAA-specific validation logic
	return nil
}

// =============================================================================
// GOVERNANCE ENFORCEMENT HELPERS
// =============================================================================

// ApplyEncryptionRules applies encryption rules to a resource configuration
func (gh *GovernanceHelper) ApplyEncryptionRules(
	config map[string]interface{},
	requirements *ResourceGovernanceRequirements,
) (map[string]interface{}, error) {

	if !requirements.EncryptionRequired {
		return config, nil
	}

	// Apply provider-specific encryption rules
	updatedConfig := make(map[string]interface{})
	for k, v := range config {
		updatedConfig[k] = v
	}

	// Add encryption configuration
	if len(requirements.EncryptionConfig) > 0 {
		updatedConfig["encryption"] = requirements.EncryptionConfig
	}

	// Apply column-level encryption
	if requirements.ColumnRequirements != nil {
		if err := gh.applyColumnEncryption(updatedConfig, requirements.ColumnRequirements); err != nil {
			return nil, err
		}
	}

	return updatedConfig, nil
}

// applyColumnEncryption applies column-level encryption rules
func (gh *GovernanceHelper) applyColumnEncryption(
	config map[string]interface{},
	columnRequirements map[string]*ColumnGovernanceRequirements,
) error {

	// Apply column-level encryption based on requirements
	// Implementation would be provider-specific

	return nil
}

// ApplyAccessControls applies access control rules to a resource configuration
func (gh *GovernanceHelper) ApplyAccessControls(
	config map[string]interface{},
	accessControls []AccessControl,
) (map[string]interface{}, error) {

	// Apply provider-specific access control rules
	// Implementation would be provider-specific

	return config, nil
}

// GenerateAuditEvent generates an audit event for governance actions
func (gh *GovernanceHelper) GenerateAuditEvent(
	ctx context.Context,
	action string,
	resource string,
	outcome string,
	details map[string]interface{},
) *AuditEvent {

	return &AuditEvent{
		EventID:      generateEventID(),
		EventType:    "governance_enforcement",
		Timestamp:    time.Now(),
		ProviderType: gh.providerType,
		Action:       action,
		Resource:     resource,
		Outcome:      outcome,
		Details:      details,
	}
}

// AuditEvent represents a governance audit event
type AuditEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Timestamp    time.Time              `json:"timestamp"`
	ProviderType string                 `json:"provider_type"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	Outcome      string                 `json:"outcome"`
	Details      map[string]interface{} `json:"details"`
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("gov_evt_%d", time.Now().UnixNano())
}

// =============================================================================
// EXAMPLE IMPLEMENTATION FOR EXTERNAL PROVIDERS
// =============================================================================

/*
Example usage in an external provider (e.g., PostgreSQL provider):

```go
package main

import (
	"context"
	"github.com/schemabounce/kolumn-sdk/core"
)

type PostgreSQLProvider struct {
	governanceHelper *core.GovernanceHelper
	governanceCtx    *core.GovernanceContext
}

func (p *PostgreSQLProvider) ConfigureGovernance(ctx context.Context, governanceCtx *core.GovernanceContext) error {
	p.governanceCtx = governanceCtx

	capabilities := &core.GovernanceCapabilities{
		SupportsEncryption:       true,
		EncryptionMethods:        []string{"column_encryption", "transparent_encryption"},
		SupportsAccessControls:   true,
		SupportsAuditLogging:     true,
		SupportsDataMasking:      true,
		SupportsRowLevelSecurity: true,
		SupportedCompliance:      []string{"GDPR", "SOX", "HIPAA"},
		GovernanceVersion:        "v1.0",
	}

	p.governanceHelper = core.NewGovernanceHelper("postgres", capabilities)

	return nil
}

func (p *PostgreSQLProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
	switch function {
	case "CreateResource":
		return p.createResourceWithGovernance(ctx, input)
	default:
		return p.baseCallFunction(ctx, function, input)
	}
}

func (p *PostgreSQLProvider) createResourceWithGovernance(ctx context.Context, input []byte) ([]byte, error) {
	var req CreateResourceRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	// Extract governance requirements
	requirements, err := p.governanceHelper.ExtractGovernanceRequirements(
		ctx, req.ResourceType, req.Config, p.governanceCtx)
	if err != nil {
		return nil, err
	}

	// Apply encryption rules
	updatedConfig, err := p.governanceHelper.ApplyEncryptionRules(req.Config, requirements)
	if err != nil {
		return nil, err
	}

	// Apply access controls
	finalConfig, err := p.governanceHelper.ApplyAccessControls(updatedConfig, requirements.AccessControls)
	if err != nil {
		return nil, err
	}

	// Create the resource with governance-enhanced configuration
	result := p.createPostgreSQLTable(finalConfig, requirements)

	// Generate audit event
	auditEvent := p.governanceHelper.GenerateAuditEvent(
		ctx, "create_table", req.Name, "success",
		map[string]interface{}{
			"encryption_applied": requirements.EncryptionRequired,
			"compliance_frameworks": requirements.ComplianceRules,
		})

	// Send audit event to governance system
	p.sendAuditEvent(auditEvent)

	return json.Marshal(result)
}
```

This example shows how external providers can:
1. Receive governance context from Kolumn
2. Extract governance requirements for their resources
3. Apply encryption, access controls, and compliance rules
4. Generate audit events for governance actions
5. Integrate seamlessly with Kolumn's governance ecosystem
*/

// =============================================================================
// GOVERNANCE MIDDLEWARE - KOLUMN SDK INTEGRATION
// =============================================================================

// ColumnGovernanceMetadata represents governance metadata for a specific column
type ColumnGovernanceMetadata struct {
	Name               string          `json:"name"`
	Classifications    []string        `json:"classifications"`
	EncryptionRequired bool            `json:"encryption_required"`
	AuditRequired      bool            `json:"audit_required"`
	AccessLevel        string          `json:"access_level"`
	RetentionPolicy    string          `json:"retention_policy"`
	ComplianceFlags    map[string]bool `json:"compliance_flags"`
}

// GovernanceMiddleware manages governance context throughout the SDK request lifecycle
type GovernanceMiddleware struct {
	context        *GovernanceContext
	columnMetadata map[string]*ColumnGovernanceMetadata
	frameworks     []string
	hasContext     bool
}

// NewGovernanceMiddleware creates a new governance middleware instance
func NewGovernanceMiddleware() *GovernanceMiddleware {
	return &GovernanceMiddleware{
		columnMetadata: make(map[string]*ColumnGovernanceMetadata),
		frameworks:     make([]string, 0),
		hasContext:     false,
	}
}

// ExtractGovernanceFromRequest extracts governance context from RPC request metadata
func (gm *GovernanceMiddleware) ExtractGovernanceFromRequest(metadata map[string]interface{}) error {
	if metadata == nil {
		gm.hasContext = false
		return nil
	}

	// Extract governance context if present
	if govData, exists := metadata["governance_context"]; exists {
		if err := gm.parseGovernanceContext(govData); err != nil {
			return fmt.Errorf("failed to parse governance context: %w", err)
		}
		gm.hasContext = true
	}

	return nil
}

// HasGovernanceContext returns true if governance context is available
func (gm *GovernanceMiddleware) HasGovernanceContext() bool {
	return gm.hasContext
}

// GetColumnGovernance returns governance metadata for a specific column
func (gm *GovernanceMiddleware) GetColumnGovernance(columnName string) (*ColumnGovernanceMetadata, error) {
	if !gm.hasContext {
		return nil, fmt.Errorf("no governance context available")
	}

	if columnMeta, exists := gm.columnMetadata[columnName]; exists {
		return columnMeta, nil
	}

	// Create default column metadata
	return &ColumnGovernanceMetadata{
		Name:               columnName,
		Classifications:    []string{},
		EncryptionRequired: false,
		AuditRequired:      false,
		AccessLevel:        "public",
		RetentionPolicy:    "",
		ComplianceFlags:    make(map[string]bool),
	}, nil
}

// GetRequiredComplianceFrameworks returns the list of required compliance frameworks
func (gm *GovernanceMiddleware) GetRequiredComplianceFrameworks() []string {
	if !gm.hasContext {
		return []string{}
	}
	return gm.frameworks
}

// parseGovernanceContext parses governance context from request metadata
func (gm *GovernanceMiddleware) parseGovernanceContext(govData interface{}) error {
	// In a real implementation, this would parse the governance context
	// For now, we'll create a stub that handles basic scenarios

	if govMap, ok := govData.(map[string]interface{}); ok {
		// Extract compliance frameworks
		if frameworks, exists := govMap["compliance_frameworks"]; exists {
			if frameworkList, ok := frameworks.([]interface{}); ok {
				gm.frameworks = make([]string, 0, len(frameworkList))
				for _, fw := range frameworkList {
					if fwStr, ok := fw.(string); ok {
						gm.frameworks = append(gm.frameworks, fwStr)
					}
				}
			}
		}

		// Extract column metadata
		if columns, exists := govMap["columns"]; exists {
			if columnList, ok := columns.([]interface{}); ok {
				for _, col := range columnList {
					if columnMap, ok := col.(map[string]interface{}); ok {
						columnMeta := &ColumnGovernanceMetadata{
							Classifications: []string{},
							ComplianceFlags: make(map[string]bool),
						}

						if name, exists := columnMap["name"]; exists {
							if nameStr, ok := name.(string); ok {
								columnMeta.Name = nameStr
							}
						}

						if classifications, exists := columnMap["classifications"]; exists {
							if classList, ok := classifications.([]interface{}); ok {
								for _, cls := range classList {
									if clsStr, ok := cls.(string); ok {
										columnMeta.Classifications = append(columnMeta.Classifications, clsStr)
									}
								}
							}
						}

						if encryption, exists := columnMap["encryption_required"]; exists {
							if encBool, ok := encryption.(bool); ok {
								columnMeta.EncryptionRequired = encBool
							}
						}

						if audit, exists := columnMap["audit_required"]; exists {
							if auditBool, ok := audit.(bool); ok {
								columnMeta.AuditRequired = auditBool
							}
						}

						if access, exists := columnMap["access_level"]; exists {
							if accessStr, ok := access.(string); ok {
								columnMeta.AccessLevel = accessStr
							}
						}

						if retention, exists := columnMap["retention_policy"]; exists {
							if retentionStr, ok := retention.(string); ok {
								columnMeta.RetentionPolicy = retentionStr
							}
						}

						if compliance, exists := columnMap["compliance_flags"]; exists {
							if complianceMap, ok := compliance.(map[string]interface{}); ok {
								for flag, value := range complianceMap {
									if valueBool, ok := value.(bool); ok {
										columnMeta.ComplianceFlags[flag] = valueBool
									}
								}
							}
						}

						if columnMeta.Name != "" {
							gm.columnMetadata[columnMeta.Name] = columnMeta
						}
					}
				}
			}
		}
	}

	return nil
}
