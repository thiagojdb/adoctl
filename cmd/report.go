package cmd

import (
	"context"
	"fmt"
	"os"

	"adoctl/pkg/devops"
	"adoctl/pkg/filter"
	"adoctl/pkg/git"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	reportRepositoryName string
	reportRepoID         string
	reportStatus         string
	reportTargetBranch   string
	reportSourceBranch   string
	reportCreator        string
	reportOutput         string
	reportCopy           bool
	reportNoWarnings     bool
	reportWorkItems      []string
	reportTitleRegex     string
	reportTitleFuzzy     string
	reportRepoRegex      string
	reportRepoFuzzy      string
	reportUseGitContext  bool
	reportNoGitContext   bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate PR message report",
	Long:  `Generate a report of pull requests with optional filtering and output options.`,
	Example: `  # Generate a report for all active PRs
  adoctl report --status active

  # Generate a report for the current repository (auto-detected from git)
  adoctl report

  # Generate a report for a specific repository
  adoctl report --repository-name myrepo

  # Generate and copy to clipboard for Teams
  adoctl report --status active --copy

  # Filter by target branch and save to file
  adoctl report --target-branch main --output pr-report.md

  # Filter by creator
  adoctl report --creator john@example.com

  # Use regex pattern for title filter
  adoctl report --title-regex ".*release.*"

  # Use fuzzy match for title
  adoctl report --title-fuzzy "login bug"

  # Hide requirement warnings
  adoctl report --status active --no-warnings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		// Determine if we should use git context
		useGitContext := reportUseGitContext && !reportNoGitContext && git.IsGitRepository()

		repoID, _, err := ResolveRepoID(svc, reportRepositoryName, reportRepoID, useGitContext)
		if err != nil {
			return fmt.Errorf("could not determine repository ID: %w", err)
		}

		creatorID, err := resolveCreator(svc, reportCreator)
		if err != nil {
			return err
		}

		prFilter := &filter.PRFilter{
			TitleRegex:   reportTitleRegex,
			TitleFuzzy:   reportTitleFuzzy,
			RepoRegex:    reportRepoRegex,
			RepoFuzzy:    reportRepoFuzzy,
			SourceBranch: reportSourceBranch,
			TargetBranch: reportTargetBranch,
			Status:       reportStatus,
			CreatorID:    creatorID,
		}

		prs, err := svc.ListPullRequestsWithFilter(ctx, repoID, prFilter)
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		return outputReport(ctx, svc, prs, reportCopy)
	},
}

func resolveCreator(svc *devops.DevOpsService, creator string) (string, error) {
	if creator == "" {
		return "", nil
	}
	return svc.ResolveCreator(creator)
}

func outputReport(ctx context.Context, svc *devops.DevOpsService, prs []models.PullRequest, shouldCopy bool) error {
	report := svc.GenerateMessageReport(ctx, prs, !reportNoWarnings, reportWorkItems)

	// Handle file output
	if reportOutput != "" {
		err := os.WriteFile(reportOutput, []byte(report), 0600)
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		fmt.Printf("Report saved to %s\n", reportOutput)
	}

	// Always print to terminal
	fmt.Println(report)

	// Copy to clipboard if requested
	if shouldCopy {
		htmlFragment := svc.GenerateHTMLMessageReport(ctx, prs, !reportNoWarnings, reportWorkItems)
		htmlContent := devops.WrapHTMLForClipboard(htmlFragment)
		plainContent := svc.GeneratePlainTextReport(ctx, prs, !reportNoWarnings, reportWorkItems)

		if err := CopyRichToClipboard(htmlContent, plainContent); err != nil {
			return fmt.Errorf("error copying to clipboard: %w", err)
		}
		fmt.Println("\n✓ Report copied to clipboard!")
	}

	return nil
}

func init() {
	reportCmd.Flags().StringVar(&reportRepositoryName, "repository-name", "", "Repository name (auto-detected from git if not specified)")
	reportCmd.Flags().StringVar(&reportRepoID, "repo-id", "", "Repository ID")
	reportCmd.Flags().StringVar(&reportStatus, "status", "all", "PR status filter")
	reportCmd.Flags().StringVar(&reportTargetBranch, "target-branch", "", "Filter by target branch")
	reportCmd.Flags().StringVar(&reportSourceBranch, "source-branch", "", "Filter by source branch")
	reportCmd.Flags().StringVar(&reportCreator, "creator", "", "Filter by creator (use 'self', name, or ID)")
	reportCmd.Flags().StringVar(&reportOutput, "output", "", "Output file path (default: stdout)")
	reportCmd.Flags().BoolVar(&reportCopy, "copy", false, "Copy report to clipboard for Teams")
	reportCmd.Flags().BoolVar(&reportNoWarnings, "no-warnings", false, "Hide requirement warnings (work items, merge conflicts)")
	reportCmd.Flags().StringArrayVar(&reportWorkItems, "work-items", []string{}, "Filter by linked work items (e.g., PBI-12345, BUG-12345)")
	reportCmd.Flags().StringVar(&reportTitleRegex, "title-regex", "", "Filter PR title by regex pattern")
	reportCmd.Flags().StringVar(&reportTitleFuzzy, "title-fuzzy", "", "Filter PR title by fuzzy match")
	reportCmd.Flags().StringVar(&reportRepoRegex, "repo-regex", "", "Filter repository by regex pattern")
	reportCmd.Flags().StringVar(&reportRepoFuzzy, "repo-fuzzy", "", "Filter repository by fuzzy match")
	reportCmd.Flags().BoolVar(&reportUseGitContext, "use-git-context", true, "Use git context for auto-detection when in a git repository")
	reportCmd.Flags().BoolVar(&reportNoGitContext, "no-git-context", false, "Disable git context auto-detection")

	reportCmd.MarkFlagsMutuallyExclusive("repository-name", "repo-id")
	reportCmd.MarkFlagsMutuallyExclusive("use-git-context", "no-git-context")
}
