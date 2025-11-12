package enterprise_safety

import (
	"context"
	"testing"
)

// TestProviderSafetyCapabilitiesInterface ensures the interface can be implemented
func TestProviderSafetyCapabilitiesInterface(t *testing.T) {
	// Test that NullSafetyProvider implements the interface
	var provider ProviderSafetyCapabilities = NewNullSafetyProvider("test")

	// Test basic interface methods
	ctx := context.Background()

	// Test ValidateOperations
	validationReq := &ValidationRequest{
		Operations:  []*DatabaseOperationSpec{},
		SafetyLevel: SafetyLevelDevelopment,
	}
	resp, err := provider.ValidateOperations(ctx, validationReq)
	if err != nil {
		t.Errorf("ValidateOperations failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected successful validation")
	}

	// Test GetProviderCapabilities
	capabilities := provider.GetProviderCapabilities()
	if capabilities == nil {
		t.Errorf("GetProviderCapabilities returned nil")
	}
	if capabilities.ProviderType != "test" {
		t.Errorf("Expected provider type 'test', got '%s'", capabilities.ProviderType)
	}

	// Test GetSafetyFeatureSupport
	features := provider.GetSafetyFeatureSupport()
	if features == nil {
		t.Errorf("GetSafetyFeatureSupport returned nil")
	}
}

// TestBaseSafetyProvider ensures base provider works correctly
func TestBaseSafetyProvider(t *testing.T) {
	provider := NewBaseSafetyProvider("base-test")

	// Test that it implements the interface
	var _ ProviderSafetyCapabilities = provider

	ctx := context.Background()

	// Test CreateBackup
	backupReq := &BackupRequest{
		BackupID: "test-backup-123",
		Objects:  []*DatabaseObject{},
	}
	backupResp, err := provider.CreateBackup(ctx, backupReq)
	if err != nil {
		t.Errorf("CreateBackup failed: %v", err)
	}
	if backupResp.BackupID != "test-backup-123" {
		t.Errorf("Expected backup ID 'test-backup-123', got '%s'", backupResp.BackupID)
	}

	// Test GenerateRollbackPlan
	rollbackReq := &RollbackPlanRequest{
		Operations: []*DatabaseOperationSpec{},
	}
	rollbackResp, err := provider.GenerateRollbackPlan(ctx, rollbackReq)
	if err != nil {
		t.Errorf("GenerateRollbackPlan failed: %v", err)
	}
	if rollbackResp.PlanID == "" {
		t.Errorf("Expected non-empty plan ID")
	}
}

// TestSafetyLevelConstants ensures safety level constants are defined
func TestSafetyLevelConstants(t *testing.T) {
	levels := []SafetyLevel{
		SafetyLevelDevelopment,
		SafetyLevelStaging,
		SafetyLevelProduction,
	}

	for _, level := range levels {
		if string(level) == "" {
			t.Errorf("Safety level should not be empty")
		}
	}
}

// TestRiskLevelConstants ensures risk level constants are defined
func TestRiskLevelConstants(t *testing.T) {
	risks := []RiskLevel{
		RiskLevelLow,
		RiskLevelMedium,
		RiskLevelHigh,
		RiskLevelCritical,
	}

	for _, risk := range risks {
		if string(risk) == "" {
			t.Errorf("Risk level should not be empty")
		}
	}
}

// TestDatabaseOperationConstants ensures operation constants are defined
func TestDatabaseOperationConstants(t *testing.T) {
	operations := []DatabaseOperation{
		OperationCreateTable,
		OperationDropTable,
		OperationAlterTable,
		OperationCreateColumn,
		OperationDropColumn,
		OperationAlterColumn,
		OperationRenameColumn,
		OperationCreateIndex,
		OperationDropIndex,
		OperationCreateFunction,
		OperationDropFunction,
		OperationCreateView,
		OperationDropView,
	}

	for _, op := range operations {
		if string(op) == "" {
			t.Errorf("Database operation should not be empty")
		}
	}
}

// TestHelperFunctions tests the helper functions
func TestHelperFunctions(t *testing.T) {
	operation := &DatabaseOperationSpec{
		Type:       OperationCreateTable,
		TableName:  "test_table",
		SchemaName: "test_schema",
	}

	// Test CreateValidationViolation
	violation := CreateValidationViolation("TEST001", SeverityError, "Test violation", operation)
	if violation.RuleID != "TEST001" {
		t.Errorf("Expected rule ID 'TEST001', got '%s'", violation.RuleID)
	}
	if violation.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got '%s'", violation.Severity)
	}

	// Test CreateValidationWarning
	warning := CreateValidationWarning("WARN001", "Test warning", operation)
	if warning.Code != "WARN001" {
		t.Errorf("Expected code 'WARN001', got '%s'", warning.Code)
	}

	// Test CreateSafetyRecommendation
	recommendation := CreateSafetyRecommendation("backup", "high", "Create backup", operation)
	if recommendation.Type != "backup" {
		t.Errorf("Expected type 'backup', got '%s'", recommendation.Type)
	}

	// Test CreateRiskAssessment
	factors := []string{"factor1", "factor2"}
	assessment := CreateRiskAssessment(RiskLevelHigh, factors)
	if assessment.OverallRisk != RiskLevelHigh {
		t.Errorf("Expected risk HIGH, got '%s'", assessment.OverallRisk)
	}
	if len(assessment.RiskFactors) != 2 {
		t.Errorf("Expected 2 risk factors, got %d", len(assessment.RiskFactors))
	}
}
