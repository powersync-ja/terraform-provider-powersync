package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ datasource.DataSource = &InstanceDataSource{}

type InstanceDataSource struct {
	client *client.Client
}

func NewInstanceDataSource() datasource.DataSource {
	return &InstanceDataSource{}
}

// ── Model types ──────────────────────────────────────────────────────────────

type instanceDSModel struct {
	OrgID                  types.String                `tfsdk:"org_id"`
	ProjectID              types.String                `tfsdk:"project_id"`
	ID                     types.String                `tfsdk:"id"`
	Name                   types.String                `tfsdk:"name"`
	Region                 types.String                `tfsdk:"region"`
	Status                 types.String                `tfsdk:"status"`
	Provisioned            types.Bool                  `tfsdk:"provisioned"`
	InstanceURL            types.String                `tfsdk:"instance_url"`
	SyncConfigContent      types.String                `tfsdk:"sync_config_content"`
	Operations             []instanceDSOperationModel  `tfsdk:"operations"`
	ReplicationConnections []instanceDSConnectionModel `tfsdk:"replication_connection"`
	ClientAuth             []instanceDSClientAuthModel `tfsdk:"client_auth"`
}

type instanceDSOperationModel struct {
	ID     types.String `tfsdk:"id"`
	Status types.String `tfsdk:"status"`
}

type instanceDSConnectionModel struct {
	Type              types.String `tfsdk:"type"`
	Name              types.String `tfsdk:"name"`
	Tag               types.String `tfsdk:"tag"`
	URI               types.String `tfsdk:"uri"`
	Hostname          types.String `tfsdk:"hostname"`
	Port              types.Int64  `tfsdk:"port"`
	Username          types.String `tfsdk:"username"`
	Database          types.String `tfsdk:"database"`
	SSLMode           types.String `tfsdk:"sslmode"`
	CACert            types.String `tfsdk:"cacert"`
	ClientCertificate types.String `tfsdk:"client_certificate"`
	PostImages        types.String `tfsdk:"post_images"`
	Schema            types.String `tfsdk:"schema"`
}

type instanceDSClientAuthModel struct {
	Supabase             types.Bool   `tfsdk:"supabase"`
	JWKSUri              types.String `tfsdk:"jwks_uri"`
	AdditionalAudiences  types.List   `tfsdk:"additional_audiences"`
	AllowTemporaryTokens types.Bool   `tfsdk:"allow_temporary_tokens"`
}

// ── Metadata ─────────────────────────────────────────────────────────────────

func (d *InstanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

// ── Schema ───────────────────────────────────────────────────────────────────

func (d *InstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing PowerSync Cloud instance by ID. Useful for referencing instances managed outside of Terraform.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID that owns the instance.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID the instance belongs to.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Instance ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Instance name.",
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "Region the instance runs in. One of: `eu`, `us`, `jp`, `au`, `br`.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Derived status: \"deploying\" while an operation is pending/running, \"active\" once the instance has a URL, otherwise \"provisioning\".",
			},
			"provisioned": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a sync config has been deployed to this instance. Despite the name, this is not a liveness signal; use `status` or `instance_url` for that.",
			},
			"instance_url": schema.StringAttribute{
				Computed:    true,
				Description: "Public endpoint URL of the instance.",
			},
			"sync_config_content": schema.StringAttribute{
				Computed:    true,
				Description: "Currently deployed sync config YAML. See https://docs.powersync.com/sync/overview.",
			},
			"operations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "In-flight or recently completed deploy operations on the instance.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Operation ID.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Operation status: pending, running, completed, or failed.",
						},
					},
				},
			},
			"replication_connection": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Configured replication connections. Sensitive fields (password, client_private_key) are not returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Database type: postgresql, mongodb, mysql, or mssql.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Display name for this connection.",
						},
						"tag": schema.StringAttribute{
							Computed:    true,
							Description: "Identifier used to reference this connection from the sync config.",
						},
						"uri": schema.StringAttribute{
							Computed:    true,
							Description: "Full connection URI, if configured.",
						},
						"hostname": schema.StringAttribute{
							Computed:    true,
							Description: "Database hostname.",
						},
						"port": schema.Int64Attribute{
							Computed:    true,
							Description: "Database port.",
						},
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Database username.",
						},
						"database": schema.StringAttribute{
							Computed:    true,
							Description: "Database name.",
						},
						"sslmode": schema.StringAttribute{
							Computed:    true,
							Description: "TLS mode.",
						},
						"cacert": schema.StringAttribute{
							Computed:    true,
							Description: "PEM-encoded CA certificate.",
						},
						"client_certificate": schema.StringAttribute{
							Computed:    true,
							Description: "PEM-encoded client certificate.",
						},
						"post_images": schema.StringAttribute{
							Computed:    true,
							Description: "MongoDB change stream mode.",
						},
						"schema": schema.StringAttribute{
							Computed:    true,
							Description: "Database schema (MSSQL only).",
						},
					},
				},
			},
			"client_auth": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Client JWT authentication configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"supabase": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether Supabase JWT validation is enabled.",
						},
						"jwks_uri": schema.StringAttribute{
							Computed:    true,
							Description: "URL of the configured external JWKS endpoint.",
						},
						"additional_audiences": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Additional valid JWT audience values.",
						},
						"allow_temporary_tokens": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether temporary tokens are allowed.",
						},
					},
				},
			},
		},
	}
}

