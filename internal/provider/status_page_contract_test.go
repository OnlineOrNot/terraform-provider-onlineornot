package provider

import (
	"context"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestStatusPageComponentResourceSchemaIncludesOpenAPIRelationships(t *testing.T) {
	var response frameworkresource.SchemaResponse
	(&StatusPageComponentResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	for _, name := range []string{"group_id", "check_ids", "heartbeat_id", "override_status"} {
		if _, ok := response.Schema.Attributes[name]; !ok {
			t.Errorf("expected component schema to include %q", name)
		}
	}
}

func TestStatusPageIncidentCreateOnlyFieldsRequireReplacement(t *testing.T) {
	var response frameworkresource.SchemaResponse
	(&StatusPageIncidentResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	if _, ok := response.Schema.Attributes["impact"].(schema.StringAttribute); !ok {
		t.Fatalf("expected incident impact to be a string attribute, got %T", response.Schema.Attributes["impact"])
	}
	for _, name := range []string{"description", "status"} {
		attribute := response.Schema.Attributes[name].(schema.StringAttribute)
		if len(attribute.PlanModifiers) == 0 {
			t.Errorf("expected incident %q changes to require replacement", name)
		}
	}
	components := response.Schema.Attributes["components"].(schema.ListNestedAttribute)
	if len(components.PlanModifiers) == 0 {
		t.Error("expected incident component changes to require replacement")
	}
	notify := response.Schema.Attributes["notify_subscribers"].(schema.BoolAttribute)
	if len(notify.PlanModifiers) == 0 {
		t.Error("expected incident notification changes to require replacement")
	}
}

func TestScheduledMaintenanceCreateOnlyFieldsRequireReplacement(t *testing.T) {
	var response frameworkresource.SchemaResponse
	(&StatusPageScheduledMaintenanceResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	description := response.Schema.Attributes["description"].(schema.StringAttribute)
	components := response.Schema.Attributes["components_affected"].(schema.ListAttribute)
	notifications := response.Schema.Attributes["notifications"].(schema.SingleNestedAttribute)
	if len(description.PlanModifiers) == 0 || len(components.PlanModifiers) == 0 || len(notifications.PlanModifiers) == 0 {
		t.Error("expected unsupported scheduled-maintenance updates to require replacement")
	}
}
