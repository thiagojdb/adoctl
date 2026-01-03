package cmd

import (
	"fmt"

	"adoctl/pkg/azure/client"
	"adoctl/pkg/devops"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"

	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/spf13/cobra"
)

var (
	mergeRepoName        string
	mergeRepoID          string
	mergePRID            int
	mergeStrategy        string
	deleteSource         bool
	commitMessage        string
	mergeSkipPolicy      bool
	mergeUseGitContext   bool
	mergeNoGitContext    bool
	abandonUseGitContext bool
	abandonNoGitContext  bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge a pull request",
	Long: `Merge a pull request in Azure DevOps with the specified parameters.
Supports different merge strategies: noFastForward, squash, rebase, rebaseMerge.`,
	Example: `  # Merge PR #123 with default settings (auto-detect repository from git)
  adoctl pr merge --pr 123

  # Merge PR with squash strategy and delete source branch
  adoctl pr merge --pr 123 --strategy squash --delete-source

  # Merge with custom commit message
  adoctl pr merge --pr 123 --message "Merged featureXYZ"

  # Merge with explicit repository
  adoctl pr merge --repo-id <repo-id> --pr 123

  # Merge without git context
  adoctl pr merge --no-git-context --repository-name my-repo --pr 123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}

		// Determine if we should use git context
		useGitContext := mergeUseGitContext && !mergeNoGitContext && git.IsGitRepository()

		repoID, repoName, err := ResolveRepoID(svc, mergeRepoName, mergeRepoID, useGitContext)
		if err != nil {
			return err
		}

		pr, err := svc.GetPullRequest(ctx, mergePRID)
		if err != nil {
			return fmt.Errorf("failed to get PR #%d: %w", mergePRID, err)
		}

		if err := validatePRRepo(pr, repoID, mergePRID); err != nil {
			return err
		}

		if pr.Status != nil && adogit.PullRequestStatus(*pr.Status) == adogit.PullRequestStatusValues.Completed {
			return fmt.Errorf("PR #%d is already completed", mergePRID)
		}

		if pr.Status != nil && adogit.PullRequestStatus(*pr.Status) == adogit.PullRequestStatusValues.Abandoned {
			return fmt.Errorf("PR #%d is abandoned", mergePRID)
		}

		title := ""
		if pr.Title != nil {
			title = *pr.Title
		}

		// Dry-run or confirm
		details := map[string]string{
			"PR":         fmt.Sprintf("#%d", mergePRID),
			"Title":      title,
			"Repository": repoName,
			"Strategy":   mergeStrategy,
		}
		if deleteSource {
			details["Delete Source Branch"] = "yes"
		}
		if commitMessage != "" {
			details["Custom Message"] = "yes"
		}

		if IsDryRun() {
			PrintDryRunAction("merge pull request", details)
			return nil
		}

		if err := RequireConfirmation("merge this pull request", details); err != nil {
			return err
		}

		var mergeStrategyPtr *client.GitPullRequestMergeStrategy
		if mergeStrategy != "" {
			strategy := client.GitPullRequestMergeStrategy(mergeStrategy)
			mergeStrategyPtr = &strategy
		}

		result, err := svc.MergePullRequest(ctx, repoID, mergePRID, mergeStrategyPtr, deleteSource, commitMessage)
		if err != nil {
			return fmt.Errorf("failed to merge PR #%d: %w", mergePRID, err)
		}

		logger.Info().Msg("Pull Request merged successfully")
		fmt.Printf("PR ID: %d\n", mergePRID)
		fmt.Printf("Status: %s\n", string(*result.Status))
		if result.Url != nil && *result.Url != "" {
			fmt.Printf("URL: %s\n", *result.Url)
		}

		return nil
	},
}

var (
	abandonRepoName string
	abandonRepoID   string
	abandonPRID     int
	abandonMessage  string
)

var abandonCmd = &cobra.Command{
	Use:   "abandon",
	Short: "Abandon a pull request",
	Long:  `Abandon a pull request in Azure DevOps with an optional comment.`,
	Example: `  # Abandon PR #123 (auto-detect repository from git)
  adoctl pr abandon --pr 123

  # Abandon with a comment
  adoctl pr abandon --pr 123 --comment "Abandoning - needs rework"

  # Abandon with explicit repository
  adoctl pr abandon --repo-id <repo-id> --pr 123

  # Abandon without git context
  adoctl pr abandon --no-git-context --repository-name my-repo --pr 123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}

		// Determine if we should use git context
		useGitContext := abandonUseGitContext && !abandonNoGitContext && git.IsGitRepository()

		repoID, repoName, err := ResolveRepoID(svc, abandonRepoName, abandonRepoID, useGitContext)
		if err != nil {
			return err
		}

		pr, err := svc.GetPullRequest(ctx, abandonPRID)
		if err != nil {
			return fmt.Errorf("failed to get PR #%d: %w", abandonPRID, err)
		}

		if err := validatePRRepo(pr, repoID, abandonPRID); err != nil {
			return err
		}

		if pr.Status != nil && adogit.PullRequestStatus(*pr.Status) == adogit.PullRequestStatusValues.Completed {
			return fmt.Errorf("PR #%d is already completed", abandonPRID)
		}

		if pr.Status != nil && adogit.PullRequestStatus(*pr.Status) == adogit.PullRequestStatusValues.Abandoned {
			return fmt.Errorf("PR #%d is already abandoned", abandonPRID)
		}

		title := ""
		if pr.Title != nil {
			title = *pr.Title
		}

		details := map[string]string{
			"PR":         fmt.Sprintf("#%d", abandonPRID),
			"Title":      title,
			"Repository": repoName,
		}
		if abandonMessage != "" {
			details["Comment"] = abandonMessage
		}

		if IsDryRun() {
			PrintDryRunAction("abandon pull request", details)
			return nil
		}

		if err := RequireConfirmation("abandon this pull request", details); err != nil {
			return err
		}

		result, err := svc.AbandonPullRequest(ctx, repoID, abandonPRID, abandonMessage)
		if err != nil {
			return fmt.Errorf("failed to abandon PR #%d: %w", abandonPRID, err)
		}

		logger.Info().Msg("Pull Request abandoned successfully")
		fmt.Printf("PR ID: %d\n", abandonPRID)
		fmt.Printf("Status: %s\n", string(*result.Status))
		if result.Url != nil && *result.Url != "" {
			fmt.Printf("URL: %s\n", *result.Url)
		}

		return nil
	},
}

