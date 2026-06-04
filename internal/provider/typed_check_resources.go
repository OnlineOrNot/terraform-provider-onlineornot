package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_check"
)

var _ resource.Resource = &DNSCheckResource{}
var _ resource.ResourceWithImportState = &DNSCheckResource{}
var _ resource.Resource = &TCPCheckResource{}
var _ resource.ResourceWithImportState = &TCPCheckResource{}

type typedCheckModel struct {
	AlertPriority                types.String `tfsdk:"alert_priority"`
	Assertions                   types.List   `tfsdk:"assertions"`
	ConfirmationPeriodSeconds    types.Int64  `tfsdk:"confirmation_period_seconds"`
	DiscordAlerts                types.List   `tfsdk:"discord_alerts"`
	Id                           types.String `tfsdk:"id"`
	IncidentIoAlerts             types.List   `tfsdk:"incident_io_alerts"`
	MicrosoftTeamsAlerts         types.List   `tfsdk:"microsoft_teams_alerts"`
	Name                         types.String `tfsdk:"name"`
	OncallAlerts                 types.List   `tfsdk:"oncall_alerts"`
	RecoveryPeriodSeconds        types.Int64  `tfsdk:"recovery_period_seconds"`
	ReminderAlertIntervalMinutes types.Int64  `tfsdk:"reminder_alert_interval_minutes"`
	SlackAlerts                  types.List   `tfsdk:"slack_alerts"`
	TelegramAlerts               types.List   `tfsdk:"telegram_alerts"`
	TestInterval                 types.Int64  `tfsdk:"test_interval"`
	TestRegions                  types.List   `tfsdk:"test_regions"`
	Timeout                      types.Int64  `tfsdk:"timeout"`
	UserAlerts                   types.List   `tfsdk:"user_alerts"`
	WebhookAlerts                types.List   `tfsdk:"webhook_alerts"`
}

type DNSCheckModel struct {
	typedCheckModel
	DNSDomain     types.String `tfsdk:"dns_domain"`
	DNSRecordType types.String `tfsdk:"dns_record_type"`
	DNSResolver   types.String `tfsdk:"dns_resolver"`
	DNSProtocol   types.String `tfsdk:"dns_protocol"`
}

type TCPCheckModel struct {
	typedCheckModel
	TCPHostname   types.String `tfsdk:"tcp_hostname"`
	TCPPort       types.Int64  `tfsdk:"tcp_port"`
	TCPIPFamily   types.String `tfsdk:"tcp_ip_family"`
	TCPData       types.String `tfsdk:"tcp_data"`
	TCPShouldFail types.Bool   `tfsdk:"tcp_should_fail"`
}

type DNSCheckResource struct {
	client *client.Client
}

type TCPCheckResource struct {
	client *client.Client
}

func NewDNSCheckResource() resource.Resource {
	return &DNSCheckResource{}
}

func NewTCPCheckResource() resource.Resource {
	return &TCPCheckResource{}
}

func (r *DNSCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_check"
}

func (r *TCPCheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_check"
}

func (r *DNSCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := typedCheckSchema(ctx, "DNS check ID")
	s.Attributes["dns_domain"] = schema.StringAttribute{Required: true, Description: "Domain name to query", MarkdownDescription: "Domain name to query"}
	s.Attributes["dns_record_type"] = schema.StringAttribute{
		Required:            true,
		Description:         "DNS record type to query",
		MarkdownDescription: "DNS record type to query",
		Validators: []validator.String{stringvalidator.OneOf(
			"A", "AAAA", "CNAME", "MX", "NS", "PTR", "SOA", "SRV", "TXT", "CAA",
		)},
	}
	s.Attributes["dns_resolver"] = schema.StringAttribute{Optional: true, Computed: true, Description: "DNS resolver to use", MarkdownDescription: "DNS resolver to use"}
	s.Attributes["dns_protocol"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "DNS protocol to use",
		MarkdownDescription: "DNS protocol to use",
		Validators:          []validator.String{stringvalidator.OneOf("UDP", "TCP", "HTTPS")},
		Default:             stringdefault.StaticString("UDP"),
	}
	resp.Schema = s
}

