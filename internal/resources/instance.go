package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

const (
	deployTimeout  = 10 * time.Minute
	destroyTimeout = 10 * time.Minute
)

var _ resource.Resource = &InstanceResource{}
var _ resource.ResourceWithImportState = &InstanceResource{}

type InstanceResource struct {
	client *client.Client
}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

// ── Model types ──────────────────────────────────────────────────────────────

type instanceModel struct {
	ID                     types.String          `tfsdk:"id"`
	OrgID                  types.String          `tfsdk:"org_id"`
	ProjectID              types.String          `tfsdk:"project_id"`
	Name                   types.String          `tfsdk:"name"`
	Region                 types.String          `tfsdk:"region"`
	SyncConfigContent      types.String          `tfsdk:"sync_config_content"`
	Status                 types.String          `tfsdk:"status"`
	Provisioned            types.Bool            `tfsdk:"provisioned"`
	InstanceURL            types.String          `tfsdk:"instance_url"`
	ReplicationConnections []connectionModel     `tfsdk:"replication_connection"`
	ClientAuth             []clientAuthModel     `tfsdk:"client_auth"`
	ProgramVersion         []programVersionModel `tfsdk:"program_version"`
}

type connectionModel struct {
	Type              types.String `tfsdk:"type"`
	Name              types.String `tfsdk:"name"`
	Tag               types.String `tfsdk:"tag"`
	URI               types.String `tfsdk:"uri"`
	Hostname          types.String `tfsdk:"hostname"`
	Port              types.Int64  `tfsdk:"port"`
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	Database          types.String `tfsdk:"database"`
	SSLMode           types.String `tfsdk:"sslmode"`
	CACert            types.String `tfsdk:"cacert"`
	ClientCertificate types.String `tfsdk:"client_certificate"`
	ClientPrivateKey  types.String `tfsdk:"client_private_key"`
	PostImages        types.String `tfsdk:"post_images"`
	Schema            types.String `tfsdk:"schema"`
}

type clientAuthModel struct {
	Supabase             types.Bool   `tfsdk:"supabase"`
	JWKSUri              types.String `tfsdk:"jwks_uri"`
	AdditionalAudiences  types.List   `tfsdk:"additional_audiences"`
	AllowTemporaryTokens types.Bool   `tfsdk:"allow_temporary_tokens"`
}

type programVersionModel struct {
	Channel      types.String `tfsdk:"channel"`
	VersionRange types.String `tfsdk:"version_range"`
}

// ── Metadata ─────────────────────────────────────────────────────────────────

