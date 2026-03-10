package cmd

import (
	"context"
	"fmt"

	"adoctl/pkg/devops"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	sprintListCurrent    bool
	sprintShowAssignedTo string
	sprintShowDebug      bool
)

var sprintListCmd = &cobra.Command{
	Use:   "list [sprint-name]",
	Short: "List sprints or show work items for a sprint",
	Long: `List all sprints/iterations, or show work items for a specific sprint.

Without arguments, lists all sprints. With a sprint name argument, shows work items
for that sprint (supports fuzzy matching).`,
	Example: `  # List all sprints
  adoctl sprint list

  # Show work items for current sprint
  adoctl sprint list --current

  # Show work items for a specific sprint (fuzzy match)
  adoctl sprint list "Reforma Tributária"

  # Show work items with filtering
  adoctl sprint list "Sprint 1" --assigned-to me`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		// If --current flag is set, find current sprint and show its work items
		if sprintListCurrent {
			return showCurrentSprintWorkItems(ctx, svc, cmd)
		}

		// If no args, list all sprints
		if len(args) == 0 {
			return listAllSprints(ctx, svc, cmd)
		}

		// Otherwise, show work items for the specified sprint
		return showSprintWorkItems(ctx, svc, cmd, args[0])
	},
}

func listAllSprints(ctx context.Context, svc *devops.DevOpsService, cmd *cobra.Command) error {
	iterations, err := svc.ListIterations(ctx, false)
	if err != nil {
		return fmt.Errorf("error listing sprints: %w", err)
	}

	return formatAndOutputIterations(cmd, iterations)
}

func showCurrentSprintWorkItems(ctx context.Context, svc *devops.DevOpsService, cmd *cobra.Command) error {
	iterations, err := svc.ListIterations(ctx, true)
	if err != nil {
		return fmt.Errorf("error finding current sprint: %w", err)
	}

	if len(iterations) == 0 {
		return fmt.Errorf("no current sprint found")
	}

	if len(iterations) > 1 {
		fmt.Printf("Warning: Multiple current sprints found, using: %s\n\n", iterations[0].Name)
	}

	return showSprintWorkItems(ctx, svc, cmd, iterations[0].Path)
}

func showSprintWorkItems(ctx context.Context, svc *devops.DevOpsService, cmd *cobra.Command, sprintPath string) error {
	// Try to resolve the iteration by name first (fuzzy search)
	iteration, warning, err := svc.FindIterationByName(ctx, sprintPath)
	if err == nil {
		if warning != "" {
			fmt.Printf("Warning: %s\n\n", warning)
		}
		sprintPath = iteration.Path
		if sprintShowDebug {
			fmt.Printf("Debug: Found iteration '%s' at path '%s'\n", iteration.Name, iteration.Path)
		}
	} else {
		if sprintShowDebug {
			fmt.Printf("Debug: Could not find iteration matching '%s': %v\n", sprintPath, err)
		}
	}

	var workItems []models.WorkItemHierarchy
	if sprintShowAssignedTo != "" {
		userIdentifier, err := svc.ResolveUserIdentifier(sprintShowAssignedTo)
		if err != nil {
			return err
		}
		if sprintShowDebug {
			fmt.Printf("Debug: Resolved '%s' to user: '%s'\n", sprintShowAssignedTo, userIdentifier)
		}
		workItems, err = svc.GetIterationWorkItemsForUser(ctx, sprintPath, userIdentifier)
		if err != nil {
			return fmt.Errorf("error getting sprint work items: %w", err)
		}
		if sprintShowDebug {
			fmt.Printf("Debug: Found %d work items for user\n", len(workItems))
		}
	} else {
		workItems, err = svc.GetIterationWorkItems(ctx, sprintPath)
		if err != nil {
			return fmt.Errorf("error getting sprint work items: %w", err)
		}
	}

	return formatAndOutputWorkItems(cmd, workItems, sprintShowAssignedTo)
}

func init() {
	sprintListCmd.Flags().BoolVarP(&sprintListCurrent, "current", "c", false, "Show work items for current sprint")
	sprintListCmd.Flags().StringVar(&sprintShowAssignedTo, "assigned-to", "", "Filter by assignee (use 'me' for current user)")
	sprintListCmd.Flags().BoolVar(&sprintShowDebug, "debug", false, "Show debug output")
}
