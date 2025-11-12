package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/schemabounce/kolumn/sdk/helpers/ui"
)

// Level represents the logging level
type Level int

const (
	LevelInfo Level = iota
	LevelDebug
	LevelWarn
	LevelError
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger represents a component-specific logger
type Logger struct {
	component string
	level     Level
	enabled   map[Level]bool
	mu        sync.RWMutex
}

// Configuration holds the logging configuration
type Configuration struct {
	DefaultLevel    Level
	ComponentLevels map[string]Level
	EnableDebug     bool
}

var (
	// Global configuration
	globalConfig = &Configuration{
		DefaultLevel:    LevelInfo,
		ComponentLevels: make(map[string]Level),
		EnableDebug:     false,
	}
	configMu sync.RWMutex

	// Pre-configured provider component loggers
	ProviderLogger   *Logger
	ConnectionLogger *Logger
	HandlerLogger    *Logger
	ValidationLogger *Logger
	SecurityLogger   *Logger
	StateLogger      *Logger
	DiscoveryLogger  *Logger
	ConfigLogger     *Logger
	RegistryLogger   *Logger
	DispatchLogger   *Logger
	SchemaLogger     *Logger

	consoleLogger = log.New(os.Stdout, "", 0)
)

func init() {
	// Initialize from environment variables
	loadEnvironmentConfig()

	// Create pre-configured loggers
	ProviderLogger = NewLogger("provider")
	ConnectionLogger = NewLogger("connection")
	HandlerLogger = NewLogger("handler")
	ValidationLogger = NewLogger("validation")
	SecurityLogger = NewLogger("security")
	StateLogger = NewLogger("state")
	DiscoveryLogger = NewLogger("discovery")
	ConfigLogger = NewLogger("config")
	RegistryLogger = NewLogger("registry")
	DispatchLogger = NewLogger("dispatch")
	SchemaLogger = NewLogger("schema")
}

// NewLogger creates a new component-specific logger
func NewLogger(component string) *Logger {
	configMu.RLock()
	defer configMu.RUnlock()

	// Get component-specific level or use default
	level := globalConfig.DefaultLevel
	if componentLevel, exists := globalConfig.ComponentLevels[component]; exists {
		level = componentLevel
	}

	logger := &Logger{
		component: component,
		level:     level,
		enabled:   make(map[Level]bool),
	}

	// Configure enabled levels
	logger.updateEnabledLevels()

	return logger
}

// updateEnabledLevels configures which levels are enabled based on configuration
// NOTE: This function expects configMu to already be held by the caller
func (l *Logger) updateEnabledLevels() {
	// Reset enabled levels
	for level := range l.enabled {
		delete(l.enabled, level)
	}

	// Always enable warn and error
	l.enabled[LevelWarn] = true
	l.enabled[LevelError] = true

	// Enable info when logger level is Info or Debug
	if l.level == LevelInfo || l.level == LevelDebug {
		l.enabled[LevelInfo] = true
	}

	// Enable debug based on configuration or level
	if globalConfig.EnableDebug || l.level == LevelDebug {
		l.enabled[LevelDebug] = true
	}
}

// Configure updates the global logging configuration
func Configure(config *Configuration) {
	configMu.Lock()
	defer configMu.Unlock()

	if config == nil {
		return
	}

	globalConfig.DefaultLevel = config.DefaultLevel

	globalConfig.EnableDebug = config.EnableDebug

	if config.ComponentLevels != nil {
		for component, level := range config.ComponentLevels {
			globalConfig.ComponentLevels[component] = level
		}
	}

	// Update all existing loggers
	updateAllLoggers()
}

// loadEnvironmentConfig loads configuration from environment variables
func loadEnvironmentConfig() {
	// Check for DEBUG environment variable
	if debug := os.Getenv("DEBUG"); debug != "" {
		if debug == "1" || strings.ToLower(debug) == "true" {
			globalConfig.EnableDebug = true
		}
	}

	// Check for component-specific debug settings
	if debugComponents := os.Getenv("DEBUG_COMPONENTS"); debugComponents != "" {
		components := strings.Split(debugComponents, ",")
		for _, component := range components {
			component = strings.TrimSpace(component)
			if component != "" {
				globalConfig.ComponentLevels[component] = LevelDebug
			}
		}
	}

	// Check for provider-specific debug setting
	if providerDebug := os.Getenv("DEBUG_PROVIDER"); providerDebug == "1" || strings.ToLower(providerDebug) == "true" {
		globalConfig.ComponentLevels["provider"] = LevelDebug
		globalConfig.ComponentLevels["handler"] = LevelDebug
		globalConfig.ComponentLevels["discovery"] = LevelDebug
	}

	// Allow explicit log level override for provider plugins
	if providerLevel := os.Getenv("KOLUMN_PROVIDER_LOG_LEVEL"); providerLevel != "" {
		switch strings.ToLower(providerLevel) {
		case "debug":
			globalConfig.DefaultLevel = LevelDebug
			globalConfig.EnableDebug = true
		case "info":
			globalConfig.DefaultLevel = LevelInfo
		case "warn", "warning":
			globalConfig.DefaultLevel = LevelWarn
		case "error":
			globalConfig.DefaultLevel = LevelError
		}
	}
}

// updateAllLoggers updates the configuration for all existing loggers
func updateAllLoggers() {
	loggers := []*Logger{
		ProviderLogger, ConnectionLogger, HandlerLogger, ValidationLogger,
		SecurityLogger, StateLogger, DiscoveryLogger, ConfigLogger,
		RegistryLogger, DispatchLogger, SchemaLogger,
	}

	for _, logger := range loggers {
		if logger != nil {
			// Update component-specific level
			if componentLevel, exists := globalConfig.ComponentLevels[logger.component]; exists {
				logger.level = componentLevel
			} else {
				logger.level = globalConfig.DefaultLevel
			}
			logger.updateEnabledLevels()
		}
	}
}

// isEnabled checks if the given level is enabled for this logger
func (l *Logger) isEnabled(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled[level]
}

// log outputs a formatted log message
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if !l.isEnabled(level) {
		return
	}

	message := fmt.Sprintf(format, sanitizeArgs(args)...)
	logLine := formatLogLine(level, l.component, message)
	consoleLogger.Println(logLine)
}

