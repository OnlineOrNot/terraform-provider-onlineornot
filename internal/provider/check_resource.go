package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_check"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CheckResource{}
var _ resource.ResourceWithImportState = &CheckResource{}

func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

func NewUptimeCheckResource() resource.Resource {
	return &CheckResource{
		typeName:        "uptime_check",
		endpointKind:    "uptime",
		forcedInputType: "UPTIME_CHECK",
	}
}

func NewBrowserCheckResource() resource.Resource {
	return &CheckResource{
		typeName:        "browser_check",
		endpointKind:    "browser",
		forcedInputType: "BROWSER_CHECK",
	}
}

type checkModel struct {
	AlertPriority                types.String `tfsdk:"alert_priority"`
	Assertions                   types.List   `tfsdk:"assertions"`
	AuthPassword                 types.String `tfsdk:"auth_password"`
	AuthUsername                 types.String `tfsdk:"auth_username"`
	Body                         types.String `tfsdk:"body"`
	ConfirmationPeriodSeconds    types.Int64  `tfsdk:"confirmation_period_seconds"`
	DiscordAlerts                types.List   `tfsdk:"discord_alerts"`
	FollowRedirects              types.Bool   `tfsdk:"follow_redirects"`
	Headers                      types.Map    `tfsdk:"headers"`
	Id                           types.String `tfsdk:"id"`
	IncidentIoAlerts             types.List   `tfsdk:"incident_io_alerts"`
	Method                       types.String `tfsdk:"method"`
	MicrosoftTeamsAlerts         types.List   `tfsdk:"microsoft_teams_alerts"`
	Muted                        types.Bool   `tfsdk:"muted"`
	Name                         types.String `tfsdk:"name"`
	OncallAlerts                 types.List   `tfsdk:"oncall_alerts"`
	Paused                       types.Bool   `tfsdk:"paused"`
	PushoverAlerts               types.List   `tfsdk:"pushover_alerts"`
	RecoveryPeriodSeconds        types.Int64  `tfsdk:"recovery_period_seconds"`
	ReminderAlertIntervalMinutes types.Int64  `tfsdk:"reminder_alert_interval_minutes"`
	Script                       types.String `tfsdk:"script"`
	SlackAlerts                  types.List   `tfsdk:"slack_alerts"`
	TelegramAlerts               types.List   `tfsdk:"telegram_alerts"`
	TestInterval                 types.Int64  `tfsdk:"test_interval"`
	TestRegions                  types.List   `tfsdk:"test_regions"`
	TextToSearchFor              types.String `tfsdk:"text_to_search_for"`
	Timeout                      types.Int64  `tfsdk:"timeout"`
	Type                         types.String `tfsdk:"type"`
	Url                          types.String `tfsdk:"url"`
	UserAlerts                   types.List   `tfsdk:"user_alerts"`
	VerifySsl                    types.Bool   `tfsdk:"verify_ssl"`
	Version                      types.String `tfsdk:"version"`
	WebhookAlerts                types.List   `tfsdk:"webhook_alerts"`
}

// CheckResource defines the resource implementation.
type CheckResource struct {
	client          *client.Client
	typeName        string
	endpointKind    string
	forcedInputType string
}

func (r *CheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	typeName := r.typeName
	if typeName == "" {
		typeName = "check"
	}
	resp.TypeName = req.ProviderTypeName + "_" + typeName
}

func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_check.CheckResourceSchema(ctx)
	resp.Schema.Attributes["paused"] = pausedAttribute("check")
	resp.Schema.Attributes["muted"] = mutedAttribute("check")

	if authUsernameAttr, ok := resp.Schema.Attributes["auth_username"].(schema.StringAttribute); ok {
		authUsernameAttr.Description = "Username to use for URLs behind HTTP Basic Auth. Set this to an empty string for an empty user-id."
		authUsernameAttr.MarkdownDescription = authUsernameAttr.Description
		resp.Schema.Attributes["auth_username"] = authUsernameAttr
	}
	if authPasswordAttr, ok := resp.Schema.Attributes["auth_password"].(schema.StringAttribute); ok {
		authPasswordAttr.Sensitive = true
		authPasswordAttr.Description = "Password to use for URLs behind HTTP Basic Auth. Empty strings are preserved."
		authPasswordAttr.MarkdownDescription = authPasswordAttr.Description
		resp.Schema.Attributes["auth_password"] = authPasswordAttr
	}

	if r.forcedInputType != "" {
		if typeAttr, ok := resp.Schema.Attributes["type"].(schema.StringAttribute); ok {
			typeAttr.Default = stringdefault.StaticString(r.forcedInputType)
			typeAttr.Description = fmt.Sprintf("Type of check. Always %s for this resource.", r.forcedInputType)
			typeAttr.MarkdownDescription = typeAttr.Description
			resp.Schema.Attributes["type"] = typeAttr
		}
	}
}