func (r *InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

// ── Schema ────────────────────────────────────────────────────────────────────

func (r *InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PowerSync Cloud instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance ID assigned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID that owns this instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID this instance belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Instance name. Must be unique within the project.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Region the instance runs in. One of: `eu`, `us`, `jp`, `au`, `br`. Defaults to the project's `default_region` when omitted. " +
					"Changing this forces a new instance (the management API does not support cross-region moves).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"sync_config_content": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Sync config YAML (bucket definitions or streams). Omit to let CI/CD or the dashboard manage the sync config independently — " +
					"Terraform will read back whatever is currently deployed. See https://docs.powersync.com/sync/overview.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Derived status: \"deploying\" while an operation is pending/running, \"active\" once the instance has a URL, otherwise \"provisioning\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// No UseStateForUnknown: `provisioned` is recorded at deploy time, and the
			// API flag it derives from is unreliable. Pinning a (possibly stale) prior
			// value into the plan can trigger "inconsistent result after apply"; letting
			// it plan as "(known after apply)" on changes keeps apply consistent.
			"provisioned": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a sync config has been deployed to this instance. Despite the name, this is not a liveness signal — use `status` or `instance_url` for that.",
			},
			"instance_url": schema.StringAttribute{
				Computed:    true,
				Description: "Public endpoint URL of the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"replication_connection": schema.ListNestedBlock{
				Description: "Source database replication connection. At most one connection is supported per instance. " +
					"Specify either `uri` *or* the individual host/port/user/pass fields — not both.",
				// PowerSync currently supports a single replication connection per instance.
				// The API models connections as a list, so when multi-connection support
				// lands this SizeAtMost(1) can be relaxed without a schema-shape change.
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Source database type. One of: `postgresql`, `mongodb`, `mysql`, `mssql`. Determines which other fields apply.",
							Validators: []validator.String{
								stringvalidator.OneOf("postgresql", "mongodb", "mysql", "mssql"),
							},
						},
						"name": schema.StringAttribute{
							Optional:    true,
							Description: "Human-readable display name for this connection. Surfaced in the PowerSync dashboard; has no functional effect.",
						},
						"tag": schema.StringAttribute{
							Optional:    true,
							Description: "Identifier used to reference this connection from the sync config (e.g. when an instance has multiple source DBs). Defaults to `default` server-side. See https://docs.powersync.com/sync/overview.",
						},
						"uri": schema.StringAttribute{
							Optional:    true,
							Description: "Full connection URI (e.g. `postgresql://user:pass@host:5432/db`, `mongodb+srv://...`, `mysql://...`). " +
								"Mutually exclusive with the host/port/username/password/database fields. Applies to all DB types.",
						},
						"hostname": schema.StringAttribute{
							Optional:    true,
							Description: "Database server hostname. Applies to all DB types when `uri` is not used.",
						},
						"port": schema.Int64Attribute{
							Optional:    true,
							Description: "Database server port. Typical defaults: PostgreSQL 5432, MongoDB 27017, MySQL 3306, MSSQL 1433. " +
								"Applies to all DB types when `uri` is not used.",
						},
						"username": schema.StringAttribute{
							Optional:    true,
							Description: "Database user PowerSync connects as. Should be a dedicated replication user with the minimum required privileges, not an admin account. " +
								"Applies to all DB types when `uri` is not used.",
						},
						"password": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "Password for the replication user. Stored server-side as a secret; redacted in plan/apply output. " +
								"Applies to all DB types when `uri` is not used.",
						},
						"database": schema.StringAttribute{
							Optional:    true,
							Description: "Database name within the server (e.g. `postgres` for PostgreSQL, the MongoDB database name, the MySQL schema name). " +
								"Applies to PostgreSQL, MongoDB, and MySQL.",
						},
						"sslmode": schema.StringAttribute{
							Optional:    true,
							Description: "TLS verification mode. PowerSync accepts only `verify-full` (default; verifies cert chain + hostname) and `verify-ca` (verifies cert chain only). " +
								"Weaker modes like `require`/`prefer`/`disable` are rejected. Applies to PostgreSQL and MySQL.",
							Validators: []validator.String{
								stringvalidator.OneOf("verify-full", "verify-ca"),
							},
						},
						"cacert": schema.StringAttribute{
							Optional:    true,
							Description: "PEM-encoded CA certificate used to verify the server cert under `verify-full`/`verify-ca`. " +
								"PowerSync bundles the CA for three managed PostgreSQL providers — Supabase, AWS RDS, and Azure Postgres — so leave this empty for those. " +
								"Supply it for any other source: other Postgres hosts, self-hosted databases, and MySQL. Applies to PostgreSQL and MySQL.",
						},
						"client_certificate": schema.StringAttribute{
							Optional:    true,
							Description: "PEM-encoded client certificate for mutual TLS (mTLS). Pair with `client_private_key`. Applies to PostgreSQL and MySQL.",
						},
						"client_private_key": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "PEM-encoded client private key for mutual TLS (mTLS). Pair with `client_certificate`. Stored server-side as a secret. Applies to PostgreSQL and MySQL.",
						},
						"post_images": schema.StringAttribute{
							Optional:    true,
							Description: "Change-stream `fullDocument` mode. One of: `off` (only the document key), `auto_configure` (PowerSync sets `changeStreamPreAndPostImages` on collections automatically), `read_only` (assume images are already configured upstream). MongoDB only.",
							Validators: []validator.String{
								stringvalidator.OneOf("off", "auto_configure", "read_only"),
							},
						},
						"schema": schema.StringAttribute{
							Optional:    true,
							Description: "Default schema to use for replicated tables (e.g. `dbo`). MSSQL only.",
						},
					},
				},
			},
			"client_auth": schema.ListNestedBlock{
				Description: "Client JWT authentication configuration.",
				// A single auth configuration per instance; the API models it as a list.
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"supabase": schema.BoolAttribute{
							Optional:    true,
							Description: "Enable Supabase JWT validation.",
						},
						"jwks_uri": schema.StringAttribute{
							Optional:    true,
							Description: "URL of an external JWKS endpoint.",
						},
						"additional_audiences": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Additional valid JWT audience values.",
						},
						"allow_temporary_tokens": schema.BoolAttribute{
							Optional:    true,
							Description: "Allow temporary tokens (useful for development).",
						},
					},
				},
			},
			"program_version": schema.ListNestedBlock{
				Description: "PowerSync service version constraint.",
				// A single version constraint per instance; the API models it as a list.
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"channel": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Release channel. Defaults to \"stable\".",
						},
						"version_range": schema.StringAttribute{
							Optional:    true,
							Description: "Semver range constraint, e.g. \"^1.0.0\".",
						},
					},
				},
			},
		},
	}
}

