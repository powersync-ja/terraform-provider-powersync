package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ datasource.DataSource = &ProjectDataSource{}

type ProjectDataSource struct {
	client *client.Client
}

type projectModel struct {
	OrgID         types.String `tfsdk:"org_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	DefaultRegion types.String `tfsdk:"default_region"`
	VCSMode       types.String `tfsdk:"vcs_mode"`
	Trial         types.Bool   `tfsdk:"trial"`
	Locked        types.Bool   `tfsdk:"locked"`
}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

func (d *ProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a PowerSync project by ID or name within an organization. Exactly one of id or name must be set.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the organization that owns the project.",
			},
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Project ID.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Project name.",
			},
			"default_region": schema.StringAttribute{
				Computed:    true,
				Description: "Default region for instances created in this project (e.g. \"eu\", \"us\").",
			},
			"vcs_mode": schema.StringAttribute{
				Computed:    true,
				Description: "Version control mode for sync rules. BASIC means rules are managed via the dashboard/API; GIT means rules are sourced from a git repository.",
			},
			"trial": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is on a trial plan.",
			},
			"locked": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is locked.",
			},
		},
	}
}

func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !state.ID.IsNull() && !state.ID.IsUnknown()
	nameSet := !state.Name.IsNull() && !state.Name.IsUnknown()

	if idSet == nameSet {
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Exactly one of id or name must be set.",
		)
		return
	}

	orgID := state.OrgID.ValueString()
	var proj *client.Project
	var err error

	if idSet {
		proj, err = d.client.GetProjectByID(ctx, orgID, state.ID.ValueString())
	} else {
		proj, err = d.client.GetProjectByName(ctx, orgID, state.Name.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch project", err.Error())
		return
	}
	if proj == nil {
		resp.Diagnostics.AddError(
			"Project not found",
			"No project matched the given id or name within the organization.",
		)
		return
	}

	state.ID = types.StringValue(proj.ID)
	state.Name = types.StringValue(proj.Name)
	state.DefaultRegion = types.StringValue(proj.DefaultRegion)
	state.VCSMode = types.StringValue(proj.VCSMode)
	state.Trial = types.BoolValue(proj.Trial)
	state.Locked = types.BoolValue(proj.Locked)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
