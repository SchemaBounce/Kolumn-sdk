package governance

import (
	"fmt"
	"strings"

	"github.com/schemabounce/kolumn/sdk/core"
)

// =============================================================================
// UNIVERSAL GOVERNANCE ABSTRACTIONS
// =============================================================================

// AuditScope defines the scope of audit logging required
type AuditScope string

const (
	AuditScopeNone     AuditScope = "none"     // No auditing required
	AuditScopeMetadata AuditScope = "metadata" // Log metadata only (who, when)
	AuditScopeRead     AuditScope = "read"     // Log read operations
	AuditScopeWrite    AuditScope = "write"    // Log write operations
	AuditScopeFull     AuditScope = "full"     // Log all operations with details
)

// AccessTier defines the access control tier required
type AccessTier string

const (
	AccessTierPublic       AccessTier = "public"       // Unrestricted access
	AccessTierInternal     AccessTier = "internal"     // Organization-wide access
	AccessTierRestricted   AccessTier = "restricted"   // Department/team access
	AccessTierConfidential AccessTier = "confidential" // Named individuals only
	AccessTierSecret       AccessTier = "secret"       // Special authorization required
)

// GovernanceRequirement represents universal governance requirements
// This abstraction can be interpreted by ANY provider using its native capabilities
type GovernanceRequirement struct {
	ProtectionLevel int                    `json:"protection_level"`   // 0-10 scale
	AuditScope      AuditScope             `json:"audit_scope"`        // none/metadata/read/write/full
	AccessTier      AccessTier             `json:"access_tier"`        // public to secret
	RetentionDays   int                    `json:"retention_days"`     // -1 for permanent, 0 for immediate deletion allowed
	ComplianceFlags map[string]bool        `json:"compliance_flags"`   // Universal compliance requirements
	Metadata        map[string]interface{} `json:"metadata,omitempty"` // Additional provider-agnostic metadata
}

// ValidationIssue represents a governance compliance validation issue
type ValidationIssue struct {
	Identifier    string `json:"identifier"`     // Column, resource, or other identifier
	RequiredLevel int    `json:"required_level"` // Required protection/audit level
	ActualLevel   int    `json:"actual_level"`   // Actual protection/audit level
	IssueType     string `json:"issue_type"`     // "protection", "audit", "access", "retention", "compliance"
	Severity      string `json:"severity"`       // "critical", "high", "medium", "low"
	Message       string `json:"message"`        // Human-readable description
	Passed        bool   `json:"passed"`         // Whether the check passed
}

// =============================================================================
// UNIVERSAL GOVERNANCE HELPER
// =============================================================================

// GovernanceHelper provides universal, provider-agnostic governance functionality
type GovernanceHelper struct {
	middleware *core.GovernanceMiddleware
}

// NewGovernanceHelper creates a new universal governance helper instance
func NewGovernanceHelper() *GovernanceHelper {
	return &GovernanceHelper{
		middleware: core.NewGovernanceMiddleware(),
	}
}

// ExtractFromRequest extracts governance context from RPC request metadata
func (gh *GovernanceHelper) ExtractFromRequest(requestMetadata map[string]interface{}) error {
	return gh.middleware.ExtractGovernanceFromRequest(requestMetadata)
}

// HasGovernance returns true if governance context is available
func (gh *GovernanceHelper) HasGovernance() bool {
	return gh.middleware.HasGovernanceContext()
}

// =============================================================================
// UNIVERSAL REQUIREMENT EXTRACTION
// =============================================================================

// GetRequirements returns abstract governance requirements for any identifier
// The identifier can be a column name, resource name, or any other entity
func (gh *GovernanceHelper) GetRequirements(identifier string) (*GovernanceRequirement, error) {
	if !gh.HasGovernance() {
		return &GovernanceRequirement{
			ProtectionLevel: 0,
			AuditScope:      AuditScopeNone,
			AccessTier:      AccessTierPublic,
			RetentionDays:   0,
			ComplianceFlags: make(map[string]bool),
		}, nil
	}

	// Start with default requirements
	requirement := &GovernanceRequirement{
		ProtectionLevel: 0,
		AuditScope:      AuditScopeNone,
		AccessTier:      AccessTierPublic,
		RetentionDays:   0,
		ComplianceFlags: make(map[string]bool),
		Metadata:        make(map[string]interface{}),
	}

	// Try to find specific column governance
	if columnGov, err := gh.middleware.GetColumnGovernance(identifier); err == nil {
		// Map specific governance to abstract levels
		requirement.ProtectionLevel = gh.mapToProtectionLevel(columnGov)
		requirement.AuditScope = gh.mapToAuditScope(columnGov)
		requirement.AccessTier = gh.mapToAccessTier(columnGov.AccessLevel)
		requirement.RetentionDays = gh.parseRetentionDays(columnGov.RetentionPolicy)
		requirement.ComplianceFlags = gh.extractComplianceFlags(columnGov)
	}

	// Apply global compliance requirements
	frameworks := gh.middleware.GetRequiredComplianceFrameworks()
	for _, framework := range frameworks {
		gh.applyFrameworkRequirements(requirement, framework)
	}

	return requirement, nil
}