// Info logs an info message (always enabled)
func (l *Logger) Info(args ...interface{}) {
	l.logAdaptive(LevelInfo, args...)
}

// Infof logs an info message (printf-style).
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Debug logs a debug message (only enabled in debug mode)
func (l *Logger) Debug(args ...interface{}) {
	l.logAdaptive(LevelDebug, args...)
}

// Debugf logs a debug message (printf-style).
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Warn logs a warning message (always enabled)
func (l *Logger) Warn(args ...interface{}) {
	l.logAdaptive(LevelWarn, args...)
}

// Warnf logs a warning message (printf-style).
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message (always enabled)
func (l *Logger) Error(args ...interface{}) {
	l.logAdaptive(LevelError, args...)
}

// Errorf logs an error message (printf-style).
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

func (l *Logger) logAdaptive(level Level, args ...interface{}) {
	if !l.isEnabled(level) || len(args) == 0 {
		return
	}

	message, ok := args[0].(string)
	if !ok {
		message = fmt.Sprint(args[0])
	}

	rest := args[1:]
	if len(rest) == 0 {
		l.log(level, "%s", message)
		return
	}

	if strings.Contains(message, "%") {
		l.log(level, message, rest...)
		return
	}

	if len(rest)%2 != 0 {
		l.log(level, "%s %v", message, rest)
		return
	}

	l.logWithFields(level, message, rest...)
}

// InfoWithFields logs an info message with structured fields
func (l *Logger) InfoWithFields(message string, fields ...interface{}) {
	if !l.isEnabled(LevelInfo) {
		return
	}
	l.logWithFields(LevelInfo, message, fields...)
}

// DebugWithFields logs a debug message with structured fields
func (l *Logger) DebugWithFields(message string, fields ...interface{}) {
	if !l.isEnabled(LevelDebug) {
		return
	}
	l.logWithFields(LevelDebug, message, fields...)
}

// WarnWithFields logs a warning message with structured fields
func (l *Logger) WarnWithFields(message string, fields ...interface{}) {
	if !l.isEnabled(LevelWarn) {
		return
	}
	l.logWithFields(LevelWarn, message, fields...)
}

// ErrorWithFields logs an error message with structured fields
func (l *Logger) ErrorWithFields(message string, fields ...interface{}) {
	if !l.isEnabled(LevelError) {
		return
	}
	l.logWithFields(LevelError, message, fields...)
}

