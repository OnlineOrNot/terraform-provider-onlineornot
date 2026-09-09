package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

type heartbeatModel struct {
	AlertPriority                types.String `tfsdk:"alert_priority"`
	DiscordAlerts                types.List   `tfsdk:"discord_alerts"`
	GracePeriod                  types.Int64  `tfsdk:"grace_period"`
	Id                           types.String `tfsdk:"id"`
	IncidentIoAlerts             types.List   `tfsdk:"incident_io_alerts"`
	MicrosoftTeamsAlerts         types.List   `tfsdk:"microsoft_teams_alerts"`
	Muted                        types.Bool   `tfsdk:"muted"`
	Name                         types.String `tfsdk:"name"`
	OncallAlerts                 types.List   `tfsdk:"oncall_alerts"`
	Paused                       types.Bool   `tfsdk:"paused"`
	PushoverAlerts               types.List   `tfsdk:"pushover_alerts"`
	ReminderAlertIntervalMinutes types.Int64  `tfsdk:"reminder_alert_interval_minutes"`
	ReportPeriod                 types.Int64  `tfsdk:"report_period"`
	ReportPeriodCron             types.String `tfsdk:"report_period_cron"`
	SlackAlerts                  types.List   `tfsdk:"slack_alerts"`
	TelegramAlerts               types.List   `tfsdk:"telegram_alerts"`
	Timezone                     types.String `tfsdk:"timezone"`
	UserAlerts                   types.List   `tfsdk:"user_alerts"`
	WebhookAlerts                types.List   `tfsdk:"webhook_alerts"`
}

func populateHeartbeatPushoverAlerts(ctx context.Context, data *heartbeatModel, alerts []string, diags *diag.Diagnostics) {
	pushoverAlerts, diagnostics := types.ListValueFrom(ctx, types.StringType, alerts)
	diags.Append(diagnostics...)
	data.PushoverAlerts = pushoverAlerts
}

func (r *HeartbeatResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat"
}

func (r *HeartbeatResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_heartbeat.HeartbeatResourceSchema(ctx)
	resp.Schema.Attributes["paused"] = pausedAttribute("heartbeat")
	resp.Schema.Attributes["muted"] = mutedAttribute("heartbeat")
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
	var data heartbeatModel

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
	if !data.TelegramAlerts.IsNull() {
		data.TelegramAlerts.ElementsAs(ctx, &hb.TelegramAlerts, false)
	}
	if !data.PushoverAlerts.IsNull() {
		data.PushoverAlerts.ElementsAs(ctx, &hb.PushoverAlerts, false)
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

	changes, err := operationalStateChanges(data.Paused, data.Muted, types.BoolValue(false), types.BoolValue(false))
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operational State", err.Error())
		return
	}

	created, err := r.client.CreateHeartbeat(hb)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create heartbeat, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	// Set computed fields to null to avoid "unknown after apply" errors
	if data.AlertPriority.IsUnknown() {
		data.AlertPriority = types.StringNull()
	}
	if data.ReportPeriod.IsUnknown() {
		data.ReportPeriod = types.Int64Null()
	}
	if data.ReportPeriodCron.IsUnknown() {
		data.ReportPeriodCron = types.StringNull()
	}
	if data.ReminderAlertIntervalMinutes.IsUnknown() {
		data.ReminderAlertIntervalMinutes = types.Int64Null()
	}
	if data.Timezone.IsUnknown() {
		data.Timezone = types.StringNull()
	}
	if data.UserAlerts.IsUnknown() {
		data.UserAlerts = types.ListNull(types.StringType)
	}
	if data.SlackAlerts.IsUnknown() {
		data.SlackAlerts = types.ListNull(types.StringType)
	}
	if data.DiscordAlerts.IsUnknown() {
		data.DiscordAlerts = types.ListNull(types.StringType)
	}
	if data.TelegramAlerts.IsUnknown() {
		data.TelegramAlerts = stringListValue(ctx, created.TelegramAlerts, &resp.Diagnostics)
	}
	if data.PushoverAlerts.IsUnknown() {
		populateHeartbeatPushoverAlerts(ctx, &data, created.PushoverAlerts, &resp.Diagnostics)
	}
	if data.WebhookAlerts.IsUnknown() {
		data.WebhookAlerts = types.ListNull(types.StringType)
	}
	if data.OncallAlerts.IsUnknown() {
		data.OncallAlerts = types.ListNull(types.StringType)
	}
	if data.IncidentIoAlerts.IsUnknown() {
		data.IncidentIoAlerts = types.ListNull(types.StringType)
	}
	if data.MicrosoftTeamsAlerts.IsUnknown() {
		data.MicrosoftTeamsAlerts = types.ListNull(types.StringType)
	}
	data.Paused = types.BoolValue(created.Status == "PAUSED")
	data.Muted = types.BoolValue(created.Status == "MUTED")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	for _, change := range changes {
		patch := &client.HeartbeatPatch{}
		applyOperationalState(change, &patch.Paused, &patch.Muted)
		created, err = r.client.UpdateHeartbeat(created.ID, patch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set heartbeat operational state, got error: %s", err))
			return
		}
		data.Paused = types.BoolValue(created.Status == "PAUSED")
		data.Muted = types.BoolValue(created.Status == "MUTED")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data heartbeatModel

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
	populateHeartbeatPushoverAlerts(ctx, &data, hb.PushoverAlerts, &resp.Diagnostics)
	data.TelegramAlerts = stringListValue(ctx, hb.TelegramAlerts, &resp.Diagnostics)
	data.Paused = types.BoolValue(hb.Status == "PAUSED")
	data.Muted = types.BoolValue(hb.Status == "MUTED")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data heartbeatModel
	var state heartbeatModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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
	if !data.TelegramAlerts.IsNull() {
		data.TelegramAlerts.ElementsAs(ctx, &hb.TelegramAlerts, false)
	}
	if !data.PushoverAlerts.IsNull() {
		data.PushoverAlerts.ElementsAs(ctx, &hb.PushoverAlerts, false)
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

	changes, err := operationalStateChanges(data.Paused, data.Muted, state.Paused, state.Muted)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operational State", err.Error())
		return
	}
	patch := &client.HeartbeatPatch{Heartbeat: hb}
	if len(changes) > 0 {
		applyOperationalState(changes[0], &patch.Paused, &patch.Muted)
	}
	updated, err := r.client.UpdateHeartbeat(data.Id.ValueString(), patch)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update heartbeat, got error: %s", err))
		return
	}
	for i := 1; i < len(changes); i++ {
		patch = &client.HeartbeatPatch{}
		applyOperationalState(changes[i], &patch.Paused, &patch.Muted)
		updated, err = r.client.UpdateHeartbeat(data.Id.ValueString(), patch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update heartbeat operational state, got error: %s", err))
			return
		}
	}
	data.Paused = types.BoolValue(updated.Status == "PAUSED")
	data.Muted = types.BoolValue(updated.Status == "MUTED")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HeartbeatResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data heartbeatModel

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