func (r *TCPCheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := typedCheckSchema(ctx, "TCP check ID")
	s.Attributes["tcp_hostname"] = schema.StringAttribute{Required: true, Description: "Hostname to connect to", MarkdownDescription: "Hostname to connect to"}
	s.Attributes["tcp_port"] = schema.Int64Attribute{
		Required:            true,
		Description:         "TCP port to connect to",
		MarkdownDescription: "TCP port to connect to",
		Validators:          []validator.Int64{int64validator.Between(1, 65535)},
	}
	s.Attributes["tcp_ip_family"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "IP family to use",
		MarkdownDescription: "IP family to use",
		Validators:          []validator.String{stringvalidator.OneOf("IPv4", "IPv6", "Any")},
		Default:             stringdefault.StaticString("Any"),
	}
	s.Attributes["tcp_data"] = schema.StringAttribute{Optional: true, Computed: true, Description: "Data to send after connecting", MarkdownDescription: "Data to send after connecting"}
	s.Attributes["tcp_should_fail"] = schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the connection is expected to fail", MarkdownDescription: "Whether the connection is expected to fail", Default: booldefault.StaticBool(false)}
	resp.Schema = s
}

func typedCheckSchema(ctx context.Context, idDescription string) schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"alert_priority": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Description:         "Alert Priority",
			MarkdownDescription: "Alert Priority",
			Validators:          []validator.String{stringvalidator.OneOf("LOW", "HIGH")},
			Default:             stringdefault.StaticString("LOW"),
		},
		"assertions": schema.ListNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"comparison": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN", "NULL", "NOT_NULL", "EMPTY", "NOT_EMPTY", "CONTAINS", "NOT_CONTAINS", "FALSE", "TRUE")}},
					"expected":   schema.StringAttribute{Required: true},
					"property":   schema.StringAttribute{Required: true},
					"type":       schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("JSON_BODY", "TEXT_BODY", "RESPONSE_HEADERS", "HTML_BODY")}},
				},
				CustomType: resource_check.AssertionsType{ObjectType: types.ObjectType{AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx)}},
			},
			Optional:            true,
			Computed:            true,
			Description:         "Assertions to run on the response",
			MarkdownDescription: "Assertions to run on the response",
		},
		"confirmation_period_seconds": schema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(0)}, Default: int64default.StaticInt64(60)},
		"discord_alerts":              stringListAttribute(),
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Description:         idDescription,
			MarkdownDescription: idDescription,
			Validators:          []validator.String{stringvalidator.LengthAtLeast(8)},
		},
		"incident_io_alerts":              stringListAttribute(),
		"microsoft_teams_alerts":          stringListAttribute(),
		"name":                            schema.StringAttribute{Required: true, Description: "Name of the monitor", MarkdownDescription: "Name of the monitor"},
		"oncall_alerts":                   stringListAttribute(),
		"recovery_period_seconds":         schema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(0)}, Default: int64default.StaticInt64(180)},
		"reminder_alert_interval_minutes": schema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(-1)}, Default: int64default.StaticInt64(1440)},
		"slack_alerts":                    stringListAttribute(),
		"telegram_alerts":                 stringListAttribute(),
		"test_interval":                   schema.Int64Attribute{Optional: true, Computed: true, Description: "Interval in seconds between checks", MarkdownDescription: "Interval in seconds between checks", Validators: []validator.Int64{int64validator.AtLeast(30)}},
		"test_regions":                    stringListAttribute(),
		"timeout":                         schema.Int64Attribute{Optional: true, Computed: true, Description: "Timeout in milliseconds", MarkdownDescription: "Timeout in milliseconds", Validators: []validator.Int64{int64validator.AtLeast(1000)}, Default: int64default.StaticInt64(10000)},
		"user_alerts":                     stringListAttribute(),
		"webhook_alerts":                  stringListAttribute(),
	}}
}