func (r *CheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data checkModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	check := checkModelToClient(ctx, &data, r.forcedInputType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	changes, err := operationalStateChanges(data.Paused, data.Muted, types.BoolValue(false), types.BoolValue(false))
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operational State", err.Error())
		return
	}

	// Create the check without PATCH-only operational fields.
	created, err := r.client.CreateTypedCheck(r.endpointKind, check)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create check, got error: %s", err))
		return
	}

	// Populate state from the API response (includes computed defaults).
	r.populateModelFromAPI(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, change := range changes {
		patch := &client.CheckPatch{}
		applyOperationalState(change, &patch.Paused, &patch.Muted)
		created, err = r.client.UpdateTypedCheck(r.endpointKind, created.ID, patch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set check operational state, got error: %s", err))
			return
		}
		r.populateModelFromAPI(ctx, &data, created, &resp.Diagnostics)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// checkModelToClient converts a Terraform check model into the API request model.
func checkModelToClient(ctx context.Context, data *checkModel, forcedInputType string, diags *diag.Diagnostics) *client.Check {
	check := &client.Check{
		Name:                         data.Name.ValueString(),
		URL:                          data.Url.ValueString(),
		TestInterval:                 int(data.TestInterval.ValueInt64()),
		TextToSearchFor:              data.TextToSearchFor.ValueString(),
		ReminderAlertIntervalMinutes: int(data.ReminderAlertIntervalMinutes.ValueInt64()),
		ConfirmationPeriodSeconds:    int(data.ConfirmationPeriodSeconds.ValueInt64()),
		RecoveryPeriodSeconds:        int(data.RecoveryPeriodSeconds.ValueInt64()),
		Timeout:                      int(data.Timeout.ValueInt64()),
		Method:                       data.Method.ValueString(),
		Body:                         data.Body.ValueString(),
		AlertPriority:                data.AlertPriority.ValueString(),
		Type:                         data.Type.ValueString(),
		Version:                      data.Version.ValueString(),
		Script:                       data.Script.ValueString(),
	}
	if forcedInputType != "" {
		check.Type = forcedInputType
	}

	if !data.AuthUsername.IsNull() && !data.AuthUsername.IsUnknown() {
		value := data.AuthUsername.ValueString()
		check.AuthUsername = &value
	}
	if !data.AuthPassword.IsNull() && !data.AuthPassword.IsUnknown() {
		value := data.AuthPassword.ValueString()
		check.AuthPassword = &value
	}
	if !data.FollowRedirects.IsNull() {
		value := data.FollowRedirects.ValueBool()
		check.FollowRedirects = &value
	}
	if !data.VerifySsl.IsNull() {
		value := data.VerifySsl.ValueBool()
		check.VerifySSL = &value
	}

	if !data.TestRegions.IsNull() && !data.TestRegions.IsUnknown() {
		diags.Append(data.TestRegions.ElementsAs(ctx, &check.TestRegions, false)...)
	}
	if !data.UserAlerts.IsNull() && !data.UserAlerts.IsUnknown() {
		diags.Append(data.UserAlerts.ElementsAs(ctx, &check.UserAlerts, false)...)
	}
	if !data.SlackAlerts.IsNull() && !data.SlackAlerts.IsUnknown() {
		diags.Append(data.SlackAlerts.ElementsAs(ctx, &check.SlackAlerts, false)...)
	}
	if !data.DiscordAlerts.IsNull() && !data.DiscordAlerts.IsUnknown() {
		diags.Append(data.DiscordAlerts.ElementsAs(ctx, &check.DiscordAlerts, false)...)
	}
	if !data.TelegramAlerts.IsNull() && !data.TelegramAlerts.IsUnknown() {
		diags.Append(data.TelegramAlerts.ElementsAs(ctx, &check.TelegramAlerts, false)...)
	}
	if !data.PushoverAlerts.IsNull() && !data.PushoverAlerts.IsUnknown() {
		diags.Append(data.PushoverAlerts.ElementsAs(ctx, &check.PushoverAlerts, false)...)
	}
	if !data.WebhookAlerts.IsNull() && !data.WebhookAlerts.IsUnknown() {
		diags.Append(data.WebhookAlerts.ElementsAs(ctx, &check.WebhookAlerts, false)...)
	}
	if !data.OncallAlerts.IsNull() && !data.OncallAlerts.IsUnknown() {
		diags.Append(data.OncallAlerts.ElementsAs(ctx, &check.OncallAlerts, false)...)
	}
	if !data.IncidentIoAlerts.IsNull() && !data.IncidentIoAlerts.IsUnknown() {
		diags.Append(data.IncidentIoAlerts.ElementsAs(ctx, &check.IncidentIOAlerts, false)...)
	}
	if !data.MicrosoftTeamsAlerts.IsNull() && !data.MicrosoftTeamsAlerts.IsUnknown() {
		diags.Append(data.MicrosoftTeamsAlerts.ElementsAs(ctx, &check.MicrosoftTeamsAlerts, false)...)
	}
	if !data.Headers.IsNull() && !data.Headers.IsUnknown() {
		diags.Append(data.Headers.ElementsAs(ctx, &check.Headers, false)...)
	}

	if !data.Assertions.IsNull() && !data.Assertions.IsUnknown() {
		var assertionValues []resource_check.AssertionsValue
		diags.Append(data.Assertions.ElementsAs(ctx, &assertionValues, false)...)
		for _, assertion := range assertionValues {
			check.Assertions = append(check.Assertions, client.Assertion{
				Type:       assertion.AssertionsType.ValueString(),
				Property:   assertion.Property.ValueString(),
				Comparison: assertion.Comparison.ValueString(),
				Expected:   assertion.Expected.ValueString(),
			})
		}
	}

	return check
}

// populateModelFromAPI updates a CheckModel with values from the API response
func (r *CheckResource) populateModelFromAPI(ctx context.Context, data *checkModel, check *client.Check, diags *diag.Diagnostics) {
	data.Id = types.StringValue(check.ID)
	data.Name = types.StringValue(check.Name)
	data.Url = types.StringValue(check.URL)
	data.Paused = types.BoolValue(check.Status == "PAUSED")
	data.Muted = types.BoolValue(check.Status == "MUTED")

	// String fields with defaults
	if check.Method != "" {
		data.Method = types.StringValue(check.Method)
	} else {
		data.Method = types.StringNull()
	}
	// API returns check_type but schema uses type
	// API returns "UPTIME"/"BROWSER" but schema expects "UPTIME_CHECK"/"BROWSER_CHECK"
	checkType := check.CheckType
	if checkType == "" {
		checkType = check.Type
	}
	switch checkType {
	case "UPTIME":
		data.Type = types.StringValue("UPTIME_CHECK")
	case "BROWSER":
		data.Type = types.StringValue("BROWSER_CHECK")
	case "UPTIME_CHECK", "BROWSER_CHECK":
		data.Type = types.StringValue(checkType)
	case "":
		data.Type = types.StringNull()
	default:
		data.Type = types.StringValue(checkType)
	}
	if check.AlertPriority != "" {
		data.AlertPriority = types.StringValue(check.AlertPriority)
	} else {
		data.AlertPriority = types.StringNull()
	}
	if check.TextToSearchFor != "" {
		data.TextToSearchFor = types.StringValue(check.TextToSearchFor)
	} else {
		data.TextToSearchFor = types.StringNull()
	}
	if check.Body != "" {
		data.Body = types.StringValue(check.Body)
	} else {
		data.Body = types.StringNull()
	}
	if check.Version != "" {
		data.Version = types.StringValue(check.Version)
	} else {
		data.Version = types.StringNull()
	}
	if check.Script != "" {
		data.Script = types.StringValue(check.Script)
	} else {
		data.Script = types.StringNull()
	}
	if check.AuthUsername != nil {
		data.AuthUsername = types.StringValue(*check.AuthUsername)
	} else {
		data.AuthUsername = types.StringNull()
	}
	if check.AuthPassword != nil {
		data.AuthPassword = types.StringValue(*check.AuthPassword)
	} else {
		data.AuthPassword = types.StringNull()
	}

	// Integer fields
	if check.TestInterval > 0 {
		data.TestInterval = types.Int64Value(int64(check.TestInterval))
	} else {
		data.TestInterval = types.Int64Null()
	}
	if check.Timeout > 0 {
		data.Timeout = types.Int64Value(int64(check.Timeout))
	} else {
		data.Timeout = types.Int64Null()
	}
	if check.ConfirmationPeriodSeconds > 0 {
		data.ConfirmationPeriodSeconds = types.Int64Value(int64(check.ConfirmationPeriodSeconds))
	} else {
		data.ConfirmationPeriodSeconds = types.Int64Null()
	}
	if check.RecoveryPeriodSeconds > 0 {
		data.RecoveryPeriodSeconds = types.Int64Value(int64(check.RecoveryPeriodSeconds))
	} else {
		data.RecoveryPeriodSeconds = types.Int64Null()
	}
	if check.ReminderAlertIntervalMinutes > 0 {
		data.ReminderAlertIntervalMinutes = types.Int64Value(int64(check.ReminderAlertIntervalMinutes))
	} else {
		data.ReminderAlertIntervalMinutes = types.Int64Null()
	}

	// Boolean fields
	if check.FollowRedirects != nil {
		data.FollowRedirects = types.BoolValue(*check.FollowRedirects)
	} else {
		data.FollowRedirects = types.BoolNull()
	}
	if check.VerifySSL != nil {
		data.VerifySsl = types.BoolValue(*check.VerifySSL)
	} else {
		data.VerifySsl = types.BoolNull()
	}

	// List fields - convert slices to Terraform lists
	if len(check.TestRegions) > 0 {
		testRegions, d := types.ListValueFrom(ctx, types.StringType, check.TestRegions)
		diags.Append(d...)
		data.TestRegions = testRegions
	} else {
		data.TestRegions = types.ListNull(types.StringType)
	}

	if len(check.UserAlerts) > 0 {
		userAlerts, d := types.ListValueFrom(ctx, types.StringType, check.UserAlerts)
		diags.Append(d...)
		data.UserAlerts = userAlerts
	} else {
		data.UserAlerts = types.ListNull(types.StringType)
	}

	if len(check.SlackAlerts) > 0 {
		slackAlerts, d := types.ListValueFrom(ctx, types.StringType, check.SlackAlerts)
		diags.Append(d...)
		data.SlackAlerts = slackAlerts
	} else {
		data.SlackAlerts = types.ListNull(types.StringType)
	}

	if len(check.DiscordAlerts) > 0 {
		discordAlerts, d := types.ListValueFrom(ctx, types.StringType, check.DiscordAlerts)
		diags.Append(d...)
		data.DiscordAlerts = discordAlerts
	} else {
		data.DiscordAlerts = types.ListNull(types.StringType)
	}

	if len(check.TelegramAlerts) > 0 {
		telegramAlerts, d := types.ListValueFrom(ctx, types.StringType, check.TelegramAlerts)
		diags.Append(d...)
		data.TelegramAlerts = telegramAlerts
	} else {
		data.TelegramAlerts = types.ListNull(types.StringType)
	}

	if len(check.PushoverAlerts) > 0 {
		pushoverAlerts, d := types.ListValueFrom(ctx, types.StringType, check.PushoverAlerts)
		diags.Append(d...)
		data.PushoverAlerts = pushoverAlerts
	} else {
		data.PushoverAlerts = types.ListNull(types.StringType)
	}

	if len(check.WebhookAlerts) > 0 {
		webhookAlerts, d := types.ListValueFrom(ctx, types.StringType, check.WebhookAlerts)
		diags.Append(d...)
		data.WebhookAlerts = webhookAlerts
	} else {
		data.WebhookAlerts = types.ListNull(types.StringType)
	}

	if len(check.OncallAlerts) > 0 {
		oncallAlerts, d := types.ListValueFrom(ctx, types.StringType, check.OncallAlerts)
		diags.Append(d...)
		data.OncallAlerts = oncallAlerts
	} else {
		data.OncallAlerts = types.ListNull(types.StringType)
	}

	if len(check.IncidentIOAlerts) > 0 {
		incidentIoAlerts, d := types.ListValueFrom(ctx, types.StringType, check.IncidentIOAlerts)
		diags.Append(d...)
		data.IncidentIoAlerts = incidentIoAlerts
	} else {
		data.IncidentIoAlerts = types.ListNull(types.StringType)
	}

	if len(check.MicrosoftTeamsAlerts) > 0 {
		msTeamsAlerts, d := types.ListValueFrom(ctx, types.StringType, check.MicrosoftTeamsAlerts)
		diags.Append(d...)
		data.MicrosoftTeamsAlerts = msTeamsAlerts
	} else {
		data.MicrosoftTeamsAlerts = types.ListNull(types.StringType)
	}

	// Map fields
	if len(check.Headers) > 0 {
		headers, d := types.MapValueFrom(ctx, types.StringType, check.Headers)
		diags.Append(d...)
		data.Headers = headers
	} else {
		data.Headers = types.MapNull(types.StringType)
	}

	// Complex nested types - Assertions
	if len(check.Assertions) > 0 {
		assertionValues := make([]resource_check.AssertionsValue, len(check.Assertions))
		for i, a := range check.Assertions {
			assertionValues[i] = resource_check.NewAssertionsValueMust(
				resource_check.AssertionsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"type":       types.StringValue(a.Type),
					"property":   types.StringValue(a.Property),
					"comparison": types.StringValue(a.Comparison),
					"expected":   types.StringValue(a.Expected),
				},
			)
		}
		assertionsElemType := resource_check.AssertionsType{
			ObjectType: types.ObjectType{
				AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx),
			},
		}
		assertionsList, d := types.ListValueFrom(ctx, assertionsElemType, assertionValues)
		diags.Append(d...)
		data.Assertions = assertionsList
	} else {
		assertionsElemType := resource_check.AssertionsType{
			ObjectType: types.ObjectType{
				AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx),
			},
		}
		data.Assertions = types.ListNull(assertionsElemType)
	}
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data checkModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get check from API
	check, err := r.client.GetTypedCheck(r.endpointKind, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read check, got error: %s", err))
		return
	}

	// Populate state from the API response
	r.populateModelFromAPI(ctx, &data, check, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data checkModel
	var state checkModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// Read current state to get the ID
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Use the ID from state (it's stable), data from plan (user's desired state)
	checkID := state.Id.ValueString()

	check := checkModelToClient(ctx, &data, r.forcedInputType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	changes, err := operationalStateChanges(data.Paused, data.Muted, state.Paused, state.Muted)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operational State", err.Error())
		return
	}

	patch := &client.CheckPatch{Check: check}
	if len(changes) > 0 {
		applyOperationalState(changes[0], &patch.Paused, &patch.Muted)
	}
	updated, err := r.client.UpdateTypedCheck(r.endpointKind, checkID, patch)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update check, got error: %s", err))
		return
	}
	for _, change := range changes[1:] {
		patch = &client.CheckPatch{}
		applyOperationalState(change, &patch.Paused, &patch.Muted)
		updated, err = r.client.UpdateTypedCheck(r.endpointKind, checkID, patch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update check operational state, got error: %s", err))
			return
		}
	}

	// Populate state from the final API response.
	r.populateModelFromAPI(ctx, &data, updated, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data checkModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the check
	err := r.client.DeleteTypedCheck(r.endpointKind, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete check, got error: %s", err))
		return
	}
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
