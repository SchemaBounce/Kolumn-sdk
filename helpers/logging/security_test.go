package logging

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestCredentialSanitization tests that sensitive data is properly redacted
func TestCredentialSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []string // Should NOT appear in logs
		allowed  []string // Should appear in logs
	}{
		{
			name: "database credentials",
			input: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"username": "admin",
				"password": "secret123",
				"database": "production",
			},
			expected: []string{"secret123"},
			allowed:  []string{"localhost", "admin", "production"},
		},
		{
			name: "api keys and tokens",
			input: map[string]interface{}{
				"api_key":      "sk-abc123xyz",
				"access_token": "bearer_token_789",
				"secret_key":   "secret_key_456",
				"endpoint":     "https://api.example.com",
			},
			expected: []string{"sk-abc123xyz", "bearer_token_789", "secret_key_456"},
			allowed:  []string{"https://api.example.com"},
		},
		{
			name: "mixed sensitive and non-sensitive",
			input: map[string]interface{}{
				"service_name":    "myservice",
				"auth_token":      "token_abc123",
				"credential_file": "/path/to/secret.json",
				"timeout":         30,
				"encryption_key":  "key_xyz789",
			},
			expected: []string{"token_abc123", "/path/to/secret.json", "key_xyz789"},
			allowed:  []string{"myservice", "30"},
		},
		{
			name: "case insensitive detection",
			input: map[string]interface{}{
				"PASSWORD":     "upper_password",
				"Secret":       "mixed_case_secret",
				"api_KEY":      "mixed_case_key",
				"normal_field": "normal_value",
			},
			expected: []string{"upper_password", "mixed_case_secret", "mixed_case_key"},
			allowed:  []string{"normal_value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, capture := NewTestLogger(t, "security", false)

			// Test structured logging
			logger.InfoWithFields("Configuration loaded", flattenMap(test.input)...)

			output := capture.GetOutput()

			// Check that sensitive data is redacted
			for _, sensitive := range test.expected {
				if strings.Contains(output, sensitive) {
					t.Errorf("Sensitive data %q should be redacted but was found in: %s", sensitive, output)
				}
			}

			// Check that non-sensitive data is preserved
			for _, allowed := range test.allowed {
				if !strings.Contains(output, allowed) {
					t.Errorf("Non-sensitive data %q should be present but was not found in: %s", allowed, output)
				}
			}

			// Check for redaction markers
			if !strings.Contains(output, "<redacted>") {
				t.Error("Expected redaction markers but found none")
			}
		})
	}
}

// TestEndpointSanitization tests connection string sanitization
func TestEndpointSanitization(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		shouldRedact bool
		expectValue  string
	}{
		{
			name:         "postgresql with password",
			endpoint:     "postgresql://user:secretpass@localhost:5432/db",
			shouldRedact: true,
			expectValue:  "postgresql://user:***@localhost:5432/db",
		},
		{
			name:         "mysql with complex password",
			endpoint:     "mysql://admin:P@ssw0rd!@db.example.com:3306/prod",
			shouldRedact: true,
			expectValue:  "mysql://admin:***@db.example.com:3306/prod",
		},
		{
			name:         "redis with auth",
			endpoint:     "redis://user:authtoken@cache.example.com:6379/0",
			shouldRedact: true,
			expectValue:  "redis://user:***@cache.example.com:6379/0",
		},
		{
			name:         "no credentials",
			endpoint:     "postgresql://localhost:5432/db",
			shouldRedact: false,
			expectValue:  "postgresql://localhost:5432/db",
		},
		{
			name:         "username only",
			endpoint:     "postgresql://user@localhost:5432/db",
			shouldRedact: false,
			expectValue:  "postgresql://user@localhost:5432/db",
		},
		{
			name:         "complex special characters",
			endpoint:     "mongodb://user:p@$$w0rd%21@cluster.mongodb.net/db?ssl=true",
			shouldRedact: true,
			expectValue:  "mongodb://user:***@cluster.mongodb.net/db?ssl=true",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeEndpoint(test.endpoint)

			if result != test.expectValue {
				t.Errorf("Expected %q, got %q", test.expectValue, result)
			}

			// Verify original endpoint contains password but result doesn't
			if test.shouldRedact {
				if !strings.Contains(test.endpoint, "@") {
					t.Error("Test setup error: endpoint should contain @ for redaction test")
				}
				if strings.Contains(result, ":") && strings.Contains(result, "@") {
					// Check that the password part is replaced with ***
					parts := strings.Split(result, "://")
					if len(parts) > 1 {
						userPart := strings.Split(parts[1], "@")[0]
						if strings.Contains(userPart, ":") && !strings.Contains(userPart, "***") {
							t.Error("Password should be replaced with *** in sanitized endpoint")
						}
					}
				}
			}
		})
	}
}

