package provider

import (
	"context"
	"fmt"

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

	// Update state from the created response
	data.Id = types.StringValue(created.ID)

	// Set all computed fields to null to avoid "unknown after apply" errors
	// The Read operation will populate actual values on subsequent refreshes
	data.TestInterval = types.Int64Null()
	data.TestRegions = types.ListNull(types.StringType)
	data.TextToSearchFor = types.StringNull()
	data.Body = types.StringNull()
	data.Version = types.StringNull()
	data.AuthUsername = types.StringNull()
	data.AuthPassword = types.StringNull()
	data.Headers = types.MapNull(types.StringType)
	data.UserAlerts = types.ListNull(types.StringType)
	data.SlackAlerts = types.ListNull(types.StringType)
	data.DiscordAlerts = types.ListNull(types.StringType)
	data.WebhookAlerts = types.ListNull(types.StringType)
	data.OncallAlerts = types.ListNull(types.StringType)
	data.IncidentIoAlerts = types.ListNull(types.StringType)
	data.MicrosoftTeamsAlerts = types.ListNull(types.StringType)

	// For the complex Assertions list, we need to use the proper element type
	assertionsElemType := resource_check.AssertionsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx),
		},
	}
	data.Assertions = types.ListNull(assertionsElemType)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	// Update state from API response
	data.Id = types.StringValue(check.ID)
	data.Name = types.StringValue(check.Name)
	data.Url = types.StringValue(check.URL)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	// Update the check
	_, err := r.client.UpdateCheck(data.Id.ValueString(), check)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update check, got error: %s", err))
		return
	}

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