// GetProtectionLevel returns the protection level (0-10) for an identifier
func (gh *GovernanceHelper) GetProtectionLevel(identifier string) int {
	req, err := gh.GetRequirements(identifier)
	if err != nil {
		return 0
	}
	return req.ProtectionLevel
}

// GetAuditScope returns the audit scope requirement for an identifier
func (gh *GovernanceHelper) GetAuditScope(identifier string) AuditScope {
	req, err := gh.GetRequirements(identifier)
	if err != nil {
		return AuditScopeNone
	}
	return req.AuditScope
}

// GetAccessTier returns the access tier requirement for an identifier
func (gh *GovernanceHelper) GetAccessTier(identifier string) AccessTier {
	req, err := gh.GetRequirements(identifier)
	if err != nil {
		return AccessTierPublic
	}
	return req.AccessTier
}

// GetRetentionDuration returns the retention requirement in days
func (gh *GovernanceHelper) GetRetentionDuration(identifier string) int {
	req, err := gh.GetRequirements(identifier)
	if err != nil {
		return 0
	}
	return req.RetentionDays
}

// GetComplianceFlags returns universal compliance requirement flags
func (gh *GovernanceHelper) GetComplianceFlags() map[string]bool {
	if !gh.HasGovernance() {
		return make(map[string]bool)
	}

	flags := make(map[string]bool)
	frameworks := gh.middleware.GetRequiredComplianceFrameworks()

	for _, framework := range frameworks {
		switch strings.ToUpper(framework) {
		case "GDPR":
			flags["erasure_capable"] = true
			flags["audit_trail"] = true
			flags["encryption_at_rest"] = true
			flags["pseudonymization"] = true
		case "SOX":
			flags["audit_trail"] = true
			flags["access_logging"] = true
			flags["change_tracking"] = true
			flags["backup_verification"] = true
		case "PCI":
			flags["encryption_at_rest"] = true
			flags["encryption_in_transit"] = true
			flags["access_logging"] = true
			flags["network_segmentation"] = true
		case "HIPAA":
			flags["encryption_at_rest"] = true
			flags["encryption_in_transit"] = true
			flags["audit_trail"] = true
			flags["minimum_necessary"] = true
		}
	}

	return flags
}

// =============================================================================
// UNIVERSAL VALIDATION
// =============================================================================

// ValidateCompliance checks if a configuration meets governance requirements
// This is completely provider-agnostic - it only validates against abstract levels
func (gh *GovernanceHelper) ValidateCompliance(config map[string]interface{}, requirements *GovernanceRequirement) []ValidationIssue {
	issues := make([]ValidationIssue, 0)

	// Validate protection level
	if actualLevel := gh.extractActualProtectionLevel(config); actualLevel < requirements.ProtectionLevel {
		issues = append(issues, ValidationIssue{
			Identifier:    getConfigIdentifier(config),
			RequiredLevel: requirements.ProtectionLevel,
			ActualLevel:   actualLevel,
			IssueType:     "protection",
			Severity:      gh.getProtectionSeverity(requirements.ProtectionLevel),
			Message:       fmt.Sprintf("Protection level %d required but only %d provided", requirements.ProtectionLevel, actualLevel),
			Passed:        false,
		})
	}

	// Validate audit scope
	if actualScope := gh.extractActualAuditScope(config); !gh.isAuditScopeAdequate(actualScope, requirements.AuditScope) {
		issues = append(issues, ValidationIssue{
			Identifier:    getConfigIdentifier(config),
			RequiredLevel: int(gh.auditScopeToLevel(requirements.AuditScope)),
			ActualLevel:   int(gh.auditScopeToLevel(actualScope)),
			IssueType:     "audit",
			Severity:      gh.getAuditSeverity(requirements.AuditScope),
			Message:       fmt.Sprintf("Audit scope '%s' required but only '%s' provided", requirements.AuditScope, actualScope),
			Passed:        false,
		})
	}

	// Validate access tier
	if actualTier := gh.extractActualAccessTier(config); !gh.isAccessTierAdequate(actualTier, requirements.AccessTier) {
		issues = append(issues, ValidationIssue{
			Identifier:    getConfigIdentifier(config),
			RequiredLevel: int(gh.accessTierToLevel(requirements.AccessTier)),
			ActualLevel:   int(gh.accessTierToLevel(actualTier)),
			IssueType:     "access",
			Severity:      gh.getAccessSeverity(requirements.AccessTier),
			Message:       fmt.Sprintf("Access tier '%s' required but only '%s' provided", requirements.AccessTier, actualTier),
			Passed:        false,
		})
	}

	// Validate compliance flags
	for flag, required := range requirements.ComplianceFlags {
		if required && !gh.hasComplianceCapability(config, flag) {
			issues = append(issues, ValidationIssue{
				Identifier:    getConfigIdentifier(config),
				RequiredLevel: 1,
				ActualLevel:   0,
				IssueType:     "compliance",
				Severity:      "high",
				Message:       fmt.Sprintf("Compliance capability '%s' is required but not provided", flag),
				Passed:        false,
			})
		}
	}

	return issues
}

