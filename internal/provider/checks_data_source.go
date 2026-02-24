package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &ChecksDataSource{}

func NewChecksDataSource() datasource.DataSource {
	return &ChecksDataSource{}
}

type ChecksDataSource struct {
	client *client.Client
}

type ChecksDataSourceModel struct {
	Checks []CheckDataModel `tfsdk:"checks"`
}

type CheckDataModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	CheckType types.String `tfsdk:"check_type"`
	Status    types.String `tfsdk:"status"`
	Method    types.String `tfsdk:"method"`
}

func (d *ChecksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_checks"
}

func (d *ChecksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of uptime checks.",
		Attributes: map[string]schema.Attribute{
			"checks": schema.ListNestedAttribute{
				Description: "List of uptime checks",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the check",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the check",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The URL being monitored",
							Computed:    true,
						},
						"check_type": schema.StringAttribute{
							Description: "The type of check (GET, POST, etc.)",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the check",
							Computed:    true,
						},
						"method": schema.StringAttribute{
							Description: "The HTTP method used",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ChecksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ChecksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ChecksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	checks, err := d.client.ListChecks()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read checks, got error: %s", err))
		return
	}

	data.Checks = make([]CheckDataModel, len(checks))
	for i, check := range checks {
		data.Checks[i] = CheckDataModel{
			ID:        types.StringValue(check.ID),
			Name:      types.StringValue(check.Name),
			URL:       types.StringValue(check.URL),
			CheckType: types.StringValue(check.CheckType),
			Status:    types.StringValue(check.Status),
			Method:    types.StringValue(check.Method),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
