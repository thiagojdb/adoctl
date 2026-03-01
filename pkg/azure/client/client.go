package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"adoctl/pkg/config"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/release"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

var (
	httpClient *http.Client
)

type Client struct {
	config         *config.AzureConfig
	Connection     *azuredevops.Connection
	GitClient      git.Client
	BuildClient    build.Client
	ReleaseClient  release.Client
	WorkItemClient workitemtracking.Client
	CoreClient     core.Client
}

func init() {
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			MaxConnsPerHost:     20,
		},
	}
}

// NewClient creates a new Azure DevOps client instance.
// Each call creates a fresh client, allowing for parallel testing and proper dependency injection.
func NewClient() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return NewClientWithConfig(cfg)
}

// NewClientWithConfig creates a new client with the provided configuration.
// This allows for dependency injection in tests and alternative configuration sources.
func NewClientWithConfig(cfg *config.Config) (*Client, error) {
	organizationURL := fmt.Sprintf("https://dev.azure.com/%s", cfg.Azure.Organization)
	connection := azuredevops.NewPatConnection(organizationURL, cfg.Azure.PersonalAccessToken)

	ctx := context.Background()

	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create build client: %w", err)
	}

	releaseClient, err := release.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create release client: %w", err)
	}

	workItemClient, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item client: %w", err)
	}

	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	return &Client{
		config:         &cfg.Azure,
		Connection:     connection,
		GitClient:      gitClient,
		BuildClient:    buildClient,
		ReleaseClient:  releaseClient,
		WorkItemClient: workItemClient,
		CoreClient:     coreClient,
	}, nil
}

func GetHttpClient() *http.Client {
	return httpClient
}

func (c *Client) GetOrganization() string {
	return c.config.Organization
}

func (c *Client) GetProject() string {
	return c.config.Project
}
