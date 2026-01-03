package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"adoctl/pkg/config"
	"adoctl/pkg/devops"
	"adoctl/pkg/logger"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	bulkCreateSourceBranch   string
	bulkCreateTargetBranches []string
	bulkCreateTitle          string
	bulkCreateDescription    string
	bulkCreateWorkItemIDs    []string
)

var bulkCreateCmd = &cobra.Command{
	Use:   "bulk-create",
	Short: "Bulk create pull requests across all repos",
	Long:  `Creates PRs across all repositories that have the specified source branch.`,
	Example: `  # Create PRs from feature branch to main in all repos
  adoctl pr bulk-create --source-branch feature --target-branch main --title "Merge feature"

  # Create PRs to multiple target branches
  adoctl pr bulk-create --source-branch develop --target-branch main --target-branch release \
    --title "Release merge"

  # Create PRs with work items linked
  adoctl pr bulk-create --source-branch feature --target-branch main \
    --title "Feature merge" --work-item-id PBI-123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}

		repos, err := svc.ListRepositories()
		if err != nil {
			return fmt.Errorf("error listing repositories: %w", err)
		}

		fmt.Printf("Searching for repos with branch '%s' and creating PRs to %v...\n\n", bulkCreateSourceBranch, bulkCreateTargetBranches)

		results := bulkCreatePRs(ctx, svc, repos)

		printBulkCreateResults(results, repos)

		return nil
	},
}

func bulkCreatePRs(ctx context.Context, svc *devops.DevOpsService, repos []models.Repository) []devops.BulkCreateResult {
	allResults := []devops.BulkCreateResult{}
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.DefaultParallelProcesses)

	for _, targetBranch := range bulkCreateTargetBranches {
		wg.Add(1)
		go func(tb string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			results, err := svc.BulkCreatePullRequests(ctx, bulkCreateSourceBranch, tb, bulkCreateTitle, bulkCreateDescription, bulkCreateWorkItemIDs)
			if err != nil {
				logger.Error().Str("target_branch", tb).Err(err).Msg("Failed to bulk create PRs")
				return
			}

			resultsMutex.Lock()
			allResults = append(allResults, results...)
			resultsMutex.Unlock()
		}(targetBranch)
	}

	wg.Wait()
	return allResults
}

func printBulkCreateResults(results []devops.BulkCreateResult, repos []models.Repository) {
	successCount := 0
	failureCount := 0

	fmt.Println("=== Bulk PR Creation Report ===")

	for _, result := range results {
		if result.Success {
			fmt.Printf("✓ %s: PR #%d created\n", result.RepoName, result.PRID)
			fmt.Printf("  URL: %s\n", result.URL)
			successCount++
		} else {
			fmt.Printf("✗ %s: Failed - %s\n", result.RepoName, result.Error)
			failureCount++
		}
		fmt.Println()
	}

	reposWithSourceBranch := 0
	for _, result := range results {
		if result.Success || (result.Error != "" && !strings.Contains(result.Error, "without source branch")) {
			reposWithSourceBranch++
		}
	}
	skippedCount := len(repos) - reposWithSourceBranch

	fmt.Println("=== Summary ===")
	fmt.Printf("Total repos checked: %d\n", len(repos))
	logger.Info().Int("count", successCount).Msg("Successfully created PRs")
	logger.Error().Int("count", failureCount).Msg("Failed to create PRs")
	fmt.Printf("Repos without source branch: %d\n", skippedCount)
}

func init() {
	bulkCreateCmd.Flags().StringVar(&bulkCreateSourceBranch, "source-branch", "", "Source branch name")
	bulkCreateCmd.Flags().StringArrayVar(&bulkCreateTargetBranches, "target-branch", []string{}, "Target branch name (can be specified multiple times)")
	bulkCreateCmd.Flags().StringVar(&bulkCreateTitle, "title", "", "PR title")
	bulkCreateCmd.Flags().StringVar(&bulkCreateDescription, "description", "", "PR description")
	bulkCreateCmd.Flags().StringArrayVar(&bulkCreateWorkItemIDs, "work-item-id", []string{}, "Work item IDs to link (can be specified multiple times)")

	bulkCreateCmd.MarkFlagRequired("source-branch")
	bulkCreateCmd.MarkFlagRequired("title")
}
