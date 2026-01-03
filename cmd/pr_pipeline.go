package cmd

import (
	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:     "pipeline",
	Aliases: []string{"pipe", "status"},
	Short:   "Get PR status and CI/CD pipeline status",
	Long:    `Get pull request status (including approvals), CI/CD pipeline status (builds and deployments) for one or more pull requests.`,
	Example: `  # Get pipeline status for a single PR
  adoctl pr pipeline --pr 123

  # Get pipeline status for multiple PRs
  adoctl pr pipeline --pr 123 --pr 456

  # Get status in modern table format
  adoctl pr pipeline --pr 123 --format modern

  # Watch PR pipeline status with refresh
  adoctl pr pipeline --pr 123 --watch

  # Read PR numbers from file
  adoctl pr pipeline --file pr-list.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delegate to pipelineStatusCmd
		return pipelineStatusCmd.RunE(cmd, args)
	},
}

func init() {
	// Copy flags from pipelineStatusCmd
	pipelineCmd.Flags().IntSliceVar(&pipelineStatusPRs, "pr", []int{}, "PR number(s) (can specify multiple)")
	pipelineCmd.Flags().StringVar(&pipelineStatusFile, "file", "", "Read PR numbers from a file")
	pipelineCmd.Flags().BoolVar(&pipelineStatusQuiet, "quiet", false, "Suppress sync messages")
	pipelineCmd.Flags().StringVar(&pipelineStatusFormat, "format", "detailed", "Output format: 'detailed' or 'modern'")
	pipelineCmd.Flags().BoolVar(&pipelineStatusCachedOnly, "cached-only", false, "Use cached data only, skip syncing")
	pipelineCmd.Flags().BoolVar(&pipelineStatusWatch, "watch", false, "Watch mode: continuously refresh PR status")
	pipelineCmd.Flags().IntVar(&pipelineStatusInterval, "interval", 30, "Refresh interval in seconds (default: 30)")
	pipelineCmd.MarkFlagsOneRequired("pr", "file")
}
