// Package state - collections.go provides resource collection management capabilities for the SDK
package state

import (
	"context"
	"fmt"
	"time"
)

// ResourceCollectionManager manages resource collections and their lifecycle
type ResourceCollectionManager struct {
	manager *DefaultManager
}

// NewResourceCollectionManager creates a new resource collection manager
func NewResourceCollectionManager(manager *DefaultManager) *ResourceCollectionManager {
	return &ResourceCollectionManager{
		manager: manager,
	}
}

// CreateCollection creates a new resource collection
func (c *ResourceCollectionManager) CreateCollection(ctx context.Context, definition *CollectionDefinition) (*ResourceCollection, error) {
	// Validate collection definition
	if err := c.validateCollectionDefinition(definition); err != nil {
		return nil, fmt.Errorf("invalid collection definition: %w", err)
	}

	// Check if collection already exists
	existing, err := c.GetCollection(ctx, definition.ID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("collection %s already exists", definition.ID)
	}

	// Verify all referenced resources exist
	if err := c.validateResourceReferences(ctx, definition.Resources); err != nil {
		return nil, fmt.Errorf("resource reference validation failed: %w", err)
	}

	// Create collection
	collection := &ResourceCollection{
		ID:           definition.ID,
		Name:         definition.Name,
		Type:         definition.Type,
		Description:  definition.Description,
		Resources:    definition.Resources,
		Metadata:     definition.Metadata,
		DesiredState: "active",
		Status: CollectionStatus{
			State:         "creating",
			Healthy:       false,
			ResourceCount: len(definition.Resources),
			HealthyCount:  0,
			ErrorCount:    0,
			LastUpdated:   time.Now(),
			Issues:        []string{},
			LastCheck:     time.Now(),
		},
		Dependencies: definition.Dependencies,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store collection metadata
	if err := c.storeCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to store collection: %w", err)
	}

	// Update resource assignments
	if err := c.assignResourcesToCollection(ctx, collection.ID, definition.Resources); err != nil {
		return nil, fmt.Errorf("failed to assign resources to collection: %w", err)
	}

	// Perform initial health check
	if err := c.updateCollectionStatus(ctx, collection); err != nil {
		// Log error but don't fail collection creation
		collection.Status.Issues = append(collection.Status.Issues,
			fmt.Sprintf("Initial health check failed: %v", err))
	}

	collection.Status.State = "active"
	if err := c.storeCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to update collection status: %w", err)
	}

	return collection, nil
}

// GetCollection retrieves a collection by ID
func (c *ResourceCollectionManager) GetCollection(ctx context.Context, collectionID string) (*ResourceCollection, error) {
	return c.loadCollection(ctx, collectionID)
}

// UpdateCollection updates an existing resource collection
func (c *ResourceCollectionManager) UpdateCollection(ctx context.Context, collectionID string, updates *CollectionUpdates) (*ResourceCollection, error) {
	// Load existing collection
	collection, err := c.loadCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection %s: %w", collectionID, err)
	}

	originalState := collection.Status.State
	collection.Status.State = "updating"
	collection.UpdatedAt = time.Now()

	// Apply updates
	if updates.Name != "" {
		collection.Name = updates.Name
	}
	if updates.Description != "" {
		collection.Description = updates.Description
	}
	if updates.DesiredState != "" {
		collection.DesiredState = updates.DesiredState
	}
	if len(updates.ResourcesAdded) > 0 {
		if err := c.validateResourceReferences(ctx, updates.ResourcesAdded); err != nil {
			return nil, fmt.Errorf("invalid resources to add: %w", err)
		}
		collection.Resources = append(collection.Resources, updates.ResourcesAdded...)
	}
	if len(updates.ResourcesRemoved) > 0 {
		collection.Resources = c.removeResourceReferences(collection.Resources, updates.ResourcesRemoved)
	}
	if len(updates.MetadataUpdates) > 0 {
		c.applyMetadataUpdates(&collection.Metadata, updates.MetadataUpdates)
	}

	// Update resource count
	collection.Status.ResourceCount = len(collection.Resources)

	// Save updated collection
	if err := c.storeCollection(ctx, collection); err != nil {
		collection.Status.State = originalState // Rollback state
		return nil, fmt.Errorf("failed to save updated collection: %w", err)
	}

	// Update resource assignments
	if len(updates.ResourcesAdded) > 0 {
		if err := c.assignResourcesToCollection(ctx, collectionID, updates.ResourcesAdded); err != nil {
			return nil, fmt.Errorf("failed to assign new resources: %w", err)
		}
	}
	if len(updates.ResourcesRemoved) > 0 {
		if err := c.unassignResourcesFromCollection(ctx, collectionID, updates.ResourcesRemoved); err != nil {
			return nil, fmt.Errorf("failed to unassign resources: %w", err)
		}
	}

	// Update collection status
	if err := c.updateCollectionStatus(ctx, collection); err != nil {
		collection.Status.Issues = append(collection.Status.Issues,
			fmt.Sprintf("Status update failed: %v", err))
	}

	collection.Status.State = "active"
	if err := c.storeCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to finalize collection update: %w", err)
	}

	return collection, nil
}

