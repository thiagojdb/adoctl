package client

import (
	"context"
	"fmt"

	"adoctl/pkg/utils"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

func (c *Client) CreatePullRequest(ctx context.Context, repositoryID string, pullRequest *git.GitPullRequest, reviewers []string) (*git.GitPullRequest, error) {
	args := git.CreatePullRequestArgs{
		RepositoryId:           &repositoryID,
		GitPullRequestToCreate: pullRequest,
	}

	result, err := c.GitClient.CreatePullRequest(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	if len(reviewers) > 0 && result.PullRequestId != nil {
		for _, reviewerID := range reviewers {
			_, err := c.AddPullRequestReviewer(ctx, repositoryID, *result.PullRequestId, reviewerID)
			if err != nil {
				return nil, fmt.Errorf("failed to add reviewer %s: %w", reviewerID, err)
			}
		}
	}

	return result, nil
}

func (c *Client) GetPullRequest(ctx context.Context, pullRequestID int) (*git.GitPullRequest, error) {
	project := c.GetProject()
	args := git.GetPullRequestByIdArgs{
		Project:       &project,
		PullRequestId: &pullRequestID,
	}

	return c.GitClient.GetPullRequestById(ctx, args)
}

func (c *Client) GetPullRequests(ctx context.Context, repositoryID string, criteria *git.GitPullRequestSearchCriteria) ([]git.GitPullRequest, error) {
	args := git.GetPullRequestsArgs{
		RepositoryId:   &repositoryID,
		SearchCriteria: criteria,
	}

	result, err := c.GitClient.GetPullRequests(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	return *result, nil
}

func (c *Client) UpdatePullRequest(ctx context.Context, repositoryID string, pullRequestID int, pullRequest *git.GitPullRequest) (*git.GitPullRequest, error) {
	args := git.UpdatePullRequestArgs{
		RepositoryId:           &repositoryID,
		PullRequestId:          &pullRequestID,
		GitPullRequestToUpdate: pullRequest,
	}

	return c.GitClient.UpdatePullRequest(ctx, args)
}

func (c *Client) GetPullRequestWorkItems(ctx context.Context, repositoryID string, pullRequestID int) ([]webapi.ResourceRef, error) {
	args := git.GetPullRequestWorkItemRefsArgs{
		RepositoryId:  &repositoryID,
		PullRequestId: &pullRequestID,
	}

	result, err := c.GitClient.GetPullRequestWorkItemRefs(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request work items: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return *result, nil
}

func (c *Client) BranchExists(ctx context.Context, repositoryID, branchName string) (bool, error) {
	branchName = normalizeBranchName(branchName)

	args := git.GetBranchArgs{
		RepositoryId: &repositoryID,
		Name:         &branchName,
	}

	_, err := c.GitClient.GetBranch(ctx, args)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (c *Client) GetActualBranchName(ctx context.Context, repositoryID, branchName string) (string, error) {
	branchName = normalizeBranchName(branchName)

	args := git.GetBranchArgs{
		RepositoryId: &repositoryID,
		Name:         &branchName,
	}

	branch, err := c.GitClient.GetBranch(ctx, args)
	if err != nil {
		return "", fmt.Errorf("failed to get branch %s: %w", branchName, err)
	}

	if branch == nil || branch.Name == nil {
		return "", fmt.Errorf("branch %s not found", branchName)
	}

	return *branch.Name, nil
}

func (c *Client) GetRefs(ctx context.Context, repositoryID string, filter *string) ([]git.GitRef, error) {
	args := git.GetRefsArgs{
		RepositoryId: &repositoryID,
		Filter:       filter,
	}

	result, err := c.GitClient.GetRefs(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get refs: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return result.Value, nil
}

func (c *Client) GetBranchCommitID(ctx context.Context, repositoryID, branchName string) (string, error) {
	branchName = normalizeBranchName(branchName)

	args := git.GetBranchArgs{
		RepositoryId: &repositoryID,
		Name:         &branchName,
	}

	branch, err := c.GitClient.GetBranch(ctx, args)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit ID: %w", err)
	}

	if branch == nil || branch.Commit == nil || branch.Commit.CommitId == nil {
		return "", fmt.Errorf("branch %s commit ID not found", branchName)
	}

	return *branch.Commit.CommitId, nil
}

func (c *Client) GetCommitDiffs(ctx context.Context, repositoryID, baseVersion, targetVersion string) (*git.GitCommitDiffs, error) {
	baseVersionType := git.GitVersionType(git.GitVersionTypeValues.Branch)
	targetVersionType := git.GitVersionType(git.GitVersionTypeValues.Branch)

	args := git.GetCommitDiffsArgs{
		RepositoryId: &repositoryID,
		BaseVersionDescriptor: &git.GitBaseVersionDescriptor{
			Version:     &baseVersion,
			VersionType: &baseVersionType,
		},
		TargetVersionDescriptor: &git.GitTargetVersionDescriptor{
			Version:     &targetVersion,
			VersionType: &targetVersionType,
		},
		Top: utils.Ptr(1000),
	}

	return c.GitClient.GetCommitDiffs(ctx, args)
}

func (c *Client) BranchesHaveChanges(ctx context.Context, repositoryID, sourceBranch, targetBranch string) (bool, error) {
	sourceCommitID, err := c.GetBranchCommitID(ctx, repositoryID, sourceBranch)
	if err != nil {
		return false, fmt.Errorf("failed to get source branch commit ID: %w", err)
	}

	targetCommitID, err := c.GetBranchCommitID(ctx, repositoryID, targetBranch)
	if err != nil {
		return false, fmt.Errorf("failed to get target branch commit ID: %w", err)
	}

	diffs, err := c.GetCommitDiffs(ctx, repositoryID, targetCommitID, sourceCommitID)
	if err != nil {
		return false, fmt.Errorf("failed to get commit diffs: %w", err)
	}

	if diffs == nil {
		return false, nil
	}

	if diffs.AheadCount != nil && *diffs.AheadCount > 0 {
		return true, nil
	}

	if diffs.ChangeCounts != nil {
		for _, count := range *diffs.ChangeCounts {
			if count > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func (c *Client) MergePullRequest(ctx context.Context, repositoryID string, pullRequestID int, completionOptions *GitPullRequestCompletionOptions) (*git.GitPullRequest, error) {
	status := git.PullRequestStatus(git.PullRequestStatusValues.Completed)
	updatedPR := &git.GitPullRequest{
		Status: &status,
	}

	if completionOptions != nil {
		sdkCompletionOptions := &git.GitPullRequestCompletionOptions{}

		if completionOptions.DeleteSourceBranch != nil {
			sdkCompletionOptions.DeleteSourceBranch = completionOptions.DeleteSourceBranch
		}
		if completionOptions.SquashMerge != nil {
			sdkCompletionOptions.SquashMerge = completionOptions.SquashMerge
		}
		if completionOptions.MergeCommitMessage != nil {
			sdkCompletionOptions.MergeCommitMessage = completionOptions.MergeCommitMessage
		}
		if completionOptions.BypassPolicy != nil {
			sdkCompletionOptions.BypassPolicy = completionOptions.BypassPolicy
		}
		if completionOptions.BypassReason != nil {
			sdkCompletionOptions.BypassReason = completionOptions.BypassReason
		}
		if completionOptions.TransitionWorkItems != nil {
			sdkCompletionOptions.TransitionWorkItems = completionOptions.TransitionWorkItems
		}

		updatedPR.CompletionOptions = sdkCompletionOptions
	}

	args := git.UpdatePullRequestArgs{
		RepositoryId:           &repositoryID,
		PullRequestId:          &pullRequestID,
		GitPullRequestToUpdate: updatedPR,
	}

	return c.GitClient.UpdatePullRequest(ctx, args)
}

func (c *Client) AbandonPullRequest(ctx context.Context, repositoryID string, pullRequestID int, abortMessage string) (*git.GitPullRequest, error) {
	status := git.PullRequestStatus(git.PullRequestStatusValues.Abandoned)
	updatedPR := &git.GitPullRequest{
		Status: &status,
	}

	if abortMessage != "" {
		updatedPR.Description = &abortMessage
	}

	args := git.UpdatePullRequestArgs{
		RepositoryId:           &repositoryID,
		PullRequestId:          &pullRequestID,
		GitPullRequestToUpdate: updatedPR,
	}

	return c.GitClient.UpdatePullRequest(ctx, args)
}

func (c *Client) GetPullRequestReviewers(ctx context.Context, repositoryID string, pullRequestID int) ([]git.IdentityRefWithVote, error) {
	args := git.GetPullRequestReviewersArgs{
		RepositoryId:  &repositoryID,
		PullRequestId: &pullRequestID,
	}

	result, err := c.GitClient.GetPullRequestReviewers(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request reviewers: %w", err)
	}

	return *result, nil
}

func (c *Client) AddPullRequestReviewer(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string) (*git.IdentityRefWithVote, error) {
	args := git.CreatePullRequestReviewerArgs{
		RepositoryId:  &repositoryID,
		PullRequestId: &pullRequestID,
		Reviewer: &git.IdentityRefWithVote{
			Id: &reviewerID,
		},
	}

	return c.GitClient.CreatePullRequestReviewer(ctx, args)
}

func (c *Client) SetPullRequestReviewerVote(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string, vote int) error {
	reviewer := &git.IdentityRefWithVote{
		Id:   &reviewerID,
		Vote: &vote,
	}

	args := git.UpdatePullRequestReviewerArgs{
		RepositoryId:  &repositoryID,
		PullRequestId: &pullRequestID,
		ReviewerId:    &reviewerID,
		Reviewer:      reviewer,
	}

	_, err := c.GitClient.UpdatePullRequestReviewer(ctx, args)
	return err
}

func (c *Client) RemovePullRequestReviewer(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string) error {
	args := git.DeletePullRequestReviewerArgs{
		RepositoryId:  &repositoryID,
		PullRequestId: &pullRequestID,
		ReviewerId:    &reviewerID,
	}

	return c.GitClient.DeletePullRequestReviewer(ctx, args)
}

func normalizeBranchName(branchName string) string {
	if len(branchName) > 0 && branchName[0] == '/' {
		branchName = branchName[1:]
	}
	return branchName
}
