package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ datasource.DataSource = &ProjectsDataSource{}

type ProjectsDataSource struct {
	client *client.Client
}

func NewProjectsDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

type projectsModel struct {
	OrgID    types.String        `tfsdk:"org_id"`
	Total    types.Int64         `tfsdk:"total"`
	Projects []projectsItemModel `tfsdk:"projects"`
}

type projectsItemModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	DefaultRegion types.String `tfsdk:"default_region"`
	VCSMode       types.String `tfsdk:"vcs_mode"`
	Trial         types.Bool   `tfsdk:"trial"`
	Locked        types.Bool   `tfsdk:"locked"`
}

func (d *ProjectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all PowerSync projects in an organization. Follows pagination to return every project.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID whose projects to list.",
			},
			"total": schema.Int64Attribute{
				Computed:    true,
				Description: "Server-reported total project count. Should match length(projects) since pagination is followed end-to-end.",
			},
			"projects": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Projects in the organization.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Project ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Project name.",
						},
						"default_region": schema.StringAttribute{
							Computed:    true,
							Description: "Default region for instances created under this project. One of: `eu`, `us`, `jp`, `au`, `br`.",
						},
						"vcs_mode": schema.StringAttribute{
							Computed:    true,
							Description: "Version control mode for the sync config: `BASIC` (dashboard/API/Terraform managed) or `GIT` (sourced from a git repo).",
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
				},
			},
		},
	}
}

func (d *ProjectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projects, total, err := d.client.ListProjects(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list projects", err.Error())
		return
	}

	items := make([]projectsItemModel, len(projects))
	for i, p := range projects {
		items[i] = projectsItemModel{
			ID:            types.StringValue(p.ID),
			Name:          types.StringValue(p.Name),
			DefaultRegion: types.StringValue(p.DefaultRegion),
			VCSMode:       types.StringValue(p.VCSMode),
			Trial:         types.BoolValue(p.Trial),
			Locked:        types.BoolValue(p.Locked),
		}
	}

	state.Total = types.Int64Value(int64(total))
	state.Projects = items
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