// ValidateAllRequirements validates all identifiers in a configuration against their requirements
func (gh *GovernanceHelper) ValidateAllRequirements(config map[string]interface{}) []ValidationIssue {
	allIssues := make([]ValidationIssue, 0)

	// Validate columns if present
	if columns, exists := config["columns"]; exists {
		if columnList, ok := columns.([]interface{}); ok {
			for _, col := range columnList {
				if columnMap, ok := col.(map[string]interface{}); ok {
					if columnName, exists := columnMap["name"]; exists {
						if columnNameStr, ok := columnName.(string); ok {
							requirements, err := gh.GetRequirements(columnNameStr)
							if err == nil {
								issues := gh.ValidateCompliance(columnMap, requirements)
								allIssues = append(allIssues, issues...)
							}
						}
					}
				}
			}
		}
	}

	// Validate resource-level requirements
	if resourceName, exists := config["name"]; exists {
		if resourceNameStr, ok := resourceName.(string); ok {
			requirements, err := gh.GetRequirements(resourceNameStr)
			if err == nil {
				issues := gh.ValidateCompliance(config, requirements)
				allIssues = append(allIssues, issues...)
			}
		}
	}

	return allIssues
}

// =============================================================================
// MAPPING FUNCTIONS - GOVERNANCE TO ABSTRACT LEVELS
// =============================================================================

// mapToProtectionLevel maps specific governance metadata to abstract protection level (0-10)
func (gh *GovernanceHelper) mapToProtectionLevel(columnGov *core.ColumnGovernanceMetadata) int {
	level := 0

	// Base level from encryption requirement
	if columnGov.EncryptionRequired {
		level = 7 // High protection by default for encryption
	}

	// Adjust based on classifications
	for _, classification := range columnGov.Classifications {
		switch strings.ToLower(classification) {
		case "public":
			if level < 1 {
				level = 1
			}
		case "internal":
			if level < 3 {
				level = 3
			}
		case "confidential", "pii", "personal":
			if level < 7 {
				level = 7
			}
		case "secret", "top_secret", "restricted":
			if level < 9 {
				level = 9
			}
		}
	}

	// Adjust based on compliance flags
	for framework := range columnGov.ComplianceFlags {
		switch strings.ToUpper(framework) {
		case "PCI", "HIPAA":
			if level < 8 {
				level = 8
			}
		case "SOX", "GDPR":
			if level < 6 {
				level = 6
			}
		}
	}

	return level
}

// mapToAuditScope maps specific governance metadata to abstract audit scope
func (gh *GovernanceHelper) mapToAuditScope(columnGov *core.ColumnGovernanceMetadata) AuditScope {
	if !columnGov.AuditRequired {
		return AuditScopeNone
	}

	// Determine scope based on classifications and compliance
	hasHighClassification := false
	for _, classification := range columnGov.Classifications {
		switch strings.ToLower(classification) {
		case "confidential", "secret", "pii", "personal":
			hasHighClassification = true
		}
	}

	if hasHighClassification {
		return AuditScopeFull
	}

	// Check compliance requirements
	for framework := range columnGov.ComplianceFlags {
		switch strings.ToUpper(framework) {
		case "SOX", "PCI", "HIPAA":
			return AuditScopeFull
		case "GDPR":
			return AuditScopeWrite
		}
	}

	return AuditScopeMetadata
}

