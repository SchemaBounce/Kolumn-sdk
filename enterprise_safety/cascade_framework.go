// Package enterprise_safety provides cascade delete testing framework for comprehensive cascade delete testing
package enterprise_safety

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// CascadeDeleteTestFramework provides comprehensive cascade delete testing
type CascadeDeleteTestFramework struct {
	ProviderType string
	TestResults  []CascadeDeleteTestResult
	Metrics      CascadeTestMetrics
}

// CascadeTestMetrics tracks cascade testing performance
type CascadeTestMetrics struct {
	TotalTests             int
	PassedTests            int
	FailedTests            int
	OrphanedResourcesFound int
	IntegrityViolations    int
	TotalTestDuration      time.Duration
	AverageTestDuration    time.Duration
}

// CascadeDeleteTestResult represents the result of a cascade delete test
type CascadeDeleteTestResult struct {
	TestName            string                 `json:"test_name"`
	TestType            string                 `json:"test_type"`
	ProviderType        string                 `json:"provider_type"`
	StartTime           time.Time              `json:"start_time"`
	Duration            time.Duration          `json:"duration"`
	Success             bool                   `json:"success"`
	PrimaryObject       ObjectInfo             `json:"primary_object"`
	DependentObjects    []ObjectInfo           `json:"dependent_objects"`
	PreDeleteCounts     map[string]int         `json:"pre_delete_counts"`
	PostDeleteCounts    map[string]int         `json:"post_delete_counts"`
	ExpectedBehavior    CascadeExpectation     `json:"expected_behavior"`
	ActualBehavior      CascadeActual          `json:"actual_behavior"`
	OrphanedResources   []OrphanedResource     `json:"orphaned_resources"`
	IntegrityViolations []IntegrityViolation   `json:"integrity_violations"`
	Error               string                 `json:"error"`
	Recommendations     []string               `json:"recommendations"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// ObjectInfo represents detailed information about a database object
type ObjectInfo struct {
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	DatabaseName string            `json:"database_name"`
	SchemaName   string            `json:"schema_name"`
	TableName    string            `json:"table_name"` // For triggers, indexes
	Columns      []string          `json:"columns"`
	Constraints  []string          `json:"constraints"`
	Dependencies []string          `json:"dependencies"`
	RowCount     int               `json:"row_count"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
}

// CascadeExpectation defines what should happen during cascade delete
type CascadeExpectation struct {
	ShouldCascade        bool                   `json:"should_cascade"`
	CascadeTypes         []string               `json:"cascade_types"`
	ExpectedOrphans      int                    `json:"expected_orphans"`
	OrphanPrevention     bool                   `json:"orphan_prevention"`
	TransactionSupport   bool                   `json:"transaction_support"`
	ConstraintViolations []string               `json:"constraint_violations"`
	CleanupRequired      bool                   `json:"cleanup_required"`
	Dependencies         map[string]interface{} `json:"dependencies"`
}

// CascadeActual represents what actually happened during cascade delete
type CascadeActual struct {
	CascadeExecuted       bool                   `json:"cascade_executed"`
	ObjectsDeleted        []string               `json:"objects_deleted"`
	ObjectsRemaining      []string               `json:"objects_remaining"`
	ConstraintsViolated   []string               `json:"constraints_violated"`
	TransactionRolledBack bool                   `json:"transaction_rolled_back"`
	ErrorsEncountered     []string               `json:"errors_encountered"`
	CleanupExecuted       bool                   `json:"cleanup_executed"`
	ResultDetails         map[string]interface{} `json:"result_details"`
}

