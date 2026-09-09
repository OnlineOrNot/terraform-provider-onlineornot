package provider

import (
	"context"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func requireSchemaFields(t *testing.T, resource frameworkresource.Resource, fields ...string) {
	t.Helper()
	var response frameworkresource.SchemaResponse
	resource.Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)
	for _, field := range fields {
		if _, ok := response.Schema.Attributes[field]; !ok {
			t.Errorf("expected schema to include %q", field)
		}
	}
}

func TestMonitorSchemasIncludeOperationalState(t *testing.T) {
	for name, resource := range map[string]frameworkresource.Resource{
		"generic":   NewCheckResource(),
		"uptime":    NewUptimeCheckResource(),
		"browser":   NewBrowserCheckResource(),
		"dns":       NewDNSCheckResource(),
		"tcp":       NewTCPCheckResource(),
		"heartbeat": NewHeartbeatResource(),
	} {
		t.Run(name, func(t *testing.T) {
			requireSchemaFields(t, resource, "paused", "muted")
		})
	}
}

func TestTypedMonitorAndHeartbeatSchemasIncludeMissingAlertChannels(t *testing.T) {
	requireSchemaFields(t, NewDNSCheckResource(), "pushover_alerts")
	requireSchemaFields(t, NewTCPCheckResource(), "pushover_alerts")
	requireSchemaFields(t, NewHeartbeatResource(), "telegram_alerts")
}
