package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &StatusPagesDataSource{}

func NewStatusPagesDataSource() datasource.DataSource {
	return &StatusPagesDataSource{}
}

type StatusPagesDataSource struct {
	client *client.Client
}

type StatusPagesDataSourceModel struct {
	StatusPages []StatusPageDataModel `tfsdk:"status_pages"`
}

type StatusPageDataModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Subdomain    types.String `tfsdk:"subdomain"`
	CustomDomain types.String `tfsdk:"custom_domain"`
}

func (d *StatusPagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_pages"
}

func (d *StatusPagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of status pages.",
		Attributes: map[string]schema.Attribute{
			"status_pages": schema.ListNestedAttribute{
				Description: "List of status pages",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the status page",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the status page",
							Computed:    true,
						},
						"subdomain": schema.StringAttribute{
							Description: "The subdomain of the status page",
							Computed:    true,
						},
						"custom_domain": schema.StringAttribute{
							Description: "The custom domain of the status page",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *StatusPagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *StatusPagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StatusPagesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	statusPages, err := d.client.ListStatusPages()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read status pages, got error: %s", err))
		return
	}

	data.StatusPages = make([]StatusPageDataModel, len(statusPages))
	for i, sp := range statusPages {
		data.StatusPages[i] = StatusPageDataModel{
			ID:        types.StringValue(sp.ID),
			Name:      types.StringValue(sp.Name),
			Subdomain: types.StringValue(sp.Subdomain),
		}
		if sp.CustomDomain != "" {
			data.StatusPages[i].CustomDomain = types.StringValue(sp.CustomDomain)
		} else {
			data.StatusPages[i].CustomDomain = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
