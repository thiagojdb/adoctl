package cmd

import (
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	syncDeploymentsForce     bool
	syncDeploymentsReleaseID int
)

var syncDeploymentsCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync deployments from Azure DevOps to local cache",
	Long:  `Sync deployments from Azure DevOps to local cache for faster searching.`,
	Example: `  # Sync deployments to local cache
  adoctl deployment sync

  # Force sync all deployments (ignore cache)
  adoctl deployment sync --force

  # Sync deployments for specific release
  adoctl deployment sync --release-id 123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("error creating service: %w", err)
		}
		defer svc.Close()

		var count int
		if syncDeploymentsReleaseID != 0 {
			count, err = svc.SyncDeploymentsWithReleaseID(syncDeploymentsForce, syncDeploymentsReleaseID)
		} else {
			count, err = svc.SyncDeployments(syncDeploymentsForce)
		}
		if err != nil {
			return fmt.Errorf("error syncing deployments: %w", err)
		}

		logger.Info().Int("count", count).Msg("Successfully synced deployments")

		return nil
	},
}

func init() {
	syncDeploymentsCmd.Flags().BoolVar(&syncDeploymentsForce, "force", false, "Force sync all deployments (ignore cache)")
	syncDeploymentsCmd.Flags().IntVar(&syncDeploymentsReleaseID, "release-id", 0, "Sync deployments for specific release ID")
}
