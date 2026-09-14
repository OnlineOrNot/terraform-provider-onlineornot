package provider

import (
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Select configured, known fields before serialization. Unlike a response
// struct's omitempty tags, PATCH must retain explicit zero, false and empty
// collections, and must not turn an unknown or unmanaged field into zero.
func configuredCheckPatch(config tfsdk.Config, model any, diags *diag.Diagnostics) map[string]any {
	var configured map[string]tftypes.Value
	if err := config.Raw.As(&configured); err != nil {
		diags.AddError("Invalid check configuration", err.Error())
		return nil
	}
	fields := map[string]any{}
	value := reflect.ValueOf(model).Elem()
	for i := 0; i < value.NumField(); i++ {
		key := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
		setting, ok := configured[key]
		if !ok || setting.IsNull() || !setting.IsFullyKnown() {
			continue
		}
		v := value.Field(i)
		// Empty configured lists/maps are explicit clears, not JSON null.
		if v.Kind() == reflect.Slice && v.IsNil() {
			v = reflect.MakeSlice(v.Type(), 0, 0)
		}
		if v.Kind() == reflect.Map && v.IsNil() {
			v = reflect.MakeMap(v.Type())
		}
		fields[key] = v.Interface()
	}
	return fields
}
