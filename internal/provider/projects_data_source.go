package provider

import (
	"context"

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
func (d *ProjectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureTypedCheckClient(req.ProviderData, &resp.Diagnostics)
}
func (d *ProjectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists all authorized projects. Select by encoded ID; names are not unique and Default identity does not depend on its name.", Attributes: map[string]schema.Attribute{"projects": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Computed: true}, "is_default": schema.BoolAttribute{Computed: true}, "created_at": schema.StringAttribute{Computed: true}, "updated_at": schema.StringAttribute{Computed: true},
	}}}}}
}
func (d *ProjectsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	projects, err := d.client.ListProjects()
	if err != nil {
		resp.Diagnostics.AddError("Unable to list projects", err.Error())
		return
	}
	data := projectsDataModel{Projects: make([]projectModel, len(projects))}
	for i, p := range projects {
		data.Projects[i] = projectModelFromAPI(&p)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
