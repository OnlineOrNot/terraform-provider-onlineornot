package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_heartbeat"
)

var _ resource.Resource = &HeartbeatResource{}
var _ resource.ResourceWithImportState = &HeartbeatResource{}

func NewHeartbeatResource() resource.Resource {
	return &HeartbeatResource{}
}

type HeartbeatResource struct {
	client *client.Client
}

func (r *HeartbeatResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat"
}

func (r *HeartbeatResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_heartbeat.HeartbeatResourceSchema(ctx)
}

func (r *HeartbeatResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HeartbeatResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_heartbeat.HeartbeatModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hb := &client.Heartbeat{
		Name:                         data.Name.ValueString(),
		GracePeriod:                  int(data.GracePeriod.ValueInt64()),
		ReportPeriod:                 int(data.ReportPeriod.ValueInt64()),
		ReportPeriodCron:             data.ReportPeriodCron.ValueString(),
		Timezone:                     data.Timezone.ValueString(),
		AlertPriority:                data.AlertPriority.ValueString(),
		ReminderAlertIntervalMinutes: int(data.ReminderAlertIntervalMinutes.ValueInt64()),
	}

	if !data.UserAlerts.IsNull() {
		data.UserAlerts.ElementsAs(ctx, &hb.UserAlerts, false)
	}
	if !data.SlackAlerts.IsNull() {
		data.SlackAlerts.ElementsAs(ctx, &hb.SlackAlerts, false)
	}
	if !data.DiscordAlerts.IsNull() {
		data.DiscordAlerts.ElementsAs(ctx, &hb.DiscordAlerts, false)
	}
	if !data.WebhookAlerts.IsNull() {
		data.WebhookAlerts.ElementsAs(ctx, &hb.WebhookAlerts, false)
	}
	if !data.OncallAlerts.IsNull() {
		data.OncallAlerts.ElementsAs(ctx, &hb.OncallAlerts, false)
	}
	if !data.IncidentIoAlerts.IsNull() {
		data.IncidentIoAlerts.ElementsAs(ctx, &hb.IncidentIOAlerts, false)
	}
	if !data.MicrosoftTeamsAlerts.IsNull() {
		data.MicrosoftTeamsAlerts.ElementsAs(ctx, &hb.MicrosoftTeamsAlerts, false)
	}

	created, err := r.client.CreateHeartbeat(hb)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create heartbeat, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_heartbeat.HeartbeatModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hb, err := r.client.GetHeartbeat(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read heartbeat, got error: %s", err))
		return
	}

	data.Id = types.StringValue(hb.ID)
	data.Name = types.StringValue(hb.Name)
	data.GracePeriod = types.Int64Value(int64(hb.GracePeriod))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_heartbeat.HeartbeatModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hb := &client.Heartbeat{
		Name:                         data.Name.ValueString(),
		GracePeriod:                  int(data.GracePeriod.ValueInt64()),
		ReportPeriod:                 int(data.ReportPeriod.ValueInt64()),
		ReportPeriodCron:             data.ReportPeriodCron.ValueString(),
		Timezone:                     data.Timezone.ValueString(),
		AlertPriority:                data.AlertPriority.ValueString(),
		ReminderAlertIntervalMinutes: int(data.ReminderAlertIntervalMinutes.ValueInt64()),
	}

	if !data.UserAlerts.IsNull() {
		data.UserAlerts.ElementsAs(ctx, &hb.UserAlerts, false)
	}
	if !data.SlackAlerts.IsNull() {
		data.SlackAlerts.ElementsAs(ctx, &hb.SlackAlerts, false)
	}
	if !data.DiscordAlerts.IsNull() {
		data.DiscordAlerts.ElementsAs(ctx, &hb.DiscordAlerts, false)
	}
	if !data.WebhookAlerts.IsNull() {
		data.WebhookAlerts.ElementsAs(ctx, &hb.WebhookAlerts, false)
	}
	if !data.OncallAlerts.IsNull() {
		data.OncallAlerts.ElementsAs(ctx, &hb.OncallAlerts, false)
	}
	if !data.IncidentIoAlerts.IsNull() {
		data.IncidentIoAlerts.ElementsAs(ctx, &hb.IncidentIOAlerts, false)
	}
	if !data.MicrosoftTeamsAlerts.IsNull() {
		data.MicrosoftTeamsAlerts.ElementsAs(ctx, &hb.MicrosoftTeamsAlerts, false)
	}

	_, err := r.client.UpdateHeartbeat(data.Id.ValueString(), hb)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update heartbeat, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_heartbeat.HeartbeatModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteHeartbeat(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete heartbeat, got error: %s", err))
		return
	}
}

func (r *HeartbeatResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
