package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

type projectIDValidator struct{}

func (projectIDValidator) Description(context.Context) string {
	return "Must be an encoded project ID, not a name or database ID."
}
func (v projectIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v projectIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !client.ValidProjectID(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid project ID", v.Description(ctx))
	}
}
func projectSelectionAttribute(replace bool) schema.StringAttribute {
	a := schema.StringAttribute{Optional: true, Computed: true, Description: "Encoded project ID. Omit on creation to use Default; omission on update retains ownership. Changes use an atomic move preserving identity and operational state. Destination variables must already exist.", Validators: []validator.String{projectIDValidator{}}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	if replace {
		a.Description = "Encoded project ID. Omit on creation to use Default. Variable moves are unsupported; changing project replaces the variable."
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
	}
	return a
}

// Save committed ownership before subsequent configuration writes, so failures
// after a successful move remain recoverable without inventing a rollback.
func moveProject(ctx context.Context, c *client.Client, heartbeat bool, id string, before, after types.String, prior tfsdk.State, state *tfsdk.State, diags *diag.Diagnostics) bool {
	if after.IsNull() || after.IsUnknown() || after.Equal(before) {
		return true
	}
	var err error
	if heartbeat {
		err = c.MoveHeartbeat(id, after.ValueString())
	} else {
		err = c.MoveCheck(id, after.ValueString())
	}
	if err != nil {
		diags.AddError("Unable to move resource", "Project move failed. Check project access and destination variable compatibility. No subsequent configuration updates were attempted.")
		return false
	}
	*state = prior
	diags.Append(state.SetAttribute(ctx, path.Root("project_id"), after)...)
	return !diags.HasError()
}
