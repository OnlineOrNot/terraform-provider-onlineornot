package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_status_page"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &StatusPageResource{}
var _ resource.ResourceWithImportState = &StatusPageResource{}

func NewStatusPageResource() resource.Resource {
	return &StatusPageResource{}
}

// StatusPageResource defines the resource implementation.
type StatusPageResource struct {
	client *client.Client
}

func (r *StatusPageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *StatusPageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_status_page.StatusPageResourceSchema(ctx)
}

func (r *StatusPageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *StatusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_status_page.StatusPageModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp := &client.StatusPage{
		Name:                  data.Name.ValueString(),
		Subdomain:             data.Subdomain.ValueString(),
		Description:           data.Description.ValueString(),
		CustomDomain:          data.CustomDomain.ValueString(),
		Password:              data.Password.ValueString(),
		HideFromSearchEngines: data.HideFromSearchEngines.ValueBool(),
	}

	if !data.AllowedIps.IsNull() {
		data.AllowedIps.ElementsAs(ctx, &sp.AllowedIPs, false)
	}

	created, err := r.client.CreateStatusPage(sp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	// Set computed fields to null to avoid "unknown after apply" errors
	if data.AllowedIps.IsUnknown() {
		data.AllowedIps = types.ListNull(types.StringType)
	}
	if data.CustomDomain.IsUnknown() {
		data.CustomDomain = types.StringNull()
	}
	if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	if data.Password.IsUnknown() {
		data.Password = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_status_page.StatusPageModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := r.client.GetStatusPage(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status page, got error: %s", err))
		return
	}

	data.Id = types.StringValue(sp.ID)
	data.Name = types.StringValue(sp.Name)
	data.Subdomain = types.StringValue(sp.Subdomain)
	data.Description = types.StringValue(sp.Description)
	data.CustomDomain = types.StringValue(sp.CustomDomain)
	data.HideFromSearchEngines = types.BoolValue(sp.HideFromSearchEngines)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_status_page.StatusPageModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp := &client.StatusPage{
		Name:                  data.Name.ValueString(),
		Subdomain:             data.Subdomain.ValueString(),
		Description:           data.Description.ValueString(),
		CustomDomain:          data.CustomDomain.ValueString(),
		Password:              data.Password.ValueString(),
		HideFromSearchEngines: data.HideFromSearchEngines.ValueBool(),
	}

	if !data.AllowedIps.IsNull() {
		data.AllowedIps.ElementsAs(ctx, &sp.AllowedIPs, false)
	}

	_, err := r.client.UpdateStatusPage(data.Id.ValueString(), sp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_status_page.StatusPageModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStatusPage(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete status page, got error: %s", err))
		return
	}
}

func (r *StatusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
