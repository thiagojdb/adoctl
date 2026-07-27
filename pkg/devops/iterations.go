package devops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"adoctl/pkg/models"
)

// ListIterations returns all iterations with optional "current" filtering
func (s *DevOpsService) ListIterations(ctx context.Context, onlyCurrent bool) ([]models.Iteration, error) {
	iterations, err := s.client.GetIterations(ctx)
	if err != nil {
		return nil, err
	}

	var result []models.Iteration
	now := time.Now()

	for _, iter := range iterations {
		iteration := mapToIteration(iter, now)

		if onlyCurrent && !iteration.IsCurrent {
			continue
		}

		result = append(result, iteration)
	}

	return result, nil
}

// mapToIteration converts a map to an Iteration model
func mapToIteration(data map[string]any, now time.Time) models.Iteration {
	iter := models.Iteration{}

	if id, ok := data["id"].(string); ok {
		iter.ID = id
	}
	if name, ok := data["name"].(string); ok {
		iter.Name = name
	}
	if path, ok := data["path"].(string); ok {
		iter.Path = path
	}
	if startDate, ok := data["startDate"].(time.Time); ok {
		iter.StartDate = &startDate
	}
	if finishDate, ok := data["finishDate"].(time.Time); ok {
		iter.FinishDate = &finishDate
	}
	if isCurrent, ok := data["isCurrent"].(bool); ok {
		iter.IsCurrent = isCurrent
	}

	return iter
}

// GetIterationWorkItems returns work items for a specific iteration
// Returns hierarchical structure with parent-child relationships
func (s *DevOpsService) GetIterationWorkItems(ctx context.Context, iterationPath string) ([]models.WorkItemHierarchy, error) {
	// Normalize iteration path
	iterationPath = normalizeIterationPath(iterationPath, s.client.GetProject())

	// Get all work items with their parent relationships
	workItems, parentMap, err := s.client.GetAllWorkItemsInIteration(ctx, iterationPath)
	if err != nil {
		return nil, err
	}

	// Build hierarchy
	hierarchy := BuildWorkItemHierarchy(workItems, parentMap)

	return hierarchy, nil
}

// GetIterationWorkItemsForUser returns work items filtered by assignee
// Uses server-side filtering via WIQL for @CurrentUser, falls back to client-side for other users
func (s *DevOpsService) GetIterationWorkItemsForUser(ctx context.Context, iterationPath string, userIdentifier string) ([]models.WorkItemHierarchy, error) {
	// Normalize iteration path
	iterationPath = normalizeIterationPath(iterationPath, s.client.GetProject())

	var workItems []map[string]any
	var parentMap map[int]int
	var err error

	if userIdentifier == "@Me" {
		// Use server-side filtering with WIQL macro
		workItems, parentMap, err = s.client.GetAllWorkItemsInIterationForUser(ctx, iterationPath, "@Me")
		if err != nil {
			return nil, err
		}
		// Build hierarchy (all items are already filtered to current user)
		hierarchy := BuildWorkItemHierarchy(workItems, parentMap)
		return markHierarchyAssigned(hierarchy, true), nil
	}

	// For other users, fetch all and filter client-side
	workItems, parentMap, err = s.client.GetAllWorkItemsInIteration(ctx, iterationPath)
	if err != nil {
		return nil, err
	}

	hierarchy := BuildWorkItemHierarchy(workItems, parentMap)
	return filterHierarchyByUser(hierarchy, userIdentifier), nil
}

// ResolveUserIdentifier resolves "me" to @Me macro, or validates email/name
func (s *DevOpsService) ResolveUserIdentifier(identifier string) (string, error) {
	if strings.ToLower(identifier) == "me" {
		// Use Azure DevOps @Me macro for WIQL queries
		return "@Me", nil
	}

	// Validate as email if it contains @
	if strings.Contains(identifier, "@") {
		return identifier, nil
	}

	// Try to match against known users
	creators := s.GetAllCreators()
	for _, creator := range creators {
		name := creator["name"].(string)
		if strings.Contains(strings.ToLower(name), strings.ToLower(identifier)) {
			return name, nil
		}
	}

	return "", fmt.Errorf("user not found: %s", identifier)
}

