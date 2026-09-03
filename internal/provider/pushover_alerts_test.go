package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_check"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider/resource_heartbeat"
)

func TestCheckResourcePopulateModelFromAPIPushoverAlerts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		alerts   []string
		expected types.List
	}{
		"populated": {
			alerts:   []string{"pushover-1"},
			expected: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("pushover-1")}),
		},
		"empty": {
			alerts:   nil,
			expected: types.ListNull(types.StringType),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := resource_check.CheckModel{
				PushoverAlerts: types.ListUnknown(types.StringType),
			}
			var diagnostics diag.Diagnostics

			resource := &CheckResource{}
			resource.populateModelFromAPI(context.Background(), &data, &client.Check{
				PushoverAlerts: test.alerts,
			}, &diagnostics)

			if diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diagnostics)
			}
			if !data.PushoverAlerts.Equal(test.expected) {
				t.Fatalf("expected pushover alerts %s, got %s", test.expected, data.PushoverAlerts)
			}
		})
	}
}

func TestPopulateHeartbeatPushoverAlerts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		alerts   []string
		expected types.List
	}{
		"populated": {
			alerts:   []string{"pushover-1"},
			expected: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("pushover-1")}),
		},
		"empty": {
			alerts:   nil,
			expected: types.ListNull(types.StringType),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := resource_heartbeat.HeartbeatModel{
				PushoverAlerts: types.ListUnknown(types.StringType),
			}
			var diagnostics diag.Diagnostics

			populateHeartbeatPushoverAlerts(context.Background(), &data, test.alerts, &diagnostics)

			if diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diagnostics)
			}
			if !data.PushoverAlerts.Equal(test.expected) {
				t.Fatalf("expected pushover alerts %s, got %s", test.expected, data.PushoverAlerts)
			}
		})
	}
}
