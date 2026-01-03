package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Commit represents a git commit
type Commit struct {
	Hash    string
	Subject string
	Body    string
}

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetGitDir returns the path to the .git directory
func GetGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryRoot returns the absolute path to the git repository root
func GetRepositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the URL of the specified remote (default: origin)
func GetRemoteURL(remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	cmd := exec.Command("git", "remote", "get-url", remote)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL for '%s': %w", remote, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRecentCommits returns the most recent commits up to the specified count
func GetRecentCommits(count int) ([]Commit, error) {
	if count <= 0 {
		count = 10
	}

	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--format=%H%x00%s%x00%b%x00%x01")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	return parseCommits(string(output)), nil
}

// parseCommits parses the output from git log
func parseCommits(output string) []Commit {
	commits := []Commit{}
	entries := strings.Split(output, "\x01")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, "\x00")
		if len(parts) >= 2 {
			commit := Commit{
				Hash:    strings.TrimSpace(parts[0]),
				Subject: strings.TrimSpace(parts[1]),
			}
			if len(parts) >= 3 {
				commit.Body = strings.TrimSpace(parts[2])
			}
			commits = append(commits, commit)
		}
	}

	return commits
}

// ExtractWorkItemID extracts a work item ID from a branch name
// Supports formats like: feature/PBI-12345, bugfix/12345, PBI-12345-feature
func ExtractWorkItemID(branchName string) string {
	// Pattern to match common work item ID formats
	patterns := []string{
		`(?i)(?:PBI|WI|BUG|TASK|FEATURE)[-_]?(\d+)`,
		`(?i)#(\d+)`,
		`(?i)^(?:feature|bugfix|hotfix|release)[-_/](\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(branchName)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// ParseRemoteURL parses an Azure DevOps remote URL and extracts organization, project, and repo
// Supports formats:
//   - https://dev.azure.com/{org}/{project}/_git/{repo}
//   - https://{org}.visualstudio.com/{project}/_git/{repo}
//   - git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
type RemoteInfo struct {
	Organization string
	Project      string
	Repository   string
}

// ParseAzureDevOpsURL parses an Azure DevOps git URL and returns the components
func ParseAzureDevOpsURL(url string) (*RemoteInfo, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty URL")
	}

	// HTTPS format: https://dev.azure.com/{org}/{project}/_git/{repo}
	httpsPattern := regexp.MustCompile(`https://dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/]+)`)
	matches := httpsPattern.FindStringSubmatch(url)
	if len(matches) == 4 {
		return &RemoteInfo{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
		}, nil
	}

	// Old Visual Studio format: https://{org}.visualstudio.com/{project}/_git/{repo}
	vsPattern := regexp.MustCompile(`https://([^\.]+)\.visualstudio\.com/([^/]+)/_git/([^/]+)`)
	matches = vsPattern.FindStringSubmatch(url)
	if len(matches) == 4 {
		return &RemoteInfo{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
		}, nil
	}

	// SSH format: git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
	sshPattern := regexp.MustCompile(`git@ssh\.dev\.azure\.com:v3/([^/]+)/([^/]+)/([^/]+)`)
	matches = sshPattern.FindStringSubmatch(url)
	if len(matches) == 4 {
		return &RemoteInfo{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
		}, nil
	}

	return nil, fmt.Errorf("unable to parse Azure DevOps URL: %s", url)
}

// GetDefaultBranch returns the default branch of the repository (usually main or master)
func GetDefaultBranch() (string, error) {
	// Try to get the default branch from git config
	cmd := exec.Command("git", "config", "init.defaultBranch")
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		if branch != "" {
			return branch, nil
		}
	}

	// Check if main exists
	cmd = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/main")
	if err := cmd.Run(); err == nil {
		return "main", nil
	}

	// Check if master exists
	cmd = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/master")
	if err := cmd.Run(); err == nil {
		return "master", nil
	}

	// Default to main if we can't determine
	return "main", nil
}

// HasUncommittedChanges checks if there are uncommitted changes in the working directory
func HasUncommittedChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetLastCommitMessage returns the subject of the most recent commit
func GetLastCommitMessage() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit message: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SuggestPRTitle generates a PR title from recent commits
func SuggestPRTitle() string {
	commits, err := GetRecentCommits(5)
	if err != nil || len(commits) == 0 {
		return ""
	}

	// Use the most recent commit subject as the title
	return commits[0].Subject
}

// GetTrackedBranch returns the upstream branch that the current branch is tracking
func GetTrackedBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no upstream branch configured: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a branch exists locally
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branch))
	return cmd.Run() == nil
}

// GetRemoteBranches returns a list of remote branches
func GetRemoteBranches(remote string) ([]string, error) {
	if remote == "" {
		remote = "origin"
	}
	cmd := exec.Command("git", "branch", "-r", "--list", fmt.Sprintf("%s/*", remote))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote branches: %w", err)
	}

	var branches []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		branch := strings.TrimSpace(scanner.Text())
		// Remove remote prefix (e.g., "origin/")
		branch = strings.TrimPrefix(branch, remote+"/")
		// Skip HEAD
		if branch != "HEAD" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}
