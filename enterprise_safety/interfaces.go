// Package enterprise_safety provides interfaces and types for enterprise safety capabilities
// across all Kolumn database providers. This package defines the standard interface that
// all providers must implement to support enterprise safety features.
package enterprise_safety

import (
	"context"
)

// ProviderSafetyCapabilities is the main interface that all providers must implement
// to support enterprise safety features. This interface provides a unified API for
// validation, backup, rollback, and other safety operations across all providers.
type ProviderSafetyCapabilities interface {
	// Validation capabilities
	ValidateOperations(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error)
	AnalyzeConstraints(ctx context.Context, req *ConstraintAnalysisRequest) (*ConstraintAnalysisResponse, error)
	AssessOperationRisk(ctx context.Context, req *RiskAssessmentRequest) (*RiskAssessmentResponse, error)

	// Backup capabilities
	CreateBackup(ctx context.Context, req *BackupRequest) (*BackupResponse, error)
	ValidateBackup(ctx context.Context, req *BackupValidationRequest) (*BackupValidationResponse, error)
	RestoreFromBackup(ctx context.Context, req *RestoreRequest) (*RestoreResponse, error)

	// Rollback capabilities
	GenerateRollbackPlan(ctx context.Context, req *RollbackPlanRequest) (*RollbackPlanResponse, error)
	ExecuteRollback(ctx context.Context, req *RollbackExecutionRequest) (*RollbackExecutionResponse, error)

	// Provider introspection
	GetProviderCapabilities() *ProviderCapabilityMatrix
	GetSafetyFeatureSupport() *SafetyFeatureSupport
}

// ProviderIntrospector provides database introspection capabilities
type ProviderIntrospector interface {
	GetTableDefinition(table, schema string) (*TableDefinition, error)
	GetColumnDefinition(table, schema, column string) (*ColumnDefinition, error)
	TableExists(table, schema string) (bool, error)
	ColumnExists(table, schema, column string) (bool, error)
	GetConstraints(table, schema string) ([]*ConstraintDefinition, error)
	GetIndexes(table, schema string) ([]*IndexDefinition, error)
}

// ProviderCapabilityProvider provides information about what a provider can do
type ProviderCapabilityProvider interface {
	SupportsOperation(operation DatabaseOperation) bool
	SupportsDataType(dataType string) bool
	SupportsConstraintType(constraintType ConstraintType) bool
	ValidateDataType(dataType string) error
	ValidateIdentifier(identifier string) error
	GetMaxIdentifierLength() int
	GetSupportedFeatures() []DatabaseFeature
}

// SafetyLevel defines the level of safety checks to perform
type SafetyLevel string

const (
	SafetyLevelDevelopment SafetyLevel = "DEVELOPMENT" // Minimal checks, fast execution
	SafetyLevelStaging     SafetyLevel = "STAGING"     // Enhanced validation, backup warnings
	SafetyLevelProduction  SafetyLevel = "PRODUCTION"  // Full enterprise safety, mandatory backups
)

// RiskLevel defines the risk level of an operation
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"      // Safe operation, minimal impact
	RiskLevelMedium   RiskLevel = "MEDIUM"   // Moderate risk, review recommended
	RiskLevelHigh     RiskLevel = "HIGH"     // High risk, backup required
	RiskLevelCritical RiskLevel = "CRITICAL" // Critical risk, manual approval required
)

// ValidationSeverity defines the severity of a validation issue
type ValidationSeverity string

const (
	SeverityInfo     ValidationSeverity = "INFO"     // Informational message
	SeverityWarning  ValidationSeverity = "WARNING"  // Warning, operation can proceed
	SeverityError    ValidationSeverity = "ERROR"    // Error, operation should not proceed
	SeverityCritical ValidationSeverity = "CRITICAL" // Critical error, operation blocked
)

// DatabaseOperation defines the type of database operation
type DatabaseOperation string

const (
	OperationCreateTable    DatabaseOperation = "CREATE_TABLE"
	OperationDropTable      DatabaseOperation = "DROP_TABLE"
	OperationAlterTable     DatabaseOperation = "ALTER_TABLE"
	OperationCreateColumn   DatabaseOperation = "CREATE_COLUMN"
	OperationDropColumn     DatabaseOperation = "DROP_COLUMN"
	OperationAlterColumn    DatabaseOperation = "ALTER_COLUMN"
	OperationRenameColumn   DatabaseOperation = "RENAME_COLUMN"
	OperationCreateIndex    DatabaseOperation = "CREATE_INDEX"
	OperationDropIndex      DatabaseOperation = "DROP_INDEX"
	OperationCreateFunction DatabaseOperation = "CREATE_FUNCTION"
	OperationDropFunction   DatabaseOperation = "DROP_FUNCTION"
	OperationCreateView     DatabaseOperation = "CREATE_VIEW"
	OperationDropView       DatabaseOperation = "DROP_VIEW"
)

// ConstraintType defines the type of database constraint
type ConstraintType string

const (
	ConstraintPrimaryKey ConstraintType = "PRIMARY_KEY"
	ConstraintForeignKey ConstraintType = "FOREIGN_KEY"
	ConstraintUnique     ConstraintType = "UNIQUE"
	ConstraintCheck      ConstraintType = "CHECK"
	ConstraintNotNull    ConstraintType = "NOT_NULL"
	ConstraintDefault    ConstraintType = "DEFAULT"
)

// DatabaseFeature defines features supported by a database provider
type DatabaseFeature string

const (
	FeatureTransactions        DatabaseFeature = "TRANSACTIONS"
	FeatureForeignKeys         DatabaseFeature = "FOREIGN_KEYS"
	FeatureConstraints         DatabaseFeature = "CONSTRAINTS"
	FeatureIndexes             DatabaseFeature = "INDEXES"
	FeatureViews               DatabaseFeature = "VIEWS"
	FeatureFunctions           DatabaseFeature = "FUNCTIONS"
	FeatureTriggers            DatabaseFeature = "TRIGGERS"
	FeaturePartitioning        DatabaseFeature = "PARTITIONING"
	FeatureReplication         DatabaseFeature = "REPLICATION"
	FeatureBackup              DatabaseFeature = "BACKUP"
	FeaturePointInTimeRecovery DatabaseFeature = "POINT_IN_TIME_RECOVERY"
)
