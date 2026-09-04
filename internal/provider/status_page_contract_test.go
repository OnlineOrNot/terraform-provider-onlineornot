package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStatusPageComponentResourceSchemaIncludesOpenAPIRelationships(t *testing.T) {
	var response frameworkresource.SchemaResponse
	(&StatusPageComponentResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	for _, name := range []string{"group_id", "check_ids", "heartbeat_id", "override_status"} {
		if _, ok := response.Schema.Attributes[name]; !ok {
			t.Errorf("expected component schema to include %q", name)
		}
	}
	group := response.Schema.Attributes["group_id"].(schema.StringAttribute)
	if group.Computed {
		t.Error("expected group_id to remain null when omitted so removing it clears a managed group")
	}
	overrideStatus := response.Schema.Attributes["override_status"].(schema.BoolAttribute)
	if overrideStatus.Computed || overrideStatus.Default != nil {
		t.Error("expected override_status to remain unmanaged when omitted")
	}
}

func TestComponentPatchOmitsUnmanagedRelationships(t *testing.T) {
	data := statusPageComponentModel{
		Name:           types.StringValue("API"),
		Status:         types.StringValue("OPERATIONAL"),
		GroupId:        types.StringNull(),
		CheckIds:       types.ListNull(types.StringType),
		HeartbeatId:    types.StringNull(),
		OverrideStatus: types.BoolNull(),
	}
	var diagnostics diag.Diagnostics
	patch := componentPatchFromModel(context.Background(), &data, nil, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if patch.GroupID != nil || patch.CheckIDs != nil || patch.HeartbeatID != nil || patch.OverrideStatus != nil {
		t.Fatalf("expected unmanaged relationships to be omitted, got %#v", patch)
	}
}

func TestComponentPatchClearsRemovedRelationships(t *testing.T) {
	data := statusPageComponentModel{
		Name:           types.StringValue("API"),
		Status:         types.StringValue("OPERATIONAL"),
		GroupId:        types.StringNull(),
		CheckIds:       types.ListNull(types.StringType),
		HeartbeatId:    types.StringNull(),
		OverrideStatus: types.BoolNull(),
	}
	priorCheckIDs, priorDiagnostics := types.ListValue(types.StringType, []attr.Value{types.StringValue("check123")})
	if priorDiagnostics.HasError() {
		t.Fatalf("failed to construct prior check IDs: %v", priorDiagnostics)
	}
	prior := statusPageComponentModel{
		GroupId:        types.StringValue("group123"),
		CheckIds:       priorCheckIDs,
		HeartbeatId:    types.StringValue("heart123"),
		OverrideStatus: types.BoolValue(true),
	}
	var diagnostics diag.Diagnostics
	patch := componentPatchFromModel(context.Background(), &data, &prior, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if patch.GroupID == nil || *patch.GroupID != nil {
		t.Error("expected an explicit null group_id")
	}
	if patch.CheckIDs == nil || len(*patch.CheckIDs) != 0 {
		t.Error("expected an explicit empty check_ids list")
	}
	if patch.HeartbeatID == nil || *patch.HeartbeatID != nil {
		t.Error("expected an explicit null heartbeat_id")
	}
	if patch.OverrideStatus == nil || *patch.OverrideStatus {
		t.Error("expected override_status to reset to false")
	}
}

func TestComponentPatchReportsInvalidCheckIDs(t *testing.T) {
	checkIDs, listDiagnostics := types.ListValue(types.StringType, []attr.Value{types.StringNull()})
	if listDiagnostics.HasError() {
		t.Fatalf("failed to construct check IDs: %v", listDiagnostics)
	}
	data := statusPageComponentModel{
		Name:     types.StringValue("API"),
		Status:   types.StringValue("OPERATIONAL"),
		CheckIds: checkIDs,
	}
	var diagnostics diag.Diagnostics
	componentPatchFromModel(context.Background(), &data, nil, &diagnostics)

	if !diagnostics.HasError() {
		t.Fatal("expected a null check ID to produce a conversion error")
	}
}

func TestStatusPageIncidentCreateOnlyFieldsRequireReplacement(t *testing.T) {
	var response frameworkresource.SchemaResponse
	(&StatusPageIncidentResource{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)

	impact, ok := response.Schema.Attributes["impact"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected incident impact to be a string attribute, got %T", response.Schema.Attributes["impact"])
	}
	if len(impact.PlanModifiers) == 0 {
		t.Error("expected incident impact to preserve state when omitted during another update")
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