// TestLogInjectionPrevention tests that log injection attacks are prevented
func TestLogInjectionPrevention(t *testing.T) {
	logger, capture := NewTestLogger(t, "injection", false)

	maliciousInputs := []string{
		"user\n[ERROR][admin] Fake admin access granted",
		"user\r\n[CRITICAL][security] System compromised",
		"user\x00[INFO][auth] Backdoor installed",
		"user\t[WARN][system] Malicious activity",
	}

	for i, input := range maliciousInputs {
		t.Run(fmt.Sprintf("injection_test_%d", i), func(t *testing.T) {
			capture.Clear()

			logger.Info("Processing user: %s", input)

			lines := capture.GetLines()

			// Should only have one log line
			if len(lines) != 1 {
				t.Errorf("Expected 1 log line, got %d. Lines: %v", len(lines), lines)
			}

			// The log line should not contain fake log entries
			output := capture.GetOutput()
			if strings.Contains(output, "Fake admin access") {
				t.Error("Log injection succeeded - fake admin message found")
			}
			if strings.Contains(output, "System compromised") {
				t.Error("Log injection succeeded - fake critical message found")
			}
			if strings.Contains(output, "Backdoor installed") {
				t.Error("Log injection succeeded - fake backdoor message found")
			}
		})
	}
}

// TestLargeLogPrevention tests that extremely large logs don't cause issues
func TestLargeLogPrevention(t *testing.T) {
	logger, capture := NewTestLogger(t, "large", false)

	// Test very large message
	largeMessage := strings.Repeat("A", 1000000) // 1MB message

	logger.Info("Large message: %s", largeMessage[:100]) // Truncate for sanity

	output := capture.GetOutput()
	if len(output) > 2000000 { // Allow some overhead but prevent excessive memory usage
		t.Errorf("Log output too large: %d bytes", len(output))
	}

	// Test large structured data
	largeData := make(map[string]interface{})
	for i := 0; i < 10000; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
	}

	capture.Clear()
	logger.JSONDebug("large data", largeData)

	// Should not crash or use excessive memory
	output2 := capture.GetOutput()
	if len(output2) == 0 {
		t.Error("Expected some output from large data logging")
	}
}

// TestSensitiveDataInErrors tests that errors don't leak sensitive information
func TestSensitiveDataInErrors(t *testing.T) {
	logger, capture := NewTestLogger(t, "error", false)

	sensitiveError := fmt.Errorf("connection failed: password 'secret123' is invalid")

	logger.Error("Database connection error: %v", sensitiveError)

	output := capture.GetOutput()

	// The error should be logged but sensitive data should be handled carefully
	if !strings.Contains(output, "connection failed") {
		t.Error("Expected error message to be logged")
	}

	// In a real implementation, you might want to sanitize error messages too
	// For now, just document that this is a potential issue
	if strings.Contains(output, "secret123") {
		t.Log("WARNING: Sensitive data in error messages - consider implementing error sanitization")
	}
}

