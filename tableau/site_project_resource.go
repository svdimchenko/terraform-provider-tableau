package tableau

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &siteProjectResource{}
	_ resource.ResourceWithConfigure   = &siteProjectResource{}
	_ resource.ResourceWithImportState = &siteProjectResource{}
)

func NewSiteProjectResource() resource.Resource {
	return &siteProjectResource{}
}

type siteProjectResource struct {
	client *Client
}

type siteProjectResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Site               types.String `tfsdk:"site"`
	ParentProjectID    types.String `tfsdk:"parent_project_id"`
	Description        types.String `tfsdk:"description"`
	ContentPermissions types.String `tfsdk:"content_permissions"`
	OwnerID            types.String `tfsdk:"owner_id"`
	LastUpdated        types.String `tfsdk:"last_updated"`
}

func (r *siteProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_project"
}

func (r *siteProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a project within a Tableau site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name for project",
			},
			"site": schema.StringAttribute{
				Optional:    true,
				Description: "Site ID where the project should be created (omit for default site)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_project_id": schema.StringAttribute{
				Optional:    true,
				Description: "Identifier for the parent project",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description for the project",
				Default:     stringdefault.StaticString(""),
			},
			"content_permissions": schema.StringAttribute{
				Required:    true,
				Description: "Permissions for the project content",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						"LockedToProject",
						"ManagedByOwner",
						"LockedToProjectWithoutNested",
					}...),
				},
			},
			"owner_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Identifier for the project owner",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *siteProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteProjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var siteClient *Client
	var siteID string

	if plan.Site.IsNull() || plan.Site.ValueString() == "" {
		siteClient = r.client
		siteID = r.client.SiteID
	} else {
		siteID = plan.Site.ValueString()
		var err error
		siteClient, err = r.client.NewSiteAuthenticatedClient(siteID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site client",
				"Could not create site client: "+err.Error(),
			)
			return
		}
	}

	createdProject, err := siteClient.CreateProject(
		plan.Name.ValueString(),
		getProjectIDFromCombinedID(plan.ParentProjectID.ValueString()),
		plan.Description.ValueString(),
		plan.ContentPermissions.ValueString(),
		plan.OwnerID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(GetCombinedID(createdProject.ID, siteID))
	plan.OwnerID = types.StringValue(createdProject.Owner.ID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteProjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, siteID := GetIDsFromCombinedID(state.ID.ValueString())

	var siteClient *Client
	if siteID == r.client.SiteID {
		siteClient = r.client
	} else {
		var err error
		siteClient, err = r.client.NewSiteAuthenticatedClient(siteID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site client",
				"Could not create site client: "+err.Error(),
			)
			return
		}
	}

	project, err := siteClient.GetProject(projectID)
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(project.Name)
	if project.ParentProjectID != "" {
		state.ParentProjectID = types.StringValue(GetCombinedID(project.ParentProjectID, siteID))
	}
	state.OwnerID = types.StringValue(project.Owner.ID)
	if project.Description != "" {
		state.Description = types.StringValue(project.Description)
	}
	state.ContentPermissions = types.StringValue(project.ContentPermissions)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteProjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, siteID := GetIDsFromCombinedID(plan.ID.ValueString())

	var siteClient *Client
	if siteID == r.client.SiteID {
		siteClient = r.client
	} else {
		var err error
		siteClient, err = r.client.NewSiteAuthenticatedClient(siteID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site client",
				"Could not create site client: "+err.Error(),
			)
			return
		}
	}

	updatedProject, err := siteClient.UpdateProject(
		projectID,
		plan.Name.ValueString(),
		getProjectIDFromCombinedID(plan.ParentProjectID.ValueString()),
		plan.Description.ValueString(),
		plan.ContentPermissions.ValueString(),
		plan.OwnerID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Could not update project: "+err.Error(),
		)
		return
	}

	plan.Name = types.StringValue(updatedProject.Name)
	if updatedProject.ParentProjectID != "" {
		plan.ParentProjectID = types.StringValue(GetCombinedID(updatedProject.ParentProjectID, siteID))
	}
	plan.OwnerID = types.StringValue(updatedProject.Owner.ID)
	plan.ContentPermissions = types.StringValue(updatedProject.ContentPermissions)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteProjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, siteID := GetIDsFromCombinedID(state.ID.ValueString())

	var siteClient *Client
	if siteID == r.client.SiteID {
		siteClient = r.client
	} else {
		var err error
		siteClient, err = r.client.NewSiteAuthenticatedClient(siteID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site client",
				"Could not create site client: "+err.Error(),
			)
			return
		}
	}

	err := siteClient.DeleteProject(projectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			"Could not delete project: "+err.Error(),
		)
		return
	}
}

func (r *siteProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *Client, got: %T. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *siteProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "projectName:siteID", "projectName:siteName", or "projectName" for default site
	parts := strings.Split(req.ID, ":")
	if len(parts) > 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format 'projectName:siteID', 'projectName:siteName', or 'projectName' for default site",
		)
		return
	}

	projectName := parts[0]
	var siteIdentifier string
	if len(parts) == 2 {
		siteIdentifier = parts[1]
	}

	// If no site identifier or empty, use default site
	var targetSiteID string
	var siteClient *Client
	if siteIdentifier == "" {
		targetSiteID = r.client.SiteID
		siteClient = r.client
	} else {
		// Try to find site by name or ID
		sites, err := r.client.GetSites()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting sites",
				"Could not get sites: "+err.Error(),
			)
			return
		}

		var targetSite *Site
		for _, site := range sites {
			if site.Name == siteIdentifier || site.ID == siteIdentifier {
				targetSite = &site
				break
			}
		}

		if targetSite == nil {
			resp.Diagnostics.AddError(
				"Site not found",
				"Site with name or ID '"+siteIdentifier+"' not found",
			)
			return
		}
		targetSiteID = targetSite.ID

		var err2 error
		siteClient, err2 = r.client.NewSiteAuthenticatedClient(targetSiteID)
		if err2 != nil {
			resp.Diagnostics.AddError(
				"Error creating site client",
				"Could not create site client: "+err2.Error(),
			)
			return
		}
	}

	projects, err := siteClient.GetProjects()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting projects",
			"Could not get projects: "+err.Error(),
		)
		return
	}

	var targetProject *Project
	for _, project := range projects {
		if project.Name == projectName {
			targetProject = &project
			break
		}
	}

	if targetProject == nil {
		resp.Diagnostics.AddError(
			"Project not found",
			"Project '"+projectName+"' not found in site",
		)
		return
	}

	importID := GetCombinedID(targetProject.ID, targetSiteID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), projectName)...)
	if siteIdentifier != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), targetSiteID)...)
	}
}

// getProjectIDFromCombinedID extracts the project ID from a combined ID or returns the input if it's already a simple ID.
func getProjectIDFromCombinedID(id string) string {
	if id == "" {
		return ""
	}
	// Check if it's a combined ID (contains colon)
	if strings.Contains(id, ":") {
		projectID, _ := GetIDsFromCombinedID(id)
		return projectID
	}
	return id
}
