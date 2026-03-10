package cmd

import "github.com/spf13/cobra"

var sprintCmd = &cobra.Command{
	Use:     "sprint",
	Aliases: []string{"s", "sprints", "iteration", "iter"},
	Short:   "Sprint/iteration commands",
	Long:    `Commands for managing Azure DevOps sprints and iterations`,
}

func init() {
}
