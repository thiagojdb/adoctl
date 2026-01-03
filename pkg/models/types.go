package models

import (
	"strings"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
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
