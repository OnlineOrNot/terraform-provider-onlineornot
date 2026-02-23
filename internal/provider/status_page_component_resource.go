package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_status_page_component"
)

var _ resource.Resource = &StatusPageComponentResource{}
var _ resource.ResourceWithImportState = &StatusPageComponentResource{}

func NewStatusPageComponentResource() resource.Resource {
	return &StatusPageComponentResource{}
}

type StatusPageComponentResource struct {
	client *client.Client
}

func (r *StatusPageComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_component"
}

func (r *StatusPageComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_status_page_component.StatusPageComponentResourceSchema(ctx)
}

func (r *StatusPageComponentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *StatusPageComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_status_page_component.StatusPageComponentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	comp := &client.StatusPageComponent{
		Name:   data.Name.ValueString(),
		Status: data.Status.ValueString(),
	}

	if !data.DisplayUptime.IsNull() {
		v := data.DisplayUptime.ValueBool()
		comp.DisplayUptime = &v
	}
	if !data.DisplayMetrics.IsNull() {
		v := data.DisplayMetrics.ValueBool()
		comp.DisplayMetrics = &v
	}

	created, err := r.client.CreateStatusPageComponent(data.StatusPageId.ValueString(), comp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page component, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_status_page_component.StatusPageComponentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	comp, err := r.client.GetStatusPageComponent(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status page component, got error: %s", err))
		return
	}

	data.Id = types.StringValue(comp.ID)
	data.Name = types.StringValue(comp.Name)
	data.Status = types.StringValue(comp.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_status_page_component.StatusPageComponentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	comp := &client.StatusPageComponent{
		Name:   data.Name.ValueString(),
		Status: data.Status.ValueString(),
	}

	if !data.DisplayUptime.IsNull() {
		v := data.DisplayUptime.ValueBool()
		comp.DisplayUptime = &v
	}
	if !data.DisplayMetrics.IsNull() {
		v := data.DisplayMetrics.ValueBool()
		comp.DisplayMetrics = &v
	}

	_, err := r.client.UpdateStatusPageComponent(data.StatusPageId.ValueString(), data.Id.ValueString(), comp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page component, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_status_page_component.StatusPageComponentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStatusPageComponent(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete status page component, got error: %s", err))
		return
	}
}

func (r *StatusPageComponentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: status_page_id/component_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: status_page_id/component_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
