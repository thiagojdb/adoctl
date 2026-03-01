package client

import (
	"context"
	"fmt"
	"strconv"

	"adoctl/pkg/utils"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

func (c *Client) GetBuilds(ctx context.Context, params map[string]string) ([]build.Build, error) {
	args := build.GetBuildsArgs{}

	if topStr, ok := params["$top"]; ok && topStr != "" {
		if top, err := strconv.Atoi(topStr); err == nil {
			args.Top = utils.Ptr(top)
		}
	}

	result, err := c.BuildClient.GetBuilds(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get builds: %w", err)
	}

	if result.Value == nil {
		return []build.Build{}, nil
	}

	return result.Value, nil
}

func (c *Client) GetBuildByID(ctx context.Context, buildID int) (*build.Build, error) {
	args := build.GetBuildArgs{
		BuildId: &buildID,
	}

	result, err := c.BuildClient.GetBuild(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get build by ID: %w", err)
	}

	return result, nil
}
