package cmd

import "github.com/spf13/cobra"

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build commands",
	Long:  `Commands for managing Azure DevOps builds`,
	Example: `  # Sync builds from Azure DevOps to local cache
  adoctl build sync

  # Force sync all builds (ignore cache)
  adoctl build sync --force

  # Search cached builds with filters
  adoctl build search --branch main --status completed

  # Search builds and output as JSON
  adoctl build search --status failed --json`,
}

func init() {
	AddCommand(rootCmd, buildCmd)
}
