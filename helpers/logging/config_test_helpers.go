package logging

import (
	"os"
	"testing"
)

// resetGlobalConfig resets logging configuration and clears env overrides.
func resetGlobalConfig() {
	os.Unsetenv("DEBUG")
	os.Unsetenv("DEBUG_COMPONENTS")
	os.Unsetenv("DEBUG_PROVIDER")
	os.Unsetenv("KOLUMN_PROVIDER_LOG_LEVEL")

	Configure(&Configuration{
		DefaultLevel:    LevelInfo,
		ComponentLevels: make(map[string]Level),
		EnableDebug:     false,
	})
}

// withCleanConfig ensures each test runs with a clean configuration baseline.
func withCleanConfig(t *testing.T) {
	resetGlobalConfig()
	t.Cleanup(func() {
		resetGlobalConfig()
	})
}
