package cmd

import (
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/filter"
	"adoctl/pkg/git"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	listRepoName      string
	listStatus        string
	listTargetBranch  string
	listSourceBranch  string
	listCreator       string
	listTitleFuzzy    string
	listRepoFuzzy     string
	listCurrentBranch bool
	listUseGitContext bool
	listNoGitContext  bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long: `List pull requests with optional filtering by repository, status, branch, and creator.

When run from within a git repository with an Azure DevOps remote, this command can
auto-detect the repository and filter by the current branch.`,
	Example: `  # List active PRs (default)
  adoctl pr list

  # List all PRs including completed and abandoned
  adoctl pr list --status all

  # List PRs for a specific repository
  adoctl pr list --repository-name myrepo

  # List PRs for the current repository (auto-detected from git)
  adoctl pr list --use-git-context

  # List PRs for the current branch only
  adoctl pr list --current-branch

  # List PRs targeting main branch
  adoctl pr list --target-branch main

  # List PRs by a specific creator
  adoctl pr list --creator john@example.com

  # List active PRs with fuzzy title search
  adoctl pr list --status active --title-fuzzy "login"

  # List PRs for repos matching fuzzy pattern
  adoctl pr list --repo-fuzzy "api"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		// Determine if we should use git context
		useGitContext := listUseGitContext && !listNoGitContext && git.IsGitRepository()

		// If --current-branch is set, auto-detect the current branch
		sourceBranch := listSourceBranch
		if listCurrentBranch && sourceBranch == "" {
			if branch, err := git.GetCurrentBranch(); err == nil {
				sourceBranch = branch
				logger.Debug().Str("branch", sourceBranch).Msg("Filtering by current branch")
			} else {
				return fmt.Errorf("could not determine current branch: %w", err)
			}
		}

		// Resolve repository
		repoID, _, err := ResolveRepoID(svc, listRepoName, "", useGitContext)
		if err != nil {
			// If no repo specified and we're not using git context, list all PRs across repos
			if listRepoName == "" && !useGitContext {
				repoID = ""
			} else {
				return fmt.Errorf("could not determine repository: %w", err)
			}
		}

		creatorID, err := resolveCreator(svc, listCreator)
		if err != nil {
			return err
		}

		prFilter := &filter.PRFilter{
			TitleFuzzy:   listTitleFuzzy,
			RepoFuzzy:    listRepoFuzzy,
			SourceBranch: sourceBranch,
			TargetBranch: listTargetBranch,
			Status:       listStatus,
			CreatorID:    creatorID,
		}

		prs, err := svc.ListPullRequestsWithFilter(ctx, repoID, prFilter)
		if err != nil {
			return fmt.Errorf("error listing PRs: %w", err)
		}

		logger.Info().Int("count", len(prs)).Msg("Found pull requests")

		report := svc.GenerateMessageReport(ctx, prs, true, nil)
		fmt.Println(report)

		if ShouldCopyOutput(cmd) {
			htmlFragment := svc.GenerateHTMLMessageReport(ctx, prs, true, nil)
			htmlContent := devops.WrapHTMLForClipboard(htmlFragment)
			plainContent := svc.GeneratePlainTextReport(ctx, prs, true, nil)
			if err := CopyRichToClipboard(htmlContent, plainContent); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("✓ Copied to clipboard!")
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listRepoName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	listCmd.Flags().StringVar(&listStatus, "status", "active", "PR status filter (default: active)")
	listCmd.Flags().StringVar(&listTargetBranch, "target-branch", "", "Filter by target branch")
	listCmd.Flags().StringVar(&listSourceBranch, "source-branch", "", "Filter by source branch")
	listCmd.Flags().StringVar(&listCreator, "creator", "", "Filter by creator (use 'self', name, or ID)")
	listCmd.Flags().StringVar(&listTitleFuzzy, "title-fuzzy", "", "Filter PR title by fuzzy match")
	listCmd.Flags().StringVar(&listRepoFuzzy, "repo-fuzzy", "", "Filter repository by fuzzy match")
	listCmd.Flags().BoolVar(&listCurrentBranch, "current-branch", false, "Filter PRs to only show those from the current git branch")
	listCmd.Flags().BoolVar(&listUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	listCmd.Flags().BoolVar(&listNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	listCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
