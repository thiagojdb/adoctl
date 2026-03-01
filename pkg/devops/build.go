package devops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"adoctl/pkg/cache"
	"adoctl/pkg/logger"
)

func (s *DevOpsService) SyncBuilds(force bool) (int, error) {
	if s.syncOptions.SkipSync {
		return 0, nil
	}

	if s.cache == nil {
		return 0, fmt.Errorf("cache not initialized")
	}

	now := time.Now()

	syncTime := s.getBuildSyncTime(force)

	params := map[string]string{
		"$top":    "1000",
		"minTime": syncTime.Format(time.DateTime),
	}

	azureBuilds, err := s.client.GetBuilds(context.Background(), params)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch builds from API: %w", err)
	}

	savedCount := 0
	for _, azureBuild := range azureBuilds {
		sourceBranch := ""
		if azureBuild.SourceBranch != nil && *azureBuild.SourceBranch != "" {
			sourceBranch = strings.Replace(*azureBuild.SourceBranch, "refs/heads/", "", 1)
		}

		pipeline := ""
		if azureBuild.Definition != nil && azureBuild.Definition.Name != nil {
			pipeline = *azureBuild.Definition.Name
		}

		startTime := time.Now()
		if azureBuild.StartTime != nil {
			if parsed, err := time.Parse(time.RFC3339, azureBuild.StartTime.String()); err == nil {
				startTime = parsed
			}
		}
		var endTime sql.NullTime
		if azureBuild.FinishTime != nil {
			if parsedTime, err := time.Parse(time.RFC3339, azureBuild.FinishTime.String()); err == nil {
				endTime = sql.NullTime{Time: parsedTime, Valid: true}
			}
		}

		buildID := 0
		if azureBuild.Id != nil {
			buildID = *azureBuild.Id
		}

		sourceVersion := ""
		if azureBuild.SourceVersion != nil {
			sourceVersion = *azureBuild.SourceVersion
		}

		status := ""
		if azureBuild.Status != nil {
			status = string(*azureBuild.Status)
		}

		result := ""
		if azureBuild.Result != nil {
			result = string(*azureBuild.Result)
		}

		jsonData, _ := json.Marshal(azureBuild)
		FullJSON := string(jsonData)
		buildData := cache.Build{
			BuildID:       buildID,
			Branch:        sourceBranch,
			Repository:    pipeline,
			SourceVersion: sourceVersion,
			StartTime:     startTime,
			EndTime:       endTime,
			Status:        status,
			Result:        result,
			FullJSON:      FullJSON,
		}

		err := s.cache.SaveBuild(buildData)
		if err != nil {
			logger.Warn().
				Err(err).
				Int("build_id", buildID).
				Msg("Failed to cache build, continuing sync")
			continue
		}

		savedCount++
	}

	err = s.cache.SetLastSyncTime("builds_sync", now)
	if err != nil {
		return 0, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return savedCount, nil
}

func (s *DevOpsService) getBuildSyncTime(force bool) *time.Time {
	lastSyncTime, _ := s.cache.GetLastSyncTime("builds_sync")

	if lastSyncTime == nil || force {
		newSyncTime := time.Now().AddDate(0, -1, 0)
		return &newSyncTime
	}

	return lastSyncTime
}

func (s *DevOpsService) SearchBuildsCached(filters map[string]any) ([]cache.Build, error) {
	if s.syncOptions.SkipSync || s.buildsSync {
		return s.cache.SearchBuilds(filters)
	}

	s.SyncBuilds(false)
	s.buildsSync = true

	return s.cache.SearchBuilds(filters)
}

func (s *DevOpsService) GetBuildsForCommit(commitHash, repoID, branch string) ([]cache.Build, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not initialized")
	}

	filters := map[string]any{
		"commit": commitHash,
	}

	if repoID != "" {
		filters["repository"] = repoID
	}

	if branch != "" {
		filters["branch"] = branch
	}

	return s.SearchBuildsCached(filters)
}

func FormatBuildStatus(status, result string) string {
	switch status {
	case "inProgress":
		return "Running"
	case "completed":
		switch result {
		case "succeeded":
			return "Succeeded"
		case "failed":
			return "Failed"
		case "partiallySucceeded":
			return "Partially Succeeded"
		case "canceled":
			return "Canceled"
		default:
			return "Completed"
		}
	case "notStarted":
		return "Queued"
	case "cancelling":
		return "Cancelling"
	case "postponed":
		return "Postponed"
	default:
		return status
	}
}
