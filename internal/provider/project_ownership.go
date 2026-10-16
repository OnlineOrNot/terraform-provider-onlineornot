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
	description := "Project ID. Omission on creation selects the stable Default project; later omission preserves current ownership. Changes move checks and heartbeats in place, preserving their identity."
	modifiers := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	if replacement {
		description = "Project ID. Omission on creation selects the stable Default project; later omission preserves current ownership. Changing this value requires replacement; this resource cannot move between projects."
		modifiers = append(modifiers, stringplanmodifier.RequiresReplace())
	}
	return schema.StringAttribute{Optional: true, Computed: true, Description: description, MarkdownDescription: description, PlanModifiers: modifiers, Validators: []validator.String{stringvalidator.LengthBetween(16, 128), stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9-]*[a-zA-Z-][a-zA-Z0-9-]*$"), "must be a project ID, not a project name")}}
}
