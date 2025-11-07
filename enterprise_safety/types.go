// Package enterprise_safety types for request/response structures and core data types
package enterprise_safety

import (
	"time"
)

// ValidationRequest represents a request to validate database operations
type ValidationRequest struct {
	Operations      []*DatabaseOperationSpec `json:"operations"`
	SafetyLevel     SafetyLevel              `json:"safety_level"`
	ValidationRules *ValidationRuleSet       `json:"validation_rules,omitempty"`
	Context         map[string]interface{}   `json:"context,omitempty"`
}

// ValidationResponse represents the result of validation
type ValidationResponse struct {
	Success         bool                      `json:"success"`
	Violations      []*ValidationViolation    `json:"violations,omitempty"`
	Warnings        []*ValidationWarning      `json:"warnings,omitempty"`
	Recommendations []*SafetyRecommendation   `json:"recommendations,omitempty"`
	RiskAssessment  *RiskAssessment          `json:"risk_assessment"`
}

// DatabaseOperationSpec defines a specific database operation to be performed
type DatabaseOperationSpec struct {
	Type         DatabaseOperation      `json:"type"`
	TableName    string                 `json:"table_name,omitempty"`
	SchemaName   string                 `json:"schema_name,omitempty"`
	ColumnName   string                 `json:"column_name,omitempty"`
	DataType     string                 `json:"data_type,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	SQL          string                 `json:"sql,omitempty"`
}

// ValidationViolation represents a validation rule violation
type ValidationViolation struct {
	RuleID      string              `json:"rule_id"`
	Severity    ValidationSeverity  `json:"severity"`
	Message     string              `json:"message"`
	Suggestion  string              `json:"suggestion,omitempty"`
	Operation   *DatabaseOperationSpec `json:"operation"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	Suggestion  string              `json:"suggestion,omitempty"`
	Operation   *DatabaseOperationSpec `json:"operation"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// SafetyRecommendation represents a safety recommendation
type SafetyRecommendation struct {
	Type        string              `json:"type"`
	Priority    string              `json:"priority"`
	Message     string              `json:"message"`
	Action      string              `json:"action,omitempty"`
	Operation   *DatabaseOperationSpec `json:"operation"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// RiskAssessment represents the risk assessment of operations
type RiskAssessment struct {
	OverallRisk     RiskLevel                 `json:"overall_risk"`
	RiskFactors     []string                  `json:"risk_factors"`
	DataLossRisk    RiskLevel                 `json:"data_loss_risk"`
	PerformanceImpact RiskLevel               `json:"performance_impact"`
	DowntimeRisk    RiskLevel                 `json:"downtime_risk"`
	Details         map[string]interface{}    `json:"details,omitempty"`
}

// ValidationRuleSet defines the set of validation rules to apply
type ValidationRuleSet struct {
	Rules                   []string               `json:"rules"`
	CustomRules             []*CustomValidationRule `json:"custom_rules,omitempty"`
	ProviderSpecificRules   map[string]interface{} `json:"provider_specific_rules,omitempty"`
	Severity                ValidationSeverity     `json:"severity"`
	FailOnError             bool                   `json:"fail_on_error"`
}

// CustomValidationRule defines a custom validation rule
type CustomValidationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   string                 `json:"condition"`
	Message     string                 `json:"message"`
	Severity    ValidationSeverity     `json:"severity"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ConstraintAnalysisRequest represents a request to analyze constraints
type ConstraintAnalysisRequest struct {
	TableName   string                 `json:"table_name"`
	SchemaName  string                 `json:"schema_name"`
	Operation   *DatabaseOperationSpec `json:"operation"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ConstraintAnalysisResponse represents the result of constraint analysis
type ConstraintAnalysisResponse struct {
	Constraints         []*ConstraintDefinition `json:"constraints"`
	Dependencies        []*ConstraintDependency `json:"dependencies"`
	AffectedConstraints []*ConstraintDefinition `json:"affected_constraints"`
	RiskLevel           RiskLevel               `json:"risk_level"`
	RequiredActions     []*RequiredAction       `json:"required_actions,omitempty"`
	Recommendations     []*SafetyRecommendation `json:"recommendations,omitempty"`
}

// ConstraintDefinition defines a database constraint
type ConstraintDefinition struct {
	Name         string                 `json:"name"`
	Type         ConstraintType         `json:"type"`
	TableName    string                 `json:"table_name"`
	SchemaName   string                 `json:"schema_name"`
	Columns      []string               `json:"columns"`
	ReferencedTable string               `json:"referenced_table,omitempty"`
	ReferencedColumns []string           `json:"referenced_columns,omitempty"`
	Definition   string                 `json:"definition"`
	IsDeferred   bool                   `json:"is_deferred"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ConstraintDependency represents a dependency relationship between constraints
type ConstraintDependency struct {
	Source      *ConstraintDefinition  `json:"source"`
	Target      *ConstraintDefinition  `json:"target"`
	Type        string                 `json:"type"`
	Impact      RiskLevel              `json:"impact"`
	Description string                 `json:"description"`
}

// RequiredAction represents an action that must be taken
type RequiredAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	SQL         string                 `json:"sql,omitempty"`
	Order       int                    `json:"order"`
	Required    bool                   `json:"required"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// RiskAssessmentRequest represents a request for operation risk assessment
type RiskAssessmentRequest struct {
	Operations []*DatabaseOperationSpec `json:"operations"`
	Context    map[string]interface{}   `json:"context,omitempty"`
}

// RiskAssessmentResponse represents the result of risk assessment
type RiskAssessmentResponse struct {
	Assessment      *RiskAssessment        `json:"assessment"`
	Mitigations     []*RiskMitigation      `json:"mitigations,omitempty"`
	Recommendations []*SafetyRecommendation `json:"recommendations,omitempty"`
}

// RiskMitigation represents a way to mitigate risk
type RiskMitigation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Effectiveness string               `json:"effectiveness"`
	Cost        string                 `json:"cost"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// BackupRequest represents a request to create a backup
type BackupRequest struct {
	BackupID     string                 `json:"backup_id"`
	Objects      []*DatabaseObject      `json:"objects"`
	BackupType   BackupType             `json:"backup_type"`
	BackupPolicy *BackupPolicy          `json:"backup_policy,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BackupResponse represents the result of backup creation
type BackupResponse struct {
	BackupID       string                 `json:"backup_id"`
	Provider       string                 `json:"provider"`
	CreatedAt      time.Time              `json:"created_at"`
	Objects        []*BackupObject        `json:"objects"`
	BackupSize     int64                  `json:"backup_size"`
	IntegrityCheck *BackupIntegrityResult `json:"integrity_check"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DatabaseObject represents a database object that can be backed up
type DatabaseObject struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Schema     string                 `json:"schema,omitempty"`
	Definition string                 `json:"definition,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BackupType defines the type of backup
type BackupType string

const (
	BackupTypeFull       BackupType = "FULL"        // Complete backup of specified objects
	BackupTypeIncremental BackupType = "INCREMENTAL" // Incremental backup since last backup
	BackupTypeSchema     BackupType = "SCHEMA"      // Schema-only backup
	BackupTypeData       BackupType = "DATA"        // Data-only backup
)

// BackupPolicy defines backup policies and settings
type BackupPolicy struct {
	RetentionDays   int                    `json:"retention_days"`
	CompressionType string                 `json:"compression_type,omitempty"`
	EncryptionKey   string                 `json:"encryption_key,omitempty"`
	VerifyIntegrity bool                   `json:"verify_integrity"`
	MaxBackupSize   int64                  `json:"max_backup_size,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
}

// BackupIntegrityResult represents the result of backup integrity verification
type BackupIntegrityResult struct {
	Valid       bool                   `json:"valid"`
	CheckedAt   time.Time              `json:"checked_at"`
	Issues      []string               `json:"issues,omitempty"`
	Checksums   map[string]string      `json:"checksums,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// BackupValidationRequest represents a request to validate a backup
type BackupValidationRequest struct {
	BackupID string                 `json:"backup_id"`
	Objects  []*DatabaseObject      `json:"objects,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// BackupValidationResponse represents the result of backup validation
type BackupValidationResponse struct {
	Valid       bool                   `json:"valid"`
	ValidatedAt time.Time              `json:"validated_at"`
	Issues      []string               `json:"issues,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// RestoreRequest represents a request to restore from backup
type RestoreRequest struct {
	BackupID    string                 `json:"backup_id"`
	Objects     []*DatabaseObject      `json:"objects,omitempty"`
	RestoreType RestoreType            `json:"restore_type"`
	Target      *RestoreTarget         `json:"target,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// RestoreResponse represents the result of restore operation
type RestoreResponse struct {
	Success     bool                   `json:"success"`
	RestoredAt  time.Time              `json:"restored_at"`
	Objects     []*DatabaseObject      `json:"objects"`
	Issues      []string               `json:"issues,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// RestoreType defines the type of restore operation
type RestoreType string

const (
	RestoreTypeComplete RestoreType = "COMPLETE" // Complete restore
	RestoreTypePartial  RestoreType = "PARTIAL"  // Partial restore of specified objects
	RestoreTypeInPlace  RestoreType = "IN_PLACE" // Restore in place
	RestoreTypeNewTarget RestoreType = "NEW_TARGET" // Restore to new location
)

// RestoreTarget defines where to restore backup
type RestoreTarget struct {
	Schema     string                 `json:"schema,omitempty"`
	TableName  string                 `json:"table_name,omitempty"`
	NewName    string                 `json:"new_name,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// RollbackPlanRequest represents a request to generate a rollback plan
type RollbackPlanRequest struct {
	Operations []*DatabaseOperationSpec `json:"operations"`
	Context    map[string]interface{}   `json:"context,omitempty"`
}

// RollbackPlanResponse represents a generated rollback plan
type RollbackPlanResponse struct {
	PlanID      string                   `json:"plan_id"`
	Operations  []*RollbackOperation     `json:"operations"`
	GeneratedAt time.Time                `json:"generated_at"`
	ValidUntil  time.Time                `json:"valid_until"`
	RiskLevel   RiskLevel                `json:"risk_level"`
	Notes       []string                 `json:"notes,omitempty"`
}

// RollbackOperation represents a single rollback operation
type RollbackOperation struct {
	ID             string                 `json:"id"`
	OriginalOp     *DatabaseOperationSpec `json:"original_operation"`
	RollbackSQL    string                 `json:"rollback_sql"`
	Order          int                    `json:"order"`
	RequiresBackup bool                   `json:"requires_backup"`
	RiskLevel      RiskLevel              `json:"risk_level"`
	Notes          []string               `json:"notes,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// RollbackExecutionRequest represents a request to execute rollback
type RollbackExecutionRequest struct {
	PlanID     string                 `json:"plan_id"`
	Operations []string               `json:"operations,omitempty"` // Specific operation IDs to rollback
	Context    map[string]interface{} `json:"context,omitempty"`
}

// RollbackExecutionResponse represents the result of rollback execution
type RollbackExecutionResponse struct {
	Success     bool                   `json:"success"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Operations  []string               `json:"operations"` // Executed operation IDs
	Issues      []string               `json:"issues,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// TableDefinition represents a database table definition
type TableDefinition struct {
	Name        string                 `json:"name"`
	Schema      string                 `json:"schema"`
	Columns     []*ColumnDefinition    `json:"columns"`
	Constraints []*ConstraintDefinition `json:"constraints"`
	Indexes     []*IndexDefinition     `json:"indexes"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ColumnDefinition represents a database column definition
type ColumnDefinition struct {
	Name         string                 `json:"name"`
	DataType     string                 `json:"data_type"`
	IsNullable   bool                   `json:"is_nullable"`
	DefaultValue *string                `json:"default_value,omitempty"`
	IsPrimaryKey bool                   `json:"is_primary_key"`
	IsUnique     bool                   `json:"is_unique"`
	Position     int                    `json:"position"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// IndexDefinition represents a database index definition
type IndexDefinition struct {
	Name      string                 `json:"name"`
	TableName string                 `json:"table_name"`
	Columns   []string               `json:"columns"`
	IsUnique  bool                   `json:"is_unique"`
	Type      string                 `json:"type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderCapabilityMatrix represents the capabilities of a provider
type ProviderCapabilityMatrix struct {
	ProviderType          string                    `json:"provider_type"`
	ProviderVersion       string                    `json:"provider_version"`
	SupportedOperations   []DatabaseOperation       `json:"supported_operations"`
	SupportedDataTypes    []string                  `json:"supported_data_types"`
	SupportedConstraints  []ConstraintType          `json:"supported_constraints"`
	SupportedFeatures     []DatabaseFeature         `json:"supported_features"`
	BackupCapabilities    *BackupCapabilities       `json:"backup_capabilities"`
	RollbackCapabilities  *RollbackCapabilities     `json:"rollback_capabilities"`
	ValidationCapabilities *ValidationCapabilities  `json:"validation_capabilities"`
	Limits                *ProviderLimits           `json:"limits"`
	Metadata              map[string]interface{}    `json:"metadata,omitempty"`
}

// BackupCapabilities represents backup capabilities of a provider
type BackupCapabilities struct {
	SupportsObjectLevelBackup  bool                   `json:"supports_object_level_backup"`
	SupportsDataOnlyBackup     bool                   `json:"supports_data_only_backup"`
	SupportsSchemaOnlyBackup   bool                   `json:"supports_schema_only_backup"`
	SupportsIncrementalBackup  bool                   `json:"supports_incremental_backup"`
	SupportsTransactionalBackup bool                  `json:"supports_transactional_backup"`
	SupportsEncryption         bool                   `json:"supports_encryption"`
	SupportsCompression        bool                   `json:"supports_compression"`
	MaxBackupSize              string                 `json:"max_backup_size,omitempty"`
	SupportedFormats           []string               `json:"supported_formats,omitempty"`
	Metadata                   map[string]interface{} `json:"metadata,omitempty"`
}

// RollbackCapabilities represents rollback capabilities of a provider
type RollbackCapabilities struct {
	SupportsAutomaticRollback  bool                   `json:"supports_automatic_rollback"`
	SupportsPointInTimeRecovery bool                  `json:"supports_point_in_time_recovery"`
	SupportsTransactionRollback bool                  `json:"supports_transaction_rollback"`
	MaxRollbackWindow          string                 `json:"max_rollback_window,omitempty"`
	SupportedOperations        []DatabaseOperation    `json:"supported_operations"`
	Metadata                   map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationCapabilities represents validation capabilities of a provider
type ValidationCapabilities struct {
	SupportsConstraintValidation bool                   `json:"supports_constraint_validation"`
	SupportsDataTypeValidation   bool                   `json:"supports_data_type_validation"`
	SupportsBusinessRules        bool                   `json:"supports_business_rules"`
	SupportsCustomRules          bool                   `json:"supports_custom_rules"`
	SupportedValidationLevels    []ValidationSeverity   `json:"supported_validation_levels"`
	Metadata                     map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderLimits represents provider-specific limits
type ProviderLimits struct {
	MaxIdentifierLength   int                    `json:"max_identifier_length"`
	MaxColumnsPerTable    int                    `json:"max_columns_per_table,omitempty"`
	MaxIndexesPerTable    int                    `json:"max_indexes_per_table,omitempty"`
	MaxConstraintsPerTable int                   `json:"max_constraints_per_table,omitempty"`
	MaxTableSize          string                 `json:"max_table_size,omitempty"`
	MaxDatabaseSize       string                 `json:"max_database_size,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// SafetyFeatureSupport represents the safety features supported by a provider
type SafetyFeatureSupport struct {
	ConstraintAnalysis bool                   `json:"constraint_analysis"`
	CascadeDetection   bool                   `json:"cascade_detection"`
	RollbackGeneration bool                   `json:"rollback_generation"`
	DataValidation     bool                   `json:"data_validation"`
	IntegrityChecking  bool                   `json:"integrity_checking"`
	BackupVerification bool                   `json:"backup_verification"`
	RiskAssessment     bool                   `json:"risk_assessment"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}