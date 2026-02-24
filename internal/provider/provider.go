package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

// Ensure OnlineornotProvider satisfies various provider interfaces.
var _ provider.Provider = &OnlineornotProvider{}

// OnlineornotProvider defines the provider implementation.
type OnlineornotProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// OnlineornotProviderModel describes the provider data model.
type OnlineornotProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *OnlineornotProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "onlineornot"
	resp.Version = p.version
}

func (p *OnlineornotProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The OnlineOrNot provider allows you to manage uptime checks, heartbeats, status pages, and more.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with the OnlineOrNot API. Can also be set via the ONLINEORNOT_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL for the OnlineOrNot API. Defaults to https://api.onlineornot.com.",
				Optional:    true,
			},
		},
	}
}

func (p *OnlineornotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OnlineornotProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get API key from config or environment variable
	apiKey := data.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("ONLINEORNOT_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires an API key. Set 'api_key' in the provider configuration or set the ONLINEORNOT_API_KEY environment variable.",
		)
		return
	}

	// Get base URL from config or use default
	baseURL := data.BaseURL.ValueString()

	// Create client
	c := client.NewClient(&client.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OnlineornotProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCheckResource,
		NewStatusPageResource,
		NewHeartbeatResource,
		NewWebhookResource,
		NewMaintenanceWindowResource,
		NewStatusPageComponentResource,
		NewStatusPageComponentGroupResource,
		NewStatusPageIncidentResource,
		NewStatusPageScheduledMaintenanceResource,
	}
}

func (p *OnlineornotProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewUsersDataSource,
		NewChecksDataSource,
		NewHeartbeatsDataSource,
		NewStatusPagesDataSource,
		NewWebhooksDataSource,
		NewMaintenanceWindowsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OnlineornotProvider{
			version: version,
		}
	}
}
