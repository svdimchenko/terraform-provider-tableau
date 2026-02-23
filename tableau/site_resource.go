package tableau

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &siteResource{}
	_ resource.ResourceWithConfigure   = &siteResource{}
	_ resource.ResourceWithImportState = &siteResource{}
)

func NewSiteResource() resource.Resource {
	return &siteResource{}
}

type siteResource struct {
	client *Client
}

type siteResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ContentURL  types.String `tfsdk:"content_url"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (r *siteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Tableau Site",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name for site",
			},
			"content_url": schema.StringAttribute{
				Optional:    true,
				Description: "The subdomain name of the site's URL. This value can contain only characters that are upper or lower case alphabetic characters, numbers, hyphens, or underscores.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := Site{
		Name:       plan.Name.ValueString(),
		ContentURL: plan.ContentURL.ValueString(),
	}

	createdSite, err := r.client.CreateSite(site.Name, site.ContentURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site",
			"Could not create site, unexpected error: "+err.Error(),
		)
		return
	}

	// Add provider user to the created site
	siteClient, err := r.client.NewSiteAuthenticatedClient(createdSite.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating site client",
			"Could not create site client: "+err.Error(),
		)
		return
	}

	currentUser, err := r.client.GetCurrentUser()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting current user",
			"Could not get current user: "+err.Error(),
		)
		return
	}

	_, err = siteClient.CreateUser(currentUser.Email, currentUser.Name, currentUser.FullName, "SiteAdministratorCreator", currentUser.AuthSetting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding user to site",
			"Could not add provider user to site: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(createdSite.ID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := r.client.GetSite(state.ID.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(site.ID)
	state.Name = types.StringValue(site.Name)
	state.ContentURL = types.StringValue(site.ContentURL)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := Site{
		Name:       plan.Name.ValueString(),
		ContentURL: plan.ContentURL.ValueString(),
	}

	_, err := r.client.UpdateSite(plan.ID.ValueString(), site.Name, site.ContentURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Tableau Site",
			"Could not update site, unexpected error: "+err.Error(),
		)
		return
	}

	updatedSite, err := r.client.GetSite(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tableau Site",
			"Could not read Tableau site ID "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.Name = types.StringValue(updatedSite.Name)
	plan.ContentURL = types.StringValue(updatedSite.ContentURL)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Authenticate to the target site before deletion
	siteClient, err := r.client.NewSiteAuthenticatedClient(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error authenticating to site",
			"Could not authenticate to site for deletion: "+err.Error(),
		)
		return
	}

	err = siteClient.DeleteSite(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Tableau Site",
			"Could not delete site, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *siteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Try to find site by name first, then by ID
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
		if site.Name == req.ID || site.ID == req.ID {
			targetSite = &site
			break
		}
	}

	if targetSite == nil {
		resp.Diagnostics.AddError(
			"Site not found",
			"Site with name or ID '"+req.ID+"' not found",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), targetSite.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), targetSite.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_url"), targetSite.ContentURL)...)
}
