// Package state - serialization.go provides serialization utilities for the SDK state manager
package state

import (
	"encoding/json"
	"fmt"

	"github.com/schemabounce/kolumn/sdk/types"
)

// serializeUniversalState deserializes JSON data into a UniversalState
func serializeUniversalState(data []byte, state *types.UniversalState) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data provided")
	}

	if err := json.Unmarshal(data, state); err != nil {
		return fmt.Errorf("failed to unmarshal state data: %w", err)
	}

	return nil
}

// deserializeUniversalState serializes a UniversalState into JSON data
func deserializeUniversalState(state *types.UniversalState) ([]byte, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return data, nil
}

// SerializeEnhancedState serializes an enhanced resource state to JSON
func SerializeEnhancedState(state *EnhancedResourceState) ([]byte, error) {
	if state == nil {
		return nil, fmt.Errorf("enhanced state cannot be nil")
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enhanced state: %w", err)
	}

	return data, nil
}

// DeserializeEnhancedState deserializes JSON data into an enhanced resource state
func DeserializeEnhancedState(data []byte) (*EnhancedResourceState, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	var state EnhancedResourceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enhanced state data: %w", err)
	}

	return &state, nil
}

// SerializeResourceCollection serializes a resource collection to JSON
func SerializeResourceCollection(collection *ResourceCollection) ([]byte, error) {
	if collection == nil {
		return nil, fmt.Errorf("collection cannot be nil")
	}

	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection: %w", err)
	}

	return data, nil
}

// DeserializeResourceCollection deserializes JSON data into a resource collection
func DeserializeResourceCollection(data []byte) (*ResourceCollection, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	var collection ResourceCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection data: %w", err)
	}

	return &collection, nil
}

// SerializeDriftAnalysis serializes drift analysis to JSON
func SerializeDriftAnalysis(analysis *DriftAnalysis) ([]byte, error) {
	if analysis == nil {
		return nil, fmt.Errorf("drift analysis cannot be nil")
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal drift analysis: %w", err)
	}

	return data, nil
}

// DeserializeDriftAnalysis deserializes JSON data into drift analysis
func DeserializeDriftAnalysis(data []byte) (*DriftAnalysis, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	var analysis DriftAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal drift analysis data: %w", err)
	}

	return &analysis, nil
}

// SerializeGraphAnalysis serializes graph analysis to JSON
func SerializeGraphAnalysis(analysis *GraphAnalysis) ([]byte, error) {
	if analysis == nil {
		return nil, fmt.Errorf("graph analysis cannot be nil")
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal graph analysis: %w", err)
	}

	return data, nil
}

// DeserializeGraphAnalysis deserializes JSON data into graph analysis
func DeserializeGraphAnalysis(data []byte) (*GraphAnalysis, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	var analysis GraphAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph analysis data: %w", err)
	}

	return &analysis, nil
}

// ConvertUniversalToEnhanced converts a UniversalResource to EnhancedResourceState
func ConvertUniversalToEnhanced(universal *types.UniversalResource) (*EnhancedResourceState, error) {
	if universal == nil {
		return nil, fmt.Errorf("universal resource cannot be nil")
	}

	enhanced := &EnhancedResourceState{
		Type:           universal.Type,
		Name:           universal.Name,
		Provider:       universal.Provider,
		Category:       inferCategoryFromProvider(universal.Provider),
		Dependencies:   []ResourceDependency{},
		Collections:    []string{},
		Attributes:     make(map[string]interface{}),
		ComputedAttrs:  make(map[string]interface{}),
		SensitiveAttrs: []string{},
		Metadata: ResourceMetadata{
			Tags:   make(map[string]string),
			Labels: make(map[string]string),
		},
		Instances: []ResourceInstance{},
		Mode:      string(universal.Mode),
	}

	// Convert instances
	for _, instance := range universal.Instances {
		enhancedInstance := ResourceInstance{
			IndexKey:            instance.IndexKey,
			Status:              instance.Status,
			Attributes:          instance.Attributes,
			Private:             instance.Private,
			Metadata:            ResourceMetadata{},
			Tainted:             instance.Tainted,
			Deposed:             instance.Deposed,
			CreateBeforeDestroy: instance.CreateBeforeDestroy,
		}

		// Convert metadata
		for k, v := range instance.Metadata {
			switch k {
			case "tags":
				if tags, ok := v.(map[string]interface{}); ok {
					enhancedInstance.Metadata.Tags = make(map[string]string)
					for tagKey, tagValue := range tags {
						if tagStr, ok := tagValue.(string); ok {
							enhancedInstance.Metadata.Tags[tagKey] = tagStr
						}
					}
				}
			case "labels":
				if labels, ok := v.(map[string]interface{}); ok {
					enhancedInstance.Metadata.Labels = make(map[string]string)
					for labelKey, labelValue := range labels {
						if labelStr, ok := labelValue.(string); ok {
							enhancedInstance.Metadata.Labels[labelKey] = labelStr
						}
					}
				}
			case "owner":
				if owner, ok := v.(string); ok {
					enhancedInstance.Metadata.Owner = owner
				}
			case "environment":
				if env, ok := v.(string); ok {
					enhancedInstance.Metadata.Environment = env
				}
			}
		}

		enhanced.Instances = append(enhanced.Instances, enhancedInstance)
	}

	// Set primary attributes from first instance
	if len(enhanced.Instances) > 0 {
		enhanced.Attributes = enhanced.Instances[0].Attributes
		enhanced.Metadata = enhanced.Instances[0].Metadata
	}

	// Convert dependencies from DependsOn
	for _, depID := range universal.DependsOn {
		parts := parseResourceID(depID)
		if len(parts) == 3 {
			dep := ResourceDependency{
				Provider:     parts[0],
				Type:         parts[1],
				Name:         parts[2],
				Relationship: "depends_on",
				Optional:     false,
			}
			enhanced.Dependencies = append(enhanced.Dependencies, dep)
		}
	}

	// Convert references
	for _, ref := range universal.References {
		parts := parseResourceID(ref.TargetResource)
		if len(parts) == 3 {
			dep := ResourceDependency{
				Provider:     parts[0],
				Type:         parts[1],
				Name:         parts[2],
				Relationship: ref.ReferenceType,
				Optional:     true,
			}
			enhanced.Dependencies = append(enhanced.Dependencies, dep)
		}
	}

	return enhanced, nil
}