// logWithFields outputs a structured log message with key-value pairs
func (l *Logger) logWithFields(level Level, message string, fields ...interface{}) {
	var fieldPairs []string

	// Process fields in key-value pairs
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := sanitizeFieldValue(key, fields[i+1])
			fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", key, value))
		}
	}

	if len(fieldPairs) > 0 {
		message = fmt.Sprintf("%s %s", message, strings.Join(fieldPairs, " "))
	}

	logLine := formatLogLine(level, l.component, message)
	consoleLogger.Println(logLine)
}

// JSONDebug logs JSON data only in debug mode with human-readable context
func (l *Logger) JSONDebug(context string, jsonData interface{}) {
	if !l.isEnabled(LevelDebug) {
		return
	}

	if serialized, err := json.Marshal(jsonData); err == nil {
		l.Debug("%s: %s", context, string(serialized))
		return
	}

	humanReadable := JSONToHuman(jsonData, context)
	l.Debug("%s", humanReadable)
}

// OperationStart logs the beginning of an operation
func (l *Logger) OperationStart(operation string, target string) {
	l.Info("Starting %s operation on %s", operation, target)
}

// OperationComplete logs the completion of an operation
func (l *Logger) OperationComplete(operation string, target string) {
	l.Info("Completed %s operation on %s", operation, target)
}

// OperationFailed logs a failed operation
func (l *Logger) OperationFailed(operation string, target string, err error) {
	l.Error("Failed %s operation on %s: %v", operation, target, err)
}

func formatLogLine(level Level, component string, message string) string {
	options := ui.GetStyleOptions()
	componentName := component
	if componentName == "" {
		componentName = "logger"
	}
	return ui.FormatLogLine(level.String(), componentName, message, options)
}

var sensitiveKeyTokens = []string{
	"password",
	"secret",
	"token",
	"credential",
	"auth",
	"access_token",
	"api_key",
	"secret_key",
	"encryption_key",
}

func sanitizeArgs(args []interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}
	sanitized := make([]interface{}, len(args))
	for i, arg := range args {
		if str, ok := arg.(string); ok {
			sanitized[i] = sanitizeStringInput(str)
		} else {
			sanitized[i] = arg
		}
	}
	return sanitized
}

func sanitizeFieldValue(key string, value interface{}) interface{} {
	if shouldRedactKey(key) {
		return "<redacted>"
	}
	if str, ok := value.(string); ok {
		return sanitizeStringInput(str)
	}
	return value
}

func shouldRedactKey(key string) bool {
	lower := strings.ToLower(key)
	for _, token := range sensitiveKeyTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func sanitizeStringInput(input string) string {
	for idx, r := range input {
		if r == '\n' || r == '\r' || r == '\t' || r == 0 {
			return input[:idx]
		}
	}
	return input
}

// IsDebugEnabled returns true if debug logging is enabled for this logger
func (l *Logger) IsDebugEnabled() bool {
	return l.isEnabled(LevelDebug)
}

// GetLevel returns the current log level for this logger
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// GetComponent returns the component name for this logger
func (l *Logger) GetComponent() string {
	return l.component
}

// EnableDebug enables debug logging globally
func EnableDebug() {
	configMu.Lock()
	globalConfig.EnableDebug = true
	configMu.Unlock()
	updateAllLoggers()
}

// DisableDebug disables debug logging globally
func DisableDebug() {
	configMu.Lock()
	globalConfig.EnableDebug = false
	globalConfig.DefaultLevel = LevelInfo
	globalConfig.ComponentLevels = make(map[string]Level)
	configMu.Unlock()
	updateAllLoggers()
}

// EnableComponentDebug enables debug logging for a specific component
func EnableComponentDebug(component string) {
	configMu.Lock()
	defer configMu.Unlock()

	globalConfig.ComponentLevels[component] = LevelDebug
	updateAllLoggers()
}

// SetLogLevel sets the log level for a specific component
func SetLogLevel(component string, level Level) {
	configMu.Lock()
	defer configMu.Unlock()

	globalConfig.ComponentLevels[component] = level
	updateAllLoggers()
}

// GetGlobalDebugStatus returns whether debug logging is globally enabled
func GetGlobalDebugStatus() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig.EnableDebug
}
