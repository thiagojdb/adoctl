package devops

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"adoctl/pkg/azure/client"
	"adoctl/pkg/config"
	"adoctl/pkg/filter"
	"adoctl/pkg/models"
	"adoctl/pkg/utils"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type PRRequirementsChecker struct {
	client              *client.Client
	workItemsCache      map[string]int
	workItemsCacheMutex sync.RWMutex
	workItemsDataCache  map[string][]string
	workItemsDataMutex  sync.RWMutex
}

func NewPRRequirementsChecker(client *client.Client) *PRRequirementsChecker {
	return &PRRequirementsChecker{
		client:             client,
		workItemsCache:     make(map[string]int),
		workItemsDataCache: make(map[string][]string),
	}
}

func (c *PRRequirementsChecker) CheckPRRequirements(pr models.PullRequest) []string {
	warnings := []string{}

	if pr.MergeStatus == models.MergeStatusConflicts {
		warnings = append(warnings, "Has merge conflicts")
	}

	if pr.MergeStatus == models.MergeStatusNotSet || pr.MergeStatus == models.MergeStatusNotStarted {
		warnings = append(warnings, "Merge not checked yet")
	}

	return warnings
}

func (c *PRRequirementsChecker) GetWorkItemsCount(ctx context.Context, repoID string, prID int) int {
	cacheKey := fmt.Sprintf("%s:%d", repoID, prID)

	c.workItemsCacheMutex.RLock()
	if count, ok := c.workItemsCache[cacheKey]; ok {
		c.workItemsCacheMutex.RUnlock()
		return count
	}
	c.workItemsCacheMutex.RUnlock()

	workItems, err := c.client.GetPullRequestWorkItems(ctx, repoID, prID)
	if err != nil {
		return 0
	}

	count := len(workItems)

	c.workItemsCacheMutex.Lock()
	c.workItemsCache[cacheKey] = count
	c.workItemsCacheMutex.Unlock()

	return count
}

func (c *PRRequirementsChecker) GetWorkItemIDs(ctx context.Context, repoID string, prID int) []string {
	cacheKey := fmt.Sprintf("%s:%d", repoID, prID)

	c.workItemsDataMutex.RLock()
	if ids, ok := c.workItemsDataCache[cacheKey]; ok {
		c.workItemsDataMutex.RUnlock()
		return ids
	}
	c.workItemsDataMutex.RUnlock()

	workItems, err := c.client.GetPullRequestWorkItems(ctx, repoID, prID)
	if err != nil {
		return []string{}
	}

	ids := make([]string, 0, len(workItems))
	for _, wi := range workItems {
		if wi.Id != nil {
			ids = append(ids, *wi.Id)
		}
	}

	c.workItemsDataMutex.Lock()
	c.workItemsDataCache[cacheKey] = ids
	c.workItemsDataMutex.Unlock()

	return ids
}

func (c *PRRequirementsChecker) HasWorkItemMatch(ctx context.Context, repoID string, prID int, workItemFilters []string) bool {
	if len(workItemFilters) == 0 {
		return true
	}

	workItemIDs := c.GetWorkItemIDs(ctx, repoID, prID)

	for _, wiFilter := range workItemFilters {
		for _, wiID := range workItemIDs {
			if strings.Contains(strings.ToLower(wiID), strings.ToLower(wiFilter)) {
				return true
			}
		}
	}

	return false
}

func (c *PRRequirementsChecker) GetWorkItemsCountsBatch(ctx context.Context, items []struct {
	RepoID string
	PRID   int
}, maxWorkers int) map[string]int {
	results := make(map[string]int)
	var mutex sync.Mutex

	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		go func(repoID string, prID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			count := c.GetWorkItemsCount(ctx, repoID, prID)

			mutex.Lock()
			results[fmt.Sprintf("%s:%d", repoID, prID)] = count
			mutex.Unlock()
		}(item.RepoID, item.PRID)
	}

	wg.Wait()

	return results
}