// ConvertEnhancedToUniversal converts an EnhancedResourceState to UniversalResource
func ConvertEnhancedToUniversal(enhanced *EnhancedResourceState) (*types.UniversalResource, error) {
	if enhanced == nil {
		return nil, fmt.Errorf("enhanced resource cannot be nil")
	}

	universal := &types.UniversalResource{
		ID:         makeResourceID(enhanced.Provider, enhanced.Type, enhanced.Name),
		Type:       enhanced.Type,
		Name:       enhanced.Name,
		Provider:   enhanced.Provider,
		Mode:       types.ResourceMode(enhanced.Mode),
		Instances:  []types.ResourceInstance{},
		DependsOn:  []string{},
		References: []types.ResourceReference{},
		Metadata:   make(map[string]interface{}),
	}

	// Convert instances
	for _, instance := range enhanced.Instances {
		universalInstance := types.ResourceInstance{
			IndexKey:            instance.IndexKey,
			Status:              instance.Status,
			Attributes:          instance.Attributes,
			Private:             instance.Private,
			Metadata:            make(map[string]interface{}),
			Tainted:             instance.Tainted,
			Deposed:             instance.Deposed,
			CreateBeforeDestroy: instance.CreateBeforeDestroy,
		}

		// Convert metadata
		if len(instance.Metadata.Tags) > 0 {
			universalInstance.Metadata["tags"] = instance.Metadata.Tags
		}
		if len(instance.Metadata.Labels) > 0 {
			universalInstance.Metadata["labels"] = instance.Metadata.Labels
		}
		if instance.Metadata.Owner != "" {
			universalInstance.Metadata["owner"] = instance.Metadata.Owner
		}
		if instance.Metadata.Environment != "" {
			universalInstance.Metadata["environment"] = instance.Metadata.Environment
		}

		universal.Instances = append(universal.Instances, universalInstance)
	}

	// Set primary metadata from enhanced metadata
	if len(enhanced.Metadata.Tags) > 0 {
		universal.Metadata["tags"] = enhanced.Metadata.Tags
	}
	if len(enhanced.Metadata.Labels) > 0 {
		universal.Metadata["labels"] = enhanced.Metadata.Labels
	}
	if enhanced.Metadata.Owner != "" {
		universal.Metadata["owner"] = enhanced.Metadata.Owner
	}
	if enhanced.Metadata.Environment != "" {
		universal.Metadata["environment"] = enhanced.Metadata.Environment
	}

	// Convert dependencies
	for _, dep := range enhanced.Dependencies {
		depID := makeResourceID(dep.Provider, dep.Type, dep.Name)

		if dep.Relationship == "depends_on" && !dep.Optional {
			universal.DependsOn = append(universal.DependsOn, depID)
		} else {
			ref := types.ResourceReference{
				TargetResource:  depID,
				ReferenceType:   dep.Relationship,
				TargetAttribute: "", // Would need to be determined from the actual reference
				SourcePath:      "", // Would need to be determined from the actual reference
			}
			universal.References = append(universal.References, ref)
		}
	}

	return universal, nil
}

// Helper functions

func parseResourceID(resourceID string) []string {
	// Parse resource ID in format "provider.type.name"
	parts := []string{}
	current := ""
	dotCount := 0

	for _, char := range resourceID {
		if char == '.' && dotCount < 2 {
			parts = append(parts, current)
			current = ""
			dotCount++
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func makeResourceID(provider, resourceType, name string) string {
	return provider + "." + resourceType + "." + name
}

func inferCategoryFromProvider(provider string) string {
	switch provider {
	case "postgres", "mysql", "sqlite", "mssql", "mongodb", "redis", "elasticsearch":
		return "database"
	case "s3", "gcs", "azure_blob", "deltalake", "iceberg":
		return "storage"
	case "kafka", "kinesis", "pulsar":
		return "streaming"
	case "airflow", "dagster", "prefect", "temporal":
		return "orchestration"
	case "dbt", "airbyte", "fivetran", "spark":
		return "etl"
	default:
		return "unknown"
	}
}