func stringListAttribute() schema.ListAttribute {
	return schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true}
}

func (r *DNSCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureTypedCheckClient(req.ProviderData, &resp.Diagnostics)
}

func (r *TCPCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureTypedCheckClient(req.ProviderData, &resp.Diagnostics)
}

func configureTypedCheckClient(providerData any, diags *diag.Diagnostics) *client.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", providerData))
		return nil
	}
	return c
}

func (r *DNSCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDNSCheck(dnsModelToClient(ctx, &data, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create DNS check, got error: %s", err))
		return
	}
	populateDNSModel(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check, err := r.client.GetDNSCheck(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read DNS check, got error: %s", err))
		return
	}
	populateDNSModel(ctx, &data, check, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DNSCheckModel
	var state DNSCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateDNSCheck(state.Id.ValueString(), dnsModelToClient(ctx, &data, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update DNS check, got error: %s", err))
		return
	}
	populateDNSModel(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DNSCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDNSCheck(data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete DNS check, got error: %s", err))
	}
}

func (r *DNSCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TCPCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TCPCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTCPCheck(tcpModelToClient(ctx, &data, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create TCP check, got error: %s", err))
		return
	}
	populateTCPModel(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TCPCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TCPCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check, err := r.client.GetTCPCheck(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read TCP check, got error: %s", err))
		return
	}
	populateTCPModel(ctx, &data, check, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TCPCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TCPCheckModel
	var state TCPCheckModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateTCPCheck(state.Id.ValueString(), tcpModelToClient(ctx, &data, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update TCP check, got error: %s", err))
		return
	}
	populateTCPModel(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TCPCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TCPCheckModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTCPCheck(data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete TCP check, got error: %s", err))
	}
}

func (r *TCPCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func dnsModelToClient(ctx context.Context, data *DNSCheckModel, diags *diag.Diagnostics) *client.DNSCheck {
	check := &client.DNSCheck{
		Name:                         data.Name.ValueString(),
		TestInterval:                 int(data.TestInterval.ValueInt64()),
		ReminderAlertIntervalMinutes: int(data.ReminderAlertIntervalMinutes.ValueInt64()),
		ConfirmationPeriodSeconds:    int(data.ConfirmationPeriodSeconds.ValueInt64()),
		RecoveryPeriodSeconds:        int(data.RecoveryPeriodSeconds.ValueInt64()),
		Timeout:                      int(data.Timeout.ValueInt64()),
		AlertPriority:                data.AlertPriority.ValueString(),
		DNSDomain:                    data.DNSDomain.ValueString(),
		DNSRecordType:                data.DNSRecordType.ValueString(),
		DNSProtocol:                  data.DNSProtocol.ValueString(),
	}
	if !data.DNSResolver.IsNull() {
		v := data.DNSResolver.ValueString()
		check.DNSResolver = &v
	}
	populateClientCommon(ctx, &data.typedCheckModel, &check.TestRegions, &check.UserAlerts, &check.SlackAlerts, &check.DiscordAlerts, &check.TelegramAlerts, &check.WebhookAlerts, &check.OncallAlerts, &check.IncidentIOAlerts, &check.MicrosoftTeamsAlerts, &check.Assertions, diags)
	return check
}

func tcpModelToClient(ctx context.Context, data *TCPCheckModel, diags *diag.Diagnostics) *client.TCPCheck {
	check := &client.TCPCheck{
		Name:                         data.Name.ValueString(),
		TestInterval:                 int(data.TestInterval.ValueInt64()),
		ReminderAlertIntervalMinutes: int(data.ReminderAlertIntervalMinutes.ValueInt64()),
		ConfirmationPeriodSeconds:    int(data.ConfirmationPeriodSeconds.ValueInt64()),
		RecoveryPeriodSeconds:        int(data.RecoveryPeriodSeconds.ValueInt64()),
		Timeout:                      int(data.Timeout.ValueInt64()),
		AlertPriority:                data.AlertPriority.ValueString(),
		TCPHostname:                  data.TCPHostname.ValueString(),
		TCPPort:                      int(data.TCPPort.ValueInt64()),
		TCPIPFamily:                  data.TCPIPFamily.ValueString(),
	}
	if !data.TCPData.IsNull() {
		v := data.TCPData.ValueString()
		check.TCPData = &v
	}
	if !data.TCPShouldFail.IsNull() {
		v := data.TCPShouldFail.ValueBool()
		check.TCPShouldFail = &v
	}
	populateClientCommon(ctx, &data.typedCheckModel, &check.TestRegions, &check.UserAlerts, &check.SlackAlerts, &check.DiscordAlerts, &check.TelegramAlerts, &check.WebhookAlerts, &check.OncallAlerts, &check.IncidentIOAlerts, &check.MicrosoftTeamsAlerts, &check.Assertions, diags)
	return check
}

func populateClientCommon(ctx context.Context, data *typedCheckModel, testRegions, userAlerts, slackAlerts, discordAlerts, telegramAlerts, webhookAlerts, oncallAlerts, incidentIOAlerts, microsoftTeamsAlerts *[]string, assertions *[]client.MonitorAssertion, diags *diag.Diagnostics) {
	listElementsAs(ctx, data.TestRegions, testRegions, diags)
	listElementsAs(ctx, data.UserAlerts, userAlerts, diags)
	listElementsAs(ctx, data.SlackAlerts, slackAlerts, diags)
	listElementsAs(ctx, data.DiscordAlerts, discordAlerts, diags)
	listElementsAs(ctx, data.TelegramAlerts, telegramAlerts, diags)
	listElementsAs(ctx, data.WebhookAlerts, webhookAlerts, diags)
	listElementsAs(ctx, data.OncallAlerts, oncallAlerts, diags)
	listElementsAs(ctx, data.IncidentIoAlerts, incidentIOAlerts, diags)
	listElementsAs(ctx, data.MicrosoftTeamsAlerts, microsoftTeamsAlerts, diags)

	if !data.Assertions.IsNull() {
		var values []resource_check.AssertionsValue
		diags.Append(data.Assertions.ElementsAs(ctx, &values, false)...)
		for _, value := range values {
			*assertions = append(*assertions, client.MonitorAssertion{
				Type:       value.AssertionsType.ValueString(),
				Property:   value.Property.ValueString(),
				Comparison: value.Comparison.ValueString(),
				Expected:   value.Expected.ValueString(),
			})
		}
	}
}

func listElementsAs(ctx context.Context, value types.List, target *[]string, diags *diag.Diagnostics) {
	if !value.IsNull() {
		diags.Append(value.ElementsAs(ctx, target, false)...)
	}
}

func populateDNSModel(ctx context.Context, data *DNSCheckModel, check *client.DNSCheck, diags *diag.Diagnostics) {
	populateCommonModel(ctx, &data.typedCheckModel, check.ID, check.Name, check.TestInterval, check.ReminderAlertIntervalMinutes, check.ConfirmationPeriodSeconds, check.RecoveryPeriodSeconds, check.Timeout, check.AlertPriority, check.TestRegions, check.UserAlerts, check.SlackAlerts, check.DiscordAlerts, check.TelegramAlerts, check.WebhookAlerts, check.OncallAlerts, check.IncidentIOAlerts, check.MicrosoftTeamsAlerts, check.Assertions, diags)
	data.DNSDomain = types.StringValue(check.DNSDomain)
	data.DNSRecordType = types.StringValue(check.DNSRecordType)
	data.DNSProtocol = optionalStringValue(check.DNSProtocol)
	if check.DNSResolver != nil {
		data.DNSResolver = types.StringValue(*check.DNSResolver)
	} else {
		data.DNSResolver = types.StringNull()
	}
}

func populateTCPModel(ctx context.Context, data *TCPCheckModel, check *client.TCPCheck, diags *diag.Diagnostics) {
	populateCommonModel(ctx, &data.typedCheckModel, check.ID, check.Name, check.TestInterval, check.ReminderAlertIntervalMinutes, check.ConfirmationPeriodSeconds, check.RecoveryPeriodSeconds, check.Timeout, check.AlertPriority, check.TestRegions, check.UserAlerts, check.SlackAlerts, check.DiscordAlerts, check.TelegramAlerts, check.WebhookAlerts, check.OncallAlerts, check.IncidentIOAlerts, check.MicrosoftTeamsAlerts, check.Assertions, diags)
	data.TCPHostname = types.StringValue(check.TCPHostname)
	data.TCPPort = types.Int64Value(int64(check.TCPPort))
	data.TCPIPFamily = optionalStringValue(check.TCPIPFamily)
	if check.TCPData != nil {
		data.TCPData = types.StringValue(*check.TCPData)
	} else {
		data.TCPData = types.StringNull()
	}
	if check.TCPShouldFail != nil {
		data.TCPShouldFail = types.BoolValue(*check.TCPShouldFail)
	} else {
		data.TCPShouldFail = types.BoolNull()
	}
}

func populateCommonModel(ctx context.Context, data *typedCheckModel, id, name string, testInterval, reminderInterval, confirmationPeriod, recoveryPeriod, timeout int, alertPriority string, testRegions, userAlerts, slackAlerts, discordAlerts, telegramAlerts, webhookAlerts, oncallAlerts, incidentIOAlerts, microsoftTeamsAlerts []string, assertions []client.MonitorAssertion, diags *diag.Diagnostics) {
	data.Id = types.StringValue(id)
	data.Name = types.StringValue(name)
	data.TestInterval = optionalInt64Value(testInterval)
	data.ReminderAlertIntervalMinutes = optionalInt64Value(reminderInterval)
	data.ConfirmationPeriodSeconds = optionalInt64Value(confirmationPeriod)
	data.RecoveryPeriodSeconds = optionalInt64Value(recoveryPeriod)
	data.Timeout = optionalInt64Value(timeout)
	data.AlertPriority = optionalStringValue(alertPriority)
	data.TestRegions = stringListValue(ctx, testRegions, diags)
	data.UserAlerts = stringListValue(ctx, userAlerts, diags)
	data.SlackAlerts = stringListValue(ctx, slackAlerts, diags)
	data.DiscordAlerts = stringListValue(ctx, discordAlerts, diags)
	data.TelegramAlerts = stringListValue(ctx, telegramAlerts, diags)
	data.WebhookAlerts = stringListValue(ctx, webhookAlerts, diags)
	data.OncallAlerts = stringListValue(ctx, oncallAlerts, diags)
	data.IncidentIoAlerts = stringListValue(ctx, incidentIOAlerts, diags)
	data.MicrosoftTeamsAlerts = stringListValue(ctx, microsoftTeamsAlerts, diags)
	data.Assertions = assertionListValue(ctx, assertions, diags)
}

func optionalStringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func optionalInt64Value(value int) types.Int64 {
	if value <= 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(value))
}

func stringListValue(ctx context.Context, values []string, diags *diag.Diagnostics) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	result, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return result
}

func assertionListValue(ctx context.Context, assertions []client.MonitorAssertion, diags *diag.Diagnostics) types.List {
	elemType := resource_check.AssertionsType{ObjectType: types.ObjectType{AttrTypes: resource_check.AssertionsValue{}.AttributeTypes(ctx)}}
	if len(assertions) == 0 {
		return types.ListNull(elemType)
	}
	values := make([]resource_check.AssertionsValue, len(assertions))
	for i, assertion := range assertions {
		values[i] = resource_check.NewAssertionsValueMust(resource_check.AssertionsValue{}.AttributeTypes(ctx), map[string]attr.Value{
			"type":       types.StringValue(assertion.Type),
			"property":   types.StringValue(assertion.Property),
			"comparison": types.StringValue(assertion.Comparison),
			"expected":   types.StringValue(assertion.Expected),
		})
	}
	result, d := types.ListValueFrom(ctx, elemType, values)
	diags.Append(d...)
	return result
}
