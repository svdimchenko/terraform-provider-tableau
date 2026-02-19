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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &siteUserResource{}
	_ resource.ResourceWithConfigure   = &siteUserResource{}
	_ resource.ResourceWithImportState = &siteUserResource{}
)

func NewSiteUserResource() resource.Resource {
	return &siteUserResource{}
}

type siteUserResource struct {
	client *Client
}

type siteUserResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Site        types.String `tfsdk:"site"`
	Role        types.String `tfsdk:"role"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (r *siteUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_user"
}

func (r *siteUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Tableau Site User",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Username of the user",
			},
			"site": schema.StringAttribute{
				Required:    true,
				Description: "Site ID where the user should be added",
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Site role for the user",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						"Creator",
						"Explorer",
						"Interactor",
						"Publisher",
						"ExplorerCanPublish",
						"ServerAdministrator",
						"SiteAdministratorExplorer",
						"SiteAdministratorCreator",
						"Unlicensed",
						"Viewer",
					}...),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *siteUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteUserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that user is not the provider user
	if plan.Name.ValueString() == r.client.Username {
		resp.Diagnostics.AddError(
			"Invalid user",
			"Cannot manage the provider user with site_user resource",
		)
		return
	}

	// Create site-specific client
	siteClient, err := r.client.NewSiteAuthenticatedClient(plan.Site.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site client",
			"Could not create site client: "+err.Error(),
		)
		return
	}

	// Get user details from main client
	users, err := r.client.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting users",
			"Could not get users: "+err.Error(),
		)
		return
	}

	var targetUser *User
	for _, user := range users {
		if user.Name == plan.Name.ValueString() {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		resp.Diagnostics.AddError(
			"User not found",
			"User "+plan.Name.ValueString()+" not found",
		)
		return
	}

	// Handle ServerAdministrator role special case
	role := plan.Role.ValueString()
	if role == "ServerAdministrator" {
		// First create with SiteAdministratorCreator
		createdUser, err := siteClient.CreateUser(targetUser.Email, targetUser.Name, targetUser.FullName, "SiteAdministratorCreator", targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site user",
				"Could not create site user: "+err.Error(),
			)
			return
		}

		// Then update to ServerAdministrator
		_, err = siteClient.UpdateUser(createdUser.ID, targetUser.Email, targetUser.Name, targetUser.FullName, "ServerAdministrator", targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to ServerAdministrator",
				"Could not update site user to ServerAdministrator: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(GetCombinedID(plan.Name.ValueString(), plan.Site.ValueString()))
	} else {
		// Create user with specified role
		_, err := siteClient.CreateUser(targetUser.Email, targetUser.Name, targetUser.FullName, role, targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site user",
				"Could not create site user: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(GetCombinedID(plan.Name.ValueString(), plan.Site.ValueString()))
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userName, siteID := GetIDsFromCombinedID(state.ID.ValueString())

	// Create site-specific client
	siteClient, err := r.client.NewSiteAuthenticatedClient(siteID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site client",
			"Could not create site client: "+err.Error(),
		)
		return
	}

	// Get users from site
	users, err := siteClient.GetUsers()
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var targetUser *User
	for _, user := range users {
		if user.Name == userName {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(targetUser.Name)
	state.Site = types.StringValue(siteID)
	state.Role = types.StringValue(targetUser.SiteRole)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteUserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userName, siteID := GetIDsFromCombinedID(plan.ID.ValueString())

	// Create site-specific client
	siteClient, err := r.client.NewSiteAuthenticatedClient(siteID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site client",
			"Could not create site client: "+err.Error(),
		)
		return
	}

	// Get user from site
	users, err := siteClient.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting site users",
			"Could not get site users: "+err.Error(),
		)
		return
	}

	var targetUser *User
	for _, user := range users {
		if user.Name == userName {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		resp.Diagnostics.AddError(
			"User not found in site",
			"User "+userName+" not found in site",
		)
		return
	}

	// Handle ServerAdministrator role special case
	role := plan.Role.ValueString()
	if role == "ServerAdministrator" && targetUser.SiteRole != "ServerAdministrator" {
		// First update to SiteAdministratorCreator, then to ServerAdministrator
		_, err = siteClient.UpdateUser(targetUser.ID, targetUser.Email, targetUser.Name, targetUser.FullName, "SiteAdministratorCreator", targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to SiteAdministratorCreator",
				"Could not update site user: "+err.Error(),
			)
			return
		}

		_, err = siteClient.UpdateUser(targetUser.ID, targetUser.Email, targetUser.Name, targetUser.FullName, "ServerAdministrator", targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to ServerAdministrator",
				"Could not update site user: "+err.Error(),
			)
			return
		}
	} else {
		// Update user with specified role
		_, err = siteClient.UpdateUser(targetUser.ID, targetUser.Email, targetUser.Name, targetUser.FullName, role, targetUser.AuthSetting)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user",
				"Could not update site user: "+err.Error(),
			)
			return
		}
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userName, siteID := GetIDsFromCombinedID(state.ID.ValueString())

	// Create site-specific client
	siteClient, err := r.client.NewSiteAuthenticatedClient(siteID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site client",
			"Could not create site client: "+err.Error(),
		)
		return
	}

	// Get user from site
	users, err := siteClient.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting site users",
			"Could not get site users: "+err.Error(),
		)
		return
	}

	var targetUser *User
	for _, user := range users {
		if user.Name == userName {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		// User already doesn't exist, consider it deleted
		return
	}

	err = siteClient.DeleteUser(targetUser.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting site user",
			"Could not delete site user: "+err.Error(),
		)
		return
	}
}

func (r *siteUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "username:siteID" or "username:siteName"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format 'username:siteID' or 'username:siteName'",
		)
		return
	}

	userName := parts[0]
	siteIdentifier := parts[1]

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

	importID := GetCombinedID(userName, targetSite.ID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), userName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), targetSite.ID)...)
}
