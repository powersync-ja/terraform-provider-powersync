package client

import (
	"context"
	"fmt"
	"time"
)

// HostedSecret wraps a sensitive value. Only inline secrets are supported for now.
type HostedSecret struct {
	Secret string `json:"secret"`
}

// Connection covers all supported database types in a flat struct.
// Fields not relevant to a given type are omitted via omitempty.
type Connection struct {
	Type              string        `json:"type"`
	Name              string        `json:"name,omitempty"`
	Tag               string        `json:"tag,omitempty"`
	URI               string        `json:"uri,omitempty"`
	Hostname          string        `json:"hostname,omitempty"`
	Port              *int64        `json:"port,omitempty"`
	Username          string        `json:"username,omitempty"`
	Password          *HostedSecret `json:"password,omitempty"`
	Database          string        `json:"database,omitempty"`
	SSLMode           string        `json:"sslmode,omitempty"`
	CACert            string        `json:"cacert,omitempty"`
	ClientCertificate string        `json:"client_certificate,omitempty"`
	ClientPrivateKey  *HostedSecret `json:"client_private_key,omitempty"`
	// MongoDB only: off | auto_configure | read_only
	PostImages string `json:"post_images,omitempty"`
	// MSSQL only
	Schema string `json:"schema,omitempty"`
}

type ReplicationConfig struct {
	Connections []Connection `json:"connections,omitempty"`
}

type ClientAuthConfig struct {
	Supabase             bool     `json:"supabase,omitempty"`
	JWKSUri              string   `json:"jwks_uri,omitempty"`
	AdditionalAudiences  []string `json:"additional_audiences,omitempty"`
	AllowTemporaryTokens bool     `json:"allow_temporary_tokens,omitempty"`
}

type HostedConfig struct {
	Region      string             `json:"region"`
	Replication *ReplicationConfig `json:"replication,omitempty"`
	ClientAuth  *ClientAuthConfig  `json:"client_auth,omitempty"`
}

type ProgramVersionConstraint struct {
	Channel      string `json:"channel"`
	VersionRange string `json:"version_range,omitempty"`
}

type InstanceConfig struct {
	ID             string                   `json:"id"`
	ProjectID      string                   `json:"project_id"`
	OrgID          string                   `json:"org_id"`
	Name           string                   `json:"name"`
	Config         *HostedConfig            `json:"config"`
	SyncRules      string                   `json:"sync_rules"`
	ProgramVersion ProgramVersionConstraint `json:"program_version"`
}

type DeployOperation struct {
	ID     string `json:"id"`
	Status string `json:"status"` // pending | running | completed | failed
}

type InstanceStatus struct {
	ID          string            `json:"id"`
	// Provisioned is true when sync rules have been deployed.
	// Use InstanceURL + Operations (via DeriveStatus) for liveness.
	Provisioned bool              `json:"provisioned"`
	Operations  []DeployOperation `json:"operations"`
	InstanceURL string            `json:"instance_url,omitempty"`
}

// DeriveStatus returns a human-readable status. The API's `provisioned` flag
// is not reliable as an "is the instance live" signal (it returns false even
// for long-lived healthy instances), so we infer from operations + URL.
func (s *InstanceStatus) DeriveStatus() string {
	for _, op := range s.Operations {
		if op.Status == "pending" || op.Status == "running" {
			return "deploying"
		}
	}
	if s.InstanceURL != "" {
		return "active"
	}
	return "provisioning"
}

type Region struct {
	Name       string `json:"name"`
	Deployable bool   `json:"deployable"`
}

// InstanceSummary is the lightweight per-instance shape returned by
// /api/v1/instances/list. For full config/status, call GetInstanceConfig
// and GetInstanceStatus per instance.
type InstanceSummary struct {
	ID         string `json:"id"`
	OrgID      string `json:"org_id"`
	AppID      string `json:"app_id"`
	Name       string `json:"name"`
	HasConfig  bool   `json:"has_config"`
	Deployable bool   `json:"deployable"`
}

// --- Request / response types ---

type createInstanceRequest struct {
	OrgID  string `json:"org_id"`
	AppID  string `json:"app_id"`
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
}

type createInstanceResponse struct {
	ID string `json:"id"`
}

type DeployInstanceRequest struct {
	OrgID          string                   `json:"org_id"`
	AppID          string                   `json:"app_id"`
	ID             string                   `json:"id"`
	Name           string                   `json:"name,omitempty"`
	Config         HostedConfig             `json:"config"`
	SyncRules      string                   `json:"sync_rules,omitempty"`
	ProgramVersion ProgramVersionConstraint `json:"program_version"`
}

type deployInstanceResponse struct {
	ID          string `json:"id"`
	OperationID string `json:"operation_id"`
}

type instanceActionRequest struct {
	OrgID string `json:"org_id"`
	AppID string `json:"app_id"`
	ID    string `json:"id"`
}

