package devops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"adoctl/pkg/cache"
	"adoctl/pkg/logger"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/release"
)

func (s *DevOpsService) SyncDeployments(force bool) (int, error) {
	if s.syncOptions.SkipSync {
		return 0, nil
	}

	if s.cache == nil {
		return 0, fmt.Errorf("cache not initialized")
	}

	lastSyncTime, err := s.cache.GetLastSyncTime("deployments_sync")
	if err != nil {
		return 0, fmt.Errorf("failed to get last sync time: %w", err)
	}
	if force || lastSyncTime == nil {
		newSyncTime := time.Now().AddDate(0, -1, 0)
		lastSyncTime = &newSyncTime
	}

	params := map[string]string{
		"$top":           "1000",
		"minStartedTime": lastSyncTime.Format(time.DateTime),
	}

	azureDeployments, err := s.client.GetDeployments(context.Background(), params)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch deployments from API: %w", err)
	}

	savedCount := 0
	for _, azureDeployment := range azureDeployments {
		err := s.SyncDeployment(azureDeployment)
		if err != nil {
			continue
		}
		savedCount++
	}

	now := time.Now()
	err = s.cache.SetLastSyncTime("deployments_sync", now)
	if err != nil {
		return 0, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return savedCount, nil
}

func (s *DevOpsService) SyncDeployment(azureDeployment release.Deployment) error {
	releaseId := 0
	releaseName := ""
	repository := ""
	branch := ""
	sourceVersion := ""
	buildId := 0

	if azureDeployment.Release != nil {
		if azureDeployment.Release.Id != nil {
			releaseId = int(*azureDeployment.Release.Id)
		}
		if azureDeployment.Release.Name != nil {
			releaseName = *azureDeployment.Release.Name
		}

		if azureDeployment.Release.Artifacts != nil {
			for _, artifact := range *azureDeployment.Release.Artifacts {
				if artifact.DefinitionReference != nil {
					defRef := *artifact.DefinitionReference
					for key, value := range defRef {
						if strings.EqualFold(key, "repository") {
							if value.Name != nil {
								repository = *value.Name
							}
						}
						if strings.EqualFold(key, "branches") {
							if value.Name != nil && len(*value.Name) > 0 {
								branch = strings.Split(*value.Name, ",")[0]
							}
						}
						if strings.EqualFold(key, "sourceVersion") {
							if value.Id != nil {
								sourceVersion = *value.Id
							}
						}
						if strings.EqualFold(key, "version") {
							if value.Name != nil {
								buildId, _ = strconv.Atoi(*value.Name)
							}
						}
					}
				}
			}
		}
	}

	startTime := time.Now()
	var endTime sql.NullTime
	if azureDeployment.StartedOn != nil {
		startTime = azureDeployment.StartedOn.Time
	}
	if azureDeployment.CompletedOn != nil {
		parsedTime := azureDeployment.CompletedOn.Time
		endTime = sql.NullTime{Time: parsedTime, Valid: true}
	}

	status := ""
	if azureDeployment.DeploymentStatus != nil {
		status = string(*azureDeployment.DeploymentStatus)
	}

	FullJSON, err := json.Marshal(azureDeployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	deploymentData := cache.Deployment{
		ReleaseID:     releaseId,
		ReleaseName:   releaseName,
		Status:        status,
		StartTime:     startTime,
		EndTime:       endTime,
		Repository:    repository,
		Branch:        branch,
		SourceVersion: sourceVersion,
		BuildID:       buildId,
		FullJSON:      string(FullJSON),
	}

	err = s.cache.SaveDeployment(deploymentData)
	if err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}
	return nil
}

func (s *DevOpsService) SyncDeploymentsWithReleaseID(force bool, releaseID int) (int, error) {
	if s.cache == nil {
		return 0, fmt.Errorf("cache not initialized")
	}

	azureDeployments, err := s.client.GetDeployments(context.Background(), map[string]string{
		"definitionId": strconv.Itoa(releaseID),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to fetch deployments from API: %w", err)
	}

	savedCount := 0
	for _, azureDeployment := range azureDeployments {
		err := s.SyncDeployment(azureDeployment)
		if err != nil {
			releaseID := 0
			if azureDeployment.Release != nil && azureDeployment.Release.Id != nil {
				releaseID = int(*azureDeployment.Release.Id)
			}
			logger.Warn().
				Err(err).
				Int("release_id", releaseID).
				Msg("Failed to cache deployment, continuing sync")
			continue
		}
		savedCount++
	}

	return savedCount, nil
}

func (s *DevOpsService) SearchDeploymentsCached(filters map[string]any) ([]cache.Deployment, error) {
	if s.syncOptions.SkipSync || s.syncOptions.SyncDone {
		return s.cache.SearchDeployments(filters)
	}

	s.SyncDeployments(false)
	s.syncOptions.SyncDone = true

	return s.cache.SearchDeployments(filters)
}

func (s *DevOpsService) GetDeploymentsForBuild(buildID int) ([]DeploymentStatusInfo, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not initialized")
	}

	filters := map[string]any{
		"build_id": buildID,
	}

	cachedDeployments, err := s.SearchDeploymentsCached(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search deployments in cache: %w", err)
	}

	deployments := []DeploymentStatusInfo{}
	for _, cachedDeployment := range cachedDeployments {
		var fullData map[string]any
		if err := json.Unmarshal([]byte(cachedDeployment.FullJSON), &fullData); err == nil {
			var webURL string
			if links, ok := fullData["_links"].(map[string]any); ok {
				if web, ok := links["web"].(map[string]any); ok {
					webURL, _ = web["href"].(string)
				}
			}

			environment := ""
			if releaseEnv, ok := fullData["releaseEnvironment"].(map[string]any); ok {
				environment, _ = releaseEnv["name"].(string)
			}

			operationStatus := ""
			if opStatus, ok := fullData["operationStatus"].(string); ok {
				operationStatus = opStatus
			}

			deploymentID := 0
			if id, ok := fullData["id"].(float64); ok {
				deploymentID = int(id)
			}

			deployments = append(deployments, DeploymentStatusInfo{
				ReleaseID:       cachedDeployment.ReleaseID,
				ReleaseName:     cachedDeployment.ReleaseName,
				DeploymentID:    deploymentID,
				Status:          cachedDeployment.Status,
				Environment:     environment,
				OperationStatus: operationStatus,
				URL:             webURL,
				StartTime:       cachedDeployment.StartTime,
			})
		}
	}

	return deployments, nil
}

func FormatDeploymentStatus(status, operationStatus string) string {
	switch status {
	case "inProgress":
		return "Running"
	case "succeeded":
		switch operationStatus {
		case "approved":
			return "Approved"
		default:
			return "Completed"
		}
	case "failed":
		return "Failed"
	case "partiallySucceeded":
		return "Partially Succeeded"
	case "notDeployed":
		return "Not Deployed"
	default:
		switch operationStatus {
		case "queued", "queuedForAgent", "queuedForPipeline":
			return "Queued"
		case "pending":
			return "Pending"
		case "rejected":
			return "Rejected"
		case "approved":
			return "Approved"
		default:
			return status
		}
	}
}
