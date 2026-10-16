package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

type ProjectsDataSource struct{ client *client.Client }
type projectsDataModel struct {
	Projects []projectModel `tfsdk:"projects"`
}

func NewProjectsDataSource() datasource.DataSource { return &ProjectsDataSource{} }
func (d *ProjectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists organization projects. Requires PROJECTS:READ.", Attributes: map[string]schema.Attribute{
		"projects": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Project ID."}, "name": schema.StringAttribute{Computed: true, Description: "Project name."}, "is_default": schema.BoolAttribute{Computed: true, Description: "Legacy Default-project flag. Null when the API omits it."}, "created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp."}, "updated_at": schema.StringAttribute{Computed: true, Description: "Last update timestamp."},
		}}},
	}}
}

func (d *ProjectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected data source configure type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
	}
}

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	rows, err := d.client.ListProjects()
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", err.Error())
		return
	}
	data := projectsDataModel{Projects: make([]projectModel, len(rows))}
	for i, p := range rows {
		data.Projects[i] = projectModelFromAPI(&p)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
