package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
	"github.com/powersync/terraform-provider-powersync/internal/datasources"
	"github.com/powersync/terraform-provider-powersync/internal/resources"
)

const (
	defaultAccountsURL   = "https://accounts.powersync.com"
	defaultManagementURL = "https://powersync-api.journeyapps.com"
)

var _ provider.Provider = &PowerSyncProvider{}

type PowerSyncProvider struct {
	version string
}

type providerModel struct {
	AdminToken    types.String `tfsdk:"admin_token"`
	AccountsURL   types.String `tfsdk:"accounts_url"`
	ManagementURL types.String `tfsdk:"management_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PowerSyncProvider{version: version}
	}
}

func (p *PowerSyncProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "powersync"
	resp.Version = p.version
}

func (p *PowerSyncProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage PowerSync Cloud organizations, projects, and instances.",
		Attributes: map[string]schema.Attribute{
			"admin_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PowerSync personal access token. Can also be set via the PS_PAT_TOKEN environment variable.",
			},
			"accounts_url": schema.StringAttribute{
				Optional:    true,
				Description: "PowerSync accounts service URL. Defaults to https://accounts.powersync.com. Override for staging.",
			},
			"management_url": schema.StringAttribute{
				Optional:    true,
				Description: "PowerSync management API URL. Defaults to https://powersync-api.journeyapps.com. Override for staging.",
			},
		},
	}
}

func (p *PowerSyncProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("PS_PAT_TOKEN")
	if !config.AdminToken.IsNull() && !config.AdminToken.IsUnknown() {
		token = config.AdminToken.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing admin token",
			"Set the admin_token provider attribute or the PS_PAT_TOKEN environment variable.",
		)
		return
	}

	accountsURL := defaultAccountsURL
	if !config.AccountsURL.IsNull() && !config.AccountsURL.IsUnknown() {
		accountsURL = config.AccountsURL.ValueString()
	}

	managementURL := defaultManagementURL
	if !config.ManagementURL.IsNull() && !config.ManagementURL.IsUnknown() {
		managementURL = config.ManagementURL.ValueString()
	}

	c := client.New(accountsURL, managementURL, token)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *PowerSyncProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewOrganizationDataSource,
		datasources.NewProjectDataSource,
		datasources.NewProjectsDataSource,
		datasources.NewInstanceDataSource,
		datasources.NewInstancesDataSource,
	}
}

func (p *PowerSyncProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewInstanceResource,
	}
}
