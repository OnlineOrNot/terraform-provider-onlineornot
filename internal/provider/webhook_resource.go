package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_webhook"
)

var _ resource.Resource = &WebhookResource{}
var _ resource.ResourceWithImportState = &WebhookResource{}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

type WebhookResource struct {
	client *client.Client
}

func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_webhook.WebhookResourceSchema(ctx)
}

func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh := &client.Webhook{
		URL:         data.Url.ValueString(),
		Description: data.Description.ValueString(),
	}

	if !data.Events.IsNull() {
		data.Events.ElementsAs(ctx, &wh.Events, false)
	}
	if !data.CheckIds.IsNull() {
		data.CheckIds.ElementsAs(ctx, &wh.CheckIDs, false)
	}
	if !data.HeartbeatIds.IsNull() {
		data.HeartbeatIds.ElementsAs(ctx, &wh.HeartbeatIDs, false)
	}
	if !data.StatusPageIds.IsNull() {
		data.StatusPageIds.ElementsAs(ctx, &wh.StatusPageIDs, false)
	}

	created, err := r.client.CreateWebhook(wh)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook, got error: %s", err))
		return
	}

	data.Id = types.StringValue(created.ID)

	// Set computed fields to null if not provided by user to avoid "unknown after apply" errors
	if data.CheckIds.IsUnknown() {
		data.CheckIds = types.ListNull(types.StringType)
	}
	if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	if data.HeartbeatIds.IsUnknown() {
		data.HeartbeatIds = types.ListNull(types.StringType)
	}
	if data.StatusPageIds.IsUnknown() {
		data.StatusPageIds = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, err := r.client.GetWebhook(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook, got error: %s", err))
		return
	}

	data.Id = types.StringValue(wh.ID)
	data.Url = types.StringValue(wh.URL)
	data.Description = types.StringValue(wh.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh := &client.Webhook{
		URL:         data.Url.ValueString(),
		Description: data.Description.ValueString(),
	}

	if !data.Events.IsNull() {
		data.Events.ElementsAs(ctx, &wh.Events, false)
	}
	if !data.CheckIds.IsNull() {
		data.CheckIds.ElementsAs(ctx, &wh.CheckIDs, false)
	}
	if !data.HeartbeatIds.IsNull() {
		data.HeartbeatIds.ElementsAs(ctx, &wh.HeartbeatIDs, false)
	}
	if !data.StatusPageIds.IsNull() {
		data.StatusPageIds.ElementsAs(ctx, &wh.StatusPageIDs, false)
	}

	_, err := r.client.UpdateWebhook(data.Id.ValueString(), wh)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update webhook, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhook(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook, got error: %s", err))
		return
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
