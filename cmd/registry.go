package cmd

import "github.com/spf13/cobra"

func RegisterCommands(root *cobra.Command) {
	root.AddCommand(versionCmd)
	root.AddCommand(clipboardServeCmd)

	root.AddCommand(prCmd)
	root.AddCommand(buildCmd)
	root.AddCommand(deploymentCmd)
	root.AddCommand(reposCmd)
	root.AddCommand(workItemCmd)
	root.AddCommand(reportCmd)
	root.AddCommand(configCmd)
	root.AddCommand(hooksCmd)

	prCmd.AddCommand(
		createCmd,
		bulkCreateCmd,
		listCmd,
		statusCmd,
		linkWorkItemsCmd,
		mergeCmd,
		abandonCmd,
		approveCmd,
		pipelineCmd,
	)

	buildCmd.AddCommand(
		syncBuildsCmd,
		searchBuildsCmd,
	)

	deploymentCmd.AddCommand(
		syncDeploymentsCmd,
		searchDeploymentsCmd,
	)
}
