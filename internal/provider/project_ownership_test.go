package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_check"
)

func passesValidators(validators []validator.String, value string) bool {
	resp := &validator.StringResponse{}
	for _, v := range validators {
		v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("value"), ConfigValue: types.StringValue(value)}, resp)
	}
	return !resp.Diagnostics.HasError()
}

// Project IDs are either current 12-character lowercase IDs or legacy 16-128
// character IDs containing a letter or hyphen. Hand-written and generated
// schemas must agree with the API contract.
func TestProjectIDValidators(t *testing.T) {
	generated := resource_check.CheckResourceSchema(context.Background()).Attributes["project_id"].(schema.StringAttribute).Validators
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"izkpm55j334u", true},
		{"123456789012", true},
		{"projectSource123456", true},
		{"a1b2c3d4e5f6g7h8", true},
		{"1234567890123456-", true},
		{strings.Repeat("a", 128), true},
		{"IZKPM55J334U", false},
		{"izkpm55j334", false},
		{"izkpm55j334uu", false},
		{"abcdefghijklmno", false},
		{"1234567890123456", false},
		{"bad", false},
		{"my project name!", false},
		{strings.Repeat("a", 129), false},
	} {
		if got := passesValidators(projectIDValidators(), tc.value); got != tc.valid {
			t.Errorf("hand-written validators: %q valid=%v, want %v", tc.value, got, tc.valid)
		}
		if got := passesValidators(generated, tc.value); got != tc.valid {
			t.Errorf("generated validators: %q valid=%v, want %v", tc.value, got, tc.valid)
		}
	}
}

func TestTokenResourceAcceptsTeamsScope(t *testing.T) {
	resp := &frameworkresource.SchemaResponse{}
	NewTokenResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, resp)
	scope := resp.Schema.Attributes["grants"].(schema.SetNestedAttribute).NestedObject.Attributes["scope"].(schema.StringAttribute)
	if !passesValidators(scope.Validators, "TEAMS") {
		t.Fatal("TEAMS scope rejected")
	}
	if passesValidators(scope.Validators, "TEAM") {
		t.Fatal("unknown scope accepted")
	}
}
