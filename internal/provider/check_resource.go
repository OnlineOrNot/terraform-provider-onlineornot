package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// CheckResource defines the resource implementation.
type CheckResource struct {
	client *client.Client
}

func (r *CheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_check.CheckResourceSchema(ctx)
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
	var data resource_check.CheckModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to API model
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
		AuthUsername:                 data.AuthUsername.ValueString(),
		AuthPassword:                 data.AuthPassword.ValueString(),
	}

	if !data.FollowRedirects.IsNull() {
		v := data.FollowRedirects.ValueBool()
		check.FollowRedirects = &v
	}

	if !data.VerifySsl.IsNull() {
		v := data.VerifySsl.ValueBool()
		check.VerifySSL = &v
	}

	// Convert string lists
	if !data.TestRegions.IsNull() {
		data.TestRegions.ElementsAs(ctx, &check.TestRegions, false)
	}
	if !data.UserAlerts.IsNull() {
		data.UserAlerts.ElementsAs(ctx, &check.UserAlerts, false)
	}
	if !data.SlackAlerts.IsNull() {
		data.SlackAlerts.ElementsAs(ctx, &check.SlackAlerts, false)
	}
	if !data.DiscordAlerts.IsNull() {
		data.DiscordAlerts.ElementsAs(ctx, &check.DiscordAlerts, false)
	}
	if !data.WebhookAlerts.IsNull() {
		data.WebhookAlerts.ElementsAs(ctx, &check.WebhookAlerts, false)
	}
	if !data.OncallAlerts.IsNull() {
		data.OncallAlerts.ElementsAs(ctx, &check.OncallAlerts, false)
	}
	if !data.IncidentIoAlerts.IsNull() {
		data.IncidentIoAlerts.ElementsAs(ctx, &check.IncidentIOAlerts, false)
	}
	if !data.MicrosoftTeamsAlerts.IsNull() {
		data.MicrosoftTeamsAlerts.ElementsAs(ctx, &check.MicrosoftTeamsAlerts, false)
	}

	// Create the check
	created, err := r.client.CreateCheck(check)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create check, got error: %s", err))
		return
	}

	// Populate state from the API response (includes computed defaults)
	r.populateModelFromAPI(ctx, &data, created, &resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// populateModelFromAPI updates a CheckModel with values from the API response
func (r *CheckResource) populateModelFromAPI(ctx context.Context, data *resource_check.CheckModel, check *client.Check, diags *diag.Diagnostics) {
	data.Id = types.StringValue(check.ID)
	data.Name = types.StringValue(check.Name)
	data.Url = types.StringValue(check.URL)

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
	if check.AuthUsername != "" {
		data.AuthUsername = types.StringValue(check.AuthUsername)
	} else {
		data.AuthUsername = types.StringNull()
	}
	if check.AuthPassword != "" {
		data.AuthPassword = types.StringValue(check.AuthPassword)
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
	var data resource_check.CheckModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get check from API
	check, err := r.client.GetCheck(data.Id.ValueString())
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
	var data resource_check.CheckModel
	var state resource_check.CheckModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// Read current state to get the ID
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Use the ID from state (it's stable), data from plan (user's desired state)
	checkID := state.Id.ValueString()

	// Convert to API model
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
		AuthUsername:                 data.AuthUsername.ValueString(),
		AuthPassword:                 data.AuthPassword.ValueString(),
	}

	if !data.FollowRedirects.IsNull() {
		v := data.FollowRedirects.ValueBool()
		check.FollowRedirects = &v
	}

	if !data.VerifySsl.IsNull() {
		v := data.VerifySsl.ValueBool()
		check.VerifySSL = &v
	}

	// Update the check using the ID from state
	updated, err := r.client.UpdateCheck(checkID, check)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update check, got error: %s", err))
		return
	}

	// Populate state from the API response
	r.populateModelFromAPI(ctx, &data, updated, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_check.CheckModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the check
	err := r.client.DeleteCheck(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete check, got error: %s", err))
		return
	}
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
