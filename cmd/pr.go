package cmd

import "github.com/spf13/cobra"

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Pull request commands",
	Long:  `Commands for managing Azure DevOps pull requests`,
}

func init() {
}
