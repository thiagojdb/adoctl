package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"adoctl/pkg/cache"
	"adoctl/pkg/devops"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	searchBuildsID            int
	searchBuildsBranch        string
	searchBuildsRepository    string
	searchBuildsCommit        string
	searchBuildsStatus        string
	searchBuildsStartTimeFrom string
	searchBuildsStartTimeTo   string
	searchBuildsEndTimeFrom   string
	searchBuildsEndTimeTo     string
	searchBuildsHasEndTime    string
	searchBuildsLimit         int
	searchBuildsOutput        string
	searchBuildsJSON          bool
)

var searchBuildsCmd = &cobra.Command{
	Use:   "search",
	Short: "Search builds in local cache with filters",
	Long:  `Search builds in local cache with various filters.`,
	Example: `  # Search builds by branch
  adoctl build search --branch main

  # Search builds by status
  adoctl build search --status completed

  # Search builds and output as JSON
  adoctl build search --status failed --json

  # Search with time filters
  adoctl build search --start-time-from 2024-01-01T00:00:00Z --start-time-to 2024-01-31T23:59:59Z

  # Limit results
  adoctl build search --limit 10

  # Save results to file
  adoctl build search --status completed --output builds.json --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := syncBuildsCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("failed to sync builds: %w", err)
		}

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		filters := buildSearchFilters()

		builds, err := svc.SearchBuildsCached(filters)
		if err != nil {
			return fmt.Errorf("error searching builds: %w", err)
		}

		if searchBuildsJSON {
			return outputBuildsAsJSON(builds, ShouldCopyOutput(cmd))
		}

		return outputBuilds(builds, ShouldCopyOutput(cmd))
	},
}

func buildSearchFilters() map[string]any {
	filters := make(map[string]any)

	addStringFilter(filters, "branch", searchBuildsBranch)
	addStringFilter(filters, "repository", searchBuildsRepository)
	addStringFilter(filters, "commit", searchBuildsCommit)
	addStringFilter(filters, "status", searchBuildsStatus)
	addTimeFilters(filters)
	addEndTimeFilter(filters)
	addIntFilter(filters, "build_id", searchBuildsID)
	addIntFilter(filters, "limit", searchBuildsLimit)

	return filters
}

func addStringFilter(filters map[string]any, key, value string) {
	if value != "" {
		filters[key] = value
	}
}

func addIntFilter(filters map[string]any, key string, value int) {
	if value != 0 {
		filters[key] = value
	}
}

func addTimeFilters(filters map[string]any) {
	parseAndAddTime(filters, "start_time_from", searchBuildsStartTimeFrom)
	parseAndAddTime(filters, "start_time_to", searchBuildsStartTimeTo)
	parseAndAddTime(filters, "end_time_from", searchBuildsEndTimeFrom)
	parseAndAddTime(filters, "end_time_to", searchBuildsEndTimeTo)
}

func parseAndAddTime(filters map[string]any, key, value string) {
	if value == "" {
		return
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		filters[key] = t
	}
}

func addEndTimeFilter(filters map[string]any) {
	switch searchBuildsHasEndTime {
	case "true":
		filters["has_end_time"] = true
	case "false":
		filters["has_end_time"] = false
	}
}

func outputBuildsAsJSON(builds []cache.Build, shouldCopy bool) error {
	result := []map[string]any{}
	for _, build := range builds {
		buildMap := map[string]any{
			"build_id":       build.BuildID,
			"branch":         build.Branch,
			"repository":     build.Repository,
			"source_version": build.SourceVersion,
			"start_time":     build.StartTime,
			"end_time":       nil,
			"status":         build.Status,
			"updated_at":     build.UpdatedAt,
		}
		if build.EndTime.Valid {
			buildMap["end_time"] = build.EndTime.Time
		}

		var fullData map[string]any
		if err := json.Unmarshal([]byte(build.FullJSON), &fullData); err == nil {
			buildMap["full_data"] = fullData
		}

		result = append(result, buildMap)
	}

	jsonOutputData, _ := json.MarshalIndent(result, "", "  ")
	if searchBuildsOutput != "" {
		err := os.WriteFile(searchBuildsOutput, jsonOutputData, 0600)
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		logger.Info().Str("file", searchBuildsOutput).Msg("Results saved")
	}

	// Always print to terminal
	fmt.Println(string(jsonOutputData))

	// Copy to clipboard if requested (as plain text)
	if shouldCopy {
		if err := CopyToClipboard(string(jsonOutputData)); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("✓ Copied to clipboard!")
	}

	return nil
}

func outputBuilds(builds []cache.Build, shouldCopy bool) error {
	logger.Info().Int("count", len(builds)).Msg("Found builds")

	var markdownBuilder strings.Builder

	if shouldCopy {
		markdownBuilder.WriteString("**Builds**\n\n")
	}

	fmt.Println()
	for _, build := range builds {
		// Build output
		fmt.Printf("Build ID: %d\n", build.BuildID)
		fmt.Printf("  Branch: %s\n", build.Branch)
		fmt.Printf("  Repository: %s\n", build.Repository)
		fmt.Printf("  Commit: %s\n", build.SourceVersion)
		fmt.Printf("  Start Time: %s\n", build.StartTime.Format("2006-01-02 15:04:05"))
		if build.EndTime.Valid {
			fmt.Printf("  End Time: %s\n", build.EndTime.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("  End Time: (not completed)")
		}
		fmt.Printf("  Status: %s\n", build.Status)
		fmt.Printf("  Updated At: %s\n", build.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()

		// Build Markdown for clipboard
		if shouldCopy {
			fmt.Fprintf(&markdownBuilder, "- **Build #%d** - Branch: `%s`, Repository: %s, Status: %s\n",
				build.BuildID, build.Branch, build.Repository, build.Status)
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
	searchBuildsCmd.Flags().IntVar(&searchBuildsID, "build-id", 0, "Filter by build ID")
	searchBuildsCmd.Flags().StringVar(&searchBuildsBranch, "branch", "", "Filter by branch name")
	searchBuildsCmd.Flags().StringVar(&searchBuildsRepository, "repository", "", "Filter by repository/pipeline name")
	searchBuildsCmd.Flags().StringVar(&searchBuildsCommit, "commit", "", "Filter by commit/build number")
	searchBuildsCmd.Flags().StringVar(&searchBuildsStatus, "status", "", "Filter by build status")
	searchBuildsCmd.Flags().StringVar(&searchBuildsStartTimeFrom, "start-time-from", "", "Filter builds starting after this time (RFC3339 format)")
	searchBuildsCmd.Flags().StringVar(&searchBuildsStartTimeTo, "start-time-to", "", "Filter builds starting before this time (RFC3339 format)")
	searchBuildsCmd.Flags().StringVar(&searchBuildsEndTimeFrom, "end-time-from", "", "Filter builds ending after this time (RFC3339 format)")
	searchBuildsCmd.Flags().StringVar(&searchBuildsEndTimeTo, "end-time-to", "", "Filter builds ending before this time (RFC3339 format)")
	searchBuildsCmd.Flags().StringVar(&searchBuildsHasEndTime, "has-end-time", "", "Filter by end time existence (true/false)")
	searchBuildsCmd.Flags().IntVar(&searchBuildsLimit, "limit", 0, "Limit number of results")
	searchBuildsCmd.Flags().StringVar(&searchBuildsOutput, "output", "", "Output file path (default: stdout)")
	searchBuildsCmd.Flags().BoolVar(&searchBuildsJSON, "json", false, "Output in JSON format")
}