// ── Configure ─────────────────────────────────────────────────────────────────

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = c
}

// ── Create ────────────────────────────────────────────────────────────────────

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrgID.ValueString()
	projectID := plan.ProjectID.ValueString()

	// Resolve region: use plan value or fall back to project default.
	region := plan.Region.ValueString()
	if region == "" {
		proj, err := r.client.GetProjectByID(ctx, orgID, projectID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to resolve project region", err.Error())
			return
		}
		if proj == nil {
			resp.Diagnostics.AddError("Project not found", fmt.Sprintf("project %s not found in org %s", projectID, orgID))
			return
		}
		region = proj.DefaultRegion
	}

	// Create the instance (provision only — no config yet).
	instanceID, err := r.client.CreateInstance(ctx, orgID, projectID, plan.Name.ValueString(), region)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// Write ID to state immediately so partial failures are recoverable.
	// Computed fields must be known (not unknown) before saving — use null to
	// indicate "not yet assigned" rather than leaving them as plan unknowns.
	plan.ID = types.StringValue(instanceID)
	plan.Region = types.StringValue(region)
	plan.Status = types.StringNull()
	plan.Provisioned = types.BoolNull()
	plan.InstanceURL = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Test connections before deploying.
	for _, conn := range plan.ReplicationConnections {
		if err := r.client.TestConnection(ctx, orgID, projectID, instanceID, connectionModelToClient(conn)); err != nil {
			resp.Diagnostics.AddError("Connection test failed", err.Error())
			return
		}
	}

	// Deploy.
	deployReq := buildDeployRequest(plan, instanceID, orgID, projectID)
	operationID, err := r.client.DeployInstance(ctx, deployReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to deploy instance", err.Error())
		return
	}

	if err := r.client.WaitForOperation(ctx, orgID, projectID, instanceID, operationID, deployTimeout); err != nil {
		resp.Diagnostics.AddError("Deploy did not complete", err.Error())
		return
	}

	// Refresh status. Record `provisioned` as observed at deploy time; it is not
	// re-read on later refreshes (the API flag is unreliable — see refreshStatus).
	plan.Provisioned = types.BoolValue(r.refreshStatus(ctx, &plan, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	projectID := state.ProjectID.ValueString()
	instanceID := state.ID.ValueString()

	config, err := r.client.GetInstanceConfig(ctx, orgID, projectID, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	if config == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(config.Name)

	if config.Config != nil {
		state.Region = types.StringValue(config.Config.Region)
		state.ReplicationConnections = connectionsFromAPI(config.Config.Replication, state.ReplicationConnections)
		state.ClientAuth = clientAuthFromAPI(config.Config.ClientAuth)
	}

	if config.SyncRules != "" {
		state.SyncConfigContent = types.StringValue(config.SyncRules)
	} else {
		state.SyncConfigContent = types.StringNull()
	}

	// program_version: preserve whatever the user configured in state (write-through only).

	// provisioned: preserve the value recorded at the last deploy rather than
	// re-reading the unreliable live flag (which would show phantom drift). Seed
	// it from the API only when state has no prior value — i.e. on import.
	live := r.refreshStatus(ctx, &state, &resp.Diagnostics)
	if state.Provisioned.IsNull() || state.Provisioned.IsUnknown() {
		state.Provisioned = types.BoolValue(live)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrgID.ValueString()
	projectID := plan.ProjectID.ValueString()
	instanceID := state.ID.ValueString()

	plan.ID = state.ID
	plan.Region = state.Region // region is ForceNew if configured, so state == plan here

	// Test connections before deploying.
	for _, conn := range plan.ReplicationConnections {
		if err := r.client.TestConnection(ctx, orgID, projectID, instanceID, connectionModelToClient(conn)); err != nil {
			resp.Diagnostics.AddError("Connection test failed", err.Error())
			return
		}
	}

	deployReq := buildDeployRequest(plan, instanceID, orgID, projectID)
	operationID, err := r.client.DeployInstance(ctx, deployReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to deploy instance", err.Error())
		return
	}

	if err := r.client.WaitForOperation(ctx, orgID, projectID, instanceID, operationID, deployTimeout); err != nil {
		resp.Diagnostics.AddError("Deploy did not complete", err.Error())
		return
	}

	// Record `provisioned` as observed at this deploy; see Create / refreshStatus.
	plan.Provisioned = types.BoolValue(r.refreshStatus(ctx, &plan, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	projectID := state.ProjectID.ValueString()
	instanceID := state.ID.ValueString()

	operationID, err := r.client.DestroyInstance(ctx, orgID, projectID, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to destroy instance", err.Error())
		return
	}

	if err := r.client.WaitForOperation(ctx, orgID, projectID, instanceID, operationID, destroyTimeout); err != nil {
		resp.Diagnostics.AddError("Destroy did not complete", err.Error())
	}
}

// ── Import ────────────────────────────────────────────────────────────────────

// ImportState expects the import ID in the form org_id/project_id/instance_id.
func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format: org_id/project_id/instance_id",
		)
		return
	}

	config, err := r.client.GetInstanceConfig(ctx, parts[0], parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import instance", err.Error())
		return
	}
	if config == nil {
		resp.Diagnostics.AddError("Instance not found", fmt.Sprintf("no instance %s in project %s / org %s", parts[2], parts[1], parts[0]))
		return
	}

	state := instanceModel{
		ID:        types.StringValue(config.ID),
		OrgID:     types.StringValue(parts[0]),
		ProjectID: types.StringValue(parts[1]),
		Name:      types.StringValue(config.Name),
	}

	if config.Config != nil {
		state.Region = types.StringValue(config.Config.Region)
		state.ReplicationConnections = connectionsFromAPI(config.Config.Replication, nil)
		state.ClientAuth = clientAuthFromAPI(config.Config.ClientAuth)
	} else {
		state.Region = types.StringNull()
		state.ReplicationConnections = []connectionModel{}
		state.ClientAuth = []clientAuthModel{}
	}

	state.ProgramVersion = []programVersionModel{}

	if config.SyncRules != "" {
		state.SyncConfigContent = types.StringValue(config.SyncRules)
	} else {
		state.SyncConfigContent = types.StringNull()
	}

	r.refreshStatus(ctx, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// refreshStatus updates the reliable derived fields (status, instance_url) from
// the live API and returns the API's `provisioned` flag. It deliberately does
// NOT write `provisioned` into state: that flag flip-flops (it reports false even
// for healthy long-lived instances — see client.InstanceStatus), so tracking it
// live produces phantom drift and breaks apply consistency. Callers decide what
// to do with the returned value (Create/Update record it as the deploy-time
// value; Read uses it only to seed an imported resource).
func (r *InstanceResource) refreshStatus(ctx context.Context, state *instanceModel, diags *diag.Diagnostics) bool {
	status, err := r.client.GetInstanceStatus(ctx, state.OrgID.ValueString(), state.ProjectID.ValueString(), state.ID.ValueString())
	if err != nil {
		diags.AddWarning("Could not read instance status", err.Error())
		return false
	}
	state.Status = types.StringValue(status.DeriveStatus())
	state.InstanceURL = stringOrNull(status.InstanceURL)
	return status.Provisioned
}

func buildDeployRequest(plan instanceModel, instanceID, orgID, projectID string) client.DeployInstanceRequest {
	req := client.DeployInstanceRequest{
		OrgID: orgID,
		AppID: projectID,
		ID:    instanceID,
		Name:  plan.Name.ValueString(),
		Config: client.HostedConfig{
			Region: plan.Region.ValueString(),
		},
		ProgramVersion: client.ProgramVersionConstraint{Channel: "stable"},
	}

	if !plan.SyncConfigContent.IsNull() && !plan.SyncConfigContent.IsUnknown() {
		req.SyncRules = plan.SyncConfigContent.ValueString()
	}

	if len(plan.ReplicationConnections) > 0 {
		conns := make([]client.Connection, len(plan.ReplicationConnections))
		for i, c := range plan.ReplicationConnections {
			conns[i] = connectionModelToClient(c)
		}
		req.Config.Replication = &client.ReplicationConfig{Connections: conns}
	}

	if len(plan.ClientAuth) > 0 {
		req.Config.ClientAuth = clientAuthModelToClient(plan.ClientAuth[0])
	}

	if len(plan.ProgramVersion) > 0 {
		pv := plan.ProgramVersion[0]
		req.ProgramVersion.Channel = pv.Channel.ValueString()
		if !pv.VersionRange.IsNull() && !pv.VersionRange.IsUnknown() {
			req.ProgramVersion.VersionRange = pv.VersionRange.ValueString()
		}
	}

	return req
}

func connectionModelToClient(m connectionModel) client.Connection {
	conn := client.Connection{
		Type:              m.Type.ValueString(),
		Name:              strVal(m.Name),
		Tag:               strVal(m.Tag),
		URI:               strVal(m.URI),
		Hostname:          strVal(m.Hostname),
		Username:          strVal(m.Username),
		Database:          strVal(m.Database),
		SSLMode:           strVal(m.SSLMode),
		CACert:            strVal(m.CACert),
		ClientCertificate: strVal(m.ClientCertificate),
		PostImages:        strVal(m.PostImages),
		Schema:            strVal(m.Schema),
	}
	if !m.Port.IsNull() && !m.Port.IsUnknown() {
		v := m.Port.ValueInt64()
		conn.Port = &v
	}
	if !m.Password.IsNull() && !m.Password.IsUnknown() && m.Password.ValueString() != "" {
		conn.Password = &client.HostedSecret{Secret: m.Password.ValueString()}
	}
	if !m.ClientPrivateKey.IsNull() && !m.ClientPrivateKey.IsUnknown() && m.ClientPrivateKey.ValueString() != "" {
		conn.ClientPrivateKey = &client.HostedSecret{Secret: m.ClientPrivateKey.ValueString()}
	}
	return conn
}

func clientAuthModelToClient(m clientAuthModel) *client.ClientAuthConfig {
	auth := &client.ClientAuthConfig{
		Supabase:             m.Supabase.ValueBool(),
		JWKSUri:              strVal(m.JWKSUri),
		AllowTemporaryTokens: m.AllowTemporaryTokens.ValueBool(),
	}
	if !m.AdditionalAudiences.IsNull() && !m.AdditionalAudiences.IsUnknown() {
		var audiences []string
		m.AdditionalAudiences.ElementsAs(context.Background(), &audiences, false)
		auth.AdditionalAudiences = audiences
	}
	return auth
}

func connectionsFromAPI(repl *client.ReplicationConfig, prior []connectionModel) []connectionModel {
	if repl == nil || len(repl.Connections) == 0 {
		return []connectionModel{}
	}
	result := make([]connectionModel, len(repl.Connections))
	for i, apiConn := range repl.Connections {
		m := connectionModel{
			Type:              types.StringValue(apiConn.Type),
			Name:              stringOrNull(apiConn.Name),
			Tag:               stringOrNull(apiConn.Tag),
			URI:               stringOrNull(apiConn.URI),
			Hostname:          stringOrNull(apiConn.Hostname),
			Username:          stringOrNull(apiConn.Username),
			Database:          stringOrNull(apiConn.Database),
			SSLMode:           stringOrNull(apiConn.SSLMode),
			CACert:            stringOrNull(apiConn.CACert),
			ClientCertificate: stringOrNull(apiConn.ClientCertificate),
			PostImages:        stringOrNull(apiConn.PostImages),
			Schema:            stringOrNull(apiConn.Schema),
			Port:              int64OrNull(apiConn.Port),
			// Sensitive fields default to null; preserved from prior state below.
			Password:         types.StringNull(),
			ClientPrivateKey: types.StringNull(),
		}
		// Preserve sensitive values the API does not return.
		if i < len(prior) {
			m.Password = prior[i].Password
			m.ClientPrivateKey = prior[i].ClientPrivateKey
		}
		result[i] = m
	}
	return result
}

func clientAuthFromAPI(auth *client.ClientAuthConfig) []clientAuthModel {
	if auth == nil {
		return []clientAuthModel{}
	}
	m := clientAuthModel{
		Supabase:             boolOrNull(auth.Supabase),
		JWKSUri:              stringOrNull(auth.JWKSUri),
		AllowTemporaryTokens: boolOrNull(auth.AllowTemporaryTokens),
		AdditionalAudiences:  types.ListNull(types.StringType),
	}
	if len(auth.AdditionalAudiences) > 0 {
		listVal, _ := types.ListValueFrom(context.Background(), types.StringType, auth.AdditionalAudiences)
		m.AdditionalAudiences = listVal
	}
	return []clientAuthModel{m}
}

// ── Value helpers ─────────────────────────────────────────────────────────────

func strVal(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func boolOrNull(b bool) types.Bool {
	if !b {
		return types.BoolNull()
	}
	return types.BoolValue(true)
}

func int64OrNull(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}
