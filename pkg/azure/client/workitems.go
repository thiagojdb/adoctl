package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"adoctl/pkg/utils"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

func (c *Client) GetWorkItem(ctx context.Context, id int) (map[string]any, error) {
	project := c.GetProject()
	idPtr := &id
	args := workitemtracking.GetWorkItemArgs{
		Project: &project,
		Id:      idPtr,
	}

	wi, err := c.WorkItemClient.GetWorkItem(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item %d: %w", id, err)
	}

	result, err := json.Marshal(wi)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal work item: %w", err)
	}

	var workItem map[string]any
	if err := json.Unmarshal(result, &workItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work item: %w", err)
	}

	return workItem, nil
}

func (c *Client) GetWorkItems(ctx context.Context, ids []int) ([]map[string]any, error) {
	if len(ids) == 0 {
		return []map[string]any{}, nil
	}

	project := c.GetProject()
	idsPtr := &ids
	args := workitemtracking.GetWorkItemsArgs{
		Project: &project,
		Ids:     idsPtr,
	}

	wis, err := c.WorkItemClient.GetWorkItems(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get work items: %w", err)
	}

	if wis == nil {
		return []map[string]any{}, nil
	}

	var result []map[string]any
	for _, wi := range *wis {
		data, err := json.Marshal(wi)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal work item: %w", err)
		}

		var workItem map[string]any
		if err := json.Unmarshal(data, &workItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal work item: %w", err)
		}

		result = append(result, workItem)
	}

	return result, nil
}

func (c *Client) GetWorkItemRelations(ctx context.Context, id int) ([]map[string]any, error) {
	project := c.GetProject()
	idPtr := &id
	args := workitemtracking.GetWorkItemArgs{
		Project: &project,
		Id:      idPtr,
	}

	wi, err := c.WorkItemClient.GetWorkItem(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item %d: %w", id, err)
	}

	if wi.Relations == nil || len(*wi.Relations) == 0 {
		return []map[string]any{}, nil
	}

	var result []map[string]any
	for _, rel := range *wi.Relations {
		data, err := json.Marshal(rel)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal relation: %w", err)
		}

		var relation map[string]any
		if err := json.Unmarshal(data, &relation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal relation: %w", err)
		}

		result = append(result, relation)
	}

	return result, nil
}

func (c *Client) AddWorkItemRelation(ctx context.Context, workItemID int, rel map[string]any) error {
	project := c.GetProject()

	relData, err := json.Marshal(rel)
	if err != nil {
		return fmt.Errorf("failed to marshal relation: %w", err)
	}

	var relationValue any
	if err := json.Unmarshal(relData, &relationValue); err != nil {
		return fmt.Errorf("failed to unmarshal relation value: %w", err)
	}

	op := webapi.Operation("add")
	path := "/relations/-"
	patchDocument := []webapi.JsonPatchOperation{
		{
			Op:    &op,
			Path:  &path,
			Value: relationValue,
		},
	}

	workItemIDPtr := &workItemID
	args := workitemtracking.UpdateWorkItemArgs{
		Project:  &project,
		Id:       workItemIDPtr,
		Document: &patchDocument,
	}

	_, err = c.WorkItemClient.UpdateWorkItem(ctx, args)
	if err != nil {
		return fmt.Errorf("failed to add work item relation: %w", err)
	}

	return nil
}

func (c *Client) GetWorkItemComments(ctx context.Context, id int) ([]map[string]any, error) {
	project := c.GetProject()
	idPtr := &id
	args := workitemtracking.GetCommentsArgs{
		Project:    &project,
		WorkItemId: idPtr,
	}

	comments, err := c.WorkItemClient.GetComments(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for work item %d: %w", id, err)
	}

	if comments == nil || comments.Comments == nil || len(*comments.Comments) == 0 {
		return []map[string]any{}, nil
	}

	var result []map[string]any
	for _, comment := range *comments.Comments {
		data, err := json.Marshal(comment)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal comment: %w", err)
		}

		var commentMap map[string]any
		if err := json.Unmarshal(data, &commentMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal comment: %w", err)
		}

		result = append(result, commentMap)
	}

	return result, nil
}

func (c *Client) DownloadAttachment(ctx context.Context, url, destPath string) error {
	project := c.GetProject()

	attachmentID, err := extractAttachmentIDFromURL(url)
	if err != nil {
		return fmt.Errorf("failed to extract attachment ID: %w", err)
	}

	reader, err := c.WorkItemClient.GetAttachmentZip(ctx, workitemtracking.GetAttachmentZipArgs{
		Id:       attachmentID,
		Project:  &project,
		Download: utils.Ptr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read attachment data: %w", err)
	}

	return os.WriteFile(destPath, data, 0600)
}

func extractAttachmentIDFromURL(url string) (*uuid.UUID, error) {
	re := regexp.MustCompile(`/attachments/([a-f0-9-]{36})`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return nil, fmt.Errorf("failed to extract attachment ID from URL: %s", url)
	}
	id, err := uuid.Parse(matches[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse attachment ID: %w", err)
	}
	return &id, nil
}
