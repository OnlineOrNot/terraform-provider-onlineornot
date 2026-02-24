package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &UsersDataSource{}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSource struct {
	client *client.Client
}

type UsersDataSourceModel struct {
	Users []UserModel `tfsdk:"users"`
}

type UserModel struct {
	ID        types.String `tfsdk:"id"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	Image     types.String `tfsdk:"image"`
	Role      types.String `tfsdk:"role"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of users in your organisation.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Description: "List of users",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the user",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "The user's first name",
							Computed:    true,
						},
						"last_name": schema.StringAttribute{
							Description: "The user's last name",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The user's email address",
							Computed:    true,
						},
						"image": schema.StringAttribute{
							Description: "URL to the user's avatar image",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "The role of the user in the organisation (ADMIN or STANDARD)",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.client.ListUsers()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read users, got error: %s", err))
		return
	}

	data.Users = make([]UserModel, len(users))
	for i, user := range users {
		data.Users[i] = UserModel{
			ID:   types.StringValue(user.ID),
			Role: types.StringValue(user.Role),
		}
		if user.FirstName != nil {
			data.Users[i].FirstName = types.StringValue(*user.FirstName)
		} else {
			data.Users[i].FirstName = types.StringNull()
		}
		if user.LastName != nil {
			data.Users[i].LastName = types.StringValue(*user.LastName)
		} else {
			data.Users[i].LastName = types.StringNull()
		}
		if user.Email != nil {
			data.Users[i].Email = types.StringValue(*user.Email)
		} else {
			data.Users[i].Email = types.StringNull()
		}
		if user.Image != nil {
			data.Users[i].Image = types.StringValue(*user.Image)
		} else {
			data.Users[i].Image = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