type destroyInstanceResponse struct {
	OperationID string `json:"operation_id"`
}

type testConnectionRequest struct {
	OrgID      string     `json:"org_id"`
	AppID      string     `json:"app_id"`
	ID         string     `json:"id"`
	Connection Connection `json:"connection"`
}

type testConnectionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type listRegionsResponse struct {
	Regions []Region `json:"regions"`
}

type listInstancesRequest struct {
	OrgID string `json:"org_id"`
	AppID string `json:"app_id"`
}

type listInstancesResponse struct {
	Instances []InstanceSummary `json:"instances"`
}

// --- Methods ---

func (c *Client) CreateInstance(ctx context.Context, orgID, appID, name, region string) (string, error) {
	req := createInstanceRequest{OrgID: orgID, AppID: appID, Name: name, Region: region}
	var out createInstanceResponse
	if err := c.managementPostData(ctx, "/api/v1/instances/create", req, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) DeployInstance(ctx context.Context, req DeployInstanceRequest) (string, error) {
	var out deployInstanceResponse
	if err := c.managementPostData(ctx, "/api/v1/instances/deploy", req, &out); err != nil {
		return "", err
	}
	return out.OperationID, nil
}

// GetInstanceConfig returns nil, nil when the instance does not exist (404).
// Retries on transient errors.
func (c *Client) GetInstanceConfig(ctx context.Context, orgID, appID, id string) (*InstanceConfig, error) {
	var out InstanceConfig
	err := retryTransient(ctx, func() error {
		return c.managementPostData(ctx, "/api/v1/instances/config", instanceActionRequest{OrgID: orgID, AppID: appID, ID: id}, &out)
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// GetInstanceStatus retries on transient errors.
func (c *Client) GetInstanceStatus(ctx context.Context, orgID, appID, id string) (*InstanceStatus, error) {
	var out InstanceStatus
	err := retryTransient(ctx, func() error {
		return c.managementPostData(ctx, "/api/v1/instances/status", instanceActionRequest{OrgID: orgID, AppID: appID, ID: id}, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DestroyInstance(ctx context.Context, orgID, appID, id string) (string, error) {
	var out destroyInstanceResponse
	if err := c.managementPostData(ctx, "/api/v1/instances/destroy", instanceActionRequest{OrgID: orgID, AppID: appID, ID: id}, &out); err != nil {
		return "", err
	}
	return out.OperationID, nil
}

// WaitForOperation polls GetInstanceStatus every 5s until the matching operation
// reaches completed or failed, or until the context deadline.
func (c *Client) WaitForOperation(ctx context.Context, orgID, appID, instanceID, operationID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for operation %s: %w", operationID, ctx.Err())
		case <-ticker.C:
			status, err := c.GetInstanceStatus(ctx, orgID, appID, instanceID)
			if err != nil {
				return fmt.Errorf("polling status: %w", err)
			}
			for _, op := range status.Operations {
				if op.ID == operationID {
					switch op.Status {
					case "completed":
						return nil
					case "failed":
						return fmt.Errorf("operation %s failed", operationID)
					}
				}
			}
		}
	}
}

// TestConnection verifies DB connectivity before deploying. Returns an error if
// the API reports the connection failed. Retries on transient errors — testing
// a connection is idempotent (no resources created).
func (c *Client) TestConnection(ctx context.Context, orgID, appID, instanceID string, conn Connection) error {
	req := testConnectionRequest{OrgID: orgID, AppID: appID, ID: instanceID, Connection: conn}
	var out testConnectionResponse
	err := retryTransient(ctx, func() error {
		return c.managementPostData(ctx, "/api/v1/connections/test", req, &out)
	})
	if err != nil {
		return err
	}
	if !out.Success {
		if out.Error != "" {
			return fmt.Errorf("connection test failed: %s", out.Error)
		}
		return fmt.Errorf("connection test failed")
	}
	return nil
}

// ListInstances returns the lightweight summary list for all instances in a
// project. The endpoint has no pagination — strict validation, all results in
// one shot. Retries on transient errors.
func (c *Client) ListInstances(ctx context.Context, orgID, appID string) ([]InstanceSummary, error) {
	req := listInstancesRequest{OrgID: orgID, AppID: appID}
	var out listInstancesResponse
	err := retryTransient(ctx, func() error {
		return c.managementPostData(ctx, "/api/v1/instances/list", req, &out)
	})
	if err != nil {
		return nil, err
	}
	return out.Instances, nil
}

// ListRegions retries on transient errors.
func (c *Client) ListRegions(ctx context.Context) ([]Region, error) {
	var out listRegionsResponse
	err := retryTransient(ctx, func() error {
		return c.managementGetData(ctx, "/api/v1/regions", &out)
	})
	if err != nil {
		return nil, err
	}
	return out.Regions, nil
}
