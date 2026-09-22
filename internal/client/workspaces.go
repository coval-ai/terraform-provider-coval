package client

import (
	"context"
	"net/http"
	"net/url"
)

// Workspace is a Coval organization workspace.
type Workspace struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	Status        string `json:"status"`
	WorkspaceType string `json:"workspace_type"`
	CreatedAt     string `json:"created_at"`
	LastUpdatedAt string `json:"last_updated_at"`
}

// CreateWorkspaceInput contains writable workspace fields.
type CreateWorkspaceInput struct {
	DisplayName string `json:"display_name"`
}

// UpdateWorkspaceInput contains writable workspace fields.
type UpdateWorkspaceInput struct {
	DisplayName string `json:"display_name"`
}

type workspaceEnvelope struct {
	Workspace Workspace `json:"workspace"`
}

type workspacesEnvelope struct {
	Workspaces []Workspace `json:"workspaces"`
}

// CreateWorkspace creates a custom workspace.
func (c *Client) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (Workspace, error) {
	var response workspaceEnvelope
	if err := c.Do(ctx, http.MethodPost, "workspaces", input, &response); err != nil {
		return Workspace{}, err
	}
	return response.Workspace, nil
}

// GetWorkspace retrieves a workspace by ID.
func (c *Client) GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	var response workspaceEnvelope
	if err := c.Do(ctx, http.MethodGet, "workspaces/"+url.PathEscape(id), nil, &response); err != nil {
		return Workspace{}, err
	}
	return response.Workspace, nil
}

// UpdateWorkspace changes a workspace display name.
func (c *Client) UpdateWorkspace(ctx context.Context, id string, input UpdateWorkspaceInput) (Workspace, error) {
	var response workspaceEnvelope
	if err := c.Do(ctx, http.MethodPatch, "workspaces/"+url.PathEscape(id), input, &response); err != nil {
		return Workspace{}, err
	}
	return response.Workspace, nil
}

// DeleteWorkspace soft-deletes a workspace.
func (c *Client) DeleteWorkspace(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "workspaces/"+url.PathEscape(id), nil, nil)
}

// ListWorkspaces returns active and archived workspaces.
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var response workspacesEnvelope
	if err := c.Do(ctx, http.MethodGet, "workspaces", nil, &response); err != nil {
		return nil, err
	}
	return response.Workspaces, nil
}