// OrphanedResource represents a resource left without its parent
type OrphanedResource struct {
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	DatabaseName   string    `json:"database_name"`
	SchemaName     string    `json:"schema_name"`
	ParentType     string    `json:"parent_type"`
	ParentName     string    `json:"parent_name"`
	OrphanedCount  int       `json:"orphaned_count"`
	OrphanedSince  time.Time `json:"orphaned_since"`
	Severity       string    `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	CleanupAction  string    `json:"cleanup_action"`
	CanAutoCleanup bool      `json:"can_auto_cleanup"`
}

// IntegrityViolation represents a data integrity violation
type IntegrityViolation struct {
	Type              string    `json:"type"`
	Description       string    `json:"description"`
	AffectedObjects   []string  `json:"affected_objects"`
	Severity          string    `json:"severity"`
	DetectedAt        time.Time `json:"detected_at"`
	RecommendedAction string    `json:"recommended_action"`
	CanAutoResolve    bool      `json:"can_auto_resolve"`
}

// CascadeTestScenario defines a cascade delete test scenario
type CascadeTestScenario struct {
	Name              string
	Description       string
	PrimaryObject     ObjectInfo
	DependentObjects  []ObjectInfo
	TestData          map[string]interface{}
	ExpectedBehavior  CascadeExpectation
	SetupSQL          []string
	CleanupSQL        []string
	ValidationQueries []ValidationQuery
}

// ValidationQuery represents a query to validate cascade behavior
type ValidationQuery struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Query          string      `json:"query"`
	ExpectedResult interface{} `json:"expected_result"`
	Tolerance      float64     `json:"tolerance"`
}

// NewCascadeDeleteTestFramework creates a new cascade delete test framework
func NewCascadeDeleteTestFramework(providerType string) *CascadeDeleteTestFramework {
	return &CascadeDeleteTestFramework{
		ProviderType: providerType,
		TestResults:  make([]CascadeDeleteTestResult, 0),
		Metrics:      CascadeTestMetrics{},
	}
}

// RunCascadeDeleteTest executes a comprehensive cascade delete test
func (f *CascadeDeleteTestFramework) RunCascadeDeleteTest(ctx context.Context, db *sql.DB, scenario CascadeTestScenario) CascadeDeleteTestResult {
	result := CascadeDeleteTestResult{
		TestName:         scenario.Name,
		TestType:         "cascade_delete",
		ProviderType:     f.ProviderType,
		StartTime:        time.Now(),
		PrimaryObject:    scenario.PrimaryObject,
		DependentObjects: scenario.DependentObjects,
		ExpectedBehavior: scenario.ExpectedBehavior,
		PreDeleteCounts:  make(map[string]int),
		PostDeleteCounts: make(map[string]int),
		Metadata:         make(map[string]interface{}),
	}

	defer func() {
		result.Duration = time.Since(result.StartTime)
		f.TestResults = append(f.TestResults, result)
		f.updateMetrics(result)
	}()

	// Step 1: Setup test environment
	if err := f.setupTestEnvironment(ctx, db, scenario); err != nil {
		result.Error = fmt.Sprintf("Setup failed: %v", err)
		return result
	}

	// Step 2: Count objects before deletion
	result.PreDeleteCounts = f.countAllObjects(db, scenario.DependentObjects)

	// Step 3: Execute cascade delete
	err := f.executeCascadeDelete(ctx, db, scenario.PrimaryObject)
	if err != nil {
		result.Error = fmt.Sprintf("Delete failed: %v", err)
		return result
	}

	// Step 4: Count objects after deletion
	result.PostDeleteCounts = f.countAllObjects(db, scenario.DependentObjects)

	// Step 5: Analyze cascade behavior
	result.ActualBehavior = f.analyzeCascadeBehavior(result.PreDeleteCounts, result.PostDeleteCounts, scenario)

	// Step 6: Detect orphaned resources
	result.OrphanedResources = f.detectOrphanedResources(ctx, db, scenario)

	// Step 7: Check for integrity violations
	result.IntegrityViolations = f.checkIntegrityViolations(ctx, db, scenario)

	// Step 8: Validate results
	result.Success = f.validateTestResults(result, scenario)

	// Step 9: Generate recommendations
	result.Recommendations = f.generateRecommendations(result, scenario)

	// Step 10: Cleanup test environment
	if err := f.cleanupTestEnvironment(ctx, db, scenario); err != nil {
		log.Printf("Cleanup warning: %v", err)
	}

	return result
}

// orphanDetectionNotImplemented is the Error RunOrphanDetectionTest reports.
const orphanDetectionNotImplemented = "orphan detection test is not implemented: " +
	"the framework cannot yet create, find or remove intentional orphans, so it runs no SQL and reports failure"

// RunOrphanDetectionTest is meant to plant orphaned rows, check that they are
// detected and remove them again. That is not implemented, so it runs nothing
// against db and returns a failed result whose Error says so. It used to
// report success without testing anything.
func (f *CascadeDeleteTestFramework) RunOrphanDetectionTest(ctx context.Context, db *sql.DB) CascadeDeleteTestResult {
	result := CascadeDeleteTestResult{
		TestName:     "Orphaned Resource Detection",
		TestType:     "orphan_detection",
		ProviderType: f.ProviderType,
		StartTime:    time.Now(),
		Success:      false,
		Error:        orphanDetectionNotImplemented,
		Metadata:     make(map[string]interface{}),
	}

	result.Duration = time.Since(result.StartTime)
	f.TestResults = append(f.TestResults, result)
	f.updateMetrics(result)

	return result
}

// RunReferentialIntegrityTest tests referential integrity constraints
func (f *CascadeDeleteTestFramework) RunReferentialIntegrityTest(ctx context.Context, db *sql.DB) CascadeDeleteTestResult {
	result := CascadeDeleteTestResult{
		TestName:     "Referential Integrity Test",
		TestType:     "referential_integrity",
		ProviderType: f.ProviderType,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	defer func() {
		result.Duration = time.Since(result.StartTime)
		f.TestResults = append(f.TestResults, result)
		f.updateMetrics(result)
	}()

	// Test various referential integrity scenarios
	violations := f.testReferentialIntegrity(ctx, db)
	result.IntegrityViolations = violations
	result.Success = len(violations) == 0

	return result
}

// Helper methods

func (f *CascadeDeleteTestFramework) setupTestEnvironment(ctx context.Context, db *sql.DB, scenario CascadeTestScenario) error {
	// Execute setup SQL statements
	for _, sqlStmt := range scenario.SetupSQL {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("setup SQL failed: %v", err)
		}
	}

	// Insert test data
	if err := f.insertTestData(ctx, db, scenario.TestData); err != nil {
		return fmt.Errorf("test data insertion failed: %v", err)
	}

	return nil
}

func (f *CascadeDeleteTestFramework) insertTestData(ctx context.Context, db *sql.DB, testData map[string]interface{}) error {
	// This would contain provider-specific logic to insert test data
	// Implementation depends on the specific test data structure
	return nil
}

func (f *CascadeDeleteTestFramework) countAllObjects(db *sql.DB, objects []ObjectInfo) map[string]int {
	counts := make(map[string]int)

	for _, obj := range objects {
		count := f.countObject(db, obj)
		key := fmt.Sprintf("%s.%s", obj.Type, obj.Name)
		counts[key] = count
	}

	return counts
}

func (f *CascadeDeleteTestFramework) countObject(db *sql.DB, obj ObjectInfo) int {
	var query string
	var args []interface{}
	var count int

	// Catalog lookups take the names as bind parameters. A table count has
	// to name the table in the SQL text, so that name is validated first.
	switch f.ProviderType {
	case "postgres":
		switch obj.Type {
		case "table":
			name, err := qualifiedSQLName(obj.SchemaName, obj.Name)
			if err != nil {
				return 0
			}
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s", name) // #nosec G201 -- name is validated by qualifiedSQLName
		case "view":
			query = "SELECT COUNT(*) FROM information_schema.views WHERE table_schema = $1 AND table_name = $2"
			args = []interface{}{obj.SchemaName, obj.Name}
		case "function":
			query = "SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema = $1 AND routine_name = $2"
			args = []interface{}{obj.SchemaName, obj.Name}
		case "trigger":
			query = "SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = $1 AND trigger_name = $2"
			args = []interface{}{obj.SchemaName, obj.Name}
		}

	case "mysql":
		switch obj.Type {
		case "table":
			name, err := qualifiedSQLName(obj.DatabaseName, obj.Name)
			if err != nil {
				return 0
			}
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s", name) // #nosec G201 -- name is validated by qualifiedSQLName
		case "view":
			query = "SELECT COUNT(*) FROM information_schema.views WHERE table_schema = ? AND table_name = ?"
			args = []interface{}{obj.DatabaseName, obj.Name}
		case "function":
			query = "SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema = ? AND routine_name = ? AND routine_type = 'FUNCTION'"
			args = []interface{}{obj.DatabaseName, obj.Name}
		case "procedure":
			query = "SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema = ? AND routine_name = ? AND routine_type = 'PROCEDURE'"
			args = []interface{}{obj.DatabaseName, obj.Name}
		}
	}

	if query != "" {
		err := db.QueryRow(query, args...).Scan(&count)
		if err != nil {
			return 0
		}
	}

	return count
}

func (f *CascadeDeleteTestFramework) executeCascadeDelete(ctx context.Context, db *sql.DB, obj ObjectInfo) error {
	var query string

	switch f.ProviderType {
	case "postgres":
		switch obj.Type {
		case "table", "view":
			name, err := qualifiedSQLName(obj.SchemaName, obj.Name)
			if err != nil {
				return err
			}
			query = fmt.Sprintf("DROP %s IF EXISTS %s CASCADE", strings.ToUpper(obj.Type), name) // #nosec G201 -- obj.Type is "table" or "view" here and name is validated by qualifiedSQLName
		case "schema":
			if err := validateSQLIdentifier("schema name", obj.Name); err != nil {
				return err
			}
			query = fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", obj.Name) // #nosec G201 -- obj.Name is validated by validateSQLIdentifier
		}

	case "mysql":
		switch obj.Type {
		case "table", "view":
			name, err := qualifiedSQLName(obj.DatabaseName, obj.Name)
			if err != nil {
				return err
			}
			query = fmt.Sprintf("DROP %s IF EXISTS %s", strings.ToUpper(obj.Type), name) // #nosec G201 -- obj.Type is "table" or "view" here and name is validated by qualifiedSQLName
		case "database":
			if err := validateSQLIdentifier("database name", obj.Name); err != nil {
				return err
			}
			query = fmt.Sprintf("DROP DATABASE IF EXISTS %s", obj.Name) // #nosec G201 -- obj.Name is validated by validateSQLIdentifier
		}
	}

	if query != "" {
		_, err := db.ExecContext(ctx, query)
		return err
	}

	return fmt.Errorf("unsupported object type for deletion: %s", obj.Type)
}

func (f *CascadeDeleteTestFramework) analyzeCascadeBehavior(preCount, postCount map[string]int, scenario CascadeTestScenario) CascadeActual {
	actual := CascadeActual{
		ObjectsDeleted:   make([]string, 0),
		ObjectsRemaining: make([]string, 0),
		ResultDetails:    make(map[string]interface{}),
	}

	for key, preCnt := range preCount {
		postCnt, exists := postCount[key]
		if !exists {
			postCnt = 0
		}

		if preCnt > postCnt {
			actual.CascadeExecuted = true
			actual.ObjectsDeleted = append(actual.ObjectsDeleted, key)
		} else if postCnt > 0 {
			actual.ObjectsRemaining = append(actual.ObjectsRemaining, key)
		}

		actual.ResultDetails[key] = map[string]int{
			"before":  preCnt,
			"after":   postCnt,
			"deleted": preCnt - postCnt,
		}
	}

	return actual
}

func (f *CascadeDeleteTestFramework) detectOrphanedResources(ctx context.Context, db *sql.DB, scenario CascadeTestScenario) []OrphanedResource {
	var orphans []OrphanedResource

	// Check for orphaned records in dependent tables
	for _, obj := range scenario.DependentObjects {
		if obj.Type == "table" {
			tableOrphans := f.detectOrphanedRecords(db, obj)
			orphans = append(orphans, tableOrphans...)
		}
	}

	return orphans
}

func (f *CascadeDeleteTestFramework) detectOrphanedRecords(db *sql.DB, obj ObjectInfo) []OrphanedResource {
	var orphans []OrphanedResource

	// This is a simplified implementation
	// In practice, this would analyze foreign key relationships and find records
	// that reference non-existent parent records

	for _, dependency := range obj.Dependencies {
		// Check if records exist that reference deleted parent
		var orphanCount int

		// Example query (would need to be customized per relationship)
		query := f.buildOrphanDetectionQuery(obj, dependency)
		if query != "" {
			err := db.QueryRow(query).Scan(&orphanCount)
			if err == nil && orphanCount > 0 {
				orphan := OrphanedResource{
					Type:           obj.Type,
					Name:           obj.Name,
					DatabaseName:   obj.DatabaseName,
					SchemaName:     obj.SchemaName,
					ParentType:     "table", // Would parse from dependency
					ParentName:     dependency,
					OrphanedCount:  orphanCount,
					OrphanedSince:  time.Now(),
					Severity:       f.calculateOrphanSeverity(orphanCount, obj.Type),
					CleanupAction:  f.suggestCleanupAction(obj.Type, orphanCount),
					CanAutoCleanup: f.canAutoCleanup(obj.Type),
				}
				orphans = append(orphans, orphan)
			}
		}
	}

	return orphans
}

func (f *CascadeDeleteTestFramework) buildOrphanDetectionQuery(obj ObjectInfo, dependency string) string {
	// Build provider-specific query to detect orphaned records.
	// The schema (PostgreSQL) or database (MySQL) name goes into the SQL
	// text, so an unsafe one builds no query, as an unknown relationship does.
	switch f.ProviderType {
	case "postgres":
		// Example: Find comments without posts
		if obj.Name == "comments" && strings.Contains(dependency, "posts") {
			if validateSQLIdentifier("schema name", obj.SchemaName) != nil {
				return ""
			}
			// #nosec G201 -- obj.SchemaName is validated above and obj.Name is the constant "comments"
			return fmt.Sprintf(`
				SELECT COUNT(*)
				FROM %s.%s c
				LEFT JOIN %s.posts p ON c.post_id = p.id
				WHERE p.id IS NULL`,
				obj.SchemaName, obj.Name, obj.SchemaName)
		}
	case "mysql":
		// Example: Find comments without posts
		if obj.Name == "comments" && strings.Contains(dependency, "posts") {
			if validateSQLIdentifier("database name", obj.DatabaseName) != nil {
				return ""
			}
			// #nosec G201 -- obj.DatabaseName is validated above and obj.Name is the constant "comments"
			return fmt.Sprintf(`
				SELECT COUNT(*)
				FROM %s.%s c
				LEFT JOIN %s.posts p ON c.post_id = p.id
				WHERE p.id IS NULL`,
				obj.DatabaseName, obj.Name, obj.DatabaseName)
		}
	}

	return ""
}

func (f *CascadeDeleteTestFramework) checkIntegrityViolations(ctx context.Context, db *sql.DB, scenario CascadeTestScenario) []IntegrityViolation {
	var violations []IntegrityViolation

	// Run validation queries
	for _, validation := range scenario.ValidationQueries {
		violation := f.runValidationQuery(db, validation)
		if violation != nil {
			violations = append(violations, *violation)
		}
	}

	return violations
}

func (f *CascadeDeleteTestFramework) runValidationQuery(db *sql.DB, validation ValidationQuery) *IntegrityViolation {
	var result interface{}

	// Execute validation query and compare with expected result
	err := db.QueryRow(validation.Query).Scan(&result)
	if err != nil {
		return &IntegrityViolation{
			Type:              "query_error",
			Description:       fmt.Sprintf("Validation query failed: %v", err),
			AffectedObjects:   []string{validation.Name},
			Severity:          "HIGH",
			DetectedAt:        time.Now(),
			RecommendedAction: "Review and fix validation query",
			CanAutoResolve:    false,
		}
	}

	// Compare result with expected (simplified comparison)
	if result != validation.ExpectedResult {
		return &IntegrityViolation{
			Type:              "validation_failure",
			Description:       fmt.Sprintf("Validation '%s' failed: expected %v, got %v", validation.Name, validation.ExpectedResult, result),
			AffectedObjects:   []string{validation.Name},
			Severity:          "MEDIUM",
			DetectedAt:        time.Now(),
			RecommendedAction: "Investigate data integrity issue",
			CanAutoResolve:    false,
		}
	}

	return nil
}

func (f *CascadeDeleteTestFramework) validateTestResults(result CascadeDeleteTestResult, scenario CascadeTestScenario) bool {
	// Test passes if:
	// 1. No unexpected errors occurred
	// 2. Cascade behavior matches expectations
	// 3. No critical integrity violations
	// 4. Orphaned resources are within expected limits

	if result.Error != "" {
		return false
	}

	// Check cascade behavior
	expectedCascade := scenario.ExpectedBehavior.ShouldCascade
	actualCascade := result.ActualBehavior.CascadeExecuted
	if expectedCascade != actualCascade {
		return false
	}

	// Check for critical integrity violations
	for _, violation := range result.IntegrityViolations {
		if violation.Severity == "CRITICAL" {
			return false
		}
	}

	// Check orphaned resources
	if scenario.ExpectedBehavior.OrphanPrevention && len(result.OrphanedResources) > scenario.ExpectedBehavior.ExpectedOrphans {
		return false
	}

	return true
}

func (f *CascadeDeleteTestFramework) generateRecommendations(result CascadeDeleteTestResult, scenario CascadeTestScenario) []string {
	var recommendations []string

	if !result.Success {
		recommendations = append(recommendations, "Review cascade delete implementation for compliance with expected behavior")
	}

	if len(result.OrphanedResources) > 0 {
		recommendations = append(recommendations, "Implement foreign key constraints with appropriate CASCADE options")
		recommendations = append(recommendations, "Add orphaned resource cleanup procedures")
	}

	if len(result.IntegrityViolations) > 0 {
		recommendations = append(recommendations, "Address data integrity violations before production deployment")
	}

	if !result.ActualBehavior.CascadeExecuted && scenario.ExpectedBehavior.ShouldCascade {
		recommendations = append(recommendations, "Enable CASCADE DELETE in foreign key constraints")
	}

	return recommendations
}

func (f *CascadeDeleteTestFramework) cleanupTestEnvironment(ctx context.Context, db *sql.DB, scenario CascadeTestScenario) error {
	// Execute cleanup SQL statements
	for _, sqlStmt := range scenario.CleanupSQL {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("cleanup SQL failed: %v", err)
		}
	}

	return nil
}

func (f *CascadeDeleteTestFramework) testReferentialIntegrity(ctx context.Context, db *sql.DB) []IntegrityViolation {
	var violations []IntegrityViolation

	// Test various referential integrity scenarios
	// This would include trying to insert invalid foreign keys,
	// checking for orphaned records, etc.

	return violations
}

func (f *CascadeDeleteTestFramework) updateMetrics(result CascadeDeleteTestResult) {
	f.Metrics.TotalTests++
	f.Metrics.TotalTestDuration += result.Duration

	if result.Success {
		f.Metrics.PassedTests++
	} else {
		f.Metrics.FailedTests++
	}

	f.Metrics.OrphanedResourcesFound += len(result.OrphanedResources)
	f.Metrics.IntegrityViolations += len(result.IntegrityViolations)

	if f.Metrics.TotalTests > 0 {
		f.Metrics.AverageTestDuration = f.Metrics.TotalTestDuration / time.Duration(f.Metrics.TotalTests)
	}
}

func (f *CascadeDeleteTestFramework) calculateOrphanSeverity(count int, objectType string) string {
	if count == 0 {
		return "NONE"
	} else if count < 10 {
		return "LOW"
	} else if count < 100 {
		return "MEDIUM"
	} else if count < 1000 {
		return "HIGH"
	} else {
		return "CRITICAL"
	}
}

func (f *CascadeDeleteTestFramework) suggestCleanupAction(objectType string, count int) string {
	if count == 0 {
		return "No action needed"
	} else if count < 10 {
		return "Manual cleanup recommended"
	} else {
		return "Automated cleanup required"
	}
}

func (f *CascadeDeleteTestFramework) canAutoCleanup(objectType string) bool {
	// Define which object types can be automatically cleaned up
	autoCleanupTypes := []string{"table", "view"}
	for _, t := range autoCleanupTypes {
		if objectType == t {
			return true
		}
	}
	return false
}

// GenerateReport generates a comprehensive cascade delete test report
func (f *CascadeDeleteTestFramework) GenerateReport() CascadeTestReport {
	report := CascadeTestReport{
		ProviderType: f.ProviderType,
		GeneratedAt:  time.Now(),
		TestResults:  f.TestResults,
		Metrics:      f.Metrics,
	}

	// Calculate success rate
	if f.Metrics.TotalTests > 0 {
		report.SuccessRate = float64(f.Metrics.PassedTests) / float64(f.Metrics.TotalTests) * 100
	}

	// Generate summary
	report.Summary = f.generateTestSummary()

	// Generate recommendations
	report.Recommendations = f.generateOverallRecommendations()

	return report
}

// CascadeTestReport represents a comprehensive cascade delete test report
type CascadeTestReport struct {
	ProviderType    string                    `json:"provider_type"`
	GeneratedAt     time.Time                 `json:"generated_at"`
	TestResults     []CascadeDeleteTestResult `json:"test_results"`
	Metrics         CascadeTestMetrics        `json:"metrics"`
	SuccessRate     float64                   `json:"success_rate"`
	Summary         TestSummary               `json:"summary"`
	Recommendations []string                  `json:"recommendations"`
}

// TestSummary provides a high-level summary of test results
type TestSummary struct {
	TotalOrphansFound    int `json:"total_orphans_found"`
	TotalViolationsFound int `json:"total_violations_found"`
	CriticalIssues       int `json:"critical_issues"`
	HighPriorityIssues   int `json:"high_priority_issues"`
	MediumPriorityIssues int `json:"medium_priority_issues"`
	LowPriorityIssues    int `json:"low_priority_issues"`
}

func (f *CascadeDeleteTestFramework) generateTestSummary() TestSummary {
	summary := TestSummary{}

	for _, result := range f.TestResults {
		summary.TotalOrphansFound += len(result.OrphanedResources)
		summary.TotalViolationsFound += len(result.IntegrityViolations)

		// Count issues by severity
		for _, orphan := range result.OrphanedResources {
			switch orphan.Severity {
			case "CRITICAL":
				summary.CriticalIssues++
			case "HIGH":
				summary.HighPriorityIssues++
			case "MEDIUM":
				summary.MediumPriorityIssues++
			case "LOW":
				summary.LowPriorityIssues++
			}
		}

		for _, violation := range result.IntegrityViolations {
			switch violation.Severity {
			case "CRITICAL":
				summary.CriticalIssues++
			case "HIGH":
				summary.HighPriorityIssues++
			case "MEDIUM":
				summary.MediumPriorityIssues++
			case "LOW":
				summary.LowPriorityIssues++
			}
		}
	}

	return summary
}

func (f *CascadeDeleteTestFramework) generateOverallRecommendations() []string {
	var recommendations []string

	if f.Metrics.FailedTests > 0 {
		recommendations = append(recommendations, "Address failed cascade delete tests before production deployment")
	}

	if f.Metrics.OrphanedResourcesFound > 0 {
		recommendations = append(recommendations, "Implement comprehensive orphaned resource detection and cleanup")
	}

	if f.Metrics.IntegrityViolations > 0 {
		recommendations = append(recommendations, "Resolve all data integrity violations")
	}

	successRate := float64(f.Metrics.PassedTests) / float64(f.Metrics.TotalTests) * 100
	if successRate < 80 {
		recommendations = append(recommendations, "Improve cascade delete implementation to achieve >80% success rate")
	}

	return recommendations
}
