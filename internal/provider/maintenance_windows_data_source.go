package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &MaintenanceWindowsDataSource{}

func NewMaintenanceWindowsDataSource() datasource.DataSource {
	return &MaintenanceWindowsDataSource{}
}

type MaintenanceWindowsDataSource struct {
	client *client.Client
}

type MaintenanceWindowsDataSourceModel struct {
	MaintenanceWindows []MaintenanceWindowDataModel `tfsdk:"maintenance_windows"`
}

type MaintenanceWindowDataModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	StartDate       types.String `tfsdk:"start_date"`
	DurationMinutes types.Int64  `tfsdk:"duration_minutes"`
	Timezone        types.String `tfsdk:"timezone"`
}

func (d *MaintenanceWindowsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_windows"
}

func (d *MaintenanceWindowsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of maintenance windows.",
		Attributes: map[string]schema.Attribute{
			"maintenance_windows": schema.ListNestedAttribute{
				Description: "List of maintenance windows",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the maintenance window",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the maintenance window",
							Computed:    true,
						},
						"start_date": schema.StringAttribute{
							Description: "The start time of the maintenance window",
							Computed:    true,
						},
						"duration_minutes": schema.Int64Attribute{
							Description: "The duration of the maintenance window in minutes",
							Computed:    true,
						},
						"timezone": schema.StringAttribute{
							Description: "The timezone for the maintenance window",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *MaintenanceWindowsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MaintenanceWindowsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MaintenanceWindowsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	windows, err := d.client.ListMaintenanceWindows()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read maintenance windows, got error: %s", err))
		return
	}

	data.MaintenanceWindows = make([]MaintenanceWindowDataModel, len(windows))
	for i, mw := range windows {
		data.MaintenanceWindows[i] = MaintenanceWindowDataModel{
			ID:              types.StringValue(mw.ID),
			Name:            types.StringValue(mw.Name),
			StartDate:       types.StringValue(mw.StartDate),
			DurationMinutes: types.Int64Value(int64(mw.DurationMinutes)),
			Timezone:        types.StringValue(mw.Timezone),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
