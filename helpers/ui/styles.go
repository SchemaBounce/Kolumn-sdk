package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Reset / ANSI color codes
const (
	Reset = "\033[0m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
	BgGray   = "\033[100m"
)

// Provider-specific operations
const (
	OpInit     = "INIT"
	OpPlan     = "PLAN"
	OpApply    = "APPLY"
	OpDestroy  = "DESTROY"
	OpValidate = "VALIDATE"
	OpFormat   = "FORMAT"
	OpDownload = "DOWNLOAD"
	OpUpload   = "UPLOAD"
	OpPackage  = "PACKAGE"
	OpProvider = "PROVIDER"
	OpDatabase = "DATABASE"
	OpCloud    = "CLOUD"
	OpSecurity = "SECURITY"
	OpConfig   = "CONFIG"
	OpState    = "STATE"
	OpCheck    = "CHECK"
	OpFail     = "FAIL"
	OpNext     = "NEXT"
	OpItem     = "ITEM"

	OpSuccess = "SUCCESS"
	OpError   = "ERROR"
	OpWarning = "WARNING"
	OpInfo    = "INFO"
	OpDebug   = "DEBUG"
	OpQuery   = "QUERY"
)

// Resource identifiers
const (
	ResTable          = "TABLE"
	ResView           = "VIEW"
	ResFunction       = "FUNCTION"
	ResIndex          = "INDEX"
	ResTrigger        = "TRIGGER"
	ResObject         = "OBJECT"
	ResUser           = "USER"
	ResRole           = "ROLE"
	ResPermission     = "PERMISSION"
	ResPolicy         = "POLICY"
	ResClassification = "CLASSIFICATION"
	ResResource       = "RESOURCE"
	ResTopic          = "TOPIC"
	ResBucket         = "BUCKET"
	ResStream         = "STREAM"
	ResQueue          = "QUEUE"
	ResCluster        = "CLUSTER"
)

const DefaultProviderName = "KOLUMN"

type ColorScheme struct {
	Success   string
	Error     string
	Warning   string
	Info      string
	Debug     string
	Highlight string
	Muted     string
	Primary   string
	Secondary string
}

var DefaultColorScheme = ColorScheme{
	Success:   Green,
	Warning:   Yellow,
	Error:     Red,
	Info:      Blue,
	Debug:     Gray,
	Highlight: BrightCyan,
	Muted:     Gray,
	Primary:   BrightBlue,
	Secondary: Cyan,
}

var NoColorScheme = ColorScheme{}

type ProgressStyle struct {
	StartChar    string
	ProgressChar string
	EmptyChar    string
	EndChar      string
	Width        int
	Color        string
}

var DefaultProgressStyle = ProgressStyle{
	StartChar:    "[",
	ProgressChar: "█",
	EmptyChar:    "░",
	EndChar:      "]",
	Width:        30,
	Color:        BrightBlue,
}

var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type MessageStyle struct {
	Prefix       string
	Color        string
	SchemaPrefix string
	Bold         bool
	Underline    bool
}

var (
	SuccessStyle = MessageStyle{Prefix: "SUCCESS", Color: Green, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpSuccess), Bold: true}
	ErrorStyle   = MessageStyle{Prefix: "ERROR", Color: Red, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpError), Bold: true}
	WarningStyle = MessageStyle{Prefix: "WARNING", Color: Yellow, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpWarning), Bold: true}
	InfoStyle    = MessageStyle{Prefix: "INFO", Color: Blue, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpInfo)}
	DebugStyle   = MessageStyle{Prefix: "DEBUG", Color: Gray, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpDebug)}

	InitStyle     = MessageStyle{Prefix: "INIT", Color: BrightCyan, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpInit), Bold: true}
	PlanStyle     = MessageStyle{Prefix: "PLAN", Color: BrightYellow, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpPlan), Bold: true}
	ApplyStyle    = MessageStyle{Prefix: "APPLY", Color: BrightGreen, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpApply), Bold: true}
	DestroyStyle  = MessageStyle{Prefix: "DESTROY", Color: BrightRed, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpDestroy), Bold: true}
	ValidateStyle = MessageStyle{Prefix: "VALIDATE", Color: BrightBlue, SchemaPrefix: BuildProviderPrefix(DefaultProviderName, OpValidate), Bold: true}
)

type StyleOptions struct {
	UseColors   bool
	UsePrefixes bool
	UseBold     bool
	Compact     bool
}

var DefaultStyleOptions = StyleOptions{
	UseColors:   true,
	UsePrefixes: true,
	UseBold:     true,
	Compact:     false,
}