func (s *DevOpsService) CreatePullRequest(ctx context.Context, repositoryID, sourceBranch, targetBranch, title, description string, reviewers []string, workItemIDs []string, skipChangeCheck bool) (*models.PullRequest, error) {
	if !skipChangeCheck {
		hasChanges, err := s.client.BranchesHaveChanges(ctx, repositoryID, sourceBranch, targetBranch)
		if err != nil {
			return nil, fmt.Errorf("failed to check for changes between branches: %w", err)
		}

		if !hasChanges {
			return nil, fmt.Errorf("no changes to merge from '%s' to '%s'", sourceBranch, targetBranch)
		}
	}

	sourceRefName := fmt.Sprintf("refs/heads/%s", sourceBranch)
	targetRefName := fmt.Sprintf("refs/heads/%s", targetBranch)
	status := git.PullRequestStatus(git.PullRequestStatusValues.Active)

	pr := &git.GitPullRequest{
		SourceRefName: &sourceRefName,
		TargetRefName: &targetRefName,
		Title:         &title,
		Description:   &description,
		Status:        &status,
		IsDraft:       utils.Ptr(false),
	}

	result, err := s.client.CreatePullRequest(ctx, repositoryID, pr, []string{})
	if err != nil {
		return nil, err
	}

	if len(workItemIDs) > 0 {
		prID := 0
		if result.PullRequestId != nil {
			prID = *result.PullRequestId
		}

		err := s.LinkWorkItemsToPullRequest(repositoryID, prID, workItemIDs)
		if err != nil {
			return nil, err
		}
	}

	return utils.Ptr(models.PullRequestFromAzure(result)), nil
}

func (s *DevOpsService) BulkCreatePullRequests(ctx context.Context, sourceBranch, targetBranch, title, description string, workItemIDs []string) ([]BulkCreateResult, error) {
	repos, err := s.ListRepositories()
	if err != nil {
		return nil, err
	}

	results := []BulkCreateResult{}
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.DefaultParallelProcesses)

	for _, repo := range repos {
		wg.Add(1)
		go func(r models.Repository) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			repoID := r.ID
			repoName := r.Name

			actualSourceBranch, err := s.client.GetActualBranchName(ctx, repoID, sourceBranch)
			if err != nil {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("Source branch '%s' not found", sourceBranch),
				})
				resultsMutex.Unlock()
				return
			}

			actualTargetBranch, err := s.client.GetActualBranchName(ctx, repoID, targetBranch)
			if err != nil {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("Target branch '%s' not found", targetBranch),
				})
				resultsMutex.Unlock()
				return
			}

			hasChanges, err := s.client.BranchesHaveChanges(ctx, repoID, actualSourceBranch, actualTargetBranch)
			if err != nil {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("Error checking for changes: %v", err),
				})
				resultsMutex.Unlock()
				return
			}

			if !hasChanges {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    "No changes to merge",
				})
				resultsMutex.Unlock()
				return
			}

			existingPRs, err := s.ListPullRequests(ctx, repoID, "active", actualTargetBranch, actualSourceBranch, "")
			if err != nil {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("Error checking for existing PRs: %v", err),
				})
				resultsMutex.Unlock()
				return
			}

			if len(existingPRs) > 0 {
				prID := existingPRs[0].ID
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("PR already exists: #%d", prID),
				})
				resultsMutex.Unlock()
				return
			}

			result, err := s.CreatePullRequest(ctx, repoID, actualSourceBranch, actualTargetBranch, title, description, []string{}, workItemIDs, true)
			if err != nil {
				resultsMutex.Lock()
				results = append(results, BulkCreateResult{
					RepoName: repoName,
					Success:  false,
					Error:    fmt.Sprintf("Error creating PR: %v", err),
				})
				resultsMutex.Unlock()
				return
			}

			resultsMutex.Lock()
			results = append(results, BulkCreateResult{
				RepoName: repoName,
				Success:  true,
				PRID:     result.ID,
				URL:      result.URL,
			})
			resultsMutex.Unlock()
		}(repo)
	}

	wg.Wait()

	return results, nil
}

func (s *DevOpsService) GetPullRequest(ctx context.Context, pullrequestid int) (*git.GitPullRequest, error) {
	return s.client.GetPullRequest(ctx, pullrequestid)
}

