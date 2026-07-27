package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// PullRequestStatus represents the status of a pull request
type PullRequestStatus string

const (
	PRStatusActive    PullRequestStatus = "active"
	PRStatusCompleted PullRequestStatus = "completed"
	PRStatusAbandoned PullRequestStatus = "abandoned"
	PRStatusAll       PullRequestStatus = "all"
	PRStatusNotSet    PullRequestStatus = "notSet"
)

// MergeStatus represents the merge status of a pull request
type MergeStatus string

const (
	MergeStatusNotSet     MergeStatus = "notSet"
	MergeStatusNotStarted MergeStatus = "notStarted"
	MergeStatusConflicts  MergeStatus = "conflicts"
	MergeStatusSucceeded  MergeStatus = "succeeded"
)

// Repository represents an Azure DevOps Git repository
type Repository struct {
	ID        string
	Name      string
	URL       string
	RemoteURL string
	Project   Project
}

// Project represents an Azure DevOps project
type Project struct {
	ID   string
	Name string
}

// Identity represents a user or group identity
type Identity struct {
	ID          string
	DisplayName string
	UniqueName  string
	URL         string
	ImageURL    string
	Descriptor  string
}

// PullRequest represents a Git pull request
type PullRequest struct {
	ID           int
	Title        string
	Description  string
	Status       PullRequestStatus
	SourceBranch string
	TargetBranch string
	URL          string
	MergeStatus  MergeStatus
	Repository   Repository
	CreatedBy    Identity
	IsDraft      bool
}

// PullRequestFromAzure converts an Azure DevOps GitPullRequest to our domain model
func PullRequestFromAzure(pr *git.GitPullRequest) PullRequest {
	if pr == nil {
		return PullRequest{}
	}

	result := PullRequest{
		Repository:   RepositoryFromAzure(pr.Repository),
		CreatedBy:    IdentityFromAzure(pr.CreatedBy),
		SourceBranch: dereferenceString(pr.SourceRefName),
		TargetBranch: dereferenceString(pr.TargetRefName),
		Title:        dereferenceString(pr.Title),
		Description:  dereferenceString(pr.Description),
		URL:          dereferenceString(pr.Url),
	}

	if pr.PullRequestId != nil {
		result.ID = *pr.PullRequestId
	}

	if pr.MergeStatus != nil {
		result.MergeStatus = MergeStatus(*pr.MergeStatus)
	}

	if pr.Status != nil {
		result.Status = PullRequestStatus(*pr.Status)
	}

	if pr.IsDraft != nil {
		result.IsDraft = *pr.IsDraft
	}

	return result
}

// RepositoryFromAzure converts an Azure DevOps GitRepository to our domain model
func RepositoryFromAzure(repo *git.GitRepository) Repository {
	if repo == nil {
		return Repository{}
	}

	result := Repository{
		ID:        dereferenceGUID(repo.Id),
		Name:      dereferenceString(repo.Name),
		URL:       dereferenceString(repo.Url),
		RemoteURL: dereferenceString(repo.RemoteUrl),
	}

	if repo.Project != nil {
		result.Project = Project{
			ID:   dereferenceGUID(repo.Project.Id),
			Name: dereferenceString(repo.Project.Name),
		}
	}

	return result
}

// IdentityFromAzure converts an Azure DevOps IdentityRef to our domain model
func IdentityFromAzure(ref *webapi.IdentityRef) Identity {
	if ref == nil {
		return Identity{}
	}

	return Identity{
		ID:          dereferenceString(ref.Id),
		DisplayName: dereferenceString(ref.DisplayName),
		UniqueName:  dereferenceString(ref.UniqueName),
		URL:         dereferenceString(ref.Url),
		ImageURL:    dereferenceString(ref.ImageUrl),
		Descriptor:  dereferenceString(ref.Descriptor),
	}
}

// GetSourceBranchName returns the source branch name without refs/heads/ prefix
func (pr *PullRequest) GetSourceBranchName() string {
	return normalizeBranchName(pr.SourceBranch)
}

// GetTargetBranchName returns the target branch name without refs/heads/ prefix
func (pr *PullRequest) GetTargetBranchName() string {
	return normalizeBranchName(pr.TargetBranch)
}

// HasMergeConflicts returns true if the PR has merge conflicts
func (pr *PullRequest) HasMergeConflicts() bool {
	return pr.MergeStatus == MergeStatusConflicts
}

