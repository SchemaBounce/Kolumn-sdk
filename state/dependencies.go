// Package state - dependencies.go provides dependency graph management capabilities for the SDK
package state

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

// DependencyManager provides graph analysis capabilities for resource dependencies
type DependencyManager struct {
	manager *DefaultManager
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager(manager *DefaultManager) *DependencyManager {
	return &DependencyManager{
		manager: manager,
	}
}

// AnalyzeGraph performs comprehensive analysis of the resource dependency graph
func (d *DependencyManager) AnalyzeGraph(ctx context.Context, stateName string) (*GraphAnalysis, error) {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	graph := d.buildResourceGraph(state)

	analysis := &GraphAnalysis{
		NodeCount:             len(graph.Nodes),
		EdgeCount:             len(graph.Edges),
		AnalysisTime:          time.Now(),
		CyclicDependencies:    []DependencyCycle{},
		OrphanedResources:     []string{},
		CriticalPathResources: []string{},
		ComponentsByProvider:  make(map[string]int),
		ComponentsByCategory:  make(map[string]int),
		DependencyLevels:      []DependencyLevel{},
	}

	// Build adjacency lists for analysis
	adjList := d.buildAdjacencyList(graph)
	reverseAdjList := d.buildReverseAdjacencyList(graph)

	// Analyze graph structure
	analysis.CyclicDependencies = d.findCycles(graph, adjList)
	analysis.OrphanedResources = d.findOrphanedResources(graph, adjList, reverseAdjList)
	analysis.CriticalPathResources = d.findCriticalPath(graph, adjList)
	analysis.ComponentsByProvider = d.analyzeByProvider(graph)
	analysis.ComponentsByCategory = d.analyzeByCategory(graph)
	analysis.DependencyLevels = d.calculateDependencyLevels(graph, adjList)

	// Calculate metrics
	analysis.MaxDependencyDepth = d.calculateMaxDepth(adjList)
	analysis.AverageDependencyDepth = d.calculateAverageDepth(graph, adjList)
	analysis.MostDependentResource = d.findMostDependentResource(graph, adjList)
	analysis.MostReferencedResource = d.findMostReferencedResource(graph, reverseAdjList)

	return analysis, nil
}

// FindExecutionOrder determines the order in which resources should be created/updated
func (d *DependencyManager) FindExecutionOrder(ctx context.Context, stateName string, resourceIDs []string) ([]ExecutionBatch, error) {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	graph := d.buildResourceGraph(state)

	// Filter graph to only include specified resources
	filteredGraph := d.filterGraph(graph, resourceIDs)

	// Check for cycles in the filtered graph
	adjList := d.buildAdjacencyList(filteredGraph)
	cycles := d.findCycles(filteredGraph, adjList)
	if len(cycles) > 0 {
		return nil, fmt.Errorf("cyclic dependencies detected, cannot determine execution order: %v", cycles)
	}

	// Perform topological sort to determine execution order
	return d.topologicalSort(filteredGraph, adjList), nil
}

// ValidateDependencies validates that all dependencies are satisfied and reachable
func (d *DependencyManager) ValidateDependencies(ctx context.Context, stateName string, resourceID string) (*DependencyValidation, error) {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	resourceMap := make(map[string]*types.UniversalResource)
	for i, resource := range state.Resources {
		id := d.makeResourceID(resource.Provider, resource.Type, resource.Name)
		resourceMap[id] = &state.Resources[i]
	}

	targetResource, exists := resourceMap[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	validation := &DependencyValidation{
		ResourceID:           resourceID,
		Valid:                true,
		ValidationTime:       time.Now(),
		MissingDependencies:  []string{},
		InvalidDependencies:  []string{},
		CircularDependencies: []string{},
		RecommendedActions:   []string{},
	}

	// Validate explicit dependencies
	for _, depID := range targetResource.DependsOn {
		if _, exists := resourceMap[depID]; !exists {
			validation.Valid = false
			validation.MissingDependencies = append(validation.MissingDependencies, depID)
			validation.RecommendedActions = append(validation.RecommendedActions,
				fmt.Sprintf("Create missing dependency: %s", depID))
		} else {
			// Check for circular dependencies
			if d.hasCircularDependency(resourceMap, resourceID, depID, make(map[string]bool)) {
				validation.Valid = false
				validation.CircularDependencies = append(validation.CircularDependencies, depID)
				validation.RecommendedActions = append(validation.RecommendedActions,
					fmt.Sprintf("Resolve circular dependency with: %s", depID))
			}
		}
	}

	// Validate reference dependencies
	for _, ref := range targetResource.References {
		if _, exists := resourceMap[ref.TargetResource]; !exists {
			validation.Valid = false
			validation.MissingDependencies = append(validation.MissingDependencies, ref.TargetResource)
			validation.RecommendedActions = append(validation.RecommendedActions,
				fmt.Sprintf("Create missing referenced resource: %s", ref.TargetResource))
		}
	}

	return validation, nil
}

// GetImpactAnalysis analyzes what would be impacted by changes to a resource
func (d *DependencyManager) GetImpactAnalysis(ctx context.Context, stateName string, resourceID string, changeType string) (*ImpactAnalysis, error) {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	graph := d.buildResourceGraph(state)

	// Build reverse adjacency list to find what depends on this resource
	reverseAdjList := d.buildReverseAdjacencyList(graph)

	analysis := &ImpactAnalysis{
		ResourceID:      resourceID,
		ChangeType:      changeType,
		AnalysisTime:    time.Now(),
		DirectImpacts:   []ImpactedResource{},
		IndirectImpacts: []ImpactedResource{},
		CriticalImpacts: []ImpactedResource{},
	}

	// Find resources that are impacted by changes to this resource
	visited := make(map[string]bool)
	d.findImpactedResources(graph, reverseAdjList, resourceID, 0, visited, analysis)

	// Classify impacts based on change type
	d.classifyImpacts(analysis, changeType)

	return analysis, nil
}

// GetDependencyChain gets the full dependency chain for a resource
func (d *DependencyManager) GetDependencyChain(ctx context.Context, stateName string, resourceID string) (*DependencyChain, error) {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	graph := d.buildResourceGraph(state)
	adjList := d.buildAdjacencyList(graph)

	chain := &DependencyChain{
		ResourceID:  resourceID,
		Levels:      []ChainLevel{},
		TotalDepth:  0,
		GeneratedAt: time.Now(),
	}

	visited := make(map[string]bool)
	d.buildDependencyChain(graph, adjList, resourceID, 0, visited, chain)

	return chain, nil
}

// AddDependency adds a dependency between two resources
func (d *DependencyManager) AddDependency(ctx context.Context, stateName string, fromResourceID, toResourceID string, dependencyType string) error {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Validate that both resources exist
	fromExists := false
	toExists := false
	for _, resource := range state.Resources {
		resourceID := d.makeResourceID(resource.Provider, resource.Type, resource.Name)
		if resourceID == fromResourceID {
			fromExists = true
		}
		if resourceID == toResourceID {
			toExists = true
		}
	}

	if !fromExists {
		return fmt.Errorf("source resource %s not found", fromResourceID)
	}
	if !toExists {
		return fmt.Errorf("target resource %s not found", toResourceID)
	}

	// Check for circular dependency
	resourceMap := make(map[string]*types.UniversalResource)
	for i, resource := range state.Resources {
		id := d.makeResourceID(resource.Provider, resource.Type, resource.Name)
		resourceMap[id] = &state.Resources[i]
	}

	if d.hasCircularDependency(resourceMap, fromResourceID, toResourceID, make(map[string]bool)) {
		return fmt.Errorf("adding dependency would create circular dependency")
	}

	// Add dependency to state
	dependency := types.Dependency{
		ID:             fmt.Sprintf("%s->%s", fromResourceID, toResourceID),
		ResourceID:     fromResourceID,
		DependsOnID:    toResourceID,
		DependencyType: types.DependencyType(dependencyType),
		Optional:       false,
	}

	state.Dependencies = append(state.Dependencies, dependency)

	// Save updated state
	return d.manager.PutState(ctx, stateName, state)
}

// RemoveDependency removes a dependency between two resources
func (d *DependencyManager) RemoveDependency(ctx context.Context, stateName string, fromResourceID, toResourceID string) error {
	state, err := d.manager.GetState(ctx, stateName)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Find and remove the dependency
	dependencyID := fmt.Sprintf("%s->%s", fromResourceID, toResourceID)
	newDependencies := make([]types.Dependency, 0)
	found := false

	for _, dep := range state.Dependencies {
		if dep.ID != dependencyID {
			newDependencies = append(newDependencies, dep)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("dependency not found: %s", dependencyID)
	}

	state.Dependencies = newDependencies

	// Save updated state
	return d.manager.PutState(ctx, stateName, state)
}

// Helper methods

func (d *DependencyManager) buildResourceGraph(state *types.UniversalState) *ResourceGraph {
	graph := &ResourceGraph{
		Nodes: make([]ResourceNode, 0),
		Edges: make([]ResourceEdge, 0),
	}

	// Create nodes for all resources
	for _, resource := range state.Resources {
		node := ResourceNode{
			ID:       d.makeResourceID(resource.Provider, resource.Type, resource.Name),
			Type:     resource.Type,
			Provider: resource.Provider,
			Category: d.inferCategory(resource.Provider),
			Name:     resource.Name,
			State:    string(resource.Instances[0].Status),
			Metadata: make(map[string]interface{}),
		}

		// Add metadata from resource
		for k, v := range resource.Metadata {
			node.Metadata[k] = v
		}

		graph.Nodes = append(graph.Nodes, node)
	}

	// Create edges for dependencies
	for _, dep := range state.Dependencies {
		edge := ResourceEdge{
			From:         dep.ResourceID,
			To:           dep.DependsOnID,
			Relationship: string(dep.DependencyType),
			Optional:     dep.Optional,
			Weight:       1,
		}
		graph.Edges = append(graph.Edges, edge)
	}

	// Create edges for explicit depends_on relationships
	for _, resource := range state.Resources {
		resourceID := d.makeResourceID(resource.Provider, resource.Type, resource.Name)
		for _, depID := range resource.DependsOn {
			edge := ResourceEdge{
				From:         resourceID,
				To:           depID,
				Relationship: "depends_on",
				Optional:     false,
				Weight:       1,
			}
			graph.Edges = append(graph.Edges, edge)
		}
	}

	// Create edges for reference relationships
	for _, resource := range state.Resources {
		resourceID := d.makeResourceID(resource.Provider, resource.Type, resource.Name)
		for _, ref := range resource.References {
			edge := ResourceEdge{
				From:         resourceID,
				To:           ref.TargetResource,
				Relationship: ref.ReferenceType,
				Optional:     true,
				Weight:       1,
			}
			graph.Edges = append(graph.Edges, edge)
		}
	}

	return graph
}

func (d *DependencyManager) buildAdjacencyList(graph *ResourceGraph) map[string][]string {
	adjList := make(map[string][]string)

	// Initialize all nodes
	for _, node := range graph.Nodes {
		adjList[node.ID] = []string{}
	}

	// Add edges
	for _, edge := range graph.Edges {
		adjList[edge.From] = append(adjList[edge.From], edge.To)
	}

	return adjList
}

func (d *DependencyManager) buildReverseAdjacencyList(graph *ResourceGraph) map[string][]string {
	reverseAdjList := make(map[string][]string)

	// Initialize all nodes
	for _, node := range graph.Nodes {
		reverseAdjList[node.ID] = []string{}
	}

	// Add reverse edges
	for _, edge := range graph.Edges {
		reverseAdjList[edge.To] = append(reverseAdjList[edge.To], edge.From)
	}

	return reverseAdjList
}

func (d *DependencyManager) findCycles(graph *ResourceGraph, adjList map[string][]string) []DependencyCycle {
	var cycles []DependencyCycle
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, node := range graph.Nodes {
		if !visited[node.ID] {
			path := []string{}
			cyclePath := d.dfsForCycles(node.ID, adjList, visited, recStack, path)
			if len(cyclePath) > 0 {
				cycles = append(cycles, DependencyCycle{
					Resources: cyclePath,
					Length:    len(cyclePath),
				})
			}
		}
	}

	return cycles
}

func (d *DependencyManager) dfsForCycles(nodeID string, adjList map[string][]string, visited, recStack map[string]bool, path []string) []string {
	visited[nodeID] = true
	recStack[nodeID] = true
	path = append(path, nodeID)

	for _, neighbor := range adjList[nodeID] {
		if !visited[neighbor] {
			if cyclePath := d.dfsForCycles(neighbor, adjList, visited, recStack, path); len(cyclePath) > 0 {
				return cyclePath
			}
		} else if recStack[neighbor] {
			// Found a cycle
			cycleStart := -1
			for i, resource := range path {
				if resource == neighbor {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return path[cycleStart:]
			}
		}
	}

	recStack[nodeID] = false
	return []string{}
}

func (d *DependencyManager) findOrphanedResources(graph *ResourceGraph, adjList, reverseAdjList map[string][]string) []string {
	var orphaned []string

	for _, node := range graph.Nodes {
		// A resource is orphaned if it has no dependencies and no dependents
		hasDependencies := len(adjList[node.ID]) > 0
		hasDependents := len(reverseAdjList[node.ID]) > 0

		if !hasDependencies && !hasDependents {
			orphaned = append(orphaned, node.ID)
		}
	}

	return orphaned
}

func (d *DependencyManager) findCriticalPath(graph *ResourceGraph, adjList map[string][]string) []string {
	// Find the longest path in the DAG (critical path)
	maxDepth := 0
	var criticalPath []string

	for _, node := range graph.Nodes {
		visited := make(map[string]bool)
		depth, path := d.findLongestPath(node.ID, adjList, visited)
		if depth > maxDepth {
			maxDepth = depth
			criticalPath = path
		}
	}

	return criticalPath
}

func (d *DependencyManager) findLongestPath(nodeID string, adjList map[string][]string, visited map[string]bool) (int, []string) {
	visited[nodeID] = true

	maxDepth := 0
	var longestPath []string

	for _, neighbor := range adjList[nodeID] {
		if !visited[neighbor] {
			depth, path := d.findLongestPath(neighbor, adjList, visited)
			if depth > maxDepth {
				maxDepth = depth
				longestPath = path
			}
		}
	}

	visited[nodeID] = false
	return maxDepth + 1, append([]string{nodeID}, longestPath...)
}

func (d *DependencyManager) analyzeByProvider(graph *ResourceGraph) map[string]int {
	counts := make(map[string]int)
	for _, node := range graph.Nodes {
		counts[node.Provider]++
	}
	return counts
}

func (d *DependencyManager) analyzeByCategory(graph *ResourceGraph) map[string]int {
	counts := make(map[string]int)
	for _, node := range graph.Nodes {
		counts[node.Category]++
	}
	return counts
}

func (d *DependencyManager) calculateDependencyLevels(graph *ResourceGraph, adjList map[string][]string) []DependencyLevel {
	// Calculate levels using topological sort
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for _, node := range graph.Nodes {
		inDegree[node.ID] = 0
	}

	// Calculate in-degrees
	for _, neighbors := range adjList {
		for _, neighbor := range neighbors {
			inDegree[neighbor]++
		}
	}

	var levels []DependencyLevel
	level := 0

	for {
		var currentLevel []string

		// Find nodes with in-degree 0
		for nodeID, degree := range inDegree {
			if degree == 0 {
				currentLevel = append(currentLevel, nodeID)
			}
		}

		if len(currentLevel) == 0 {
			break
		}

		levels = append(levels, DependencyLevel{
			Level:     level,
			Resources: currentLevel,
		})

		// Remove current level nodes and update in-degrees
		for _, nodeID := range currentLevel {
			delete(inDegree, nodeID)
			for _, neighbor := range adjList[nodeID] {
				if _, exists := inDegree[neighbor]; exists {
					inDegree[neighbor]--
				}
			}
		}

		level++
	}

	return levels
}

func (d *DependencyManager) calculateMaxDepth(adjList map[string][]string) int {
	maxDepth := 0

	for nodeID := range adjList {
		visited := make(map[string]bool)
		depth := d.calculateDepthFromNode(nodeID, adjList, visited)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

func (d *DependencyManager) calculateDepthFromNode(nodeID string, adjList map[string][]string, visited map[string]bool) int {
	visited[nodeID] = true
	maxDepth := 0

	for _, neighbor := range adjList[nodeID] {
		if !visited[neighbor] {
			depth := d.calculateDepthFromNode(neighbor, adjList, visited)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	visited[nodeID] = false
	return maxDepth + 1
}

func (d *DependencyManager) calculateAverageDepth(graph *ResourceGraph, adjList map[string][]string) float64 {
	totalDepth := 0

	for _, node := range graph.Nodes {
		visited := make(map[string]bool)
		depth := d.calculateDepthFromNode(node.ID, adjList, visited)
		totalDepth += depth
	}

	if len(graph.Nodes) == 0 {
		return 0
	}

	return float64(totalDepth) / float64(len(graph.Nodes))
}

func (d *DependencyManager) findMostDependentResource(graph *ResourceGraph, adjList map[string][]string) string {
	maxDependencies := 0
	var mostDependent string

	for _, node := range graph.Nodes {
		dependencyCount := len(adjList[node.ID])
		if dependencyCount > maxDependencies {
			maxDependencies = dependencyCount
			mostDependent = node.ID
		}
	}

	return mostDependent
}

func (d *DependencyManager) findMostReferencedResource(graph *ResourceGraph, reverseAdjList map[string][]string) string {
	maxReferences := 0
	var mostReferenced string

	for _, node := range graph.Nodes {
		referenceCount := len(reverseAdjList[node.ID])
		if referenceCount > maxReferences {
			maxReferences = referenceCount
			mostReferenced = node.ID
		}
	}

	return mostReferenced
}

func (d *DependencyManager) filterGraph(graph *ResourceGraph, resourceIDs []string) *ResourceGraph {
	resourceSet := make(map[string]bool)
	for _, id := range resourceIDs {
		resourceSet[id] = true
	}

	filtered := &ResourceGraph{
		Nodes: []ResourceNode{},
		Edges: []ResourceEdge{},
	}

	// Filter nodes
	for _, node := range graph.Nodes {
		if resourceSet[node.ID] {
			filtered.Nodes = append(filtered.Nodes, node)
		}
	}

	// Filter edges
	for _, edge := range graph.Edges {
		if resourceSet[edge.From] && resourceSet[edge.To] {
			filtered.Edges = append(filtered.Edges, edge)
		}
	}

	return filtered
}

func (d *DependencyManager) topologicalSort(graph *ResourceGraph, adjList map[string][]string) []ExecutionBatch {
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for _, node := range graph.Nodes {
		inDegree[node.ID] = 0
	}

	// Calculate in-degrees
	for _, neighbors := range adjList {
		for _, neighbor := range neighbors {
			inDegree[neighbor]++
		}
	}

	var batches []ExecutionBatch
	batchNum := 0

	for {
		var currentBatch []string

		// Find nodes with in-degree 0
		for nodeID, degree := range inDegree {
			if degree == 0 {
				currentBatch = append(currentBatch, nodeID)
			}
		}

		if len(currentBatch) == 0 {
			break
		}

		// Sort for consistent ordering
		sort.Strings(currentBatch)

		batches = append(batches, ExecutionBatch{
			BatchNumber:    batchNum,
			Resources:      currentBatch,
			CanParallelize: true, // Resources in same batch can be executed in parallel
		})

		// Remove current batch nodes and update in-degrees
		for _, nodeID := range currentBatch {
			delete(inDegree, nodeID)
			for _, neighbor := range adjList[nodeID] {
				if _, exists := inDegree[neighbor]; exists {
					inDegree[neighbor]--
				}
			}
		}

		batchNum++
	}

	return batches
}

func (d *DependencyManager) hasCircularDependency(resourceMap map[string]*types.UniversalResource, sourceID, targetID string, visited map[string]bool) bool {
	// Check if targetID eventually depends on sourceID (which would create a cycle)
	return d.dependsOn(resourceMap, targetID, sourceID, make(map[string]bool))
}

func (d *DependencyManager) dependsOn(resourceMap map[string]*types.UniversalResource, resourceID, targetID string, visited map[string]bool) bool {
	if resourceID == targetID {
		return true
	}

	if visited[resourceID] {
		return false // Already checked this path
	}

	visited[resourceID] = true

	if resource, exists := resourceMap[resourceID]; exists {
		// Check explicit depends_on
		for _, depID := range resource.DependsOn {
			if d.dependsOn(resourceMap, depID, targetID, visited) {
				return true
			}
		}

		// Check references
		for _, ref := range resource.References {
			if d.dependsOn(resourceMap, ref.TargetResource, targetID, visited) {
				return true
			}
		}
	}

	return false
}

func (d *DependencyManager) findImpactedResources(graph *ResourceGraph, reverseAdjList map[string][]string, resourceID string, depth int, visited map[string]bool, analysis *ImpactAnalysis) {
	if visited[resourceID] {
		return
	}

	visited[resourceID] = true

	for _, impactedID := range reverseAdjList[resourceID] {
		impacted := ImpactedResource{
			ResourceID: impactedID,
			ImpactType: d.determineImpactType(depth),
			Depth:      depth + 1,
		}

		if depth == 0 {
			analysis.DirectImpacts = append(analysis.DirectImpacts, impacted)
		} else {
			analysis.IndirectImpacts = append(analysis.IndirectImpacts, impacted)
		}

		// Recursively find further impacts
		d.findImpactedResources(graph, reverseAdjList, impactedID, depth+1, visited, analysis)
	}
}

func (d *DependencyManager) determineImpactType(depth int) string {
	if depth == 0 {
		return "direct"
	} else if depth <= 2 {
		return "indirect"
	} else {
		return "cascading"
	}
}

func (d *DependencyManager) classifyImpacts(analysis *ImpactAnalysis, changeType string) {
	criticalChangeTypes := []string{"delete", "destroy", "major_update"}

	for _, criticalType := range criticalChangeTypes {
		if changeType == criticalType {
			// Mark all direct impacts as critical for destructive changes
			for _, impact := range analysis.DirectImpacts {
				analysis.CriticalImpacts = append(analysis.CriticalImpacts, impact)
			}
			break
		}
	}
}

func (d *DependencyManager) buildDependencyChain(graph *ResourceGraph, adjList map[string][]string, resourceID string, level int, visited map[string]bool, chain *DependencyChain) {
	if visited[resourceID] {
		return
	}

	visited[resourceID] = true

	// Ensure we have enough levels
	for len(chain.Levels) <= level {
		chain.Levels = append(chain.Levels, ChainLevel{
			Level:     len(chain.Levels),
			Resources: []string{},
		})
	}

	chain.Levels[level].Resources = append(chain.Levels[level].Resources, resourceID)

	if level > chain.TotalDepth {
		chain.TotalDepth = level
	}

	// Recursively add dependencies
	for _, depID := range adjList[resourceID] {
		d.buildDependencyChain(graph, adjList, depID, level+1, visited, chain)
	}

	visited[resourceID] = false
}

func (d *DependencyManager) makeResourceID(provider, resourceType, name string) string {
	return provider + "." + resourceType + "." + name
}

func (d *DependencyManager) inferCategory(provider string) string {
	// Simple categorization based on provider name
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

// Graph analysis result types

// GraphAnalysis represents comprehensive analysis of the resource dependency graph
type GraphAnalysis struct {
	NodeCount              int               `json:"node_count"`
	EdgeCount              int               `json:"edge_count"`
	MaxDependencyDepth     int               `json:"max_dependency_depth"`
	AverageDependencyDepth float64           `json:"average_dependency_depth"`
	CyclicDependencies     []DependencyCycle `json:"cyclic_dependencies"`
	OrphanedResources      []string          `json:"orphaned_resources"`
	CriticalPathResources  []string          `json:"critical_path_resources"`
	MostDependentResource  string            `json:"most_dependent_resource"`
	MostReferencedResource string            `json:"most_referenced_resource"`
	ComponentsByProvider   map[string]int    `json:"components_by_provider"`
	ComponentsByCategory   map[string]int    `json:"components_by_category"`
	DependencyLevels       []DependencyLevel `json:"dependency_levels"`
	AnalysisTime           time.Time         `json:"analysis_time"`
}

// DependencyCycle represents a circular dependency
type DependencyCycle struct {
	Resources []string `json:"resources"`
	Length    int      `json:"length"`
}

// DependencyLevel represents resources at a specific dependency level
type DependencyLevel struct {
	Level     int      `json:"level"`
	Resources []string `json:"resources"`
}

// ExecutionBatch represents a batch of resources that can be executed in parallel
type ExecutionBatch struct {
	BatchNumber    int      `json:"batch_number"`
	Resources      []string `json:"resources"`
	CanParallelize bool     `json:"can_parallelize"`
}

// DependencyValidation represents the result of dependency validation
type DependencyValidation struct {
	ResourceID           string    `json:"resource_id"`
	Valid                bool      `json:"valid"`
	ValidationTime       time.Time `json:"validation_time"`
	MissingDependencies  []string  `json:"missing_dependencies"`
	InvalidDependencies  []string  `json:"invalid_dependencies"`
	CircularDependencies []string  `json:"circular_dependencies"`
	RecommendedActions   []string  `json:"recommended_actions"`
}

// ImpactAnalysis represents analysis of what would be impacted by resource changes
type ImpactAnalysis struct {
	ResourceID      string             `json:"resource_id"`
	ChangeType      string             `json:"change_type"`
	AnalysisTime    time.Time          `json:"analysis_time"`
	DirectImpacts   []ImpactedResource `json:"direct_impacts"`
	IndirectImpacts []ImpactedResource `json:"indirect_impacts"`
	CriticalImpacts []ImpactedResource `json:"critical_impacts"`
}

// ImpactedResource represents a resource that would be impacted by changes
type ImpactedResource struct {
	ResourceID string `json:"resource_id"`
	ImpactType string `json:"impact_type"` // direct, indirect, cascading
	Depth      int    `json:"depth"`
}

// DependencyChain represents the full dependency chain for a resource
type DependencyChain struct {
	ResourceID  string       `json:"resource_id"`
	Levels      []ChainLevel `json:"levels"`
	TotalDepth  int          `json:"total_depth"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// ChainLevel represents resources at a specific level in the dependency chain
type ChainLevel struct {
	Level     int      `json:"level"`
	Resources []string `json:"resources"`
}