// DeleteCollection deletes a resource collection
func (c *ResourceCollectionManager) DeleteCollection(ctx context.Context, collectionID string, deleteResources bool) error {
	// Load collection
	collection, err := c.loadCollection(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to load collection %s: %w", collectionID, err)
	}

	// Check for dependent collections
	dependentCollections, err := c.findDependentCollections(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to check for dependent collections: %w", err)
	}
	if len(dependentCollections) > 0 {
		return fmt.Errorf("cannot delete collection %s: dependent collections exist: %v",
			collectionID, dependentCollections)
	}

	// Update collection state
	collection.Status.State = "deleting"
	collection.UpdatedAt = time.Now()
	if err := c.storeCollection(ctx, collection); err != nil {
		return fmt.Errorf("failed to update collection state: %w", err)
	}

	// Optionally delete resources (this would require integration with providers)
	if deleteResources {
		if err := c.deleteCollectionResources(ctx, collection); err != nil {
			return fmt.Errorf("failed to delete collection resources: %w", err)
		}
	} else {
		// Just unassign resources from collection
		if err := c.unassignResourcesFromCollection(ctx, collectionID, collection.Resources); err != nil {
			return fmt.Errorf("failed to unassign resources: %w", err)
		}
	}

	// Delete collection metadata
	if err := c.deleteCollection(ctx, collectionID); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

// ListCollections lists all collections
func (c *ResourceCollectionManager) ListCollections(ctx context.Context) ([]*ResourceCollection, error) {
	return c.listAllCollections(ctx)
}

// ListCollectionsByType lists all collections of a specific type
func (c *ResourceCollectionManager) ListCollectionsByType(ctx context.Context, collectionType string) ([]*ResourceCollection, error) {
	allCollections, err := c.listAllCollections(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*ResourceCollection
	for _, collection := range allCollections {
		if collection.Type == collectionType {
			filtered = append(filtered, collection)
		}
	}

	return filtered, nil
}

// GetCollectionResources gets all resources in a collection with their current state
func (c *ResourceCollectionManager) GetCollectionResources(ctx context.Context, collectionID string) ([]*EnhancedResourceState, error) {
	collection, err := c.loadCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection %s: %w", collectionID, err)
	}

	resources := make([]*EnhancedResourceState, 0)
	for _, ref := range collection.Resources {
		resource, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			// Resource might not exist anymore, skip it
			continue
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// GetCollectionStatus gets the current status of a collection
func (c *ResourceCollectionManager) GetCollectionStatus(ctx context.Context, collectionID string) (*CollectionStatus, error) {
	collection, err := c.loadCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection %s: %w", collectionID, err)
	}

	// Update status
	if err := c.updateCollectionStatus(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to update collection status: %w", err)
	}

	// Save updated status
	if err := c.storeCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to save collection status: %w", err)
	}

	return &collection.Status, nil
}

// ValidateCollection validates the integrity of a collection
func (c *ResourceCollectionManager) ValidateCollection(ctx context.Context, collectionID string) (*CollectionValidation, error) {
	collection, err := c.loadCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection %s: %w", collectionID, err)
	}

	validation := &CollectionValidation{
		CollectionID:     collectionID,
		Valid:            true,
		ValidationTime:   time.Now(),
		MissingResources: []string{},
		InvalidResources: []string{},
		Issues:           []string{},
		Recommendations:  []string{},
	}

	// Validate resource references
	for _, ref := range collection.Resources {
		resource, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			validation.Valid = false
			resourceID := c.makeResourceID(ref.Provider, ref.Type, ref.Name)
			validation.MissingResources = append(validation.MissingResources, resourceID)
			validation.Issues = append(validation.Issues,
				fmt.Sprintf("Resource %s not found", resourceID))
			continue
		}

		// Validate resource state
		if resource.LifecycleState == "error" || resource.LifecycleState == "deleted" {
			validation.Valid = false
			resourceID := c.makeResourceID(ref.Provider, ref.Type, ref.Name)
			validation.InvalidResources = append(validation.InvalidResources, resourceID)
			validation.Issues = append(validation.Issues,
				fmt.Sprintf("Resource %s in invalid state: %s", resourceID, resource.LifecycleState))
		}

		// Check if resource is assigned to multiple collections
		if len(resource.Collections) > 1 {
			validation.Recommendations = append(validation.Recommendations,
				fmt.Sprintf("Resource %s is assigned to multiple collections",
					c.makeResourceID(ref.Provider, ref.Type, ref.Name)))
		}
	}

	// Validate collection dependencies
	for _, dep := range collection.Dependencies {
		dependentCollection, err := c.loadCollection(ctx, dep.CollectionID)
		if err != nil {
			validation.Valid = false
			validation.Issues = append(validation.Issues,
				fmt.Sprintf("Dependent collection %s not found", dep.CollectionID))
			continue
		}

		if dependentCollection.Status.State != "active" {
			validation.Issues = append(validation.Issues,
				fmt.Sprintf("Dependent collection %s not in active state", dep.CollectionID))
		}
	}

	return validation, nil
}