// IsMergeable returns true if the PR can be merged without conflicts
func (pr *PullRequest) IsMergeable() bool {
	return pr.MergeStatus == MergeStatusSucceeded
}

// normalizeBranchName removes refs/heads/ or refs/tags/ prefix from branch names
func normalizeBranchName(refName string) string {
	const refsHeads = "refs/heads/"
	const refsTags = "refs/tags/"

	if strings.HasPrefix(refName, refsHeads) {
		return refName[len(refsHeads):]
	}

	if strings.HasPrefix(refName, refsTags) {
		return refName[len(refsTags):]
	}

	return refName
}

// dereferenceString safely dereferences a string pointer
func dereferenceString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// dereferenceGUID safely dereferences a UUID pointer
func dereferenceGUID(g *uuid.UUID) string {
	if g == nil {
		return ""
	}
	return g.String()
}

// Iteration represents a sprint/iteration in Azure DevOps
type Iteration struct {
	ID         string
	Name       string
	Path       string
	StartDate  *time.Time
	FinishDate *time.Time
	IsCurrent  bool
}

// WorkItemHierarchy represents a work item with its children
type WorkItemHierarchy struct {
	ID               int
	Title            string
	Type             string
	State            string
	AssignedTo       string
	AssignedToUnique string
	URL              string
	Children         []WorkItemHierarchy
	IsAssignedToUser bool
	ParentID         *int
}

// IterationFromAzure converts an Azure DevOps WorkItemClassificationNode to our domain model
func IterationFromAzure(node *workitemtracking.WorkItemClassificationNode, projectName string) Iteration {
	if node == nil {
		return Iteration{}
	}

	result := Iteration{
		Name: dereferenceString(node.Name),
	}

	if node.Identifier != nil {
		result.ID = node.Identifier.String()
	}

	// Build the full path
	if node.Path != nil {
		result.Path = dereferenceString(node.Path)
	} else if node.Name != nil {
		result.Path = projectName + "\\" + *node.Name
	}

	// Extract dates from attributes
	if node.Attributes != nil {
		attrs := *node.Attributes
		if startDate, ok := attrs["startDate"].(string); ok && startDate != "" {
			if t, err := time.Parse(time.RFC3339, startDate); err == nil {
				result.StartDate = &t
			}
		}
		if finishDate, ok := attrs["finishDate"].(string); ok && finishDate != "" {
			if t, err := time.Parse(time.RFC3339, finishDate); err == nil {
				result.FinishDate = &t
			}
		}
	}

	// Determine if current
	now := time.Now()
	if result.StartDate != nil && result.FinishDate != nil {
		result.IsCurrent = isTimeWithinRangeInclusive(now, *result.StartDate, *result.FinishDate)
	}

	return result
}

// WorkItemHierarchyFromAzure converts a work item map to WorkItemHierarchy
func WorkItemHierarchyFromAzure(workItem map[string]any) WorkItemHierarchy {
	if workItem == nil {
		return WorkItemHierarchy{}
	}

	result := WorkItemHierarchy{}

	// Extract ID
	if id, ok := workItem["id"].(float64); ok {
		result.ID = int(id)
	}

	// Extract fields
	if fields, ok := workItem["fields"].(map[string]any); ok {
		result.Title = getFieldString(fields, "System.Title")
		result.Type = getFieldString(fields, "System.WorkItemType")
		result.State = getFieldString(fields, "System.State")

		// AssignedTo can be a string or an identity object.
		if assignedTo, ok := fields["System.AssignedTo"].(map[string]any); ok {
			result.AssignedTo = getFieldString(assignedTo, "displayName")
			result.AssignedToUnique = getFirstNonEmptyField(assignedTo, "uniqueName", "mailAddress")
		} else {
			result.AssignedTo = getFieldString(fields, "System.AssignedTo")
			result.AssignedToUnique = result.AssignedTo
		}
	}

	// Extract URL
	result.URL = getFieldString(workItem, "url")
	if result.URL == "" {
		// Try to get HTML URL from _links
		if links, ok := workItem["_links"].(map[string]any); ok {
			if html, ok := links["html"].(map[string]any); ok {
				result.URL = getFieldString(html, "href")
			}
		}
	}

	return result
}

// getFieldString safely extracts a string value from a map
func getFieldString(m map[string]any, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFirstNonEmptyField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := getFieldString(m, key); value != "" {
			return value
		}
	}
	return ""
}

func isTimeWithinRangeInclusive(now, start, finish time.Time) bool {
	return !now.Before(start) && !now.After(finish)
}
