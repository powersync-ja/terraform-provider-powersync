package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

type ProjectResource struct {
	client *client.Client
}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type projectModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	Name          types.String `tfsdk:"name"`
	Region        types.String `tfsdk:"region"`
	VCSMode       types.String `tfsdk:"vcs_mode"`
	Trial         types.Bool   `tfsdk:"trial"`
	Locked        types.Bool   `tfsdk:"locked"`
	ForceDestroy  types.Bool   `tfsdk:"force_destroy"`
}

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PowerSync project. Creating a project mirrors the dashboard's create flow; most internal fields (template_id, source, features, vcs_mode) are hardcoded.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Project ID assigned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID that will own the project. Changing this forces a new project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable project name. Only mutable field on update.",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Default region for instances created under this project. One of: `eu`, `us`, `jp`, `au`, `br`. " +
					"Surfaces as `default_region` on the `powersync_project` data source. Changing this forces a new project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vcs_mode": schema.StringAttribute{
				Computed:    true,
				Description: "Version control mode for the sync config. Always `BASIC` today.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"trial": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is on a trial plan.",
			},
			"locked": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is locked.",
			},
			"force_destroy": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, deleting this project will cascade-destroy any instances under it that are NOT managed by this Terraform configuration. Defaults to false; destroy is refused if non-tracked instances exist.",
			},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proj, err := r.client.CreateProject(ctx, plan.OrgID.ValueString(), plan.Name.ValueString(), plan.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", err.Error())
		return
	}

	applyProjectToModel(proj, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proj, err := r.client.GetProjectByID(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read project", err.Error())
		return
	}
	if proj == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyProjectToModel(proj, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proj, err := r.client.UpdateProject(ctx, state.ID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update project", err.Error())
		return
	}

	applyProjectToModel(proj, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	projectID := state.ID.ValueString()

	if !state.ForceDestroy.ValueBool() {
		instances, err := r.client.ListInstances(ctx, orgID, projectID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to check for instances before destroy", err.Error())
			return
		}
		if len(instances) > 0 {
			resp.Diagnostics.AddError(
				"Project has remaining instances",
				fmt.Sprintf("Project %s has %d instance(s) that are not being destroyed by this Terraform run. Set force_destroy = true to cascade-delete them, or destroy them first.", projectID, len(instances)),
			)
			return
		}
	}

	if err := r.client.DeleteProject(ctx, orgID, projectID); err != nil {
		resp.Diagnostics.AddError("Failed to delete project", err.Error())
		return
	}
}

// ImportState expects the import ID in the form org_id/project_id.
func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format: org_id/project_id",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func applyProjectToModel(p *client.Project, m *projectModel) {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	m.Region = types.StringValue(p.DefaultRegion)
	m.VCSMode = types.StringValue(p.VCSMode)
	m.Trial = types.BoolValue(p.Trial)
	m.Locked = types.BoolValue(p.Locked)
}
