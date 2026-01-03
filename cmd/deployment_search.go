package cmd

import (
	"fmt"
	"strings"
	"time"

	"adoctl/pkg/cache"
	"adoctl/pkg/devops"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	searchDeploymentsID               int
	searchDeploymentsReleaseName      string
	searchDeploymentsStatus           string
	searchDeploymentsRepository       string
	searchDeploymentsBranch           string
	searchDeploymentsStartTimeFrom    string
	searchDeploymentsStartTimeTo      string
	searchDeploymentsEndTimeFrom      string
	searchDeploymentsEndTimeTo        string
	searchDeploymentsArtifactDateFrom string
	searchDeploymentsArtifactDateTo   string
	searchDeploymentsHasEndTime       string
	searchDeploymentsLimit            int
)

var searchDeploymentsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search deployments in local cache with filters",
	Long:  `Search deployments in local cache with various filters.`,
	Example: `  # Search deployments by status
  adoctl deployment search --status succeeded

  # Search deployments by repository
  adoctl deployment search --repository myrepo

  # Search deployments by branch
  adoctl deployment search --branch main

  # Search with time filters
  adoctl deployment search --start-time-from 2024-01-01T00:00:00Z

  # Limit results
  adoctl deployment search --limit 10

  # Filter by release name
  adoctl deployment search --release-name "Release 1.0"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		syncDeploymentsCmd.RunE(cmd, args)

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("error creating service: %w", err)
		}
		defer svc.Close()

		filters := buildDeploymentSearchFilters()

		deployments, err := svc.SearchDeploymentsCached(filters)
		if err != nil {
			return fmt.Errorf("error searching deployments: %w", err)
		}

		return outputDeployments(deployments, ShouldCopyOutput(cmd))
	},
}

func buildDeploymentSearchFilters() map[string]any {
	filters := make(map[string]any)

	if searchDeploymentsID != 0 {
		filters["release_id"] = searchDeploymentsID
	}
	if searchDeploymentsReleaseName != "" {
		filters["release_name"] = searchDeploymentsReleaseName
	}
	if searchDeploymentsStatus != "" {
		filters["status"] = searchDeploymentsStatus
	}
	if searchDeploymentsRepository != "" {
		filters["repository"] = searchDeploymentsRepository
	}
	if searchDeploymentsBranch != "" {
		filters["branch"] = searchDeploymentsBranch
	}
	if searchDeploymentsStartTimeFrom != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsStartTimeFrom); err == nil {
			filters["start_time_from"] = t
		}
	}
	if searchDeploymentsStartTimeTo != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsStartTimeTo); err == nil {
			filters["start_time_to"] = t
		}
	}
	if searchDeploymentsEndTimeFrom != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsEndTimeFrom); err == nil {
			filters["end_time_from"] = t
		}
	}
	if searchDeploymentsEndTimeTo != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsEndTimeTo); err == nil {
			filters["end_time_to"] = t
		}
	}
	if searchDeploymentsArtifactDateFrom != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsArtifactDateFrom); err == nil {
			filters["artifact_date_from"] = t
		}
	}
	if searchDeploymentsArtifactDateTo != "" {
		if t, err := time.Parse(time.RFC3339, searchDeploymentsArtifactDateTo); err == nil {
			filters["artifact_date_to"] = t
		}
	}
	if searchDeploymentsHasEndTime != "" {
		switch searchDeploymentsHasEndTime {
		case "true":
			filters["has_end_time"] = true
		case "false":
			filters["has_end_time"] = false
		}
	}
	if searchDeploymentsLimit != 0 {
		filters["limit"] = searchDeploymentsLimit
	}

	return filters
}

func outputDeployments(deployments []cache.Deployment, shouldCopy bool) error {
	logger.Info().Int("count", len(deployments)).Msg("Found deployments")

	var markdownBuilder strings.Builder

	if shouldCopy {
		markdownBuilder.WriteString("**Deployments**\n\n")
	}

	fmt.Println()
	for _, deployment := range deployments {
		// Build output
		fmt.Printf("Release ID: %d\n", deployment.ReleaseID)
		fmt.Printf("  Release Name: %s\n", deployment.ReleaseName)
		fmt.Printf("  Status: %s\n", deployment.Status)
		fmt.Printf("  Start Time: %s\n", deployment.StartTime.Format("2006-01-02 15:04:05"))
		if deployment.EndTime.Valid {
			fmt.Printf("  End Time: %s\n", deployment.EndTime.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("  End Time: (not completed)")
		}
		if deployment.Repository != "" {
			fmt.Printf("  Repository: %s\n", deployment.Repository)
		}
		if deployment.Branch != "" {
			fmt.Printf("  Branch: %s\n", deployment.Branch)
		}
		if deployment.SourceVersion != "" {
			fmt.Printf("  Source Version: %s\n", deployment.SourceVersion)
		}
		if deployment.BuildID != 0 {
			fmt.Printf("  Build ID: %d\n", deployment.BuildID)
		}
		fmt.Println()

		// Build Markdown for clipboard
		if shouldCopy {
			fmt.Fprintf(&markdownBuilder, "- **Release #%d**: %s - Status: %s\n",
				deployment.ReleaseID, deployment.ReleaseName, deployment.Status)
			if deployment.Repository != "" {
				fmt.Fprintf(&markdownBuilder, "  - Repository: %s\n", deployment.Repository)
			}
			if deployment.Branch != "" {
				fmt.Fprintf(&markdownBuilder, "  - Branch: `%s`\n", deployment.Branch)
			}
			markdownBuilder.WriteString("\n")
		}
	}

	if shouldCopy {
		if err := CopyToClipboard(markdownBuilder.String()); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("✓ Copied to clipboard!")
	}

	return nil
}

func init() {
	searchDeploymentsCmd.Flags().IntVar(&searchDeploymentsID, "release-id", 0, "Filter by release ID")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsReleaseName, "release-name", "", "Filter by release name")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsStatus, "status", "", "Filter by deployment status")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsRepository, "repository", "", "Filter by repository/pipeline name")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsBranch, "branch", "", "Filter by branch name")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsStartTimeFrom, "start-time-from", "", "Filter deployments starting after this time (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsStartTimeTo, "start-time-to", "", "Filter deployments starting before this time (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsEndTimeFrom, "end-time-from", "", "Filter deployments ending after this time (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsEndTimeTo, "end-time-to", "", "Filter deployments ending before this time (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsArtifactDateFrom, "artifact-date-from", "", "Filter by artifact date starting from (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsArtifactDateTo, "artifact-date-to", "", "Filter by artifact date ending to (RFC3339 format)")
	searchDeploymentsCmd.Flags().StringVar(&searchDeploymentsHasEndTime, "has-end-time", "", "Filter by end time existence (true/false)")
	searchDeploymentsCmd.Flags().IntVar(&searchDeploymentsLimit, "limit", 0, "Limit number of results")
}
