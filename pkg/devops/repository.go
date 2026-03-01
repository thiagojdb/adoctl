package devops

import (
	"fmt"
	"strings"

	"adoctl/pkg/logger"
	"adoctl/pkg/models"
)

func (s *DevOpsService) ListRepositories() ([]models.Repository, error) {
	if s.cache != nil {
		cachedRepos, err := s.cache.GetRepositories()
		if err == nil && cachedRepos != nil {
			return cachedRepos, nil
		}
	}

	repos, err := s.client.GetRepositories(nil)
	if err != nil {
		return nil, err
	}

	result := make([]models.Repository, 0, len(repos))
	for _, repo := range repos {
		result = append(result, models.RepositoryFromAzure(&repo))
	}

	if s.cache != nil {
		if err := s.cache.SetRepositories(result); err != nil {
			logger.Warn().Err(err).Msg("Failed to cache repositories")
		}
	}

	return result, nil
}

func (s *DevOpsService) GetRepositoryID(repositoryName string) (string, error) {
	repos, err := s.ListRepositories()
	if err != nil {
		return "", err
	}

	for _, repo := range repos {
		if strings.EqualFold(repo.Name, repositoryName) {
			return repo.ID, nil
		}
	}

	return "", fmt.Errorf("repository '%s' not found", repositoryName)
}
