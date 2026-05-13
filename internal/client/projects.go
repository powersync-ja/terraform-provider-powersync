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
