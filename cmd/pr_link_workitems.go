package cmd

import (
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/logger"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	linkPRIDs       []string
	linkWorkItemIDs []string
)

var linkWorkItemsCmd = &cobra.Command{
	Use:   "link-workitems",
	Short: "Link work items to pull requests",
	Long:  `Link multiple work items to one or more existing pull requests.`,
	Example: `  # Link a single work item to a PR
  adoctl pr link-workitems --pr-id 123 --work-item-id 456

  # Link multiple work items to a single PR
  adoctl pr link-workitems --pr-id 123 --work-item-id 456 --work-item-id 789

  # Link work items to multiple PRs
  adoctl pr link-workitems --pr-id 123 --pr-id 456 --work-item-id 789`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(linkPRIDs) == 0 {
			return fmt.Errorf("at least one PR ID must be specified")
		}
		if len(linkWorkItemIDs) == 0 {
			return fmt.Errorf("at least one work item ID must be specified")
		}

		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		prs, err := svc.ListPullRequests(ctx, "", "all", "", "", "")
		if err != nil {
			return fmt.Errorf("failed to list PRs: %w", err)
		}

		prRepoMap := buildPRRepoMap(prs)

		successCount, failureCount := linkWorkItemsToPRs(svc, prRepoMap)

		printLinkWorkItemsSummary(successCount, failureCount)

		return nil
	},
}

func buildPRRepoMap(prs []models.PullRequest) map[int]string {
	prRepoMap := map[int]string{}
	for _, pr := range prs {
		prRepoMap[pr.ID] = pr.Repository.ID
	}
	return prRepoMap
}

func linkWorkItemsToPRs(svc *devops.DevOpsService, prRepoMap map[int]string) (int, int) {
	successCount := 0
	failureCount := 0

	fmt.Printf("Linking work items %v to PRs %v...\n\n", linkWorkItemIDs, linkPRIDs)

	for _, prIDStr := range linkPRIDs {
		prID := 0
		if _, err := fmt.Sscanf(prIDStr, "%d", &prID); err != nil {
			logger.Error().Str("pr_id_str", prIDStr).Err(err).Msg("Failed to parse PR ID")
			failureCount++
			continue
		}

		repoID, exists := prRepoMap[prID]
		if !exists {
			logger.Error().Int("pr_id", prID).Msg("PR not found")
			failureCount++
			continue
		}

		err := svc.LinkWorkItemsToPullRequest(repoID, prID, linkWorkItemIDs)
		if err != nil {
			logger.Error().Int("pr_id", prID).Err(err).Msg("Failed to link work items")
			failureCount++
			continue
		}

		fmt.Printf("✓ PR #%d: Successfully linked work items %v\n", prID, linkWorkItemIDs)
		successCount++
	}

	return successCount, failureCount
}

func printLinkWorkItemsSummary(successCount, failureCount int) {
	fmt.Println()
	fmt.Println("=== Summary ===")
	logger.Info().Int("count", successCount).Msg("Successfully linked work items")
	logger.Error().Int("count", failureCount).Msg("Failed to link work items")
}

func init() {
	linkWorkItemsCmd.Flags().StringArrayVar(&linkPRIDs, "pr-id", []string{}, "Pull request IDs to link (can be specified multiple times)")
	linkWorkItemsCmd.Flags().StringArrayVar(&linkWorkItemIDs, "work-item-id", []string{}, "Work item IDs to link (can be specified multiple times)")
}
