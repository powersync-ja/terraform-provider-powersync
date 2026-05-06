package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ datasource.DataSource = &OrganizationDataSource{}

type OrganizationDataSource struct {
	client *client.Client
}

type organizationModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

func (d *OrganizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a PowerSync organization by ID. The name is populated from the API response.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Organization name, populated from the API.",
			},
		},
	}
}

func (d *OrganizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state organizationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := d.client.GetOrganizationByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch organization", err.Error())
		return
	}

	state.Name = types.StringValue(org.Label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
