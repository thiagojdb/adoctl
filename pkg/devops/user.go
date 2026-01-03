package devops

import (
	"context"
	"fmt"
	"strings"
)

func (s *DevOpsService) GetCurrentUserID() (string, error) {
	prs, err := s.ListPullRequests(context.Background(), "", "all", "", "", "")
	if err != nil {
		return "", err
	}

	if len(prs) == 0 {
		return "", fmt.Errorf("no PRs found. Cannot determine current user ID.")
	}

	type creatorInfo struct {
		name       string
		count      int
		latestDate string
	}
	creators := map[string]*creatorInfo{}

	for _, pr := range prs {
		creatorID := pr.CreatedBy.ID
		displayName := pr.CreatedBy.DisplayName

		if _, ok := creators[creatorID]; !ok {
			creators[creatorID] = &creatorInfo{
				name:       displayName,
				count:      0,
				latestDate: "",
			}
		}
		creators[creatorID].count++
	}

	var maxCreator string
	maxCount := 0

	for id, creator := range creators {
		if creator.count > maxCount {
			maxCount = creator.count
			maxCreator = id
		}
	}

	return maxCreator, nil
}

func (s *DevOpsService) GetAllCreators() map[string]map[string]any {
	if s.cache != nil {
		cachedUsers, err := s.cache.GetUsers()
		if err == nil && cachedUsers != nil {
			creators := map[string]map[string]any{}
			for id, user := range cachedUsers {
				creators[id] = map[string]any{
					"id":    id,
					"name":  user.Name,
					"count": 0,
				}
			}
			return creators
		}
	}

	prs, _ := s.ListPullRequests(context.Background(), "", "all", "", "", "")
	creators := map[string]map[string]any{}

	for _, pr := range prs {
		creatorID := pr.CreatedBy.ID
		displayName := pr.CreatedBy.DisplayName

		if _, ok := creators[creatorID]; !ok {
			creators[creatorID] = map[string]any{
				"id":    creatorID,
				"name":  displayName,
				"count": 0,
			}
		}
		creators[creatorID]["count"] = creators[creatorID]["count"].(int) + 1
	}

	if s.cache != nil {
		usersMap := map[string]map[string]any{}
		for id, creator := range creators {
			usersMap[id] = map[string]any{
				"name": creator["name"],
			}
		}
		s.cache.SetUsers(usersMap)
	}

	return creators
}

func (s *DevOpsService) ResolveCreator(creatorInput string) (string, error) {
	if strings.ToLower(creatorInput) == "self" {
		creatorID, err := s.GetCurrentUserID()
		if err != nil {
			return "", fmt.Errorf("error getting current user ID: %v", err)
		}
		return creatorID, nil
	}

	creators := s.GetAllCreators()

	matchingCreators := []map[string]any{}
	for _, creator := range creators {
		name := creator["name"].(string)
		if strings.Contains(strings.ToLower(name), strings.ToLower(creatorInput)) {
			matchingCreators = append(matchingCreators, creator)
		}
	}

	if len(matchingCreators) == 0 {
		return "", fmt.Errorf("no creators found matching '%s'", creatorInput)
	}

	if len(matchingCreators) > 1 {
		return "", fmt.Errorf("multiple creators found")
	}

	return matchingCreators[0]["id"].(string), nil
}
