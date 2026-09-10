package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	for _, name := range []string{"description", "custom_domain", "password"} {
		a := resp.Schema.Attributes[name].(schema.StringAttribute)
		a.Computed = false
		if name == "password" {
			a.Sensitive = true
		}
		resp.Schema.Attributes[name] = a
	}
	ips := resp.Schema.Attributes["allowed_ips"].(schema.ListAttribute)
	ips.Computed = false
	resp.Schema.Attributes["allowed_ips"] = ips
	id := resp.Schema.Attributes["id"].(schema.StringAttribute)
	id.Optional = false
	id.PlanModifiers = append(id.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["id"] = id

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

	sp := statusPageInput(ctx, &data)

	created, err := r.client.CreateStatusPage(sp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)
	if data.HideFromSearchEngines.IsUnknown() {
		data.HideFromSearchEngines = types.BoolValue(created.HideFromSearchEngines)
	}

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
	// The create endpoint ignores these inputs. Apply them through the update
	// endpoint after saving the ID so a failed update does not orphan the page.
	if sp.Description != nil || sp.AllowedIPs != nil || sp.HideFromSearchEngines != nil {
		if _, err := r.client.UpdateStatusPage(created.ID, sp); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Status page created, but unable to configure settings: %s", err))
		}
	}

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
	data.Description = statusPageString(sp.Description, data.Description)
	if normalizedStatusPageDomain(sp.CustomDomain) != normalizedStatusPageDomain(data.CustomDomain.ValueString()) {
		data.CustomDomain = statusPageString(sp.CustomDomain, data.CustomDomain)
	}
	data.HideFromSearchEngines = types.BoolValue(sp.HideFromSearchEngines)
	if len(sp.AllowedIPs) == 0 && data.AllowedIps.IsNull() {
		data.AllowedIps = types.ListNull(types.StringType)
	} else {
		ips := sp.AllowedIPs
		if ips == nil {
			ips = []string{}
		}
		data.AllowedIps, _ = types.ListValueFrom(ctx, types.StringType, ips)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior resource_status_page.StatusPageModel

	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp := statusPageInput(ctx, &data)
	if data.Password.IsNull() && !prior.Password.IsNull() {
		empty := ""
		sp.Password = &empty
	}

	// Removal from configuration must clear fields whose API omission preserves them.
	if sp.Description == nil {
		empty := ""
		sp.Description = &empty
	}
	if sp.AllowedIPs == nil {
		empty := []string{}
		sp.AllowedIPs = &empty
	}
	updated, err := r.client.UpdateStatusPage(data.Id.ValueString(), sp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page, got error: %s", err))
		return
	}

	if data.HideFromSearchEngines.IsUnknown() {
		data.HideFromSearchEngines = types.BoolValue(updated.HideFromSearchEngines)
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

func statusPageInput(ctx context.Context, data *resource_status_page.StatusPageModel) *client.StatusPageInput {
	input := &client.StatusPageInput{Name: data.Name.ValueString(), Subdomain: data.Subdomain.ValueString()}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		v := data.Description.ValueString()
		input.Description = &v
	}
	if !data.CustomDomain.IsNull() && !data.CustomDomain.IsUnknown() {
		v := data.CustomDomain.ValueString()
		input.CustomDomain = &v
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		v := data.Password.ValueString()
		input.Password = &v
	}
	if !data.HideFromSearchEngines.IsNull() && !data.HideFromSearchEngines.IsUnknown() {
		v := data.HideFromSearchEngines.ValueBool()
		input.HideFromSearchEngines = &v
	}
	if !data.AllowedIps.IsNull() && !data.AllowedIps.IsUnknown() {
		ips := make([]string, 0, len(data.AllowedIps.Elements()))
		data.AllowedIps.ElementsAs(ctx, &ips, false)
		input.AllowedIPs = &ips
	}
	return input
}

// The API returns null for absent strings and IP lists. Preserve configured
// empty values, but retain null for omitted values and imports.
func statusPageString(value string, prior types.String) types.String {
	if value == "" && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func normalizedStatusPageDomain(domain string) string {
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	return strings.TrimSuffix(domain, "/")
}
