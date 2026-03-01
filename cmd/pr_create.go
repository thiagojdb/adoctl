package cmd

import (
	"fmt"
	"strings"

	"adoctl/pkg/devops"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	createRepoName      string
	createRepoID        string
	createSourceBranch  string
	createTargetBranch  string
	createTitle         string
	createDescription   string
	createReviewers     []string
	createWorkItemIDs   []string
	createUseGitContext bool
	createNoGitContext  bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pull request",
	Long: `Create a new pull request in Azure DevOps with the specified parameters.

When run from within a git repository with an Azure DevOps remote, this command can
auto-detect the repository, source branch, and target branch. It can also extract
work item IDs from branch names like "feature/PBI-12345".`,
	Example: `  # Create a PR with explicit settings
  adoctl pr create --repository-name myrepo --source-branch feature --target-branch main --title "My feature"

  # Create a PR using git context (auto-detect repo and branches)
  adoctl pr create --title "My feature"

  # Create a PR with reviewers and description
  adoctl pr create --repository-name myrepo --source-branch feature --target-branch main \
    --title "My feature" --description "This PR adds new functionality" \
    --reviewers user1@domain.com --reviewers user2@domain.com

  # Create a PR and link work items (auto-extracted from branch like feature/PBI-123)
  adoctl pr create --title "Fix issue"

  # Create a PR with explicit work items
  adoctl pr create --repository-name myrepo --source-branch feature --target-branch main \
    --title "Fix bug 123" --description "This PR fixes bug 123" --work-item-id 123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("error creating service: %w", err)
		}
		defer svc.Close()

		// Determine if we should use git context
		useGitContext := createUseGitContext && !createNoGitContext && git.IsGitRepository()

		// Resolve repository
		repoID, repoName, err := ResolveRepoID(svc, createRepoName, createRepoID, useGitContext)
		if err != nil {
			return fmt.Errorf("could not determine repository: %w", err)
		}

		logger.Debug().Str("repoID", repoID).Str("repoName", repoName).Msg("Using repository")

		// Resolve source branch
		sourceBranch, err := ResolveSourceBranch(createSourceBranch, useGitContext)
		if err != nil {
			return fmt.Errorf("could not determine source branch: %w", err)
		}

		// Resolve target branch
		targetBranch, err := ResolveTargetBranch(createTargetBranch, useGitContext)
		if err != nil {
			return fmt.Errorf("could not determine target branch: %w", err)
		}

		// Get or suggest PR title
		title := createTitle
		if title == "" {
			title = SuggestPRTitle("", useGitContext)
			if title != "" {
				logger.Info().Str("title", title).Msg("Using PR title from recent commit")
			}
		}
		if title == "" {
			return fmt.Errorf("PR title is required. Use --title to specify")
		}

		// Extract work items from branch if not explicitly provided
		workItemIDs := ExtractWorkItemsFromBranch(createWorkItemIDs, useGitContext)
		if len(workItemIDs) > 0 && len(createWorkItemIDs) == 0 {
			logger.Info().Strs("workItemIDs", workItemIDs).Msg("Auto-linked work items from branch name")
		}

		logger.Debug().
			Str("sourceBranch", sourceBranch).
			Str("targetBranch", targetBranch).
			Str("title", title).
			Msg("Creating PR")

		result, err := svc.CreatePullRequest(ctx, repoID, sourceBranch, targetBranch, title, createDescription, createReviewers, workItemIDs, true)
		if err != nil {
			return fmt.Errorf("error creating PR: %w", err)
		}

		logger.Info().Msg("Pull Request created successfully")

		prID := result.ID
		url := result.URL

		// Print to terminal
		fmt.Printf("PR ID: %d\n", prID)
		fmt.Printf("Repository: %s\n", repoName)
		fmt.Printf("Branch: %s → %s\n", sourceBranch, targetBranch)
		if url != "" {
			fmt.Printf("URL: %s\n", url)
		}
		if len(workItemIDs) > 0 {
			fmt.Printf("Linked Work Items: %v\n", workItemIDs)
		}

		// Copy to clipboard if requested
		if ShouldCopyOutput(cmd) {
			var markdownBuilder strings.Builder

			// Markdown format for Teams compatibility
			if url != "" {
				fmt.Fprintf(&markdownBuilder, "**PR Created:** [#%d: %s](%s)\n\n", prID, title, url)
			} else {
				fmt.Fprintf(&markdownBuilder, "**PR Created:** #%d: %s\n\n", prID, title)
			}
			fmt.Fprintf(&markdownBuilder, "- **Repository:** %s\n", repoName)
			fmt.Fprintf(&markdownBuilder, "- **Branch:** `%s` → `%s`\n", sourceBranch, targetBranch)
			if len(workItemIDs) > 0 {
				fmt.Fprintf(&markdownBuilder, "- **Linked Work Items:** %v\n", workItemIDs)
			}

			if err := CopyToClipboard(markdownBuilder.String()); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("\n✓ Copied to clipboard!")
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	createCmd.Flags().StringVar(&createRepoID, "repo-id", "", "Repository ID (alternative to --repository-name)")
	createCmd.Flags().StringVar(&createSourceBranch, "source-branch", "", "Source branch name (auto-detected from current git branch if not specified)")
	createCmd.Flags().StringVar(&createTargetBranch, "target-branch", "", "Target branch name (auto-detected from git upstream or defaults to 'main' if not specified)")
	createCmd.Flags().StringVar(&createTitle, "title", "", "PR title (auto-suggested from recent commit if not specified)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "PR description")
	createCmd.Flags().StringArrayVar(&createReviewers, "reviewers", []string{}, "List of reviewer IDs")
	createCmd.Flags().StringArrayVar(&createWorkItemIDs, "work-item-id", []string{}, "Work item IDs to link (can be specified multiple times, auto-extracted from branch name if not specified)")
	createCmd.Flags().BoolVar(&createUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	createCmd.Flags().BoolVar(&createNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	createCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	createCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
