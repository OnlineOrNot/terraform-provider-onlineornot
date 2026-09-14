package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Preserve omitted configuration, including prior nulls. Defaults apply only
// during creation. Callers must exclude values derived from other fields.
func preserveCheckConfiguration(s *schema.Schema, excluded ...string) {
	for name, attribute := range s.Attributes {
		if !attribute.IsOptional() || !attribute.IsComputed() {
			continue
		}
		skip := name == "paused" || name == "muted"
		for _, key := range excluded {
			skip = skip || key == name
		}
		if skip {
			continue
		}
		switch a := attribute.(type) {
		case schema.StringAttribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedString{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		case schema.Int64Attribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedInt64{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		case schema.BoolAttribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedBool{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		case schema.ListAttribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedList{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		case schema.MapAttribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedMap{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		case schema.ListNestedAttribute:
			a.PlanModifiers = append(a.PlanModifiers, omittedList{creationDefault: a.Default})
			a.Default = nil
			s.Attributes[name] = a
		}
	}
}

type omittedString struct{ creationDefault defaults.String }

func (m omittedString) Description(context.Context) string {
	return "Preserve omitted configuration on update; default only on creation."
}
func (m omittedString) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }
func (m omittedString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if m.creationDefault != nil {
		var result defaults.StringResponse
		m.creationDefault.DefaultString(ctx, defaults.StringRequest{Path: req.Path}, &result)
		resp.Diagnostics.Append(result.Diagnostics...)
		resp.PlanValue = result.PlanValue
	}
}

type omittedInt64 struct{ creationDefault defaults.Int64 }

func (m omittedInt64) Description(context.Context) string {
	return "Preserve omitted configuration on update; default only on creation."
}
func (m omittedInt64) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }
func (m omittedInt64) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if m.creationDefault != nil {
		var result defaults.Int64Response
		m.creationDefault.DefaultInt64(ctx, defaults.Int64Request{Path: req.Path}, &result)
		resp.Diagnostics.Append(result.Diagnostics...)
		resp.PlanValue = result.PlanValue
	}
}

type omittedBool struct{ creationDefault defaults.Bool }

func (m omittedBool) Description(context.Context) string {
	return "Preserve omitted configuration on update; default only on creation."
}
func (m omittedBool) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }
func (m omittedBool) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if m.creationDefault != nil {
		var result defaults.BoolResponse
		m.creationDefault.DefaultBool(ctx, defaults.BoolRequest{Path: req.Path}, &result)
		resp.Diagnostics.Append(result.Diagnostics...)
		resp.PlanValue = result.PlanValue
	}
}

type omittedList struct{ creationDefault defaults.List }

func (m omittedList) Description(context.Context) string {
	return "Preserve omitted configuration on update; default only on creation."
}
func (m omittedList) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }
func (m omittedList) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if m.creationDefault != nil {
		var result defaults.ListResponse
		m.creationDefault.DefaultList(ctx, defaults.ListRequest{Path: req.Path}, &result)
		resp.Diagnostics.Append(result.Diagnostics...)
		resp.PlanValue = result.PlanValue
	}
}

type omittedMap struct{ creationDefault defaults.Map }

func (m omittedMap) Description(context.Context) string {
	return "Preserve omitted configuration on update; default only on creation."
}
func (m omittedMap) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }
func (m omittedMap) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if m.creationDefault != nil {
		var result defaults.MapResponse
		m.creationDefault.DefaultMap(ctx, defaults.MapRequest{Path: req.Path}, &result)
		resp.Diagnostics.Append(result.Diagnostics...)
		resp.PlanValue = result.PlanValue
	}
}

// Paused and muted are projections of one API status, not independent flags.
type omittedOperationalState struct{ other string }

func (m omittedOperationalState) Description(context.Context) string {
	return "Preserve status unless the other operational state is enabled."
}
func (m omittedOperationalState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m omittedOperationalState) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var other types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(m.other), &other)...)
	if other.IsUnknown() {
		return
	}
	if other.ValueBool() {
		resp.PlanValue = types.BoolValue(false)
	} else {
		resp.PlanValue = req.StateValue
	}
}
