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
				Description: "Username of the user (will be imported from Active Directory)",
			},
			"site": schema.StringAttribute{
				Optional:    true,
				Description: "Site ID where the user should be added (omit for default site)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

	// Handle ServerAdministrator role special case
	role := plan.Role.ValueString()
	if role == "ServerAdministrator" {
		// First create with SiteAdministratorCreator
		createdUser, err := siteClient.CreateUser("", plan.Name.ValueString(), "", "SiteAdministratorCreator", "SAML")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site user",
				"Could not create site user: "+err.Error(),
			)
			return
		}

		// Then update to ServerAdministrator
		_, err = siteClient.UpdateUser(createdUser.ID, "", plan.Name.ValueString(), "", "ServerAdministrator", "SAML")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to ServerAdministrator",
				"Could not update site user to ServerAdministrator: "+err.Error(),
			)
			return
		}
	} else {
		// Create user with specified role
		_, err := siteClient.CreateUser("", plan.Name.ValueString(), "", role, "SAML")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating site user",
				"Could not create site user: "+err.Error(),
			)
			return
		}
	}

	plan.ID = types.StringValue(GetCombinedID(plan.Name.ValueString(), siteID))
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
		_, err = siteClient.UpdateUser(targetUser.ID, "", targetUser.Name, "", "SiteAdministratorCreator", "SAML")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to SiteAdministratorCreator",
				"Could not update site user: "+err.Error(),
			)
			return
		}

		_, err = siteClient.UpdateUser(targetUser.ID, "", targetUser.Name, "", "ServerAdministrator", "SAML")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating site user to ServerAdministrator",
				"Could not update site user: "+err.Error(),
			)
			return
		}
	} else {
		// Update user with specified role
		_, err = siteClient.UpdateUser(targetUser.ID, "", targetUser.Name, "", role, "SAML")
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
	// Import format: "username:siteID" or "username:siteName" or "username:" for default site
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format 'username:siteID' or 'username:siteName' or 'username:' for default site",
		)
		return
	}

	userName := parts[0]
	siteIdentifier := parts[1]

	// If site identifier is empty, use default site
	var targetSiteID string
	if siteIdentifier == "" {
		targetSiteID = r.client.SiteID
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
	}

	importID := GetCombinedID(userName, targetSiteID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), userName)...)
	if siteIdentifier != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), targetSiteID)...)
	}
}
