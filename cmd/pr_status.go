package cmd

import (
	"context"
	"fmt"
	"strings"

	"adoctl/pkg/devops"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	statusRepoName      string
	statusRepoID        string
	statusBranch        string
	statusUseGitContext bool
	statusNoGitContext  bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of pull request for current branch",
	Long: `Show the status of the pull request associated with the current git branch.

When run from within a git repository, this command automatically detects the current
branch and finds any associated pull requests. It displays PR details, review status,
and CI/build status.`,
	Example: `  # Show PR status for current branch
  adoctl pr status

  # Show PR status for a specific branch
  adoctl pr status --branch feature/my-feature

  # Show PR status for a specific repository
  adoctl pr status --repository-name myrepo

  # Show PR status without using git context
  adoctl pr status --no-git-context --repository-name myrepo --branch feature-branch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		// Determine if we should use git context
		useGitContext := statusUseGitContext && !statusNoGitContext && git.IsGitRepository()

		// Resolve repository
		repoID, repoName, err := ResolveRepoID(svc, statusRepoName, statusRepoID, useGitContext)
		if err != nil {
			return fmt.Errorf("could not determine repository: %w", err)
		}

		logger.Debug().Str("repoID", repoID).Str("repoName", repoName).Msg("Using repository")

		// Resolve branch
		branch := statusBranch
		if branch == "" && useGitContext {
			var currentBranch string
			currentBranch, err = git.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("could not determine current branch: %w", err)
			}
			branch = currentBranch
			logger.Debug().Str("branch", branch).Msg("Using current git branch")
		}

		if branch == "" {
			return fmt.Errorf("branch not specified and could not be auto-detected. Use --branch or run from a git repository")
		}

		// Find PRs for this branch
		prs, err := svc.ListPullRequests(ctx, repoID, "all", "", branch, "")
		if err != nil {
			return fmt.Errorf("error finding pull requests: %w", err)
		}

		// Filter to only PRs from this source branch
		var matchingPRs []models.PullRequest
		for _, pr := range prs {
			if strings.EqualFold(pr.SourceBranch, fmt.Sprintf("refs/heads/%s", branch)) {
				matchingPRs = append(matchingPRs, pr)
			}
		}

		if len(matchingPRs) == 0 {
			fmt.Printf("No pull requests found for branch '%s' in repository '%s'\n", branch, repoName)
			return nil
		}

		// Display each PR and collect for copy
		shouldCopy := ShouldCopyOutput(cmd)
		var markdownBuilder strings.Builder

		if shouldCopy {
			markdownBuilder.WriteString("**Pull Request Status**\n\n")
		}

		for _, pr := range matchingPRs {
			plain, markdown := formatPRStatus(ctx, svc, repoID, pr)
			fmt.Print(plain)
			fmt.Println()

			if shouldCopy {
				markdownBuilder.WriteString(markdown)
			}
		}

		if shouldCopy {
			if err := CopyToClipboard(markdownBuilder.String()); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("✓ Copied to clipboard!")
		}

		return nil
	},
}

// displayPRStatus displays PR status to terminal (kept for compatibility)
//
//nolint:unused
func displayPRStatus(ctx context.Context, svc *devops.DevOpsService, repoID string, pr models.PullRequest) error {
	plain, _ := formatPRStatus(ctx, svc, repoID, pr)
	fmt.Print(plain)
	return nil
}

// formatPRStatus returns both plain text and markdown formatted PR status
func formatPRStatus(ctx context.Context, svc *devops.DevOpsService, repoID string, pr models.PullRequest) (string, string) {
	var plainBuilder strings.Builder
	var markdownBuilder strings.Builder

	prID := pr.ID
	title := pr.Title
	status := string(pr.Status)

	sourceRef := strings.Replace(pr.SourceBranch, "refs/heads/", "", 1)
	targetRef := strings.Replace(pr.TargetBranch, "refs/heads/", "", 1)

	url := pr.URL

	// Plain text output
	fmt.Fprintf(&plainBuilder, "PR #%d: %s\n", prID, title)
	fmt.Fprintf(&plainBuilder, "  Status: %s\n", status)
	fmt.Fprintf(&plainBuilder, "  Branch: %s → %s\n", sourceRef, targetRef)

	// Markdown output with clickable links for Teams
	if url != "" {
		fmt.Fprintf(&markdownBuilder, "- **[PR #%d: %s](%s)**\n", prID, title, url)
	} else {
		fmt.Fprintf(&markdownBuilder, "- **PR #%d: %s**\n", prID, title)
	}
	fmt.Fprintf(&markdownBuilder, "  Status: %s | Branch: `%s` → `%s`", status, sourceRef, targetRef)

	// Author info
	author := pr.CreatedBy.DisplayName
	if author != "" {
		fmt.Fprintf(&plainBuilder, "  Author: %s\n", author)
		fmt.Fprintf(&markdownBuilder, " | Author: %s", author)
	}

	// URL
	if url != "" {
		fmt.Fprintf(&plainBuilder, "  URL: %s\n", url)
	}

	markdownBuilder.WriteString("\n")

	// Get reviewers
	reviewers, err := svc.GetPullRequestReviewers(ctx, repoID, prID)
	if err == nil && len(reviewers) > 0 {
		plainBuilder.WriteString("  Reviewers:\n")
		for _, reviewer := range reviewers {
			displayName := ""
			if reviewer.DisplayName != nil {
				displayName = *reviewer.DisplayName
			}
			voteText := formatVote(reviewer.Vote)
			fmt.Fprintf(&plainBuilder, "    - %s: %s\n", displayName, voteText)
		}
	}

	// Get work items
	workItems, err := svc.Client().GetPullRequestWorkItems(ctx, repoID, prID)
	if err == nil && len(workItems) > 0 {
		fmt.Fprintf(&plainBuilder, "  Linked Work Items: %d\n", len(workItems))
		for _, wi := range workItems {
			if wi.Id != nil {
				fmt.Fprintf(&plainBuilder, "    - %s\n", *wi.Id)
			}
		}
	}

	markdownBuilder.WriteString("\n")

	return plainBuilder.String(), markdownBuilder.String()
}

func init() {
	statusCmd.Flags().StringVar(&statusRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	statusCmd.Flags().StringVar(&statusRepoID, "repo-id", "", "Repository ID (alternative to --repository-name)")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "Branch name (auto-detected from current git branch if not specified)")
	statusCmd.Flags().BoolVar(&statusUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	statusCmd.Flags().BoolVar(&statusNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	statusCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	statusCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
