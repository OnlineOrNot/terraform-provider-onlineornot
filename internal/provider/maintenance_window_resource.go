package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_maintenance_window"
)

var _ resource.Resource = &MaintenanceWindowResource{}
var _ resource.ResourceWithImportState = &MaintenanceWindowResource{}

func NewMaintenanceWindowResource() resource.Resource {
	return &MaintenanceWindowResource{}
}

type MaintenanceWindowResource struct {
	client *client.Client
}

func (r *MaintenanceWindowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *MaintenanceWindowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_maintenance_window.MaintenanceWindowResourceSchema(ctx)
}

func (r *MaintenanceWindowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MaintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_maintenance_window.MaintenanceWindowModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mw := &client.MaintenanceWindow{
		Name:            data.Name.ValueString(),
		StartDate:       data.StartDate.ValueString(),
		DurationMinutes: int(data.DurationMinutes.ValueInt64()),
		Timezone:        data.Timezone.ValueString(),
	}

	if !data.DaysOfWeek.IsNull() {
		data.DaysOfWeek.ElementsAs(ctx, &mw.DaysOfWeek, false)
	}
	if !data.Checks.IsNull() {
		data.Checks.ElementsAs(ctx, &mw.Checks, false)
	}
	if !data.Heartbeats.IsNull() {
		data.Heartbeats.ElementsAs(ctx, &mw.Heartbeats, false)
	}

	created, err := r.client.CreateMaintenanceWindow(mw)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create maintenance window, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	// Set computed fields to null if not provided by user to avoid "unknown after apply" errors
	if data.Checks.IsUnknown() {
		data.Checks = types.ListNull(types.StringType)
	}
	if data.Heartbeats.IsUnknown() {
		data.Heartbeats = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_maintenance_window.MaintenanceWindowModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mw, err := r.client.GetMaintenanceWindow(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read maintenance window, got error: %s", err))
		return
	}

	data.Id = types.StringValue(mw.ID)
	data.Name = types.StringValue(mw.Name)
	data.StartDate = types.StringValue(mw.StartDate)
	data.DurationMinutes = types.Int64Value(int64(mw.DurationMinutes))
	data.Timezone = types.StringValue(mw.Timezone)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_maintenance_window.MaintenanceWindowModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mw := &client.MaintenanceWindow{
		Name:            data.Name.ValueString(),
		StartDate:       data.StartDate.ValueString(),
		DurationMinutes: int(data.DurationMinutes.ValueInt64()),
		Timezone:        data.Timezone.ValueString(),
	}

	if !data.DaysOfWeek.IsNull() {
		data.DaysOfWeek.ElementsAs(ctx, &mw.DaysOfWeek, false)
	}
	if !data.Checks.IsNull() {
		data.Checks.ElementsAs(ctx, &mw.Checks, false)
	}
	if !data.Heartbeats.IsNull() {
		data.Heartbeats.ElementsAs(ctx, &mw.Heartbeats, false)
	}

	_, err := r.client.UpdateMaintenanceWindow(data.Id.ValueString(), mw)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update maintenance window, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MaintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_maintenance_window.MaintenanceWindowModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMaintenanceWindow(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete maintenance window, got error: %s", err))
		return
	}
}

func (r *MaintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
