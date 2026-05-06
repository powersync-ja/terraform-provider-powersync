package client

import "context"

// Organization represents a PowerSync organization.
// The API field is "label"; we surface it as Name to callers.
type Organization struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (c *Client) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	var out Organization
	if err := c.post(ctx, "/api/accounts/v5/organizations/get", map[string]string{"id": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
