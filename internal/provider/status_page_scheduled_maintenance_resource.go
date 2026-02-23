package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_status_page_scheduled_maintenance"
)

var _ resource.Resource = &StatusPageScheduledMaintenanceResource{}
var _ resource.ResourceWithImportState = &StatusPageScheduledMaintenanceResource{}

func NewStatusPageScheduledMaintenanceResource() resource.Resource {
	return &StatusPageScheduledMaintenanceResource{}
}

type StatusPageScheduledMaintenanceResource struct {
	client *client.Client
}

func (r *StatusPageScheduledMaintenanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_scheduled_maintenance"
}

func (r *StatusPageScheduledMaintenanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_status_page_scheduled_maintenance.StatusPageScheduledMaintenanceResourceSchema(ctx)
}

func (r *StatusPageScheduledMaintenanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StatusPageScheduledMaintenanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_status_page_scheduled_maintenance.StatusPageScheduledMaintenanceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sm := &client.StatusPageScheduledMaintenance{
		Title:           data.Title.ValueString(),
		Description:     data.Description.ValueString(),
		StartDate:       data.StartDate.ValueString(),
		DurationMinutes: int(data.DurationMinutes.ValueInt64()),
	}

	if !data.ComponentsAffected.IsNull() {
		data.ComponentsAffected.ElementsAs(ctx, &sm.ComponentsAffected, false)
	}

	// Handle notifications nested object
	if !data.Notifications.IsNull() && !data.Notifications.IsUnknown() {
		sm.Notifications = &client.ScheduledMaintenanceNotifications{
			AnHourBefore: data.Notifications.AnHourBefore.ValueBool(),
			AtStart:      data.Notifications.AtStart.ValueBool(),
			AtEnd:        data.Notifications.AtEnd.ValueBool(),
		}
	}

	created, err := r.client.CreateStatusPageScheduledMaintenance(data.StatusPageId.ValueString(), sm)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled maintenance, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageScheduledMaintenanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_status_page_scheduled_maintenance.StatusPageScheduledMaintenanceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sm, err := r.client.GetStatusPageScheduledMaintenance(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled maintenance, got error: %s", err))
		return
	}

	data.Id = types.StringValue(sm.ID)
	data.Title = types.StringValue(sm.Title)
	data.Description = types.StringValue(sm.Description)
	data.StartDate = types.StringValue(sm.StartDate)
	data.DurationMinutes = types.Int64Value(int64(sm.DurationMinutes))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageScheduledMaintenanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_status_page_scheduled_maintenance.StatusPageScheduledMaintenanceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sm := &client.StatusPageScheduledMaintenance{
		Title:           data.Title.ValueString(),
		Description:     data.Description.ValueString(),
		StartDate:       data.StartDate.ValueString(),
		DurationMinutes: int(data.DurationMinutes.ValueInt64()),
	}

	if !data.ComponentsAffected.IsNull() {
		data.ComponentsAffected.ElementsAs(ctx, &sm.ComponentsAffected, false)
	}

	// Handle notifications nested object
	if !data.Notifications.IsNull() && !data.Notifications.IsUnknown() {
		sm.Notifications = &client.ScheduledMaintenanceNotifications{
			AnHourBefore: data.Notifications.AnHourBefore.ValueBool(),
			AtStart:      data.Notifications.AtStart.ValueBool(),
			AtEnd:        data.Notifications.AtEnd.ValueBool(),
		}
	}

	_, err := r.client.UpdateStatusPageScheduledMaintenance(data.StatusPageId.ValueString(), data.Id.ValueString(), sm)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update scheduled maintenance, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageScheduledMaintenanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_status_page_scheduled_maintenance.StatusPageScheduledMaintenanceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStatusPageScheduledMaintenance(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled maintenance, got error: %s", err))
		return
	}
}

func (r *StatusPageScheduledMaintenanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: status_page_id/maintenance_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: status_page_id/maintenance_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