// Helper methods

func (c *ResourceCollectionManager) validateCollectionDefinition(definition *CollectionDefinition) error {
	if definition.ID == "" {
		return fmt.Errorf("collection ID is required")
	}
	if definition.Name == "" {
		return fmt.Errorf("collection name is required")
	}
	if definition.Type == "" {
		return fmt.Errorf("collection type is required")
	}
	if len(definition.Resources) == 0 {
		return fmt.Errorf("collection must have at least one resource")
	}

	// Validate resource references
	for _, ref := range definition.Resources {
		if ref.Provider == "" || ref.Type == "" || ref.Name == "" {
			return fmt.Errorf("resource reference must have provider, type, and name")
		}
	}

	return nil
}

func (c *ResourceCollectionManager) validateResourceReferences(ctx context.Context, resources []ResourceReference) error {
	for _, ref := range resources {
		_, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			return fmt.Errorf("resource %s.%s.%s not found", ref.Provider, ref.Type, ref.Name)
		}
	}
	return nil
}

func (c *ResourceCollectionManager) assignResourcesToCollection(ctx context.Context, collectionID string, resources []ResourceReference) error {
	for _, ref := range resources {
		resource, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			continue // Skip missing resources
		}

		// Add collection to resource's collection list
		found := false
		for _, existingCollection := range resource.Collections {
			if existingCollection == collectionID {
				found = true
				break
			}
		}
		if !found {
			resource.Collections = append(resource.Collections, collectionID)
		}

		// Save updated resource
		if err := c.storeEnhancedResource(ctx, resource); err != nil {
			return fmt.Errorf("failed to update resource %s: %w",
				c.makeResourceID(ref.Provider, ref.Type, ref.Name), err)
		}
	}
	return nil
}

