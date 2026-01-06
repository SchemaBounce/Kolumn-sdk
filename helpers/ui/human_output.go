package ui

import "strings"

// HumanStatusColor returns the color to use for a human-facing status tag.
func HumanStatusColor(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "IN_SYNC", "APPLY", "APPLIED":
		return BrightGreen
	case "DRIFT", "VALIDATION":
		return BrightYellow
	case "ERROR", "APPLY_FAILED", "FAILED":
		return BrightRed
	default:
		return ""
	}
}

func levelColor(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR":
		return BrightRed
	case "WARN", "WARNING":
		return BrightYellow
	case "DEBUG":
		return Gray
	default:
		return BrightBlue
	}
}

func formatTag(tag string, color string, options StyleOptions) string {
	text := "[" + tag + "]"
	if options.UseColors && color != "" {
		text = color + text + Reset
	}
	if options.UseBold {
		text = Bold + text + Reset
	}
	return text
}

// FormatHumanStatusLine renders a human-first line:
// [KOLUMN-LEVEL] [PROVIDER] [STATUS] subject detail
// Color is applied to the bracketed tags only.
func FormatHumanStatusLine(level, provider, status, subject, detail string, options StyleOptions) string {
	levelTag := strings.ToUpper(strings.TrimSpace(level))
	if levelTag == "" {
		levelTag = "INFO"
	}
	providerTag := strings.ToUpper(strings.TrimSpace(provider))
	if providerTag == "" {
		providerTag = DefaultProviderName
	}
	statusTag := strings.ToUpper(strings.TrimSpace(status))
	if statusTag == "" {
		statusTag = "STATUS"
	}

	if !options.UsePrefixes {
		var messageParts []string
		if subject != "" {
			messageParts = append(messageParts, subject)
		}
		if detail != "" {
			messageParts = append(messageParts, detail)
		}
		return strings.Join(messageParts, " ")
	}

	tagColor := HumanStatusColor(statusTag)
	if tagColor == "" {
		tagColor = levelColor(levelTag)
	}

	parts := []string{
		formatTag("KOLUMN-"+levelTag, tagColor, options),
		formatTag(providerTag, tagColor, options),
		formatTag(statusTag, tagColor, options),
	}
	if subject != "" {
		parts = append(parts, subject)
	}
	if detail != "" {
		parts = append(parts, detail)
	}

	sep := " "
	if options.Compact {
		sep = ""
	}
	return strings.Join(parts, sep)
}

// RedCaret returns a red caret marker (respects color options).
func RedCaret(options StyleOptions) string {
	if options.UseColors {
		return BrightRed + "^" + Reset
	}
	return "^"
}

// RenderLineWithCaret inserts a caret at the 1-based column position and appends an optional annotation.
func RenderLineWithCaret(line string, column int, annotation string, options StyleOptions) string {
	caret := RedCaret(options)
	runes := []rune(line)
	if column < 1 {
		column = len(runes) + 1
	}
	if column > len(runes)+1 {
		column = len(runes) + 1
	}

	insertPos := column - 1
	var sb strings.Builder
	sb.WriteString(string(runes[:insertPos]))
	sb.WriteString(caret)
	sb.WriteString(string(runes[insertPos:]))

	result := sb.String()
	if annotation != "" {
		if !strings.HasSuffix(result, " ") {
			result += " "
		}
		result += annotation
	}
	return result
}

// FormatTerraformStyle renders output in Terraform's clean resource-first format:
// resource_type.resource_name: Action...
// resource_type.resource_name: Action complete [duration]
func FormatTerraformStyle(resourceType, resourceName, action string, options StyleOptions) string {
	resourceType = strings.TrimSpace(resourceType)
	resourceName = strings.TrimSpace(resourceName)
	action = strings.TrimSpace(action)

	// Build resource identifier
	var resource string
	if resourceName != "" {
		resource = resourceType + "." + resourceName
	} else {
		resource = resourceType
	}

	// Colorize based on action
	if options.UseColors {
		actionLower := strings.ToLower(action)
		switch {
		case strings.Contains(actionLower, "creat"):
			resource = BrightGreen + resource + Reset
		case strings.Contains(actionLower, "updat"), strings.Contains(actionLower, "modif"):
			resource = BrightYellow + resource + Reset
		case strings.Contains(actionLower, "delet"), strings.Contains(actionLower, "destroy"):
			resource = BrightRed + resource + Reset
		case strings.Contains(actionLower, "refresh"), strings.Contains(actionLower, "read"):
			resource = Cyan + resource + Reset
		default:
			resource = BrightBlue + resource + Reset
		}
	}

	if action == "" {
		return resource
	}

	return resource + ": " + action
}

