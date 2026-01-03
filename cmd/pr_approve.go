package cmd

import (
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"

	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/spf13/cobra"
)

var (
	approveRepoName      string
	approveRepoID        string
	approvePRID          int
	approveUseGitContext bool
	approveNoGitContext  bool
)

func validatePRRepo(pr *adogit.GitPullRequest, repoID string, prID int) error {
	if pr.Repository != nil && pr.Repository.Id != nil && pr.Repository.Id.String() != "" && pr.Repository.Id.String() != repoID {
		return fmt.Errorf("PR #%d does not belong to repository %s", prID, repoID)
	}
	return nil
}

func formatVote(vote *int) string {
	v := 0
	if vote != nil {
		v = *vote
	}
	switch v {
	case 10:
		return "Approved"
	case 5:
		return "Approved with suggestions"
	case 0:
		return "No vote"
	case -5:
		return "Waiting for author"
	case -10:
		return "Rejected"
	default:
		return fmt.Sprintf("Vote: %d", v)
	}
}

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve a pull request",
	Long:  `Approve a pull request by setting your vote to approved.`,
	Example: `  # Approve PR #123 (auto-detect repository from git)
  adoctl pr approve --pr 123

  # Approve PR #123 with explicit repository
  adoctl pr approve --repo-id <repo-id> --pr 123

  # Approve using repository name
  adoctl pr approve --repository-name my-repo --pr 123

  # Approve without git context
  adoctl pr approve --no-git-context --repository-name my-repo --pr 123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		// Determine if we should use git context
		useGitContext := approveUseGitContext && !approveNoGitContext && git.IsGitRepository()

		repoID, _, err := ResolveRepoID(svc, approveRepoName, approveRepoID, useGitContext)
		if err != nil {
			return err
		}

		pr, err := svc.GetPullRequest(ctx, approvePRID)
		if err != nil {
			return fmt.Errorf("failed to get PR #%d: %w", approvePRID, err)
		}

		if err := validatePRRepo(pr, repoID, approvePRID); err != nil {
			return err
		}

		reviewers, err := svc.GetPullRequestReviewers(ctx, repoID, approvePRID)
		if err != nil {
			return fmt.Errorf("failed to get reviewers: %w", err)
		}

		currentUserID := ""
		for _, reviewer := range reviewers {
			if reviewer.Vote != nil && *reviewer.Vote != 0 && reviewer.Id != nil {
				currentUserID = *reviewer.Id
				break
			}
		}

		if currentUserID == "" {
			return fmt.Errorf("current user is not a reviewer on this PR")
		}

		err = svc.SetPullRequestReviewerVote(ctx, repoID, approvePRID, currentUserID, 10)
		if err != nil {
			return fmt.Errorf("failed to approve PR: %w", err)
		}

		logger.Info().Msg("PR approved successfully")

		return nil
	},
}

func init() {
	approveCmd.Flags().StringVar(&approveRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	approveCmd.Flags().StringVar(&approveRepoID, "repo-id", "", "Repository ID (alternative to --repository-name)")
	approveCmd.Flags().IntVar(&approvePRID, "pr", 0, "Pull request ID")
	approveCmd.Flags().BoolVar(&approveUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	approveCmd.Flags().BoolVar(&approveNoGitContext, "no-git-context", false, "Disable git context auto-detection")
	approveCmd.MarkFlagRequired("pr")
	approveCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	approveCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
