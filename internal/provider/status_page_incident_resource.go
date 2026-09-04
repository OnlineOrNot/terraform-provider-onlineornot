package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_status_page_incident"
)

var _ resource.Resource = &StatusPageIncidentResource{}
var _ resource.ResourceWithImportState = &StatusPageIncidentResource{}

func NewStatusPageIncidentResource() resource.Resource {
	return &StatusPageIncidentResource{}
}

type StatusPageIncidentResource struct {
	client *client.Client
}

type statusPageIncidentModel struct {
	Components        types.List   `tfsdk:"components"`
	Description       types.String `tfsdk:"description"`
	Id                types.String `tfsdk:"id"`
	Impact            types.String `tfsdk:"impact"`
	NotifySubscribers types.Bool   `tfsdk:"notify_subscribers"`
	Status            types.String `tfsdk:"status"`
	StatusPageId      types.String `tfsdk:"status_page_id"`
	Title             types.String `tfsdk:"title"`
}

func (r *StatusPageIncidentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_incident"
}

func (r *StatusPageIncidentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_status_page_incident.StatusPageIncidentResourceSchema(ctx)

	description := s.Attributes["description"].(schema.StringAttribute)
	description.PlanModifiers = append(description.PlanModifiers, stringplanmodifier.RequiresReplace())
	s.Attributes["description"] = description
	status := s.Attributes["status"].(schema.StringAttribute)
	status.PlanModifiers = append(status.PlanModifiers, stringplanmodifier.RequiresReplace())
	s.Attributes["status"] = status
	components := s.Attributes["components"].(schema.ListNestedAttribute)
	components.PlanModifiers = append(components.PlanModifiers, listplanmodifier.RequiresReplace())
	s.Attributes["components"] = components
	notifySubscribers := s.Attributes["notify_subscribers"].(schema.BoolAttribute)
	notifySubscribers.PlanModifiers = append(notifySubscribers.PlanModifiers, boolplanmodifier.RequiresReplace())
	s.Attributes["notify_subscribers"] = notifySubscribers

	s.Attributes["impact"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "Impact of the incident.",
		MarkdownDescription: "Impact of the incident.",
		Validators: []validator.String{
			stringvalidator.OneOf("MAJOR_OUTAGE", "PARTIAL_OUTAGE", "DEGRADED_PERFORMANCE", "NO_IMPACT", "MAINTENANCE"),
		},
	}
	resp.Schema = s
}

func (r *StatusPageIncidentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StatusPageIncidentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data statusPageIncidentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	incident := &client.StatusPageIncident{
		Title:       data.Title.ValueString(),
		Description: data.Description.ValueString(),
		Status:      data.Status.ValueString(),
	}

	if !data.NotifySubscribers.IsNull() {
		v := data.NotifySubscribers.ValueBool()
		incident.NotifySubscribers = &v
	}

	// Handle components list - this is a nested object list
	if !data.Components.IsNull() {
		var components []resource_status_page_incident.ComponentsValue
		resp.Diagnostics.Append(data.Components.ElementsAs(ctx, &components, false)...)
		for _, comp := range components {
			incident.Components = append(incident.Components, client.StatusPageIncidentComponent{
				ID:     comp.Id.ValueString(),
				Status: comp.Status.ValueString(),
			})
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateStatusPageIncident(data.StatusPageId.ValueString(), incident)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page incident, got error: %s", err))
		return
	}

	impactConfigured := !data.Impact.IsNull() && !data.Impact.IsUnknown()
	requestedPatch := incidentPatchFromModel(&data)
	data.Id = types.StringValue(created.ID)
	populateStatusPageIncidentIdentity(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if impactConfigured {
		updated, err := r.client.UpdateStatusPageIncident(data.StatusPageId.ValueString(), created.ID, requestedPatch)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set status page incident impact, got error: %s", err))
			return
		}
		populateStatusPageIncidentIdentity(&data, updated)
	}

	// Set computed fields to null if not provided by user to avoid "unknown after apply" errors
	if data.Components.IsUnknown() {
		componentsElemType := resource_status_page_incident.ComponentsType{
			ObjectType: types.ObjectType{
				AttrTypes: resource_status_page_incident.ComponentsValue{}.AttributeTypes(ctx),
			},
		}
		data.Components = types.ListNull(componentsElemType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageIncidentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data statusPageIncidentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	incident, err := r.client.GetStatusPageIncident(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status page incident, got error: %s", err))
		return
	}

	populateStatusPageIncidentIdentity(&data, incident)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageIncidentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data statusPageIncidentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateStatusPageIncident(data.StatusPageId.ValueString(), data.Id.ValueString(), incidentPatchFromModel(&data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page incident, got error: %s", err))
		return
	}
	populateStatusPageIncidentIdentity(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func incidentPatchFromModel(data *statusPageIncidentModel) *client.StatusPageIncidentPatch {
	patch := &client.StatusPageIncidentPatch{Title: data.Title.ValueString()}
	if !data.Impact.IsNull() && !data.Impact.IsUnknown() {
		v := data.Impact.ValueString()
		patch.Impact = &v
	}
	return patch
}

func populateStatusPageIncidentIdentity(data *statusPageIncidentModel, incident *client.StatusPageIncident) {
	data.Id = types.StringValue(incident.ID)
	data.Title = types.StringValue(incident.Title)
	if incident.Impact == nil {
		data.Impact = types.StringNull()
	} else {
		data.Impact = types.StringValue(*incident.Impact)
	}
}

func (r *StatusPageIncidentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data statusPageIncidentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStatusPageIncident(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete status page incident, got error: %s", err))
		return
	}
}

func (r *StatusPageIncidentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: status_page_id/incident_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: status_page_id/incident_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