func (c *ResourceCollectionManager) unassignResourcesFromCollection(ctx context.Context, collectionID string, resources []ResourceReference) error {
	for _, ref := range resources {
		resource, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			continue // Skip missing resources
		}

		// Remove collection from resource's collection list
		var updatedCollections []string
		for _, existingCollection := range resource.Collections {
			if existingCollection != collectionID {
				updatedCollections = append(updatedCollections, existingCollection)
			}
		}
		resource.Collections = updatedCollections

		// Save updated resource
		if err := c.storeEnhancedResource(ctx, resource); err != nil {
			return fmt.Errorf("failed to update resource %s: %w",
				c.makeResourceID(ref.Provider, ref.Type, ref.Name), err)
		}
	}
	return nil
}

func (c *ResourceCollectionManager) updateCollectionStatus(ctx context.Context, collection *ResourceCollection) error {
	healthyCount := 0
	errorCount := 0
	var issues []string

	for _, ref := range collection.Resources {
		resource, err := c.loadEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name)
		if err != nil {
			errorCount++
			issues = append(issues, fmt.Sprintf("Resource %s.%s.%s not found",
				ref.Provider, ref.Type, ref.Name))
			continue
		}

		switch resource.LifecycleState {
		case "active":
			if !resource.DriftDetected {
				healthyCount++
			} else {
				issues = append(issues, fmt.Sprintf("Resource %s has configuration drift",
					c.makeResourceID(ref.Provider, ref.Type, ref.Name)))
			}
		case "error", "failed":
			errorCount++
			issues = append(issues, fmt.Sprintf("Resource %s in error state",
				c.makeResourceID(ref.Provider, ref.Type, ref.Name)))
		}
	}

	collection.Status.HealthyCount = healthyCount
	collection.Status.ErrorCount = errorCount
	collection.Status.Issues = issues
	collection.Status.LastCheck = time.Now()
	collection.Status.LastUpdated = time.Now()

	// Determine overall health
	totalResources := len(collection.Resources)
	if totalResources == 0 {
		collection.Status.Healthy = false
		collection.Status.State = "empty"
	} else if errorCount == 0 && healthyCount == totalResources {
		collection.Status.Healthy = true
		if collection.Status.State != "deleting" && collection.Status.State != "updating" {
			collection.Status.State = "active"
		}
	} else if errorCount > totalResources/2 {
		collection.Status.Healthy = false
		collection.Status.State = "degraded"
	} else {
		collection.Status.Healthy = false
		collection.Status.State = "partial"
	}

	return nil
}

func (c *ResourceCollectionManager) findDependentCollections(ctx context.Context, collectionID string) ([]string, error) {
	allCollections, err := c.listAllCollections(ctx)
	if err != nil {
		return nil, err
	}

	var dependents []string
	for _, collection := range allCollections {
		for _, dep := range collection.Dependencies {
			if dep.CollectionID == collectionID {
				dependents = append(dependents, collection.ID)
				break
			}
		}
	}

	return dependents, nil
}

func (c *ResourceCollectionManager) deleteCollectionResources(ctx context.Context, collection *ResourceCollection) error {
	// This would require integration with the actual providers to delete resources
	// For now, we'll just remove them from state
	for _, ref := range collection.Resources {
		if err := c.deleteEnhancedResource(ctx, ref.Provider, ref.Type, ref.Name); err != nil {
			return fmt.Errorf("failed to delete resource %s: %w",
				c.makeResourceID(ref.Provider, ref.Type, ref.Name), err)
		}
	}

	return nil
}

func (c *ResourceCollectionManager) removeResourceReferences(current []ResourceReference, toRemove []ResourceReference) []ResourceReference {
	removeSet := make(map[string]bool)
	for _, ref := range toRemove {
		key := fmt.Sprintf("%s.%s.%s", ref.Provider, ref.Type, ref.Name)
		removeSet[key] = true
	}

	var result []ResourceReference
	for _, ref := range current {
		key := fmt.Sprintf("%s.%s.%s", ref.Provider, ref.Type, ref.Name)
		if !removeSet[key] {
			result = append(result, ref)
		}
	}

	return result
}