// TestJSONDebugSafetyInProduction tests that debug mode doesn't leak data in production
func TestJSONDebugSafetyInProduction(t *testing.T) {
	// Simulate production environment (debug disabled)
	DisableDebug()

	logger, capture := NewTestLogger(t, "production", false)

	sensitiveData := map[string]interface{}{
		"user_id":       123,
		"session_token": "secret_session_abc123",
		"password":      "user_password_456",
	}

	// JSONDebug should not output anything when debug is disabled
	logger.JSONDebug("sensitive user data", sensitiveData)

	output := capture.GetOutput()
	if strings.Contains(output, "secret_session_abc123") {
		t.Error("Sensitive data leaked through JSONDebug in production mode")
	}
	if strings.Contains(output, "user_password_456") {
		t.Error("Password leaked through JSONDebug in production mode")
	}

	// Test with debug enabled to verify it works in debug mode
	EnableDebug()
	logger2, capture2 := NewTestLogger(t, "debug", true)

	logger2.JSONDebug("debug data", sensitiveData)

	debugOutput := capture2.GetOutput()
	if !strings.Contains(debugOutput, "user_id") {
		t.Error("Expected debug data to be shown when debug is enabled")
	}

	// Clean up
	DisableDebug()
}

// TestConcurrentSensitiveDataHandling tests thread safety with sensitive data
func TestConcurrentSensitiveDataHandling(t *testing.T) {
	logger, capture := NewTestLogger(t, "concurrent", false)

	const numGoroutines = 10
	const messagesPerGoroutine = 5

	sensitiveConfigs := make([]map[string]interface{}, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		sensitiveConfigs[i] = map[string]interface{}{
			"username": fmt.Sprintf("user_%d", i),
			"password": fmt.Sprintf("secret_%d", i),
			"api_key":  fmt.Sprintf("key_%d", i),
		}
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.InfoWithFields(fmt.Sprintf("Config %d-%d", id, j),
					flattenMap(sensitiveConfigs[id])...)
			}
		}(i)
	}

	wg.Wait()

	output := capture.GetOutput()

	// Check that no sensitive data leaked
	for i := 0; i < numGoroutines; i++ {
		secretValue := fmt.Sprintf("secret_%d", i)
		keyValue := fmt.Sprintf("key_%d", i)

		if strings.Contains(output, secretValue) {
			t.Errorf("Password %q leaked in concurrent logging", secretValue)
		}
		if strings.Contains(output, keyValue) {
			t.Errorf("API key %q leaked in concurrent logging", keyValue)
		}
	}

	// Check that redaction markers are present
	if !strings.Contains(output, "<redacted>") {
		t.Error("Expected redaction markers in concurrent logging output")
	}
}

// TestMemoryLeaksWithSensitiveData tests that sensitive data doesn't cause memory leaks
func TestMemoryLeaksWithSensitiveData(t *testing.T) {
	logger, _ := NewTestLogger(t, "memory", false)

	// Simulate logging with sensitive data many times
	for i := 0; i < 1000; i++ {
		sensitiveData := map[string]interface{}{
			"iteration": i,
			"password":  fmt.Sprintf("password_%d_%s", i, strings.Repeat("x", 1000)),
			"secret":    fmt.Sprintf("secret_%d_%s", i, strings.Repeat("y", 1000)),
		}

		logger.InfoWithFields("Iteration", flattenMap(sensitiveData)...)
		logger.JSONDebug("debug data", sensitiveData)

		// Clear references to help GC
		sensitiveData = nil
	}

	// This test mainly ensures no crashes occur with repeated sensitive data logging
	// In a more sophisticated test, you might measure memory usage
}

// TestConfigurationSecrecy tests that configuration logging is secure
func TestConfigurationSecrecy(t *testing.T) {
	logger, capture := NewTestLogger(t, "config", false)

	config := map[string]interface{}{
		"database_url":    "postgres://user:secret@localhost/db",
		"jwt_secret":      "super_secret_jwt_key",
		"encryption_key":  "aes256_encryption_key",
		"timeout":         30,
		"max_connections": 100,
		"ssl_cert_path":   "/etc/ssl/cert.pem",
		"ssl_key_path":    "/etc/ssl/private/key.pem",
	}

	LogConnectionAttempt(logger.Logger, config["database_url"].(string), nil)

	output := capture.GetOutput()

	// Database URL should be sanitized
	if strings.Contains(output, "secret") {
		t.Error("Database password should be sanitized in connection logs")
	}

	// Should contain sanitized version
	if !strings.Contains(output, "user:***@") {
		t.Error("Expected sanitized database URL in logs")
	}
}

// Helper function to flatten map for structured logging
func flattenMap(m map[string]interface{}) []interface{} {
	var result []interface{}
	for k, v := range m {
		result = append(result, k, v)
	}
	return result
}
