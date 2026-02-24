package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func (r *StatusPageIncidentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_incident"
}

func (r *StatusPageIncidentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_status_page_incident.StatusPageIncidentResourceSchema(ctx)
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
	var data resource_status_page_incident.StatusPageIncidentModel

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
		data.Components.ElementsAs(ctx, &components, false)
		for _, comp := range components {
			incident.Components = append(incident.Components, client.StatusPageIncidentComponent{
				ID:     comp.Id.ValueString(),
				Status: comp.Status.ValueString(),
			})
		}
	}

	created, err := r.client.CreateStatusPageIncident(data.StatusPageId.ValueString(), incident)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create status page incident, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

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
	var data resource_status_page_incident.StatusPageIncidentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	incident, err := r.client.GetStatusPageIncident(data.StatusPageId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status page incident, got error: %s", err))
		return
	}

	data.Id = types.StringValue(incident.ID)
	data.Title = types.StringValue(incident.Title)
	data.Description = types.StringValue(incident.Description)
	data.Status = types.StringValue(incident.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageIncidentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_status_page_incident.StatusPageIncidentModel

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

	// Handle components list
	if !data.Components.IsNull() {
		var components []resource_status_page_incident.ComponentsValue
		data.Components.ElementsAs(ctx, &components, false)
		for _, comp := range components {
			incident.Components = append(incident.Components, client.StatusPageIncidentComponent{
				ID:     comp.Id.ValueString(),
				Status: comp.Status.ValueString(),
			})
		}
	}

	_, err := r.client.UpdateStatusPageIncident(data.StatusPageId.ValueString(), data.Id.ValueString(), incident)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update status page incident, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StatusPageIncidentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_status_page_incident.StatusPageIncidentModel

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
