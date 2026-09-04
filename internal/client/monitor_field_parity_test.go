package client

import (
	"encoding/json"
	"testing"
)

func boolPointer(value bool) *bool { return &value }

func payloadMap(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	return payload
}

func requireNoOperationalState(t *testing.T, payload map[string]any) {
	t.Helper()
	for _, field := range []string{"paused", "muted"} {
		if _, ok := payload[field]; ok {
			t.Errorf("create payload must not contain %q: %#v", field, payload)
		}
	}
}

func requireSingleOperationalState(t *testing.T, payload map[string]any, field string, value bool) {
	t.Helper()
	if payload[field] != value {
		t.Errorf("expected %s=%v, got %#v", field, value, payload[field])
	}
	other := "muted"
	if field == "muted" {
		other = "paused"
	}
	if _, ok := payload[other]; ok {
		t.Errorf("PATCH payload cannot contain both operational fields: %#v", payload)
	}
}

func TestCheckCreateAndPatchPayloadsSeparateOperationalState(t *testing.T) {
	check := &Check{Name: "API", URL: "https://example.com"}
	requireNoOperationalState(t, payloadMap(t, check))
	requireSingleOperationalState(t, payloadMap(t, &CheckPatch{Check: check, Paused: boolPointer(true)}), "paused", true)
}

func TestTypedCheckPayloadsIncludePushoverOnlyOnSupportedBasePayload(t *testing.T) {
	dns := &DNSCheck{Name: "DNS", DNSDomain: "example.com", DNSRecordType: "A", PushoverAlerts: []string{"push-1"}}
	tcp := &TCPCheck{Name: "TCP", TCPHostname: "example.com", TCPPort: 443, PushoverAlerts: []string{"push-2"}}

	for name, test := range map[string]struct {
		create any
		patch  any
		alert  string
	}{
		"dns": {dns, &DNSCheckPatch{DNSCheck: dns, Muted: boolPointer(true)}, "push-1"},
		"tcp": {tcp, &TCPCheckPatch{TCPCheck: tcp, Paused: boolPointer(true)}, "push-2"},
	} {
		t.Run(name, func(t *testing.T) {
			create := payloadMap(t, test.create)
			requireNoOperationalState(t, create)
			alerts, ok := create["pushover_alerts"].([]any)
			if !ok || len(alerts) != 1 || alerts[0] != test.alert {
				t.Errorf("expected pushover alert %q, got %#v", test.alert, create["pushover_alerts"])
			}
			patch := payloadMap(t, test.patch)
			if name == "tcp" {
				requireSingleOperationalState(t, patch, "paused", true)
			} else {
				requireSingleOperationalState(t, patch, "muted", true)
			}
		})
	}
}

func TestHeartbeatCreateAndPatchPayloadsSeparateOperationalState(t *testing.T) {
	heartbeat := &Heartbeat{Name: "Cron", GracePeriod: 60, TelegramAlerts: []string{"telegram-1"}}
	create := payloadMap(t, heartbeat)
	requireNoOperationalState(t, create)
	alerts, ok := create["telegram_alerts"].([]any)
	if !ok || len(alerts) != 1 || alerts[0] != "telegram-1" {
		t.Errorf("expected Telegram alert in create payload, got %#v", create["telegram_alerts"])
	}

	patch := payloadMap(t, &HeartbeatPatch{Heartbeat: heartbeat, Muted: boolPointer(true)})
	requireSingleOperationalState(t, patch, "muted", true)
}