func GetStyleOptions() StyleOptions {
	opts := DefaultStyleOptions
	if runtime.GOOS == "windows" {
		opts.UseColors = false
	}
	if os.Getenv("NO_COLOR") != "" {
		opts.UseColors = false
		opts.UseBold = false
	}
	return opts
}

func Colorize(text, color string, use bool) string {
	if !use || color == "" {
		return text
	}
	return color + text + Reset
}

func MakeBold(text string, use bool) string {
	if !use {
		return text
	}
	return Bold + text + Reset
}

func MakeDim(text string, use bool) string {
	if !use {
		return text
	}
	return Dim + text + Reset
}

func MakeUnderline(text string, use bool) string {
	if !use {
		return text
	}
	return Underline + text + Reset
}

func FormatMessageWithStyle(message string, style MessageStyle, options StyleOptions) string {
	var parts []string
	if options.UsePrefixes && style.SchemaPrefix != "" {
		prefix := "[" + style.SchemaPrefix + "]"
		if options.UseColors && style.Color != "" {
			prefix = style.Color + prefix + Reset
		}
		if options.UseBold && style.Bold {
			prefix = Bold + prefix + Reset
		}
		if style.Underline && options.UseColors {
			prefix = Underline + prefix + Reset
		}
		parts = append(parts, prefix)
	}
	parts = append(parts, message)
	sep := " "
	if options.Compact {
		sep = ""
	}
	return strings.Join(parts, sep)
}

func ProgressBar(current, total int, style ProgressStyle, useColors bool) string {
	if total <= 0 {
		return ""
	}
	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(style.Width))
	var bar strings.Builder
	bar.WriteString(style.StartChar)
	for i := 0; i < filled; i++ {
		bar.WriteString(style.ProgressChar)
	}
	for i := filled; i < style.Width; i++ {
		bar.WriteString(style.EmptyChar)
	}
	bar.WriteString(style.EndChar)
	result := bar.String()
	if useColors && style.Color != "" {
		result = style.Color + result + Reset
	}
	return fmt.Sprintf("%s %3.0f%%", result, percentage*100)
}

func FormatDuration(duration string) string {
	return duration
}

