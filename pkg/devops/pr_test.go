package devops

import (
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

func TestBuildCreatePullRequestSetsAutoCompleteIdentity(t *testing.T) {
	autoCompleteSetBy := &webapi.IdentityRef{
		Id:          ptr("user-id"),
		DisplayName: ptr("Jane Doe"),
		UniqueName:  ptr("jane.doe@example.com"),
	}

	pr := buildCreatePullRequest("feature", "main", "My feature", "Adds functionality", autoCompleteSetBy)

	if pr.AutoCompleteSetBy == nil {
		t.Fatal("AutoCompleteSetBy = nil, want identity ref")
	}
	if pr.AutoCompleteSetBy.Id == nil || *pr.AutoCompleteSetBy.Id != "user-id" {
		t.Fatalf("AutoCompleteSetBy.Id = %v, want user-id", pr.AutoCompleteSetBy.Id)
	}
	if pr.AutoCompleteSetBy.DisplayName == nil || *pr.AutoCompleteSetBy.DisplayName != "Jane Doe" {
		t.Fatalf("AutoCompleteSetBy.DisplayName = %v, want Jane Doe", pr.AutoCompleteSetBy.DisplayName)
	}
	if pr.AutoCompleteSetBy.UniqueName == nil || *pr.AutoCompleteSetBy.UniqueName != "jane.doe@example.com" {
		t.Fatalf("AutoCompleteSetBy.UniqueName = %v, want jane.doe@example.com", pr.AutoCompleteSetBy.UniqueName)
	}
}

func TestBuildCreatePullRequestLeavesAutoCompleteUnset(t *testing.T) {
	pr := buildCreatePullRequest("feature", "main", "My feature", "Adds functionality", nil)

	if pr.AutoCompleteSetBy != nil {
		t.Fatal("AutoCompleteSetBy = non-nil, want nil")
	}
}

func ptr[T any](v T) *T {
	return &v
}
