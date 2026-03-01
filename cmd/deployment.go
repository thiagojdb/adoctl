package cmd

import "github.com/spf13/cobra"

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Deployment commands",
	Long:  `Commands for managing Azure DevOps deployments`,
	Example: `  # Sync deployments from Azure DevOps to local cache
  adoctl deployment sync

  # Force sync all deployments (ignore cache)
  adoctl deployment sync --force

  # Sync deployments for specific release
  adoctl deployment sync --release-id 123

  # Search cached deployments with filters
  adoctl deployment search --status succeeded

  # Search deployments by repository
  adoctl deployment search --repository myrepo`,
}

func init() {
	AddCommand(rootCmd, deploymentCmd)
}
