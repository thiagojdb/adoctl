package devops

import (
	"context"
	"fmt"
	"strings"
)

func (s *DevOpsService) LinkWorkItemsToPullRequest(repositoryID string, pullRequestID int, workItemIDs []string) error {
	pr, err := s.client.GetPullRequest(context.Background(), pullRequestID)
	if err != nil {
		return fmt.Errorf("failed to get PR: %w", err)
	}

	if pr.Repository != nil && pr.Repository.Id != nil && pr.Repository.Id.String() != repositoryID {
		return fmt.Errorf("PR #%d does not belong to repository %s", pullRequestID, repositoryID)
	}

	if pr.Repository == nil || pr.Repository.Project == nil || pr.Repository.Project.Id == nil {
		return fmt.Errorf("PR #%d has no project information", pullRequestID)
	}

	projectID := pr.Repository.Project.Id.String()
	artifactURL := fmt.Sprintf("vstfs:///Git/PullRequestId/%s%%2F%s%%2F%d", projectID, repositoryID, pullRequestID)

	for _, wiIDStr := range workItemIDs {
		wiID := 0
		fmt.Sscanf(wiIDStr, "%d", &wiID)

		relation := map[string]any{
			"rel": "ArtifactLink",
			"url": artifactURL,
			"attributes": map[string]any{
				"name": "Pull Request",
			},
		}

		err := s.client.AddWorkItemRelation(context.Background(), wiID, relation)
		if err != nil {
			if strings.Contains(err.Error(), "Relation already exists") {
				continue
			}
			return fmt.Errorf("failed to link work item %s to PR: %w", wiIDStr, err)
		}
	}

	return nil
}
