package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"adoctl/pkg/config"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// GetIterations returns all iterations (sprints) for the project
func (c *Client) GetIterations(ctx context.Context) ([]map[string]any, error) {
	project := c.GetProject()
	depth := 10
	structureGroup := workitemtracking.TreeStructureGroupValues.Iterations

	args := workitemtracking.GetClassificationNodeArgs{
		Project:        &project,
		StructureGroup: &structureGroup,
		Depth:          &depth,
	}

	node, err := c.WorkItemClient.GetClassificationNode(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get iterations: %w", err)
	}

	if node == nil {
		return []map[string]any{}, nil
	}

	// Flatten the tree structure into a list of iterations
	var result []map[string]any
	iterations := flattenClassificationNode(node, project)
	result = append(result, iterations...)

	return result, nil
}

// GetCurrentIterations returns iterations where current date is within start/end dates
func (c *Client) GetCurrentIterations(ctx context.Context) ([]map[string]any, error) {
	iterations, err := c.GetIterations(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var current []map[string]any

	for _, iter := range iterations {
		if startDate, ok := iter["startDate"].(time.Time); ok {
			if finishDate, ok := iter["finishDate"].(time.Time); ok {
				if isTimeWithinRangeInclusive(now, startDate, finishDate) {
					current = append(current, iter)
				}
			}
		}
	}

	return current, nil
}

// GetWorkItemsInIteration returns work items assigned to a specific iteration path
func (c *Client) GetWorkItemsInIteration(ctx context.Context, iterationPath string) ([]map[string]any, error) {
	project := c.GetProject()

	// Build WIQL query to find work items by iteration path
	wiql := fmt.Sprintf(
		"SELECT [System.Id], [System.Title], [System.WorkItemType], [System.State], [System.AssignedTo] "+
			"FROM workitems "+
			"WHERE [System.IterationPath] = '%s' "+
			"ORDER BY [System.WorkItemType], [System.Id]",
		iterationPath,
	)

	args := workitemtracking.QueryByWiqlArgs{
		Project: &project,
		Wiql: &workitemtracking.Wiql{
			Query: &wiql,
		},
	}

	result, err := c.WorkItemClient.QueryByWiql(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query work items: %w", err)
	}

	if result == nil || result.WorkItems == nil || len(*result.WorkItems) == 0 {
		return []map[string]any{}, nil
	}

	// Extract work item IDs
	var ids []int
	for _, wi := range *result.WorkItems {
		if wi.Id != nil {
			ids = append(ids, *wi.Id)
		}
	}

	// Fetch full work item details
	return c.GetWorkItems(ctx, ids)
}

// flattenClassificationNode recursively flattens a classification node tree
func flattenClassificationNode(node *workitemtracking.WorkItemClassificationNode, project string) []map[string]any {
	if node == nil {
		return nil
	}

	var result []map[string]any

	// Only add leaf nodes or nodes with dates (actual sprints)
	// Skip the root "Iterations" node itself
	if node.Name != nil && *node.Name != "" && *node.Name != "Iterations" {
		iteration := classificationNodeToMap(node, project)
		if iteration != nil {
			result = append(result, iteration)
		}
	}

	// Recursively process children
	if node.Children != nil {
		for _, child := range *node.Children {
			children := flattenClassificationNode(&child, project)
			result = append(result, children...)
		}
	}

	return result
}

// classificationNodeToMap converts a classification node to a map
func classificationNodeToMap(node *workitemtracking.WorkItemClassificationNode, project string) map[string]any {
	if node == nil {
		return nil
	}

	result := make(map[string]any)

	if node.Identifier != nil {
		result["id"] = node.Identifier.String()
	}
	if node.Name != nil {
		result["name"] = *node.Name
	}
	if node.Path != nil {
		result["path"] = *node.Path
	} else if node.Name != nil {
		result["path"] = project + "\\" + *node.Name
	}

	// Extract dates from attributes
	if node.Attributes != nil {
		attrs := *node.Attributes
		if startDate, ok := attrs["startDate"].(string); ok && startDate != "" {
			if t, err := time.Parse(time.RFC3339, startDate); err == nil {
				result["startDate"] = t
			}
		}
		if finishDate, ok := attrs["finishDate"].(string); ok && finishDate != "" {
			if t, err := time.Parse(time.RFC3339, finishDate); err == nil {
				result["finishDate"] = t
			}
		}
	}

	// Check if current
	now := time.Now()
	if startDate, ok := result["startDate"].(time.Time); ok {
		if finishDate, ok := result["finishDate"].(time.Time); ok {
			result["isCurrent"] = isTimeWithinRangeInclusive(now, startDate, finishDate)
		}
	}

	return result
}

func isTimeWithinRangeInclusive(now, startDate, finishDate time.Time) bool {
	return !now.Before(startDate) && !now.After(finishDate)
}

// GetWorkItemRelationsBatch fetches relations for multiple work items in parallel
func (c *Client) GetWorkItemRelationsBatch(ctx context.Context, ids []int) (map[int][]map[string]any, error) {
	result := make(map[int][]map[string]any)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Use configured threadpool size for parallel processing
	maxWorkers := config.GetParallelProcesses()
	semaphore := make(chan struct{}, maxWorkers)

	for _, id := range ids {
		wg.Add(1)
		go func(workItemID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			relations, err := c.GetWorkItemRelations(ctx, workItemID)
			if err != nil {
				// Continue on error - item may not exist or lack permissions
				return
			}

			mutex.Lock()
			result[workItemID] = relations
			mutex.Unlock()
		}(id)
	}

	wg.Wait()
	return result, nil
}

// GetWorkItemsWithParents returns work items along with their parent relationships
func (c *Client) GetWorkItemsWithParents(ctx context.Context, ids []int) ([]map[string]any, map[int]int, error) {
	// Get the work items
	workItems, err := c.GetWorkItems(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	// Build map of parent relationships
	parentMap := make(map[int]int) // childID -> parentID

	for _, wi := range workItems {
		id := 0
		if idFloat, ok := wi["id"].(float64); ok {
			id = int(idFloat)
		}
		if id == 0 {
			continue
		}

		// Get relations for this work item
		relations, err := c.GetWorkItemRelations(ctx, id)
		if err != nil {
			continue
		}

		for _, rel := range relations {
			if relType, ok := rel["rel"].(string); ok && relType == "System.LinkTypes.Hierarchy-Reverse" {
				// This is a parent link
				if url, ok := rel["url"].(string); ok {
					var parentID int
					if _, err := fmt.Sscanf(url, "%*[^/]/%d", &parentID); err == nil {
						parentMap[id] = parentID
					}
				}
			}
		}
	}

	return workItems, parentMap, nil
}

// ParseWorkItemIDFromURL extracts work item ID from an Azure DevOps URL
func ParseWorkItemIDFromURL(url string) int {
	// URL format: https://dev.azure.com/.../_apis/wit/workItems/12345
	// Extract the last number from the URL
	parts := strings.Split(url, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "" {
			continue
		}
		var id int
		if _, err := fmt.Sscanf(parts[i], "%d", &id); err == nil {
			return id
		}
	}
	return 0
}

// GetAllWorkItemsInIteration returns all work items with their hierarchy information
func (c *Client) GetAllWorkItemsInIteration(ctx context.Context, iterationPath string) ([]map[string]any, map[int]int, error) {
	return c.GetAllWorkItemsInIterationForUser(ctx, iterationPath, "")
}

// GetAllWorkItemsInIterationForUser returns work items filtered by assignee
// If assignedTo is empty, returns all work items
// If assignedTo is "@Me", uses the WIQL macro for current user
func (c *Client) GetAllWorkItemsInIterationForUser(ctx context.Context, iterationPath string, assignedTo string) ([]map[string]any, map[int]int, error) {
	project := c.GetProject()

	// Build WIQL query
	var userFilter string
	if assignedTo != "" {
		userFilter = fmt.Sprintf(" AND [System.AssignedTo] = %s", assignedTo)
	}

	wiql := fmt.Sprintf(
		"SELECT [System.Id], [System.Title], [System.WorkItemType], [System.State], [System.AssignedTo] "+
			"FROM workitems "+
			"WHERE [System.IterationPath] = '%s'%s "+
			"ORDER BY [System.Id]",
		iterationPath,
		userFilter,
	)

	args := workitemtracking.QueryByWiqlArgs{
		Project: &project,
		Wiql: &workitemtracking.Wiql{
			Query: &wiql,
		},
	}

	result, err := c.WorkItemClient.QueryByWiql(ctx, args)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query work items: %w", err)
	}

	if result == nil || result.WorkItems == nil || len(*result.WorkItems) == 0 {
		return []map[string]any{}, map[int]int{}, nil
	}

	// Extract work item IDs
	var ids []int
	idSet := make(map[int]bool)
	for _, wi := range *result.WorkItems {
		if wi.Id != nil {
			ids = append(ids, *wi.Id)
			idSet[*wi.Id] = true
		}
	}

	if len(ids) == 0 {
		return []map[string]any{}, map[int]int{}, nil
	}

	// Get full work item details with relations
	workItems, err := c.GetWorkItemsWithRelations(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	// Build parent map from relations
	parentMap := make(map[int]int)
	for _, wi := range workItems {
		id := 0
		if idFloat, ok := wi["id"].(float64); ok {
			id = int(idFloat)
		}
		if id == 0 {
			continue
		}

		if relations, ok := wi["relations"].([]any); ok {
			for _, rel := range relations {
				if relMap, ok := rel.(map[string]any); ok {
					if relType, ok := relMap["rel"].(string); ok {
						if relType == "System.LinkTypes.Hierarchy-Reverse" {
							// Parent link
							if url, ok := relMap["url"].(string); ok {
								parentID := ParseWorkItemIDFromURL(url)
								if parentID > 0 && idSet[parentID] {
									parentMap[id] = parentID
								}
							}
						}
					}
				}
			}
		}
	}

	return workItems, parentMap, nil
}

// GetWorkItemsWithRelations fetches work items with their relations expanded
func (c *Client) GetWorkItemsWithRelations(ctx context.Context, ids []int) ([]map[string]any, error) {
	if len(ids) == 0 {
		return []map[string]any{}, nil
	}

	project := c.GetProject()
	expand := workitemtracking.WorkItemExpandValues.Relations

	args := workitemtracking.GetWorkItemsArgs{
		Project: &project,
		Ids:     &ids,
		Expand:  &expand,
	}

	wis, err := c.WorkItemClient.GetWorkItems(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get work items: %w", err)
	}

	if wis == nil {
		return []map[string]any{}, nil
	}

	var result []map[string]any
	for _, wi := range *wis {
		data, err := json.Marshal(wi)
		if err != nil {
			continue
		}

		var workItem map[string]any
		if err := json.Unmarshal(data, &workItem); err != nil {
			continue
		}

		result = append(result, workItem)
	}

	return result, nil
}
