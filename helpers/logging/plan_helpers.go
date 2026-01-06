// Package logging provides plan-specific logging quality helpers for Kolumn providers.
// These helpers ensure brief, actionable CLI output with NOOP detection for existing data.
package logging

import (
	"strings"

	"github.com/schemabounce/kolumn/sdk/core"
)

// StripTemplateContext removes the _template_context field from config maps
// to prevent template metadata from polluting config snapshots and logs.
// This should be called before processing resource configurations for plan output.
func StripTemplateContext(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return nil
	}
	result := make(map[string]interface{}, len(config))
	for k, v := range config {
		if k == "_template_context" {
			continue
		}
		result[k] = v
	}
	return result
}

// BuildResourceSummary creates a PlanResourceSummary from a PlanResource.
// This is the foundation for generating clean CLI logging output.
func BuildResourceSummary(resource core.PlanResource) core.PlanResourceSummary {
	cleanConfig := StripTemplateContext(resource.Config)
	return core.PlanResourceSummary{
		ResourceType:   resource.ResourceType,
		Name:           resource.Name,
		Action:         resource.Action,
		ConfigSnapshot: cleanConfig,
	}
}

// BuildResourceSummaryWithNOOP creates a PlanResourceSummary with NOOP detection.
// If exists is true, the action is changed to "noop" with the provided reason.
func BuildResourceSummaryWithNOOP(resource core.PlanResource, exists bool, reason string) core.PlanResourceSummary {
	summary := BuildResourceSummary(resource)
	if exists && resource.Action == "create" {
		summary.Action = "noop"
		summary.Reason = reason
	}
	return summary
}

// IsInsertResourceType returns true if the resource type represents an insert/seed operation.
// Insert resources typically end with "_insert" and require NOOP detection.
func IsInsertResourceType(resourceType string) bool {
	return strings.HasSuffix(resourceType, "_insert")
}

// ExtractInsertConfig extracts common insert configuration fields.
// Returns tableName, values map, and uniqueKeys slice.
func ExtractInsertConfig(config map[string]interface{}) (tableName string, values map[string]interface{}, uniqueKeys []string) {
	if config == nil {
		return "", nil, nil
	}

	// Extract table name (could be "table", "collection", "entity", etc.)
	if t, ok := config["table"].(string); ok {
		tableName = t
	} else if t, ok := config["collection"].(string); ok {
		tableName = t
	} else if t, ok := config["entity"].(string); ok {
		tableName = t
	}

	// Extract values
	if v, ok := config["values"].(map[string]interface{}); ok {
		values = v
	}

	// Extract unique keys
	if uk, ok := config["unique_keys"].([]interface{}); ok {
		uniqueKeys = make([]string, 0, len(uk))
		for _, k := range uk {
			if s, ok := k.(string); ok {
				uniqueKeys = append(uniqueKeys, s)
			}
		}
	} else if uk, ok := config["unique_keys"].([]string); ok {
		uniqueKeys = uk
	}

	return tableName, values, uniqueKeys
}

// BuildUniqueKeyFilter creates a filter map for existence checking based on unique keys.
// This is useful for building WHERE clauses or query filters.
func BuildUniqueKeyFilter(values map[string]interface{}, uniqueKeys []string) map[string]interface{} {
	if len(uniqueKeys) == 0 || values == nil {
		return nil
	}

	filter := make(map[string]interface{}, len(uniqueKeys))
	for _, key := range uniqueKeys {
		if val, ok := values[key]; ok {
			filter[key] = val
		}
	}

	if len(filter) == 0 {
		return nil
	}
	return filter
}

// PlanLoggingContext provides context for plan resource evaluation logging.
type PlanLoggingContext struct {
	ProviderName   string
	Operation      string
	TotalCount     int
	ProcessedCount int
}

// NewPlanLoggingContext creates a new plan logging context.
func NewPlanLoggingContext(providerName string, totalResources int) *PlanLoggingContext {
	return &PlanLoggingContext{
		ProviderName:   providerName,
		Operation:      "Plan",
		TotalCount:     totalResources,
		ProcessedCount: 0,
	}
}

// IncrementProcessed increments the processed resource counter.
func (c *PlanLoggingContext) IncrementProcessed() {
	c.ProcessedCount++
}

// LogPlanStart logs the start of a plan operation.
func LogPlanStart(logger *Logger, ctx *PlanLoggingContext) {
	logger.Info("Planning %d resources for %s", ctx.TotalCount, ctx.ProviderName)
}

// LogPlanComplete logs the completion of a plan operation.
func LogPlanComplete(logger *Logger, ctx *PlanLoggingContext, summaries []core.PlanResourceSummary) {
	createCount := 0
	updateCount := 0
	deleteCount := 0
	noopCount := 0

	for _, s := range summaries {
		switch s.Action {
		case "create":
			createCount++
		case "update":
			updateCount++
		case "delete":
			deleteCount++
		case "noop":
			noopCount++
		}
	}

	if createCount > 0 || updateCount > 0 || deleteCount > 0 {
		logger.Info("Plan complete: %d create, %d update, %d delete, %d unchanged",
			createCount, updateCount, deleteCount, noopCount)
	} else {
		logger.Info("Plan complete: no changes required (%d resources unchanged)", noopCount)
	}
}

// LogResourceSummary logs a single resource summary in concise format.
func LogResourceSummary(logger *Logger, summary core.PlanResourceSummary) {
	if summary.Reason != "" {
		logger.Debug("%s.%s: %s (%s)", summary.ResourceType, summary.Name, summary.Action, summary.Reason)
	} else {
		logger.Debug("%s.%s: %s", summary.ResourceType, summary.Name, summary.Action)
	}
}
