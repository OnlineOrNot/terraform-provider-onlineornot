package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ resource.ResourceWithImportState = &StatusPageOrderResource{}

// StatusPageOrderResource owns an entire ordered collection, not its members.
type StatusPageOrderResource struct {
	client             *client.Client
	kind, name, idsKey string
}

func NewStatusPageComponentOrderResource() resource.Resource {
	return &StatusPageOrderResource{kind: "components", name: "status_page_component_order", idsKey: "component_ids"}
}
func NewStatusPageGroupOrderResource() resource.Resource {
	return &StatusPageOrderResource{kind: "groups", name: "status_page_group_order", idsKey: "group_ids"}
}
func NewStatusPageGroupComponentOrderResource() resource.Resource {
	return &StatusPageOrderResource{kind: "group_components", name: "status_page_group_component_order", idsKey: "component_ids"}
}
func (r *StatusPageOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}
func (r *StatusPageOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	idValidators := []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9_-]+$`), "must be a nonempty path-safe ID (letters, digits, underscores, hyphens)")}
	resp.Schema = schema.Schema{
		Description: "Owns the complete ordered membership of one status-page scope. Use only one ordering resource per scope. Destroy relinquishes ownership without changing the remote layout or deleting members.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true, Description: "Ordering scope identity.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"status_page_id": schema.StringAttribute{Required: true, Description: "Status page ID. Changes replace this ownership resource.", Validators: idValidators, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			r.idsKey:         schema.ListAttribute{Required: true, ElementType: types.StringType, Description: "Complete list of IDs in desired display order. No omissions or duplicates; use an empty list only for an empty scope. Membership is managed separately by component/group resources.", Validators: []validator.List{listvalidator.UniqueValues(), listvalidator.ValueStringsAre(idValidators...)}},
		},
	}
	if r.kind == "group_components" {
		resp.Schema.Attributes["group_id"] = schema.StringAttribute{Required: true, Description: "Component group ID. Changes replace this ownership resource.", Validators: idValidators, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	}
}
func (r *StatusPageOrderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
	}
}

type orderAttributeReader interface {
	GetAttribute(context.Context, path.Path, interface{}) diag.Diagnostics
}

func (r *StatusPageOrderResource) scope(ctx context.Context, reader orderAttributeReader, diags *diag.Diagnostics) client.StatusPageOrderScope {
	var page types.String
	diags.Append(reader.GetAttribute(ctx, path.Root("status_page_id"), &page)...)
	s := client.StatusPageOrderScope{PageID: page.ValueString(), Kind: r.kind}
	if r.kind == "group_components" {
		var group types.String
		diags.Append(reader.GetAttribute(ctx, path.Root("group_id"), &group)...)
		s.GroupID = group.ValueString()
	}
	if !client.ValidOrderingID(s.PageID) || (r.kind == "group_components" && !client.ValidOrderingID(s.GroupID)) {
		diags.AddError("Invalid ordering identity", "Page and group IDs must be known, nonempty path-safe IDs.")
	}
	return s
}
func (r *StatusPageOrderResource) save(ctx context.Context, state *tfsdk.State, s client.StatusPageOrderScope, ids []string, diags *diag.Diagnostics) {
	id := s.PageID
	if r.kind == "group_components" {
		id += "/" + s.GroupID
	}
	diags.Append(state.SetAttribute(ctx, path.Root("id"), id)...)
	diags.Append(state.SetAttribute(ctx, path.Root("status_page_id"), s.PageID)...)
	if r.kind == "group_components" {
		diags.Append(state.SetAttribute(ctx, path.Root("group_id"), s.GroupID)...)
	}
	if ids == nil {
		ids = []string{}
	}
	list, ds := types.ListValueFrom(ctx, types.StringType, ids)
	diags.Append(ds...)
	diags.Append(state.SetAttribute(ctx, path.Root(r.idsKey), list)...)
}
func (r *StatusPageOrderResource) apply(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	s := r.scope(ctx, plan, diags)
	var list types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(r.idsKey), &list)...)
	if diags.HasError() {
		return
	}
	if list.IsNull() || list.IsUnknown() {
		diags.AddError("Invalid ordered membership", "The complete ordered list must be known at apply.")
		return
	}
	var ids []string
	diags.Append(list.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !client.ValidOrderingID(id) || seen[id] {
			diags.AddError("Invalid ordered membership", "IDs must be path-safe, nonempty, and unique.")
			return
		}
		seen[id] = true
	}
	if err := r.client.CheckStatusPageOrderParent(s); err != nil {
		diags.AddError("Unable to read ordering parent", err.Error())
		return
	}
	current, err := r.client.ListStatusPageOrder(s)
	if err != nil {
		diags.AddError("Unable to read ordered membership", err.Error())
		return
	}
	complete := len(current) == len(ids)
	for _, id := range current {
		if !seen[id] {
			complete = false
		}
	}
	if !complete {
		diags.AddAttributeError(path.Root(r.idsKey), "Incomplete ordered membership", "Supply every current member of this scope exactly once, and no other IDs. Ordering never creates, deletes, or moves members. Update membership separately before retrying.")
		return
	}
	if err := r.client.SetStatusPageOrder(s, ids); err != nil {
		diags.AddError("Unable to update ordering", err.Error())
		return
	}
	// Save ownership even if verification subsequently fails, allowing recovery.
	r.save(ctx, state, s, ids, diags)
	actual, err := r.client.ListStatusPageOrder(s)
	if err != nil {
		diags.AddError("Unable to verify ordering", err.Error())
		return
	}
	if !slices.Equal(actual, ids) {
		diags.AddError("Ordering changed during apply", "The API order does not match the requested order. Avoid simultaneous membership/layout edits and retry.")
	}
}
func (r *StatusPageOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}
func (r *StatusPageOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}
func (r *StatusPageOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	s := r.scope(ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.CheckStatusPageOrderParent(s); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read ordering parent", err.Error())
		return
	}
	ids, err := r.client.ListStatusPageOrder(s)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read ordered membership", err.Error())
		return
	}
	r.save(ctx, &resp.State, s, ids, &resp.Diagnostics)
}
func (r *StatusPageOrderResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Ownership only: never reset ranks or delete/move components or groups.
}
func (r *StatusPageOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	want := 1
	if r.kind == "group_components" {
		want = 2
	}
	valid := len(parts) == want
	for _, part := range parts {
		valid = valid && client.ValidOrderingID(part)
	}
	if !valid {
		resp.Diagnostics.AddError("Invalid import ID", "Use pageID, or pageID/groupID for within-group ordering, with path-safe nonempty IDs.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	if want == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
	}
}
