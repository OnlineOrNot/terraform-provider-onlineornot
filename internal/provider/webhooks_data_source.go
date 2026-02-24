package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &WebhooksDataSource{}

func NewWebhooksDataSource() datasource.DataSource {
	return &WebhooksDataSource{}
}

type WebhooksDataSource struct {
	client *client.Client
}

type WebhooksDataSourceModel struct {
	Webhooks []WebhookDataModel `tfsdk:"webhooks"`
}

type WebhookDataModel struct {
	ID          types.String `tfsdk:"id"`
	URL         types.String `tfsdk:"url"`
	Description types.String `tfsdk:"description"`
}

func (d *WebhooksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhooks"
}

func (d *WebhooksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of webhooks.",
		Attributes: map[string]schema.Attribute{
			"webhooks": schema.ListNestedAttribute{
				Description: "List of webhooks",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the webhook",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The webhook endpoint URL",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the webhook",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *WebhooksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *WebhooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WebhooksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhooks, err := d.client.ListWebhooks()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhooks, got error: %s", err))
		return
	}

	data.Webhooks = make([]WebhookDataModel, len(webhooks))
	for i, wh := range webhooks {
		data.Webhooks[i] = WebhookDataModel{
			ID:          types.StringValue(wh.ID),
			URL:         types.StringValue(wh.URL),
			Description: types.StringValue(wh.Description),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
