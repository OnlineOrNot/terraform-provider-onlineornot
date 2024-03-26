// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure OnlineornotProvider satisfies various provider interfaces.
var _ provider.Provider = &OnlineornotProvider{}
var _ provider.ProviderWithFunctions = &OnlineornotProvider{}

// OnlineornotProvider defines the provider implementation.
type OnlineornotProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// OnlineornotProviderModel describes the provider data model.
type OnlineornotProviderModel struct {
	ApiHost   types.String `tfsdk:"api_host"`
	ApiKey    types.String `tfsdk:"api_key" sensitive:"true"`
	ApiScheme types.String `tfsdk:"api_scheme"`
}

func (p *OnlineornotProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "onlineornot"
	resp.Version = p.version
}

func (p *OnlineornotProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_host": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Required: true,
			},
			"api_scheme": schema.StringAttribute{
				Optional: true,
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

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	c := client.NewClient(&client.Config{
		APIKey:    data.ApiKey.ValueString(),
		APIHost:   data.ApiHost.ValueString(),
		ApiScheme: data.ApiScheme.ValueString(),
		Debug:     false,
	})
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *OnlineornotProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
	}
}

func (p *OnlineornotProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OnlineornotProvider{
			version: version,
		}
	}
}
