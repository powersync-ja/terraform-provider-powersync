package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-uuid"
)

// Project represents a PowerSync project (called "app" in the API).
type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DefaultRegion string `json:"default_region"`
	VCSMode       string `json:"vcs_mode"`
	Trial         bool   `json:"trial"`
	Locked        bool   `json:"locked"`
}

type listProjectsRequest struct {
	OrgID  *string `json:"org_id,omitempty"`
	ID     *string `json:"id,omitempty"`
	Cursor string  `json:"cursor,omitempty"`
	Limit  int     `json:"limit,omitempty"`
}

type listProjectsResponse struct {
	Objects []Project `json:"objects"`
	Total   int       `json:"total"`
	More    bool      `json:"more"`
	Cursor  string    `json:"cursor"`
}

type listProjectsAPIResponse struct {
	Data listProjectsResponse `json:"data"`
}

// ListProjects fetches every project in an org by following cursor pagination.
// Returns the accumulated list and the server-reported total from the last page
// (useful as a sanity check; the two should match).
func (c *Client) ListProjects(ctx context.Context, orgID string) ([]Project, int, error) {
	var all []Project
	var cursor string
	var total int
	for {
		req := listProjectsRequest{OrgID: &orgID, Cursor: cursor}
		var out listProjectsAPIResponse
		if err := c.post(ctx, "/api/accounts/v5/apps/list", req, &out); err != nil {
			return nil, 0, err
		}
		all = append(all, out.Data.Objects...)
		total = out.Data.Total
		if !out.Data.More {
			break
		}
		if out.Data.Cursor == "" {
			// Defensive: server says more but gave no cursor — avoid infinite loop.
			break
		}
		cursor = out.Data.Cursor
	}
	return all, total, nil
}

// GetProjectByID fetches a single project via /apps/get. Returns nil, nil
// when the project does not exist (404).
func (c *Client) GetProjectByID(ctx context.Context, orgID, id string) (*Project, error) {
	var out Project
	err := c.postData(ctx, "/api/accounts/v5/apps/get", map[string]string{"id": id}, &out)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// --- Create / Update / Delete ---

type projectSource struct {
	Type       string                  `json:"type"`
	Properties projectSourceProperties `json:"properties"`
}

type projectSourceProperties struct {
	ID string `json:"id"`
}

type createProjectRequest struct {
	Name       string        `json:"name"`
	OrgID      string        `json:"org_id"`
	Region     string        `json:"region"`
	Source     projectSource `json:"source"`
	VCSMode    string        `json:"vcs_mode"`
	Features   []string      `json:"features"`
	TemplateID string        `json:"template_id"`
}

// CreateProject creates a new project. Most fields are hardcoded to match the
// dashboard's create flow — only name, org, and region are user-controllable.
// source.properties.id is a fresh UUID per create.
func (c *Client) CreateProject(ctx context.Context, orgID, name, region string) (*Project, error) {
	sourceID, err := uuid.GenerateUUID()
	if err != nil {
		return nil, fmt.Errorf("generate source id: %w", err)
	}
	req := createProjectRequest{
		Name:   name,
		OrgID:  orgID,
		Region: region,
		Source: projectSource{
			Type:       "INTERNAL",
			Properties: projectSourceProperties{ID: sourceID},
		},
		VCSMode:    "BASIC",
		Features:   []string{"powersync"},
		TemplateID: "standard-powersync",
	}
	var out Project
	if err := c.alphaPostData(ctx, "/api/v1/apps/create", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type updateProjectRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpdateProject changes the project's name. Per the accounts API, name is the
// only mutable field.
func (c *Client) UpdateProject(ctx context.Context, id, name string) (*Project, error) {
	req := updateProjectRequest{ID: id, Name: name}
	var out Project
	if err := c.postData(ctx, "/api/accounts/v5/apps/update", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type deleteProjectRequest struct {
	OrgID string `json:"org_id"`
	AppID string `json:"app_id"`
}

// DeleteProject removes the project. The API cascade-destroys all instances
// under the project — callers should check for non-Terraform-managed instances
// first if they want to avoid silent data loss.
func (c *Client) DeleteProject(ctx context.Context, orgID, id string) error {
	req := deleteProjectRequest{OrgID: orgID, AppID: id}
	return c.alphaPost(ctx, "/api/v1/apps/delete", req, nil)
}

// GetProjectByName fetches the full project list and returns the first name match.
// Returns nil, nil when no match is found.
func (c *Client) GetProjectByName(ctx context.Context, orgID, name string) (*Project, error) {
	projects, _, err := c.ListProjects(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].Name == name {
			return &projects[i], nil
		}
	}
	return nil, nil
}
