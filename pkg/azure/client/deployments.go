package client

import (
	"context"
	"strconv"

	"adoctl/pkg/utils"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/release"
)

func (c *Client) GetDeployments(ctx context.Context, params map[string]string) ([]release.Deployment, error) {
	args := release.GetDeploymentsArgs{
		Project: utils.Ptr(c.GetProject()),
	}

	if topStr, ok := params["$top"]; ok {
		if top, err := strconv.Atoi(topStr); err == nil {
			args.Top = &top
		}
	}

	if definitionIdStr, ok := params["definitionId"]; ok {
		if definitionId, err := strconv.Atoi(definitionIdStr); err == nil {
			args.DefinitionId = &definitionId
		}
	}

	if definitionEnvIdStr, ok := params["definitionEnvironmentId"]; ok {
		if definitionEnvId, err := strconv.Atoi(definitionEnvIdStr); err == nil {
			args.DefinitionEnvironmentId = &definitionEnvId
		}
	}

	if statusStr, ok := params["status"]; ok {
		status := release.DeploymentStatus(statusStr)
		args.DeploymentStatus = &status
	}

	if queryOrderStr, ok := params["queryOrder"]; ok {
		queryOrder := release.ReleaseQueryOrder(queryOrderStr)
		args.QueryOrder = &queryOrder
	}

	if sourceBranch, ok := params["sourceBranch"]; ok {
		args.SourceBranch = &sourceBranch
	}

	result, err := c.ReleaseClient.GetDeployments(ctx, args)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

func (c *Client) GetDeploymentByID(ctx context.Context, deploymentID int) (*release.Deployment, error) {
	args := release.GetDeploymentsArgs{
		Project: utils.Ptr(c.GetProject()),
		Top:     utils.Ptr(1),
	}

	result, err := c.ReleaseClient.GetDeployments(ctx, args)
	if err != nil {
		return nil, err
	}

	for i := range result.Value {
		if result.Value[i].Id != nil && *result.Value[i].Id == deploymentID {
			return &result.Value[i], nil
		}
	}

	return nil, nil
}
