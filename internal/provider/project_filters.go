package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func projectFilterAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: "Filter by one project ID, or unassigned for resources without a project. Omit both project_id and project_ids to list across all projects. Cannot be combined with project_ids.",
		Validators:  []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("project_ids"))},
	}
}

func projectFiltersAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: "Filter by up to 100 project IDs. Include unassigned for resources without a project. Omit both project_id and project_ids to list across all projects. Cannot be combined with project_id.",
		Validators:  []validator.List{listvalidator.SizeAtMost(100), listvalidator.ConflictsWith(path.MatchRoot("project_id"))},
	}
}

func projectFilterValues(ctx context.Context, projectID types.String, projectIDs types.List, resp *datasource.ReadResponse) []string {
	if !projectIDs.IsNull() {
		var ids []string
		resp.Diagnostics.Append(projectIDs.ElementsAs(ctx, &ids, false)...)
		return ids
	}
	if id := projectID.ValueString(); id != "" {
		return []string{id}
	}
	return nil
}
