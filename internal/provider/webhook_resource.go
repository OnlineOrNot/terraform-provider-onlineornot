package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"

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

	wh, diags := webhookRequestFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateWebhook(wh)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(populateWebhookModel(ctx, &data, created)...)

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
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(populateWebhookModel(ctx, &data, wh)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, diags := webhookRequestFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateWebhook(data.Id.ValueString(), wh)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update webhook, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(populateWebhookModel(ctx, &data, updated)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_webhook.WebhookModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhook(data.Id.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook, got error: %s", err))
		return
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func webhookRequestFromModel(ctx context.Context, data *resource_webhook.WebhookModel) (*client.WebhookRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	wh := &client.WebhookRequest{URL: data.Url.ValueString()}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		description := data.Description.ValueString()
		wh.Description = &description
	}
	diags.Append(data.Events.ElementsAs(ctx, &wh.Events, false)...)
	associations := []struct {
		value  types.List
		target **[]string
	}{
		{data.CheckIds, &wh.CheckIDs}, {data.HeartbeatIds, &wh.HeartbeatIDs}, {data.StatusPageIds, &wh.StatusPageIDs},
	}
	for _, association := range associations {
		if association.value.IsNull() || association.value.IsUnknown() {
			continue
		}
		ids := []string{}
		diags.Append(association.value.ElementsAs(ctx, &ids, false)...)
		*association.target = &ids
	}
	return wh, diags
}

func populateWebhookModel(ctx context.Context, data *resource_webhook.WebhookModel, wh *client.Webhook) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Id = types.StringValue(wh.ID)
	data.Url = types.StringValue(wh.URL)
	// The API stores an omitted create description as null. Preserve Terraform's
	// null rather than introducing an empty-string diff on every refresh.
	if wh.Description == "" && (data.Description.IsNull() || data.Description.IsUnknown()) {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(wh.Description)
	}
	lists := []struct {
		target *types.List
		ids    []string
	}{
		{&data.Events, wh.Events}, {&data.CheckIds, wh.CheckIDs},
		{&data.HeartbeatIds, wh.HeartbeatIDs}, {&data.StatusPageIds, wh.StatusPageIDs},
	}
	for _, list := range lists {
		// SQL aggregates do not promise order. Retain configured ordering only when
		// membership matches; changed membership must still appear as drift.
		if !list.target.IsNull() && !list.target.IsUnknown() {
			var previous []string
			diags.Append(list.target.ElementsAs(ctx, &previous, false)...)
			actual := slices.Clone(list.ids)
			slices.Sort(previous)
			slices.Sort(actual)
			if slices.Equal(previous, actual) {
				continue
			}
		}
		if len(list.ids) == 0 && list.target.IsNull() {
			continue
		}
		value, ds := types.ListValueFrom(ctx, types.StringType, list.ids)
		diags.Append(ds...)
		*list.target = value
	}
	return diags
}