// mapToAccessTier maps access level to abstract access tier
func (gh *GovernanceHelper) mapToAccessTier(accessLevel string) AccessTier {
	switch strings.ToLower(accessLevel) {
	case "public":
		return AccessTierPublic
	case "internal", "organization":
		return AccessTierInternal
	case "restricted", "department", "team":
		return AccessTierRestricted
	case "confidential", "private":
		return AccessTierConfidential
	case "secret", "top_secret":
		return AccessTierSecret
	default:
		return AccessTierPublic
	}
}

// parseRetentionDays parses retention policy to days
func (gh *GovernanceHelper) parseRetentionDays(retentionPolicy string) int {
	if retentionPolicy == "" {
		return 0
	}

	policy := strings.ToLower(retentionPolicy)
	switch {
	case strings.Contains(policy, "permanent") || strings.Contains(policy, "forever"):
		return -1
	case strings.Contains(policy, "immediate"):
		return 0
	case strings.Contains(policy, "1 year") || strings.Contains(policy, "365"):
		return 365
	case strings.Contains(policy, "7 years") || strings.Contains(policy, "2555"):
		return 2555
	case strings.Contains(policy, "30 days") || strings.Contains(policy, "month"):
		return 30
	case strings.Contains(policy, "90 days") || strings.Contains(policy, "quarter"):
		return 90
	default:
		return 365 // Default to 1 year
	}
}

// extractComplianceFlags extracts compliance flags from column governance
func (gh *GovernanceHelper) extractComplianceFlags(columnGov *core.ColumnGovernanceMetadata) map[string]bool {
	flags := make(map[string]bool)

	if columnGov.EncryptionRequired {
		flags["encryption_at_rest"] = true
	}

	if columnGov.AuditRequired {
		flags["audit_trail"] = true
	}

	for framework := range columnGov.ComplianceFlags {
		switch strings.ToUpper(framework) {
		case "GDPR":
			flags["erasure_capable"] = true
			flags["pseudonymization"] = true
		case "SOX":
			flags["change_tracking"] = true
			flags["access_logging"] = true
		case "PCI":
			flags["network_segmentation"] = true
		case "HIPAA":
			flags["minimum_necessary"] = true
		}
	}

	return flags
}

// applyFrameworkRequirements applies compliance framework requirements to the requirement
func (gh *GovernanceHelper) applyFrameworkRequirements(requirement *GovernanceRequirement, framework string) {
	switch strings.ToUpper(framework) {
	case "GDPR":
		if requirement.ProtectionLevel < 6 {
			requirement.ProtectionLevel = 6
		}
		if requirement.AuditScope == AuditScopeNone {
			requirement.AuditScope = AuditScopeWrite
		}
		requirement.ComplianceFlags["erasure_capable"] = true
		requirement.ComplianceFlags["audit_trail"] = true

	case "SOX":
		if requirement.ProtectionLevel < 5 {
			requirement.ProtectionLevel = 5
		}
		if requirement.AuditScope == AuditScopeNone {
			requirement.AuditScope = AuditScopeFull
		}
		requirement.ComplianceFlags["change_tracking"] = true
		requirement.ComplianceFlags["access_logging"] = true

	case "PCI":
		if requirement.ProtectionLevel < 8 {
			requirement.ProtectionLevel = 8
		}
		if requirement.AuditScope == AuditScopeNone {
			requirement.AuditScope = AuditScopeFull
		}
		requirement.ComplianceFlags["encryption_at_rest"] = true
		requirement.ComplianceFlags["encryption_in_transit"] = true

	case "HIPAA":
		if requirement.ProtectionLevel < 8 {
			requirement.ProtectionLevel = 8
		}
		if requirement.AuditScope == AuditScopeNone {
			requirement.AuditScope = AuditScopeFull
		}
		requirement.ComplianceFlags["minimum_necessary"] = true
		requirement.ComplianceFlags["audit_trail"] = true
	}
}

// =============================================================================
// VALIDATION HELPER FUNCTIONS
// =============================================================================

