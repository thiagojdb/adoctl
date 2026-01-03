package cmd

import (
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var syncBuildsForce bool

var syncBuildsCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync builds from Azure DevOps to local cache",
	Long:  `Sync builds from Azure DevOps to local cache for faster searching.`,
	Example: `  # Sync builds to local cache
  adoctl build sync

  # Force sync all builds (ignore cache)
  adoctl build sync --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		count, err := svc.SyncBuilds(syncBuildsForce)
		if err != nil {
			return fmt.Errorf("error syncing builds: %w", err)
		}
		logger.Info().Int("count", count).Msg("Successfully synced builds")

		return nil
	},
}

func init() {
	syncBuildsCmd.Flags().BoolVar(&syncBuildsForce, "force", false, "Force sync all builds (ignore cache)")
}
