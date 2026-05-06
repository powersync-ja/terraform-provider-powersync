package client

import "context"

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
	OrgID *string `json:"org_id,omitempty"`
	ID    *string `json:"id,omitempty"`
	Limit int     `json:"limit"`
}

type listProjectsResponse struct {
	Objects []Project `json:"objects"`
	Total   int       `json:"total"`
}

type listProjectsAPIResponse struct {
	Data listProjectsResponse `json:"data"`
}

func (c *Client) ListProjects(ctx context.Context, orgID string) ([]Project, error) {
	req := listProjectsRequest{OrgID: &orgID, Limit: 100}
	var out listProjectsAPIResponse
	if err := c.post(ctx, "/api/accounts/v5/apps/list", req, &out); err != nil {
		return nil, err
	}
	return out.Data.Objects, nil
}

// GetProjectByID filters the apps/list endpoint by org + id.
// Returns nil, nil when no match is found.
func (c *Client) GetProjectByID(ctx context.Context, orgID, id string) (*Project, error) {
	req := listProjectsRequest{OrgID: &orgID, ID: &id, Limit: 1}
	var out listProjectsAPIResponse
	if err := c.post(ctx, "/api/accounts/v5/apps/list", req, &out); err != nil {
		return nil, err
	}
	if len(out.Data.Objects) == 0 {
		return nil, nil
	}
	return &out.Data.Objects[0], nil
}

// GetProjectByName fetches the full project list and returns the first name match.
// Returns nil, nil when no match is found.
func (c *Client) GetProjectByName(ctx context.Context, orgID, name string) (*Project, error) {
	projects, err := c.ListProjects(ctx, orgID)
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