func Indent(text string, level int) string {
	indent := strings.Repeat("  ", level)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

func Box(text string, options StyleOptions) string {
	lines := strings.Split(text, "\n")
	max := 0
	for _, line := range lines {
		if len(line) > max {
			max = len(line)
		}
	}
	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", max+2) + "┐\n")
	for _, line := range lines {
		padding := max - len(line)
		b.WriteString("│ " + line + strings.Repeat(" ", padding) + " │\n")
	}
	b.WriteString("└" + strings.Repeat("─", max+2) + "┘")
	return b.String()
}

func Table(headers []string, rows [][]string, options StyleOptions) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	var out strings.Builder
	out.WriteString("│")
	for i, header := range headers {
		padding := widths[i] - len(header)
		if options.UseColors {
			header = Bold + header + Reset
		}
		out.WriteString(" " + header + strings.Repeat(" ", padding) + " │")
	}
	out.WriteString("\n├")
	for i, width := range widths {
		out.WriteString(strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			out.WriteString("┼")
		}
	}
	out.WriteString("┤\n")
	for _, row := range rows {
		out.WriteString("│")
		for i, cell := range row {
			if i < len(widths) {
				padding := widths[i] - len(cell)
				out.WriteString(" " + cell + strings.Repeat(" ", padding) + " │")
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

func BuildProviderPrefix(provider, operation string) string {
	return NormalizeProviderName(provider) + "-" + operation
}

func BuildProviderPrefixWithSchema(schema interface{}, operation string) string {
	if schemaMap, ok := schema.(map[string]interface{}); ok {
		if displayName, ok := schemaMap["display_name"].(string); ok && displayName != "" {
			return strings.ToUpper(displayName) + "-" + operation
		}
		if name, ok := schemaMap["name"].(string); ok && name != "" {
			return NormalizeProviderName(name) + "-" + operation
		}
	}
	return BuildProviderPrefix(DefaultProviderName, operation)
}

func BuildResourcePrefix(provider, resourceType string) string {
	return NormalizeProviderName(provider) + "-" + GetResourceTypeName(resourceType)
}

func NormalizeProviderName(provider string) string {
	if provider == "" {
		return DefaultProviderName
	}
	return strings.ToUpper(provider)
}

func GetResourceTypeName(resourceType string) string {
	r := strings.ToLower(resourceType)
	switch {
	case strings.Contains(r, "table"):
		return ResTable
	case strings.Contains(r, "topic"):
		return ResTopic
	case strings.Contains(r, "bucket"):
		return ResBucket
	case strings.Contains(r, "stream"):
		return ResStream
	case strings.Contains(r, "queue"):
		return ResQueue
	case strings.Contains(r, "cluster"):
		return ResCluster
	case strings.Contains(r, "view"):
		return ResView
	case strings.Contains(r, "function"):
		return ResFunction
	case strings.Contains(r, "index"):
		return ResIndex
	case strings.Contains(r, "trigger"):
		return ResTrigger
	case strings.Contains(r, "schema"):
		return ResObject
	case strings.Contains(r, "user"):
		return ResUser
	case strings.Contains(r, "role"):
		return ResRole
	case strings.Contains(r, "permission"):
		return ResPermission
	case strings.Contains(r, "policy"):
		return ResPolicy
	case strings.Contains(r, "classification"):
		return ResClassification
	default:
		return ResResource
	}
}

func GetOperationType(operation string) string {
	switch strings.ToLower(operation) {
	case "create", "creating":
		return OpCheck
	case "update", "updating":
		return OpConfig
	case "delete", "deleting", "destroy", "destroying":
		return OpFail
	case "read", "reading":
		return OpInfo
	case "plan", "planning":
		return OpPlan
	case "apply", "applying":
		return OpApply
	case "validate", "validating":
		return OpValidate
	case "init", "initializing":
		return OpInit
	case "success":
		return OpSuccess
	case "error":
		return OpError
	case "warning":
		return OpWarning
	case "info":
		return OpInfo
	case "debug":
		return OpDebug
	default:
		return OpNext
	}
}

func OperationPrefix(operation string) string {
	return BuildProviderPrefix(DefaultProviderName, GetOperationType(operation))
}

func FormatResourceMessage(resourceType, name, action string, options StyleOptions) string {
	var parts []string
	if options.UsePrefixes {
		prefix := "[" + ResourceTypePrefix(resourceType) + "]"
		if options.UseColors {
			prefix = BrightBlue + prefix + Reset
		}
		parts = append(parts, prefix)
	}
	resource := resourceType + "." + name
	if options.UseColors {
		resource = Bold + resource + Reset
	}
	actionText := action
	if options.UseColors {
		actionText = Cyan + actionText + Reset
	}
	parts = append(parts, resource+": "+actionText)
	sep := " "
	if options.Compact {
		sep = ""
	}
	return strings.Join(parts, sep)
}

func FormatProviderMessage(providerType, action string, options StyleOptions) string {
	var parts []string
	if options.UsePrefixes {
		prefix := "[" + BuildProviderPrefix(DefaultProviderName, OpProvider) + "]"
		if options.UseColors {
			prefix = BrightMagenta + prefix + Reset
		}
		parts = append(parts, prefix)
	}
	provider := "provider." + providerType
	if options.UseColors {
		provider = Bold + provider + Reset
	}
	actionText := action
	if options.UseColors {
		actionText = Cyan + actionText + Reset
	}
	parts = append(parts, provider+": "+actionText)
	sep := " "
	if options.Compact {
		sep = ""
	}
	return strings.Join(parts, sep)
}

func ResourceTypePrefix(resourceType string) string {
	return BuildResourcePrefix(DefaultProviderName, resourceType)
}

func StyleForSeverity(severity string) MessageStyle {
	switch strings.ToUpper(severity) {
	case "ERROR":
		return ErrorStyle
	case "WARNING", "WARN":
		return WarningStyle
	case "DEBUG":
		return DebugStyle
	case "SUCCESS":
		return SuccessStyle
	default:
		return InfoStyle
	}
}

func OperationForSeverity(severity string) string {
	switch strings.ToUpper(severity) {
	case "ERROR":
		return OpError
	case "WARNING", "WARN":
		return OpWarning
	case "DEBUG":
		return OpDebug
	case "SUCCESS":
		return OpSuccess
	default:
		return OpInfo
	}
}

func FormatComponentTag(component string, options StyleOptions) string {
	component = strings.TrimSpace(component)
	if component == "" {
		return ""
	}
	component = strings.ToUpper(component)
	if len(component) > 18 {
		component = component[:18]
	}
	label := fmt.Sprintf("%-18s │", component)
	if options.UseColors {
		return MakeDim(label, true)
	}
	return label
}

func FormatLogLine(severity string, component string, message string, options StyleOptions) string {
	style := StyleForSeverity(severity)
	style.SchemaPrefix = BuildProviderPrefix(DefaultProviderName, OperationForSeverity(severity))
	componentTag := FormatComponentTag(component, options)
	if componentTag != "" {
		message = componentTag + " " + message
	}
	return FormatMessageWithStyle(message, style, options)
}
