//go:build windows

package ui

// detectTerminalWidth returns 0 on Windows, falling back to COLUMNS env var or default.
// Windows terminal width detection requires kernel32.dll GetConsoleScreenBufferInfo
// which is not worth the complexity for the SDK's minimal dependency policy.
func detectTerminalWidth() int {
	return 0
}
