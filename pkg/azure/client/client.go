package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"adoctl/pkg/config"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/release"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
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

// GetCurrentUser returns the display name of the user associated with the PAT token
// by making a direct API call to the Azure DevOps profile endpoint
func (c *Client) GetCurrentUser(ctx context.Context) (string, error) {
	// Make a direct API call to the profile endpoint
	// The PAT token is associated with a specific user, and we can get that user's info
	// from the _apis/connectionData endpoint

	organizationURL := fmt.Sprintf("https://dev.azure.com/%s", c.config.Organization)
	reqURL := fmt.Sprintf("%s/_apis/connectionData", organizationURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header with PAT
	// The Connection stores the PAT in the authorizationString field
	// We need to use the same format as the SDK: "Basic base64(PAT:)"
	auth := c.Connection.AuthorizationString
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		AuthenticatedUser struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			UniqueName  string `json:"uniqueName"`
		} `json:"authenticatedUser"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.AuthenticatedUser.DisplayName == "" {
		return "", fmt.Errorf("could not determine current user name")
	}

	return result.AuthenticatedUser.DisplayName, nil
}

// GetCurrentUserIdentity returns the authenticated user's identity details.
func (c *Client) GetCurrentUserIdentity(ctx context.Context) (*webapi.IdentityRef, error) {
	organizationURL := fmt.Sprintf("https://dev.azure.com/%s", c.config.Organization)
	reqURL := fmt.Sprintf("%s/_apis/connectionData", organizationURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.Connection.AuthorizationString)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var result struct {
		AuthenticatedUser struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			UniqueName  string `json:"uniqueName"`
		} `json:"authenticatedUser"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.AuthenticatedUser.ID == "" {
		return nil, fmt.Errorf("could not determine current user identity")
	}

	return &webapi.IdentityRef{
		Id:          &result.AuthenticatedUser.ID,
		DisplayName: &result.AuthenticatedUser.DisplayName,
		UniqueName:  &result.AuthenticatedUser.UniqueName,
	}, nil
}
