package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &HeartbeatsDataSource{}

func NewHeartbeatsDataSource() datasource.DataSource {
	return &HeartbeatsDataSource{}
}

type HeartbeatsDataSource struct {
	client *client.Client
}

type HeartbeatsDataSourceModel struct {
	Heartbeats []HeartbeatDataModel `tfsdk:"heartbeats"`
}

type HeartbeatDataModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GracePeriod types.Int64  `tfsdk:"grace_period"`
}

func (d *HeartbeatsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeats"
}

func (d *HeartbeatsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of heartbeats.",
		Attributes: map[string]schema.Attribute{
			"heartbeats": schema.ListNestedAttribute{
				Description: "List of heartbeats",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the heartbeat",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the heartbeat",
							Computed:    true,
						},
						"grace_period": schema.Int64Attribute{
							Description: "The grace period in seconds",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *HeartbeatsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HeartbeatsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HeartbeatsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	heartbeats, err := d.client.ListHeartbeats()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read heartbeats, got error: %s", err))
		return
	}

	data.Heartbeats = make([]HeartbeatDataModel, len(heartbeats))
	for i, hb := range heartbeats {
		data.Heartbeats[i] = HeartbeatDataModel{
			ID:          types.StringValue(hb.ID),
			Name:        types.StringValue(hb.Name),
			GracePeriod: types.Int64Value(int64(hb.GracePeriod)),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
