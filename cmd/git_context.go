package cmd

import (
	"context"
	"fmt"
	"strings"

	"adoctl/pkg/devops"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"
)

// GitContext holds information about the current git repository context
type GitContext struct {
	RemoteInfo   *git.RemoteInfo
	Branch       string
	RemoteURL    string
	RepoID       string
	RepoName     string
	WorkItemID   string
	RecentCommit string
	IsGitRepo    bool
}

// GetGitContext detects and returns the current git context
func GetGitContext() *GitContext {
	ctx := &GitContext{
		IsGitRepo: git.IsGitRepository(),
	}

	if !ctx.IsGitRepo {
		return ctx
	}

	// Get current branch
	branch, err := git.GetCurrentBranch()
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to get current branch")
	} else {
		ctx.Branch = branch
		ctx.WorkItemID = git.ExtractWorkItemID(branch)
	}

	// Get remote URL
	remoteURL, err := git.GetRemoteURL("origin")
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to get remote URL")
	} else {
		ctx.RemoteURL = remoteURL
		if info, err := git.ParseAzureDevOpsURL(remoteURL); err == nil {
			ctx.RemoteInfo = info
		}
	}

	// Get recent commit for title suggestion
	if msg, err := git.GetLastCommitMessage(); err == nil {
		ctx.RecentCommit = msg
	}

	return ctx
}

// ResolveRepoFromGitContext attempts to resolve the repository ID and name from git context
// It matches the git remote URL against available Azure DevOps repositories
func ResolveRepoFromGitContext(ctx context.Context, svc *devops.DevOpsService) (string, string, error) {
	gitCtx := GetGitContext()

	if !gitCtx.IsGitRepo {
		return "", "", fmt.Errorf("not in a git repository")
	}

	if gitCtx.RemoteInfo == nil {
		return "", "", fmt.Errorf("could not parse git remote URL")
	}

	logger.Debug().
		Str("organization", gitCtx.RemoteInfo.Organization).
		Str("project", gitCtx.RemoteInfo.Project).
		Str("repository", gitCtx.RemoteInfo.Repository).
		Msg("Detected git remote")

	// List all repositories and find a match
	repos, err := svc.ListRepositories()
	if err != nil {
		return "", "", fmt.Errorf("failed to list repositories: %w", err)
	}

	// Try to match by repository name (case-insensitive)
	targetRepoName := gitCtx.RemoteInfo.Repository
	for _, repo := range repos {
		if strings.EqualFold(repo.Name, targetRepoName) {
			logger.Debug().
				Str("repoName", repo.Name).
				Str("repoID", repo.ID).
				Msg("Matched repository from git context")
			return repo.ID, repo.Name, nil
		}
	}

	return "", "", fmt.Errorf("repository '%s' not found in Azure DevOps project", targetRepoName)
}

// ResolveRepoID resolves a repository ID from various sources:
// 1. Explicit repoID parameter
// 2. Explicit repoName parameter (lookup by name)
// 3. Git context auto-detection
func ResolveRepoID(svc *devops.DevOpsService, repoName, repoID string, useGitContext bool) (string, string, error) {
	// Priority 1: Explicit repoID
	if repoID != "" {
		// If we have repoID but no name, try to look up the name
		if repoName == "" {
			repos, err := svc.ListRepositories()
			if err == nil {
				for _, repo := range repos {
					if repo.ID == repoID {
						return repoID, repo.Name, nil
					}
				}
			}
		}
		return repoID, repoName, nil
	}

	// Priority 2: Explicit repoName
	if repoName != "" {
		id, err := svc.GetRepositoryID(repoName)
		if err != nil {
			return "", "", fmt.Errorf("could not find repository '%s': %w", repoName, err)
		}
		return id, repoName, nil
	}

	// Priority 3: Git context auto-detection
	if useGitContext {
		ctx, cancel := GetContext()
		defer cancel()

		id, name, err := ResolveRepoFromGitContext(ctx, svc)
		if err == nil {
			logger.Debug().
				Str("repoID", id).
				Str("repoName", name).
				Msg("Resolved repository from git context")
			return id, name, nil
		}
		logger.Debug().Err(err).Msg("Failed to resolve repository from git context")
	}

	return "", "", fmt.Errorf("repository not specified and could not be auto-detected from git context. Use --repository-name or --repo-id")
}

// ResolveSourceBranch resolves the source branch from various sources
func ResolveSourceBranch(branch string, useGitContext bool) (string, error) {
	// Priority 1: Explicit branch
	if branch != "" {
		return branch, nil
	}

	// Priority 2: Git context
	if useGitContext && git.IsGitRepository() {
		currentBranch, err := git.GetCurrentBranch()
		if err == nil {
			logger.Debug().
				Str("branch", currentBranch).
				Msg("Resolved source branch from git context")
			return currentBranch, nil
		}
	}

	return "", fmt.Errorf("source branch not specified and could not be auto-detected from git context. Use --source-branch")
}

// ResolveTargetBranch resolves the target branch from various sources
func ResolveTargetBranch(branch string, useGitContext bool) (string, error) {
	// Priority 1: Explicit branch
	if branch != "" {
		return branch, nil
	}

	// Priority 2: Git context - try to get tracked branch or default
	if useGitContext && git.IsGitRepository() {
		// Try to get the tracked upstream branch
		if tracked, err := git.GetTrackedBranch(); err == nil {
			// Extract just the branch name from "origin/branch"
			parts := strings.SplitN(tracked, "/", 2)
			if len(parts) == 2 {
				logger.Debug().
					Str("branch", parts[1]).
					Msg("Resolved target branch from tracked upstream")
				return parts[1], nil
			}
		}

		// Fall back to default branch
		if defaultBranch, err := git.GetDefaultBranch(); err == nil {
			logger.Debug().
				Str("branch", defaultBranch).
				Msg("Resolved target branch to default")
			return defaultBranch, nil
		}
	}

	return "", fmt.Errorf("target branch not specified and could not be auto-detected from git context. Use --target-branch")
}

// SuggestPRTitle generates a PR title from git context
func SuggestPRTitle(explicitTitle string, useGitContext bool) string {
	// Priority 1: Explicit title
	if explicitTitle != "" {
		return explicitTitle
	}

	// Priority 2: Recent commit message
	if useGitContext {
		if title := git.SuggestPRTitle(); title != "" {
			return title
		}
	}

	return ""
}

// ExtractWorkItemsFromBranch extracts work item IDs from the current branch name
func ExtractWorkItemsFromBranch(explicitIDs []string, useGitContext bool) []string {
	// Priority 1: Explicit work item IDs
	if len(explicitIDs) > 0 {
		return explicitIDs
	}

	// Priority 2: Extract from branch name
	if useGitContext && git.IsGitRepository() {
		if branch, err := git.GetCurrentBranch(); err == nil {
			if workItemID := git.ExtractWorkItemID(branch); workItemID != "" {
				logger.Debug().
					Str("workItemID", workItemID).
					Str("branch", branch).
					Msg("Extracted work item ID from branch name")
				return []string{workItemID}
			}
		}
	}

	return []string{}
}
