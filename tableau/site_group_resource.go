package tableau

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

var (
	_ resource.Resource                = &siteGroupResource{}
	_ resource.ResourceWithConfigure   = &siteGroupResource{}
	_ resource.ResourceWithImportState = &siteGroupResource{}
)

func NewSiteGroupResource() resource.Resource {
	return &siteGroupResource{}
}

type siteGroupResource struct {
	client *Client
}

type siteGroupResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Site             types.String `tfsdk:"site"`
	DomainName       types.String `tfsdk:"domain_name"`
	MinimumSiteRole  types.String `tfsdk:"minimum_site_role"`
	GrantLicenseMode types.String `tfsdk:"grant_license_mode"`
}

func (r *siteGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_group"
}

func (r *siteGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Active Directory group imports to Tableau sites.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the Active Directory group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"site": schema.StringAttribute{
				Optional:    true,
				Description: "Site ID where the group should be imported (omit for default site)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain_name": schema.StringAttribute{
				Optional:    true,
				Description: "Active Directory domain name (auto-extracted from name if not provided)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"minimum_site_role": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum site role for the group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"grant_license_mode": schema.StringAttribute{
				Optional:    true,
				Description: "Grant license mode (onLogin or onSync)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *siteGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteGroupResourceModel
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

	domainName := plan.DomainName.ValueString()
	minimumSiteRole := plan.MinimumSiteRole.ValueString()
	grantLicenseMode := plan.GrantLicenseMode.ValueString()

	createdGroup, err := siteClient.ImportGroup(plan.Name.ValueString(), domainName, minimumSiteRole, grantLicenseMode, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing group",
			"Could not import group: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(GetCombinedID(createdGroup.ID, siteID))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, siteID := GetIDsFromCombinedID(state.ID.ValueString())

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

	group, err := siteClient.GetGroup(groupID)
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(group.Name)

	if group.Import != nil {
		if group.Import.DomainName != nil {
			state.DomainName = types.StringValue(*group.Import.DomainName)
		} else {
			state.DomainName = types.StringNull()
		}
		if group.Import.MinimumSiteRole != nil {
			state.MinimumSiteRole = types.StringValue(*group.Import.MinimumSiteRole)
		} else {
			state.MinimumSiteRole = types.StringNull()
		}
		if group.Import.GrantLicenseMode != nil {
			state.GrantLicenseMode = types.StringValue(*group.Import.GrantLicenseMode)
		} else {
			state.GrantLicenseMode = types.StringNull()
		}
	} else {
		state.DomainName = types.StringNull()
		state.MinimumSiteRole = types.StringNull()
		state.GrantLicenseMode = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Site groups cannot be updated. Please delete and recreate the resource.",
	)
}

func (r *siteGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, siteID := GetIDsFromCombinedID(state.ID.ValueString())

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

	err := siteClient.DeleteGroup(groupID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group",
			"Could not delete group: "+err.Error(),
		)
		return
	}
}

func (r *siteGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "groupName:siteID", "groupName:siteName", or "groupName" for default site
	parts := strings.Split(req.ID, ":")
	if len(parts) > 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format 'groupName:siteID', 'groupName:siteName', or 'groupName' for default site",
		)
		return
	}

	groupName := parts[0]
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

	groups, err := siteClient.GetGroups()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting groups",
			"Could not get groups: "+err.Error(),
		)
		return
	}

	var targetGroup *Group
	for _, group := range groups {
		if group.Name == groupName {
			targetGroup = &group
			break
		}
	}

	if targetGroup == nil {
		resp.Diagnostics.AddError(
			"Group not found",
			"Group '"+groupName+"' not found in site",
		)
		return
	}

	importID := GetCombinedID(targetGroup.ID, targetSiteID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), groupName)...)
	if siteIdentifier != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), targetSiteID)...)
	}

	// Set import-related attributes from the group
	if targetGroup.Import != nil {
		if targetGroup.Import.DomainName != nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), *targetGroup.Import.DomainName)...)
		}
		if targetGroup.Import.MinimumSiteRole != nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("minimum_site_role"), *targetGroup.Import.MinimumSiteRole)...)
		}
		if targetGroup.Import.GrantLicenseMode != nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("grant_license_mode"), *targetGroup.Import.GrantLicenseMode)...)
		}
	}
}
