package models

import (
	"testing"
	"time"
)

func TestWorkItemHierarchyFromAzurePreservesAssignedToIdentityFields(t *testing.T) {
	workItem := map[string]any{
		"id": 123.0,
		"fields": map[string]any{
			"System.Title":        "Example task",
			"System.WorkItemType": "Task",
			"System.State":        "Active",
			"System.AssignedTo": map[string]any{
				"displayName": "Jane Doe",
				"uniqueName":  "jane.doe@example.com",
			},
		},
	}

	result := WorkItemHierarchyFromAzure(workItem)

	if result.AssignedTo != "Jane Doe" {
		t.Fatalf("AssignedTo = %q, want %q", result.AssignedTo, "Jane Doe")
	}
	if result.AssignedToUnique != "jane.doe@example.com" {
		t.Fatalf("AssignedToUnique = %q, want %q", result.AssignedToUnique, "jane.doe@example.com")
	}
}

func TestIsTimeWithinRangeInclusive(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	finish := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	if !isTimeWithinRangeInclusive(start, start, finish) {
		t.Fatal("expected start boundary to be included")
	}
	if !isTimeWithinRangeInclusive(finish, start, finish) {
		t.Fatal("expected finish boundary to be included")
	}
	if isTimeWithinRangeInclusive(start.Add(-time.Nanosecond), start, finish) {
		t.Fatal("expected time before start to be excluded")
	}
	if isTimeWithinRangeInclusive(finish.Add(time.Nanosecond), start, finish) {
		t.Fatal("expected time after finish to be excluded")
	}
}
