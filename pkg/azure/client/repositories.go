package client

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func (c *Client) GetRepositories(ctx context.Context) ([]git.GitRepository, error) {
	args := git.GetRepositoriesArgs{}

	response, err := c.GitClient.GetRepositories(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	if response == nil {
		return []git.GitRepository{}, nil
	}

	return *response, nil
}

func (c *Client) GetRepository(ctx context.Context, repositoryID string) (*git.GitRepository, error) {
	args := git.GetRepositoryArgs{
		RepositoryId: &repositoryID,
	}

	return c.GitClient.GetRepository(ctx, args)
}
