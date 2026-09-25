package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ resource.ResourceWithImportState = &EnvironmentVariableResource{}

type EnvironmentVariableResource struct{ client *client.Client }
type environmentVariableModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Reference    types.String `tfsdk:"reference"`
	Type         types.String `tfsdk:"type"`
	Value        types.String `tfsdk:"value"`
	ValueVersion types.String `tfsdk:"value_version"`
}

func NewEnvironmentVariableResource() resource.Resource { return &EnvironmentVariableResource{} }
func (r *EnvironmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}
func (r *EnvironmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages a secret environment variable. The API never returns its value; external value changes cannot be detected on refresh.", Attributes: map[string]schema.Attribute{
		"id":            schema.StringAttribute{Computed: true, Description: "Environment variable ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":          schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`), "must be an uppercase environment variable name")}, Description: "Uppercase name used in {{NAME}} references; renames update referencing checks."},
		"reference":     schema.StringAttribute{Computed: true, Description: "Non-sensitive {{NAME}} reference for use in check headers and other templated fields.", PlanModifiers: []planmodifier.String{environmentVariableReferencePlanModifier{}}},
		"type":          schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("secret")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Only secret variables are supported. Type is immutable."},
		"value":         schema.StringAttribute{Optional: true, WriteOnly: true, Sensitive: true, Description: "Secret value, supplied on create and when changing value_version. Never saved in state or plan. Supply from an ephemeral value; ordinary Terraform variables/configuration may be retained outside provider state."},
		"value_version": schema.StringAttribute{Required: true, Description: "Non-secret change trigger. Change this string AND supply value to replace the remote value. Terraform cannot detect out-of-band changes to secret values."},
	}}
}
func (r *EnvironmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
	}
}

// Never surface API errors: even validation failures may echo submitted secret values.
const environmentVariableError = "Environment variable API request failed. Check permissions, feature availability, and input constraints. Response details are withheld to protect secret values."

func (r *EnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config environmentVariableModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Value.IsNull() || config.Value.IsUnknown() {
		resp.Diagnostics.AddError("Missing secret value", "Supply value when creating a secret environment variable.")
		return
	}
	value := config.Value.ValueString()
	result, err := r.client.CreateEnvironmentVariable(client.EnvironmentVariableWrite{Name: data.Name.ValueString(), Type: "secret", Value: &value})
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", environmentVariableError)
		return
	}
	if result.Type != "secret" {
		resp.Diagnostics.AddError("Unexpected environment variable type", "The API did not create a secret environment variable.")
		return
	}
	data.ID = types.StringValue(result.ID)
	data.Reference = environmentVariableReference(data.Name)
	data.Value = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *EnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.GetEnvironmentVariable(data.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment variable", environmentVariableError)
		return
	}
	if result.Type != "secret" {
		resp.Diagnostics.AddError("Unexpected environment variable type", "The remote variable is not a secret.")
		return
	}
	data.Name = types.StringValue(result.Name)
	data.Reference = environmentVariableReference(data.Name)
	data.Type = types.StringValue(result.Type)
	// value_version represents user intent, not a remote revision; the API does not expose secret value changes.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *EnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, config environmentVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	changed := !plan.ValueVersion.Equal(state.ValueVersion)
	input := client.EnvironmentVariableWrite{}
	if !plan.Name.Equal(state.Name) {
		input.Name = plan.Name.ValueString()
	}
	if changed {
		if config.Value.IsNull() || config.Value.IsUnknown() {
			resp.Diagnostics.AddError("Missing replacement secret", "Supply value when changing value_version.")
			return
		}
		value := config.Value.ValueString()
		input.Value = &value
	}
	if input.Name == "" && input.Value == nil {
		plan.Reference = environmentVariableReference(plan.Name)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}
	result, err := r.client.UpdateEnvironmentVariable(state.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", environmentVariableError)
		return
	}
	if result.Type != "secret" {
		resp.Diagnostics.AddError("Unexpected environment variable type", "The remote variable is not a secret.")
		return
	}
	plan.ID = state.ID
	plan.Reference = environmentVariableReference(plan.Name)
	plan.Value = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *EnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEnvironmentVariable(data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting environment variable", environmentVariableError)
	}
}
func (r *EnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(req.ID) {
		resp.Diagnostics.AddError("Invalid environment variable ID", "Import requires an alphanumeric environment variable ID.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func environmentVariableReference(name types.String) types.String {
	return types.StringValue("{{" + name.ValueString() + "}}")
}

// Compute the reference from the planned name, not the prior reference: a rename
// must propagate the new template to checks in the same Terraform plan.
type environmentVariableReferencePlanModifier struct{}

func (environmentVariableReferencePlanModifier) Description(context.Context) string {
	return "Derive the reference from the planned environment variable name."
}
func (m environmentVariableReferencePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (environmentVariableReferencePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var name types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &name)...)
	if resp.Diagnostics.HasError() || name.IsNull() || name.IsUnknown() {
		return
	}
	resp.PlanValue = environmentVariableReference(name)
}