// extractActualProtectionLevel extracts the actual protection level from configuration
func (gh *GovernanceHelper) extractActualProtectionLevel(config map[string]interface{}) int {
	level := 0

	// Check for encryption indicators
	if encrypted, exists := config["encrypted"]; exists && encrypted.(bool) {
		level = 7
	}

	// Check for protection level directly
	if protectionLevel, exists := config["protection_level"]; exists {
		if levelInt, ok := protectionLevel.(int); ok {
			level = levelInt
		}
	}

	return level
}

// extractActualAuditScope extracts the actual audit scope from configuration
func (gh *GovernanceHelper) extractActualAuditScope(config map[string]interface{}) AuditScope {
	if audited, exists := config["audited"]; exists && audited.(bool) {
		return AuditScopeFull
	}

	if auditScope, exists := config["audit_scope"]; exists {
		if scopeStr, ok := auditScope.(string); ok {
			return AuditScope(scopeStr)
		}
	}

	return AuditScopeNone
}

// extractActualAccessTier extracts the actual access tier from configuration
func (gh *GovernanceHelper) extractActualAccessTier(config map[string]interface{}) AccessTier {
	if accessLevel, exists := config["access_level"]; exists {
		if levelStr, ok := accessLevel.(string); ok {
			return gh.mapToAccessTier(levelStr)
		}
	}

	return AccessTierPublic
}

// isAuditScopeAdequate checks if actual audit scope meets requirements
func (gh *GovernanceHelper) isAuditScopeAdequate(actual, required AuditScope) bool {
	actualLevel := gh.auditScopeToLevel(actual)
	requiredLevel := gh.auditScopeToLevel(required)
	return actualLevel >= requiredLevel
}

// isAccessTierAdequate checks if actual access tier meets requirements
func (gh *GovernanceHelper) isAccessTierAdequate(actual, required AccessTier) bool {
	actualLevel := gh.accessTierToLevel(actual)
	requiredLevel := gh.accessTierToLevel(required)
	return actualLevel >= requiredLevel
}

// hasComplianceCapability checks if configuration has a specific compliance capability
func (gh *GovernanceHelper) hasComplianceCapability(config map[string]interface{}, capability string) bool {
	// This is provider-agnostic - just checks if the capability is mentioned in config
	if capabilities, exists := config["compliance_capabilities"]; exists {
		if capList, ok := capabilities.([]interface{}); ok {
			for _, cap := range capList {
				if capStr, ok := cap.(string); ok && capStr == capability {
					return true
				}
			}
		}
	}
	return false
}

// =============================================================================
// LEVEL CONVERSION UTILITIES
// =============================================================================

// auditScopeToLevel converts audit scope to numeric level for comparison
func (gh *GovernanceHelper) auditScopeToLevel(scope AuditScope) int {
	switch scope {
	case AuditScopeNone:
		return 0
	case AuditScopeMetadata:
		return 1
	case AuditScopeRead:
		return 2
	case AuditScopeWrite:
		return 3
	case AuditScopeFull:
		return 4
	default:
		return 0
	}
}

// accessTierToLevel converts access tier to numeric level for comparison
func (gh *GovernanceHelper) accessTierToLevel(tier AccessTier) int {
	switch tier {
	case AccessTierPublic:
		return 0
	case AccessTierInternal:
		return 1
	case AccessTierRestricted:
		return 2
	case AccessTierConfidential:
		return 3
	case AccessTierSecret:
		return 4
	default:
		return 0
	}
}

// Severity mapping functions
func (gh *GovernanceHelper) getProtectionSeverity(level int) string {
	switch {
	case level >= 9:
		return "critical"
	case level >= 7:
		return "high"
	case level >= 4:
		return "medium"
	default:
		return "low"
	}
}

func (gh *GovernanceHelper) getAuditSeverity(scope AuditScope) string {
	switch scope {
	case AuditScopeFull:
		return "high"
	case AuditScopeWrite:
		return "medium"
	case AuditScopeRead, AuditScopeMetadata:
		return "low"
	default:
		return "low"
	}
}

func (gh *GovernanceHelper) getAccessSeverity(tier AccessTier) string {
	switch tier {
	case AccessTierSecret:
		return "critical"
	case AccessTierConfidential:
		return "high"
	case AccessTierRestricted:
		return "medium"
	default:
		return "low"
	}
}

// getConfigIdentifier extracts an identifier from configuration
func getConfigIdentifier(config map[string]interface{}) string {
	if name, exists := config["name"]; exists {
		if nameStr, ok := name.(string); ok {
			return nameStr
		}
	}
	return "unknown"
}