func (s *DevOpsService) MergePullRequest(ctx context.Context, repositoryID string, pullRequestID int, mergeStrategy *client.GitPullRequestMergeStrategy, deleteSourceBranch bool, commitMessage string) (*git.GitPullRequest, error) {
	completionOptions := &client.GitPullRequestCompletionOptions{
		MergeStrategy:      mergeStrategy,
		DeleteSourceBranch: &deleteSourceBranch,
	}
	if commitMessage != "" {
		completionOptions.MergeCommitMessage = &commitMessage
	}
	return s.client.MergePullRequest(ctx, repositoryID, pullRequestID, completionOptions)
}

func (s *DevOpsService) AbandonPullRequest(ctx context.Context, repositoryID string, pullRequestID int, abortMessage string) (*git.GitPullRequest, error) {
	return s.client.AbandonPullRequest(ctx, repositoryID, pullRequestID, abortMessage)
}

func (s *DevOpsService) GetPullRequestReviewers(ctx context.Context, repositoryID string, pullRequestID int) ([]git.IdentityRefWithVote, error) {
	return s.client.GetPullRequestReviewers(ctx, repositoryID, pullRequestID)
}

func (s *DevOpsService) AddPullRequestReviewer(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string) (*git.IdentityRefWithVote, error) {
	return s.client.AddPullRequestReviewer(ctx, repositoryID, pullRequestID, reviewerID)
}

func (s *DevOpsService) SetPullRequestReviewerVote(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string, vote int) error {
	return s.client.SetPullRequestReviewerVote(ctx, repositoryID, pullRequestID, reviewerID, vote)
}

func (s *DevOpsService) RemovePullRequestReviewer(ctx context.Context, repositoryID string, pullRequestID int, reviewerID string) error {
	return s.client.RemovePullRequestReviewer(ctx, repositoryID, pullRequestID, reviewerID)
}

func (s *DevOpsService) ListPullRequests(ctx context.Context, repositoryID, status, targetBranch, sourceBranch, creatorID string) ([]models.PullRequest, error) {
	criteria := &git.GitPullRequestSearchCriteria{}
	if status != "" {
		var statusVal git.PullRequestStatus
		if repositoryID == "" && status == "all" {
			statusVal = git.PullRequestStatus(git.PullRequestStatusValues.All)
		} else {
			statusVal = git.PullRequestStatus(status)
		}
		criteria.Status = &statusVal
	}

	var prs []git.GitPullRequest
	var err error

	if repositoryID != "" {
		prs, err = s.client.GetPullRequests(ctx, repositoryID, criteria)
	} else {
		prs, err = s.client.GetPullRequests(ctx, "", nil)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	result := make([]models.PullRequest, 0, len(prs))
	for _, pr := range prs {
		modelPR := models.PullRequestFromAzure(&pr)

		// Apply additional filtering
		include := true

		if targetBranch != "" {
			expectedRef := fmt.Sprintf("refs/heads/%s", targetBranch)
			if !strings.EqualFold(modelPR.TargetBranch, expectedRef) {
				include = false
			}
		}

		if sourceBranch != "" {
			expectedRef := fmt.Sprintf("refs/heads/%s", sourceBranch)
			if !strings.EqualFold(modelPR.SourceBranch, expectedRef) {
				include = false
			}
		}

		if creatorID != "" {
			if modelPR.CreatedBy.ID != creatorID {
				include = false
			}
		}

		if include {
			result = append(result, modelPR)
		}
	}

	return result, nil
}

func (s *DevOpsService) ListPullRequestsWithFilter(ctx context.Context, repositoryID string, prFilter *filter.PRFilter) ([]models.PullRequest, error) {
	prs, err := s.ListPullRequests(ctx, repositoryID, prFilter.Status, prFilter.TargetBranch, prFilter.SourceBranch, prFilter.CreatorID)
	if err != nil {
		return nil, err
	}

	filteredPRs := []models.PullRequest{}
	for _, pr := range prs {
		matches, err := prFilter.MatchesPR(pr)
		if err != nil {
			return nil, err
		}
		if matches {
			filteredPRs = append(filteredPRs, pr)
		}
	}

	return filteredPRs, nil
}
