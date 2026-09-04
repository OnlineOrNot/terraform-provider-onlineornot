package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

type statusPageComponentModel struct {
	CheckIds       types.List   `tfsdk:"check_ids"`
	DisplayMetrics types.Bool   `tfsdk:"display_metrics"`
	DisplayUptime  types.Bool   `tfsdk:"display_uptime"`
	GroupId        types.String `tfsdk:"group_id"`
	HeartbeatId    types.String `tfsdk:"heartbeat_id"`
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OverrideStatus types.Bool   `tfsdk:"override_status"`
	Status         types.String `tfsdk:"status"`
	StatusPageId   types.String `tfsdk:"status_page_id"`
}

func (r *StatusPageComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_component"
}

func (r *StatusPageComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_status_page_component.StatusPageComponentResourceSchema(ctx)
	s.Attributes["group_id"] = schema.StringAttribute{
		Optional:            true,
		Description:         "Component group ID. Set to null to remove the component from its group.",
		MarkdownDescription: "Component group ID. Set to null to remove the component from its group.",
	}
	s.Attributes["check_ids"] = schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Description:         "Uptime check IDs associated with this component.",
		MarkdownDescription: "Uptime check IDs associated with this component.",
	}
	s.Attributes["heartbeat_id"] = schema.StringAttribute{
		Optional:            true,
		Description:         "Heartbeat ID associated with this component.",
		MarkdownDescription: "Heartbeat ID associated with this component.",
	}
	s.Attributes["override_status"] = schema.BoolAttribute{
		Optional:            true,
		Description:         "Override status derived from an external status page.",
		MarkdownDescription: "Override status derived from an external status page.",
	}
	resp.Schema = s
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
	var data statusPageComponentModel

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

	relationshipsConfigured := componentRelationshipsConfigured(&data)
	requestedPatch := componentPatchFromModel(ctx, &data, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateStatusPageComponent(data.StatusPageId.ValueString(), comp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page component, got error: %s", err))
		return
	}

	populateStatusPageComponentModel(&data, created, requestedPatch.GroupID != nil)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if relationshipsConfigured {
		updated, err := r.client.UpdateStatusPageComponent(data.StatusPageId.ValueString(), created.ID, requestedPatch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to configure status page component relationships, got error: %s", err))
			return
		}
		populateStatusPageComponentModel(&data, updated, requestedPatch.GroupID != nil)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data statusPageComponentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	comp, err := r.client.GetStatusPageComponent(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status page component, got error: %s", err))
		return
	}

	manageGroup := !data.GroupId.IsNull() && !data.GroupId.IsUnknown()
	populateStatusPageComponentModel(&data, comp, manageGroup)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data statusPageComponentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior statusPageComponentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := componentPatchFromModel(ctx, &data, &prior, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateStatusPageComponent(data.StatusPageId.ValueString(), data.Id.ValueString(), patch)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page component, got error: %s", err))
		return
	}
	populateStatusPageComponentModel(&data, updated, patch.GroupID != nil)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func componentRelationshipsConfigured(data *statusPageComponentModel) bool {
	return (!data.GroupId.IsNull() && !data.GroupId.IsUnknown()) ||
		(!data.CheckIds.IsNull() && !data.CheckIds.IsUnknown()) ||
		(!data.HeartbeatId.IsNull() && !data.HeartbeatId.IsUnknown()) ||
		(!data.OverrideStatus.IsNull() && !data.OverrideStatus.IsUnknown())
}

func componentPatchFromModel(ctx context.Context, data, prior *statusPageComponentModel, diags *diag.Diagnostics) *client.StatusPageComponentPatch {
	patch := &client.StatusPageComponentPatch{
		Name:   data.Name.ValueString(),
		Status: data.Status.ValueString(),
	}
	if !data.DisplayUptime.IsNull() && !data.DisplayUptime.IsUnknown() {
		v := data.DisplayUptime.ValueBool()
		patch.DisplayUptime = &v
	}
	if !data.DisplayMetrics.IsNull() && !data.DisplayMetrics.IsUnknown() {
		v := data.DisplayMetrics.ValueBool()
		patch.DisplayMetrics = &v
	}
	if !data.GroupId.IsNull() && !data.GroupId.IsUnknown() {
		v := data.GroupId.ValueString()
		value := &v
		patch.GroupID = &value
	} else if data.GroupId.IsNull() && prior != nil && !prior.GroupId.IsNull() && !prior.GroupId.IsUnknown() {
		var cleared *string
		patch.GroupID = &cleared
	}
	if !data.CheckIds.IsNull() && !data.CheckIds.IsUnknown() {
		checkIDs := []string{}
		diags.Append(data.CheckIds.ElementsAs(ctx, &checkIDs, false)...)
		patch.CheckIDs = &checkIDs
	} else if data.CheckIds.IsNull() && prior != nil && !prior.CheckIds.IsNull() && !prior.CheckIds.IsUnknown() {
		checkIDs := []string{}
		patch.CheckIDs = &checkIDs
	}
	if !data.HeartbeatId.IsNull() && !data.HeartbeatId.IsUnknown() {
		v := data.HeartbeatId.ValueString()
		value := &v
		patch.HeartbeatID = &value
	} else if data.HeartbeatId.IsNull() && prior != nil && !prior.HeartbeatId.IsNull() && !prior.HeartbeatId.IsUnknown() {
		var cleared *string
		patch.HeartbeatID = &cleared
	}
	if !data.OverrideStatus.IsNull() && !data.OverrideStatus.IsUnknown() {
		v := data.OverrideStatus.ValueBool()
		patch.OverrideStatus = &v
	} else if data.OverrideStatus.IsNull() && prior != nil && !prior.OverrideStatus.IsNull() && !prior.OverrideStatus.IsUnknown() {
		v := false
		patch.OverrideStatus = &v
	}
	return patch
}

func populateStatusPageComponentModel(data *statusPageComponentModel, comp *client.StatusPageComponent, manageGroup bool) {
	data.Id = types.StringValue(comp.ID)
	data.Name = types.StringValue(comp.Name)
	data.Status = types.StringValue(comp.Status)
	if manageGroup {
		if comp.GroupID == nil {
			data.GroupId = types.StringNull()
		} else {
			data.GroupId = types.StringValue(*comp.GroupID)
		}
	}
	if comp.DisplayUptime != nil {
		data.DisplayUptime = types.BoolValue(*comp.DisplayUptime)
	}
	if comp.DisplayMetrics != nil {
		data.DisplayMetrics = types.BoolValue(*comp.DisplayMetrics)
	}
}

func (r *StatusPageComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data statusPageComponentModel

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