func (c *ResourceCollectionManager) applyMetadataUpdates(metadata *ResourceMetadata, updates map[string]interface{}) {
	for key, value := range updates {
		switch key {
		case "owner":
			if str, ok := value.(string); ok {
				metadata.Owner = str
			}
		case "environment":
			if str, ok := value.(string); ok {
				metadata.Environment = str
			}
		case "project":
			if str, ok := value.(string); ok {
				metadata.Project = str
			}
		case "team":
			if str, ok := value.(string); ok {
				metadata.ContactInfo.Team = str
			}
		default:
			// Add as tag
			if str, ok := value.(string); ok {
				if metadata.Tags == nil {
					metadata.Tags = make(map[string]string)
				}
				metadata.Tags[key] = str
			}
		}
	}
}

func (c *ResourceCollectionManager) makeResourceID(provider, resourceType, name string) string {
	return provider + "." + resourceType + "." + name
}

// Storage abstraction methods (these would be implemented based on the backend)

func (c *ResourceCollectionManager) storeCollection(ctx context.Context, collection *ResourceCollection) error {
	// This would store collection metadata in the state backend
	// For now, this is a placeholder
	return nil
}

func (c *ResourceCollectionManager) loadCollection(ctx context.Context, collectionID string) (*ResourceCollection, error) {
	// This would load collection metadata from the state backend
	// For now, this is a placeholder that returns an error
	return nil, fmt.Errorf("collection not found: %s", collectionID)
}

func (c *ResourceCollectionManager) deleteCollection(ctx context.Context, collectionID string) error {
	// This would delete collection metadata from the state backend
	// For now, this is a placeholder
	return nil
}

func (c *ResourceCollectionManager) listAllCollections(ctx context.Context) ([]*ResourceCollection, error) {
	// This would list all collections from the state backend
	// For now, this is a placeholder
	return []*ResourceCollection{}, nil
}

func (c *ResourceCollectionManager) loadEnhancedResource(ctx context.Context, provider, resourceType, name string) (*EnhancedResourceState, error) {
	// This would load enhanced resource state
	// For now, this is a placeholder that returns an error
	return nil, fmt.Errorf("resource not found: %s.%s.%s", provider, resourceType, name)
}

func (c *ResourceCollectionManager) storeEnhancedResource(ctx context.Context, resource *EnhancedResourceState) error {
	// This would store enhanced resource state
	// For now, this is a placeholder
	return nil
}

func (c *ResourceCollectionManager) deleteEnhancedResource(ctx context.Context, provider, resourceType, name string) error {
	// This would delete enhanced resource state
	// For now, this is a placeholder
	return nil
}

// Collection management types

// CollectionDefinition defines a new resource collection
type CollectionDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Resources    []ResourceReference    `json:"resources"`
	Metadata     ResourceMetadata       `json:"metadata"`
	Dependencies []CollectionDependency `json:"dependencies"`
}

// CollectionUpdates defines updates to apply to a collection
type CollectionUpdates struct {
	Name             string                 `json:"name,omitempty"`
	Description      string                 `json:"description,omitempty"`
	DesiredState     string                 `json:"desired_state,omitempty"`
	ResourcesAdded   []ResourceReference    `json:"resources_added,omitempty"`
	ResourcesRemoved []ResourceReference    `json:"resources_removed,omitempty"`
	MetadataUpdates  map[string]interface{} `json:"metadata_updates,omitempty"`
}

// CollectionValidation represents the result of collection validation
type CollectionValidation struct {
	CollectionID     string    `json:"collection_id"`
	Valid            bool      `json:"valid"`
	ValidationTime   time.Time `json:"validation_time"`
	MissingResources []string  `json:"missing_resources"`
	InvalidResources []string  `json:"invalid_resources"`
	Issues           []string  `json:"issues"`
	Recommendations  []string  `json:"recommendations"`
}
