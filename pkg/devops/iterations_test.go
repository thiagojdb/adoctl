package devops

import (
	"testing"

	"adoctl/pkg/models"
)

func TestFilterHierarchyByUserMatchesAssignedToUnique(t *testing.T) {
	items := []models.WorkItemHierarchy{
		{
			ID:               1,
			Title:            "Parent",
			Type:             "Feature",
			AssignedTo:       "Jane Doe",
			AssignedToUnique: "jane.doe@example.com",
			Children: []models.WorkItemHierarchy{
				{
					ID:               2,
					Title:            "Child",
					Type:             "Task",
					AssignedTo:       "Jane Doe",
					AssignedToUnique: "jane.doe@example.com",
				},
			},
		},
	}

	filtered := filterHierarchyByUser(items, "jane.doe@example.com")

	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if !filtered[0].IsAssignedToUser {
		t.Fatal("expected parent item to be marked as assigned")
	}
	if len(filtered[0].Children) != 1 {
		t.Fatalf("len(filtered[0].Children) = %d, want 1", len(filtered[0].Children))
	}
	if !filtered[0].Children[0].IsAssignedToUser {
		t.Fatal("expected child item to be marked as assigned")
	}
}
