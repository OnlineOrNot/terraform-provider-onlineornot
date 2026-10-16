package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func projectOwnershipAttribute(replacement bool) schema.StringAttribute {
	description := "Project ID. Omit on creation to use the API's default assignment. Omitting this attribute or setting it to null on updates preserves the current assignment; it does not move the resource to Unassigned. Unassigned resources have a null project ID in state."
	modifiers := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	if replacement {
		description += " Changing this value requires replacement."
		modifiers = append(modifiers, stringplanmodifier.RequiresReplace())
	} else {
		description += " Changing this value moves the resource to another project."
	}
	return schema.StringAttribute{Optional: true, Computed: true, Description: description, MarkdownDescription: description, PlanModifiers: modifiers, Validators: []validator.String{stringvalidator.LengthBetween(16, 128), stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9-]*[a-zA-Z-][a-zA-Z0-9-]*$"), "must be a project ID, not a project name")}}
}
