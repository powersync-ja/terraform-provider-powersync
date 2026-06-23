package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ datasource.DataSource = &InstancesDataSource{}

type InstancesDataSource struct {
	client *client.Client
}

func NewInstancesDataSource() datasource.DataSource {
	return &InstancesDataSource{}
}

type instancesModel struct {
	OrgID     types.String         `tfsdk:"org_id"`
	ProjectID types.String         `tfsdk:"project_id"`
	Instances []instancesItemModel `tfsdk:"instances"`
}

type instancesItemModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	HasConfig  types.Bool   `tfsdk:"has_config"`
	Deployable types.Bool   `tfsdk:"deployable"`
}

func (d *InstancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instances"
}

func (d *InstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all PowerSync instances in a project. Returns a lightweight summary per instance; for region, URL, status, or sync config use the `powersync_instance` data source on a specific id.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID that owns the project.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID whose instances to list.",
			},
			"instances": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Instances in the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Instance ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Instance name.",
						},
						"has_config": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the instance has been configured (region, replication, client auth). `false` for newly created instances that have never been deployed.",
						},
						"deployable": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the instance is currently in a deployed state. `false` for new, deactivated, or destroyed instances.",
						},
					},
				},
			},
		},
	}
}

func (d *InstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instancesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instances, err := d.client.ListInstances(ctx, state.OrgID.ValueString(), state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list instances", err.Error())
		return
	}

	items := make([]instancesItemModel, len(instances))
	for i, inst := range instances {
		items[i] = instancesItemModel{
			ID:         types.StringValue(inst.ID),
			Name:       types.StringValue(inst.Name),
			HasConfig:  types.BoolValue(inst.HasConfig),
			Deployable: types.BoolValue(inst.Deployable),
		}
	}

	state.Instances = items
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