// FormatTerraformStyleWithDuration includes timing information
func FormatTerraformStyleWithDuration(resourceType, resourceName, action, duration string, options StyleOptions) string {
	base := FormatTerraformStyle(resourceType, resourceName, action, options)
	if duration == "" {
		return base
	}

	durationStr := "[" + duration + "]"
	if options.UseColors {
		durationStr = Gray + durationStr + Reset
	}
	return base + " " + durationStr
}

// ParseStructuredLog parses Go slog-style structured log lines and extracts key fields.
// Input: "2025/12/19 00:07:38 INFO Creating table backup component=postgres table=users"
// Returns: level, message, and a map of key=value fields
func ParseStructuredLog(line string) (level string, message string, fields map[string]string) {
	fields = make(map[string]string)
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", fields
	}

	// Check for timestamp pattern at start: "2006/01/02 15:04:05"
	if len(line) >= 19 && line[4] == '/' && line[7] == '/' && line[10] == ' ' && line[13] == ':' && line[16] == ':' {
		line = strings.TrimSpace(line[19:])
	}

	// Extract level (INFO, WARN, ERROR, DEBUG)
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return "", line, fields
	}

	potentialLevel := strings.ToUpper(parts[0])
	switch potentialLevel {
	case "INFO", "WARN", "WARNING", "ERROR", "DEBUG", "TRACE":
		level = potentialLevel
		line = parts[1]
	default:
		// No level prefix, treat whole line as message
		level = "INFO"
	}

	// Parse key=value pairs from the end
	// Split on spaces but respect quoted values
	tokens := tokenizeLogLine(line)
	messageTokens := []string{}
	for _, tok := range tokens {
		if idx := strings.Index(tok, "="); idx > 0 && idx < len(tok)-1 {
			key := tok[:idx]
			value := tok[idx+1:]
			// Remove quotes from value
			value = strings.Trim(value, "\"'")
			fields[key] = value
		} else {
			messageTokens = append(messageTokens, tok)
		}
	}
	message = strings.Join(messageTokens, " ")

	return level, message, fields
}

// tokenizeLogLine splits a log line respecting quoted strings
func tokenizeLogLine(line string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range line {
		switch {
		case (ch == '"' || ch == '\'') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// SimplifyProviderLog converts verbose provider logs to Terraform-style output.
// Input: "2025/12/19 00:07:38 INFO Creating table backup component=postgres-capability-provider table=provider_registry schema=schemabounce_providers backup_table=backup_schemabounce_providers_provider_registry sql=\"CREATE TABLE...\""
// Output: "postgres_table.provider_registry: Creating backup..."
func SimplifyProviderLog(line string, providerType string, options StyleOptions) string {
	level, message, fields := ParseStructuredLog(line)

	// Skip DEBUG unless verbose
	if level == "DEBUG" {
		return ""
	}

	// Try to extract resource type and name from fields
	resourceType := ""
	resourceName := ""

	// Common field patterns
	if table, ok := fields["table"]; ok {
		resourceType = providerType + "_table"
		resourceName = table
		// Include schema if present
		if schema, ok := fields["schema"]; ok && schema != "" {
			resourceName = schema + "." + resourceName
		}
	} else if view, ok := fields["view"]; ok {
		resourceType = providerType + "_view"
		resourceName = view
	} else if topic, ok := fields["topic"]; ok {
		resourceType = providerType + "_topic"
		resourceName = topic
	} else if bucket, ok := fields["bucket"]; ok {
		resourceType = providerType + "_bucket"
		resourceName = bucket
	} else if role, ok := fields["role"]; ok {
		resourceType = providerType + "_role"
		resourceName = role
	} else if name, ok := fields["name"]; ok {
		resourceType = providerType + "_resource"
		resourceName = name
	}

	// Clean up message - remove redundant info already captured in fields
	message = simplifyMessage(message)

	// If we have a resource, format Terraform-style
	if resourceType != "" && resourceName != "" {
		return FormatTerraformStyle(resourceType, resourceName, message, options)
	}

	// Fallback: just clean the message and show it simply
	if message != "" {
		if options.UseColors && level == "ERROR" {
			return BrightRed + providerType + Reset + ": " + message
		}
		if options.UseColors && level == "WARN" {
			return BrightYellow + providerType + Reset + ": " + message
		}
		return providerType + ": " + message
	}

	return ""
}

// simplifyMessage removes common noise from log messages
func simplifyMessage(msg string) string {
	// Simplify common verbose phrases
	replacements := map[string]string{
		"Creating table backup":       "Creating backup...",
		"Backup created successfully": "Backup complete",
		"Starting migration":          "Migrating...",
		"Migration completed":         "Migration complete",
		"Executing SQL":               "Executing...",
		"SQL executed successfully":   "Executed",
		"Checking resource existence": "Checking...",
		"Resource exists":             "Exists",
		"Resource does not exist":     "Creating...",
	}

	for old, new := range replacements {
		if strings.Contains(msg, old) {
			msg = strings.Replace(msg, old, new, 1)
		}
	}

	return strings.TrimSpace(msg)
}
