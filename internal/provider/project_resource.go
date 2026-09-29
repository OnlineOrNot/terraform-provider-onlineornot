package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

type ProjectResource struct{ client *client.Client }
type projectModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewProjectResource() resource.Resource { return &ProjectResource{} }
func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}
func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages an organisation project. Only empty non-Default projects can be deleted. Requires project management permissions; these do not grant resource or secret permissions.", Attributes: map[string]schema.Attribute{
		"id":         schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":       schema.StringAttribute{Required: true, Validators: []validator.String{projectNameValidator{}}, Description: "Editable display label, not a unique identifier. Use the encoded ID for selection."},
		"is_default": schema.BoolAttribute{Computed: true, Description: "Stable Default identity, independent of name."},
		"created_at": schema.StringAttribute{Computed: true}, "updated_at": schema.StringAttribute{Computed: true},
	}}
}
func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureTypedCheckClient(req.ProviderData, &resp.Diagnostics)
}
func projectModelFromAPI(p *client.Project) projectModel {
	return projectModel{ID: types.StringValue(p.ID), Name: types.StringValue(p.Name), IsDefault: types.BoolValue(p.IsDefault), CreatedAt: types.StringValue(p.CreatedAt), UpdatedAt: types.StringValue(p.UpdatedAt)}
}
func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.CreateProject(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create project", err.Error())
		return
	}
	data = projectModelFromAPI(p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetProject(data.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", err.Error())
		return
	}
	data = projectModelFromAPI(p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.UpdateProject(state.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update project", err.Error())
		return
	}
	data = projectModelFromAPI(p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteProject(data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete project", err.Error())
	}
}
func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !client.ValidProjectID(req.ID) {
		resp.Diagnostics.AddError("Invalid project ID", "Import requires an encoded project ID.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type projectNameValidator struct{}

func (projectNameValidator) Description(context.Context) string {
	return "Use a nonempty name of at most 255 characters without surrounding whitespace."
}
func (v projectNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v projectNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if strings.TrimSpace(s) != s || s == "" || len([]rune(s)) > 255 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid project name", v.Description(ctx))
	}
}