func init() {
	mergeCmd.Flags().StringVar(&mergeRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	mergeCmd.Flags().StringVar(&mergeRepoID, "repo-id", "", "Repository ID (alternative to --repository-name)")
	mergeCmd.Flags().IntVar(&mergePRID, "pr", 0, "Pull request ID to merge")
	mergeCmd.Flags().StringVar(&mergeStrategy, "strategy", "", "Merge strategy: noFastForward, squash, rebase, rebaseMerge (default: noFastForward)")
	mergeCmd.Flags().BoolVar(&deleteSource, "delete-source", false, "Delete source branch after merge")
	mergeCmd.Flags().StringVar(&commitMessage, "message", "", "Custom merge commit message")
	mergeCmd.Flags().BoolVar(&mergeSkipPolicy, "skip-policy", false, "Bypass merge policy requirements")
	mergeCmd.Flags().BoolVar(&mergeUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	mergeCmd.Flags().BoolVar(&mergeNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	mergeCmd.MarkFlagRequired("pr")
	mergeCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	mergeCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")

	abandonCmd.Flags().StringVar(&abandonRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	abandonCmd.Flags().StringVar(&abandonRepoID, "repo-id", "", "Repository ID (alternative to --repository-name)")
	abandonCmd.Flags().IntVar(&abandonPRID, "pr", 0, "Pull request ID to abandon")
	abandonCmd.Flags().StringVar(&abandonMessage, "comment", "", "Abandon comment")
	abandonCmd.Flags().BoolVar(&abandonUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	abandonCmd.Flags().BoolVar(&abandonNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	abandonCmd.MarkFlagRequired("pr")
	abandonCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	abandonCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