// ── Configure ────────────────────────────────────────────────────────────────

func (d *InstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

// ── Read ─────────────────────────────────────────────────────────────────────

func (d *InstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instanceDSModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	projectID := state.ProjectID.ValueString()
	instanceID := state.ID.ValueString()

	config, err := d.client.GetInstanceConfig(ctx, orgID, projectID, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch instance", err.Error())
		return
	}
	if config == nil {
		resp.Diagnostics.AddError(
			"Instance not found",
			fmt.Sprintf("no instance %s in project %s / org %s", instanceID, projectID, orgID),
		)
		return
	}

	state.Name = types.StringValue(config.Name)
	if config.Config != nil {
		state.Region = types.StringValue(config.Config.Region)
		state.ReplicationConnections = dsConnectionsFromAPI(config.Config.Replication)
		state.ClientAuth = dsClientAuthFromAPI(ctx, config.Config.ClientAuth)
	} else {
		state.Region = types.StringNull()
		state.ReplicationConnections = []instanceDSConnectionModel{}
		state.ClientAuth = []instanceDSClientAuthModel{}
	}
	state.SyncConfigContent = dsStringOrNull(config.SyncRules)

	status, err := d.client.GetInstanceStatus(ctx, orgID, projectID, instanceID)
	if err != nil {
		resp.Diagnostics.AddWarning("Could not read instance status", err.Error())
		state.Status = types.StringNull()
		state.Provisioned = types.BoolNull()
		state.InstanceURL = types.StringNull()
		state.Operations = []instanceDSOperationModel{}
	} else {
		state.Status = types.StringValue(status.DeriveStatus())
		state.Provisioned = types.BoolValue(status.Provisioned)
		state.InstanceURL = dsStringOrNull(status.InstanceURL)
		state.Operations = dsOperationsFromAPI(status.Operations)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func dsConnectionsFromAPI(repl *client.ReplicationConfig) []instanceDSConnectionModel {
	if repl == nil || len(repl.Connections) == 0 {
		return []instanceDSConnectionModel{}
	}
	result := make([]instanceDSConnectionModel, len(repl.Connections))
	for i, c := range repl.Connections {
		result[i] = instanceDSConnectionModel{
			Type:              types.StringValue(c.Type),
			Name:              dsStringOrNull(c.Name),
			Tag:               dsStringOrNull(c.Tag),
			URI:               dsStringOrNull(c.URI),
			Hostname:          dsStringOrNull(c.Hostname),
			Port:              dsInt64OrNull(c.Port),
			Username:          dsStringOrNull(c.Username),
			Database:          dsStringOrNull(c.Database),
			SSLMode:           dsStringOrNull(c.SSLMode),
			CACert:            dsStringOrNull(c.CACert),
			ClientCertificate: dsStringOrNull(c.ClientCertificate),
			PostImages:        dsStringOrNull(c.PostImages),
			Schema:            dsStringOrNull(c.Schema),
		}
	}
	return result
}

func dsClientAuthFromAPI(ctx context.Context, auth *client.ClientAuthConfig) []instanceDSClientAuthModel {
	if auth == nil {
		return []instanceDSClientAuthModel{}
	}
	m := instanceDSClientAuthModel{
		Supabase:             types.BoolValue(auth.Supabase),
		JWKSUri:              dsStringOrNull(auth.JWKSUri),
		AllowTemporaryTokens: types.BoolValue(auth.AllowTemporaryTokens),
		AdditionalAudiences:  types.ListNull(types.StringType),
	}
	if len(auth.AdditionalAudiences) > 0 {
		listVal, _ := types.ListValueFrom(ctx, types.StringType, auth.AdditionalAudiences)
		m.AdditionalAudiences = listVal
	}
	return []instanceDSClientAuthModel{m}
}

func dsOperationsFromAPI(ops []client.DeployOperation) []instanceDSOperationModel {
	if len(ops) == 0 {
		return []instanceDSOperationModel{}
	}
	result := make([]instanceDSOperationModel, len(ops))
	for i, op := range ops {
		result[i] = instanceDSOperationModel{
			ID:     types.StringValue(op.ID),
			Status: types.StringValue(op.Status),
		}
	}
	return result
}

func dsStringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func dsInt64OrNull(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}
