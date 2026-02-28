package client

import (
	"context"
	"fmt"
	"sync"

	"adoctl/pkg/logger"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type BulkOperationResult struct {
	Index  int
	Data   interface{}
	Error  error
	RepoID string
	PRID   int
}

func (c *Client) BulkGetPullRequests(ctx context.Context, repoPRMap map[string][]int, maxWorkers int) (map[int]*git.GitPullRequest, error) {
	results := make(map[int]*git.GitPullRequest)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for repoID, prIDs := range repoPRMap {
		for _, prID := range prIDs {
			wg.Add(1)
			go func(repoID string, prID int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				pr, err := c.GetPullRequest(ctx, prID)
				mutex.Lock()
				if err != nil {
					logger.Warn().
						Err(err).
						Int("pr_id", prID).
						Str("repo_id", repoID).
						Msg("Error getting pull request in bulk operation")
				} else {
					results[prID] = pr
				}
				mutex.Unlock()
			}(repoID, prID)
		}
	}

	wg.Wait()
	return results, nil
}

func (c *Client) BulkCheckBranches(ctx context.Context, repoBranches map[string][]string, maxWorkers int) (map[string]map[string]bool, error) {
	results := make(map[string]map[string]bool)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for repoID, branches := range repoBranches {
		results[repoID] = make(map[string]bool)
		for _, branch := range branches {
			wg.Add(1)
			go func(repoID, branch string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				exists, err := c.BranchExists(ctx, repoID, branch)
				mutex.Lock()
				results[repoID][branch] = exists && err == nil
				mutex.Unlock()
			}(repoID, branch)
		}
	}

	wg.Wait()
	return results, nil
}

func (c *Client) BulkGetPullRequestWorkItems(ctx context.Context, repoPRs map[string][]int, maxWorkers int) (map[int][]string, error) {
	results := make(map[int][]string)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for repoID, prIDs := range repoPRs {
		for _, prID := range prIDs {
			wg.Add(1)
			go func(repoID string, prID int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				workItems, err := c.GetPullRequestWorkItems(ctx, repoID, prID)
				if err != nil {
					mutex.Lock()
					results[prID] = []string{}
					mutex.Unlock()
					return
				}

				ids := make([]string, 0, len(workItems))
				for _, wi := range workItems {
					if wi.Id != nil {
						ids = append(ids, *wi.Id)
					}
				}

				mutex.Lock()
				results[prID] = ids
				mutex.Unlock()
			}(repoID, prID)
		}
	}

	wg.Wait()
	return results, nil
}

func (c *Client) BulkMergePullRequests(ctx context.Context, requests []MergeRequest, maxWorkers int) ([]BulkOperationResult, error) {
	results := make([]BulkOperationResult, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for i, req := range requests {
		wg.Add(1)
		go func(index int, req MergeRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pr, err := c.MergePullRequest(ctx, req.RepositoryID, req.PullRequestID, req.CompletionOptions)
			results[index] = BulkOperationResult{
				Index:  index,
				Data:   pr,
				Error:  err,
				RepoID: req.RepositoryID,
				PRID:   req.PullRequestID,
			}
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

type MergeRequest struct {
	RepositoryID      string
	PullRequestID     int
	CompletionOptions *GitPullRequestCompletionOptions
}

func (c *Client) BulkAbandonPullRequests(ctx context.Context, requests []AbandonRequest, maxWorkers int) ([]BulkOperationResult, error) {
	results := make([]BulkOperationResult, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for i, req := range requests {
		wg.Add(1)
		go func(index int, req AbandonRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pr, err := c.AbandonPullRequest(ctx, req.RepositoryID, req.PullRequestID, req.AbortMessage)
			results[index] = BulkOperationResult{
				Index:  index,
				Data:   pr,
				Error:  err,
				RepoID: req.RepositoryID,
				PRID:   req.PullRequestID,
			}
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

type AbandonRequest struct {
	RepositoryID  string
	PullRequestID int
	AbortMessage  string
}

func (c *Client) BulkAddReviewers(ctx context.Context, requests []ReviewerRequest, maxWorkers int) ([]BulkOperationResult, error) {
	results := make([]BulkOperationResult, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for i, req := range requests {
		wg.Add(1)
		go func(index int, req ReviewerRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			reviewer, err := c.AddPullRequestReviewer(ctx, req.RepositoryID, req.PullRequestID, req.ReviewerID)
			results[index] = BulkOperationResult{
				Index:  index,
				Data:   reviewer,
				Error:  err,
				RepoID: req.RepositoryID,
				PRID:   req.PullRequestID,
			}
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

type ReviewerRequest struct {
	RepositoryID  string
	PullRequestID int
	ReviewerID    string
}

func (c *Client) BulkGetPullRequestsByRepo(ctx context.Context, repoIDs []string, criteria *git.GitPullRequestSearchCriteria, maxWorkers int) (map[string][]git.GitPullRequest, error) {
	results := make(map[string][]git.GitPullRequest)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for _, repoID := range repoIDs {
		wg.Add(1)
		go func(repoID string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			prs, err := c.GetPullRequests(ctx, repoID, criteria)
			mutex.Lock()
			if err != nil {
				logger.Warn().
					Err(err).
					Str("repo_id", repoID).
					Msg("Error getting pull requests for repository in bulk operation")
				results[repoID] = []git.GitPullRequest{}
			} else {
				results[repoID] = prs
			}
			mutex.Unlock()
		}(repoID)
	}

	wg.Wait()
	return results, nil
}

func (c *Client) BulkGetBuilds(ctx context.Context, buildIDs []int, maxWorkers int) (map[int]*build.Build, error) {
	results := make(map[int]*build.Build)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for _, buildID := range buildIDs {
		wg.Add(1)
		go func(buildID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			build, err := c.GetBuild(ctx, buildID)
			mutex.Lock()
			if err != nil {
				logger.Warn().
					Err(err).
					Int("build_id", buildID).
					Msg("Error getting build in bulk operation")
			} else {
				results[buildID] = build
			}
			mutex.Unlock()
		}(buildID)
	}

	wg.Wait()
	return results, nil
}

func (c *Client) GetBuild(ctx context.Context, buildID int) (*build.Build, error) {
	args := build.GetBuildArgs{
		BuildId: &buildID,
	}
	return c.BuildClient.GetBuild(ctx, args)
}

func (c *Client) BulkLinkWorkItems(ctx context.Context, requests []WorkItemLinkRequest, maxWorkers int) ([]BulkOperationResult, error) {
	results := make([]BulkOperationResult, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for i, req := range requests {
		wg.Add(1)
		go func(index int, req WorkItemLinkRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			relation := map[string]any{
				"rel": "ArtifactLink",
				"url": fmt.Sprintf("vstfs:///Git/PullRequestId/%s%%2F%s%%2F%d", req.ProjectID, req.RepositoryID, req.PullRequestID),
				"attributes": map[string]any{
					"name": "Pull Request",
				},
			}

			err := c.AddWorkItemRelation(ctx, req.WorkItemID, relation)
			results[index] = BulkOperationResult{
				Index:  index,
				Error:  err,
				RepoID: req.RepositoryID,
				PRID:   req.PullRequestID,
			}
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

type WorkItemLinkRequest struct {
	ProjectID     string
	RepositoryID  string
	PullRequestID int
	WorkItemID    int
}

func (c *Client) BulkGetRepositories(ctx context.Context, repoIDs []string, maxWorkers int) (map[string]*git.GitRepository, error) {
	results := make(map[string]*git.GitRepository)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxWorkers)

	for _, repoID := range repoIDs {
		wg.Add(1)
		go func(repoID string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			repo, err := c.GetRepository(ctx, repoID)
			mutex.Lock()
			if err != nil {
				logger.Warn().
					Err(err).
					Str("repo_id", repoID).
					Msg("Error getting repository in bulk operation")
			} else {
				results[repoID] = repo
			}
			mutex.Unlock()
		}(repoID)
	}

	wg.Wait()
	return results, nil
}