// normalizeIterationPath ensures the iteration path is fully qualified for WIQL queries
func normalizeIterationPath(path, project string) string {
	// Normalize separators to backslash first
	path = strings.ReplaceAll(path, "/", `\`)

	// Remove leading backslash if present (API returns paths like \GISS\Iteration...)
	path = strings.TrimPrefix(path, `\`)

	// The API returns paths like "GISS\Iteration\Inovação\Produto\Sprint Name"
	// But WIQL queries need "GISS\Inovação\Produto\Sprint Name" (without "Iteration")
	// Remove the "Iteration" folder from the path (case-insensitive)
	iterationPrefix := project + `\Iteration\`
	if len(path) >= len(iterationPrefix) && strings.EqualFold(path[:len(iterationPrefix)], iterationPrefix) {
		// Extract the part after "Iteration\"
		path = project + `\` + path[len(iterationPrefix):]
	}

	// Clean up any leading backslash
	path = strings.TrimPrefix(path, `\`)

	// If path doesn't start with project name, prepend it
	if !strings.HasPrefix(strings.ToLower(path), strings.ToLower(project+`\`)) {
		path = project + `\` + path
	}

	return path
}

// BuildWorkItemHierarchy takes flat list of work items and builds parent-child tree
func BuildWorkItemHierarchy(workItems []map[string]any, parentMap map[int]int) []models.WorkItemHierarchy {
	if len(workItems) == 0 {
		return []models.WorkItemHierarchy{}
	}

	// Create a map of all work items by ID
	itemMap := make(map[int]*models.WorkItemHierarchy)
	for _, wi := range workItems {
		hierarchyItem := models.WorkItemHierarchyFromAzure(wi)
		if hierarchyItem.ID > 0 {
			// Must create a copy since we're taking the address
			itemCopy := hierarchyItem
			itemMap[hierarchyItem.ID] = &itemCopy
		}
	}

	// Build parent references
	for id, parentID := range parentMap {
		if item, ok := itemMap[id]; ok {
			item.ParentID = &parentID
		}
	}

	// Build the tree
	var roots []models.WorkItemHierarchy
	childrenMap := make(map[int][]*models.WorkItemHierarchy)

	for id, item := range itemMap {
		if parentID, hasParent := parentMap[id]; hasParent && itemMap[parentID] != nil {
			// This is a child item
			childrenMap[parentID] = append(childrenMap[parentID], item)
		} else {
			// This is a root item
			roots = append(roots, *item)
		}
	}

	// Recursively attach children
	for i := range roots {
		attachChildren(&roots[i], childrenMap)
	}

	// Sort roots: Features first, then PBIs/Bugs, then Tasks
	roots = sortWorkItemsByType(roots)

	return roots
}

// attachChildren recursively attaches children to their parent
func attachChildren(parent *models.WorkItemHierarchy, childrenMap map[int][]*models.WorkItemHierarchy) {
	if children, ok := childrenMap[parent.ID]; ok {
		for _, child := range children {
			childCopy := *child
			attachChildren(&childCopy, childrenMap)
			parent.Children = append(parent.Children, childCopy)
		}
		// Sort children by ID for consistent ordering
		parent.Children = sortWorkItemsByType(parent.Children)
	}
}

// sortWorkItemsByType sorts work items by type (Features first, then PBIs/Bugs, then Tasks)
func sortWorkItemsByType(items []models.WorkItemHierarchy) []models.WorkItemHierarchy {
	typeOrder := map[string]int{
		"Feature":              1,
		"Epic":                 2,
		"Product Backlog Item": 3,
		"PBI":                  3,
		"User Story":           3,
		"Bug":                  4,
		"Task":                 5,
		"Test Case":            6,
		"Issue":                7,
	}

	// Simple bubble sort by type order
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			orderI := typeOrder[items[i].Type]
			orderJ := typeOrder[items[j].Type]
			if orderI == 0 {
				orderI = 99
			}
			if orderJ == 0 {
				orderJ = 99
			}

			if orderI > orderJ || (orderI == orderJ && items[i].ID > items[j].ID) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return items
}

// markHierarchyAssigned marks all items in hierarchy as assigned/unassigned
func markHierarchyAssigned(items []models.WorkItemHierarchy, assigned bool) []models.WorkItemHierarchy {
	for i := range items {
		items[i].IsAssignedToUser = assigned
		items[i].Children = markHierarchyAssigned(items[i].Children, assigned)
	}
	return items
}

// filterHierarchyByUser filters the hierarchy to show only items assigned to user
// Parent items are included (marked) even if not assigned to maintain context
func filterHierarchyByUser(items []models.WorkItemHierarchy, user string) []models.WorkItemHierarchy {
	var result []models.WorkItemHierarchy
	userLower := strings.ToLower(user)

	for _, item := range items {
		assignedToLower := strings.ToLower(item.AssignedTo)
		assignedToUniqueLower := strings.ToLower(item.AssignedToUnique)
		isAssigned := strings.Contains(assignedToLower, userLower) || strings.Contains(assignedToUniqueLower, userLower)

		// Filter children
		filteredChildren := filterHierarchyByUser(item.Children, user)

		// Include this item if:
		// 1. It's assigned to the user, OR
		// 2. It has children that are assigned (to maintain hierarchy)
		if isAssigned || len(filteredChildren) > 0 {
			itemCopy := item
			itemCopy.IsAssignedToUser = isAssigned
			itemCopy.Children = filteredChildren
			result = append(result, itemCopy)
		}
	}

	return result
}

// FindIterationByName finds an iteration by partial name match.
// Returns the first match, and a warning message if there were multiple matches.
func (s *DevOpsService) FindIterationByName(ctx context.Context, name string) (*models.Iteration, string, error) {
	iterations, err := s.ListIterations(ctx, false)
	if err != nil {
		return nil, "", err
	}

	nameLower := strings.ToLower(name)
	var matches []models.Iteration

	for _, iter := range iterations {
		if strings.Contains(strings.ToLower(iter.Name), nameLower) {
			matches = append(matches, iter)
		} else if strings.Contains(strings.ToLower(iter.Path), nameLower) {
			matches = append(matches, iter)
		}
	}

	if len(matches) == 0 {
		// Build list of available iteration names for the error message
		var available []string
		for _, iter := range iterations {
			if len(available) < 10 { // Limit to first 10
				available = append(available, iter.Name)
			}
		}
		hint := ""
		if len(available) > 0 {
			hint = fmt.Sprintf(". Available sprints: %s", strings.Join(available, ", "))
			if len(iterations) > 10 {
				hint += fmt.Sprintf(" (and %d more)", len(iterations)-10)
			}
		}
		return nil, "", fmt.Errorf("no iteration found matching '%s'%s", name, hint)
	}

	warning := ""
	if len(matches) > 1 {
		warning = fmt.Sprintf("Multiple sprints matched '%s', using: %s", name, matches[0].Name)
	}

	return &matches[0], warning, nil
}
