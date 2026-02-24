package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSource struct {
	client *client.Client
}

type UserDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Image     types.String `tfsdk:"image"`
	Role      types.String `tfsdk:"role"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a user by email or ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the user. Provide either id or email.",
				Optional:    true,
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user's email address. Provide either id or email.",
				Optional:    true,
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
			"image": schema.StringAttribute{
				Description: "URL to the user's avatar image",
				Computed:    true,
			},
			"role": schema.StringAttribute{
				Description: "The role of the user in the organisation (ADMIN or STANDARD)",
				Computed:    true,
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either id or email is provided
	if data.ID.IsNull() && data.Email.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'email' must be provided to look up a user.",
		)
		return
	}

	// Fetch all users and find the matching one
	users, err := d.client.ListUsers()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read users, got error: %s", err))
		return
	}

	var foundUser *client.User
	for i, user := range users {
		// Match by ID if provided
		if !data.ID.IsNull() && user.ID == data.ID.ValueString() {
			foundUser = &users[i]
			break
		}
		// Match by email if provided
		if !data.Email.IsNull() && user.Email != nil && *user.Email == data.Email.ValueString() {
			foundUser = &users[i]
			break
		}
	}

	if foundUser == nil {
		if !data.ID.IsNull() {
			resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("No user found with ID '%s'", data.ID.ValueString()))
		} else {
			resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("No user found with email '%s'", data.Email.ValueString()))
		}
		return
	}

	// Populate the model
	data.ID = types.StringValue(foundUser.ID)
	data.Role = types.StringValue(foundUser.Role)

	if foundUser.Email != nil {
		data.Email = types.StringValue(*foundUser.Email)
	} else {
		data.Email = types.StringNull()
	}
	if foundUser.FirstName != nil {
		data.FirstName = types.StringValue(*foundUser.FirstName)
	} else {
		data.FirstName = types.StringNull()
	}
	if foundUser.LastName != nil {
		data.LastName = types.StringValue(*foundUser.LastName)
	} else {
		data.LastName = types.StringNull()
	}
	if foundUser.Image != nil {
		data.Image = types.StringValue(*foundUser.Image)
	} else {
		data.Image = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
