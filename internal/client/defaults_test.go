package client

import (
	"encoding/json"
	"testing"
)

func TestMonitorExplicitZeroJSON(t *testing.T) {
	for name, input := range map[string]any{
		"check create": &Check{}, "check update": &CheckPatch{Check: &Check{}},
		"dns create": &DNSCheck{}, "dns update": &DNSCheckPatch{DNSCheck: &DNSCheck{}},
		"tcp create": &TCPCheck{}, "tcp update": &TCPCheckPatch{TCPCheck: &TCPCheck{}},
		"heartbeat create": &Heartbeat{}, "heartbeat update": &HeartbeatPatch{Heartbeat: &Heartbeat{}},
	} {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			keys := []string{"reminder_alert_interval_minutes"}
			if name != "heartbeat create" && name != "heartbeat update" {
				keys = append(keys, "confirmation_period_seconds", "recovery_period_seconds")
			}
			for _, key := range keys {
				if value, ok := fields[key]; !ok || value != float64(0) {
					t.Errorf("%s omitted or changed zero: %s", key, body)
				}
			}
		})
	}
}

func TestMaintenanceEmptyDefaultsJSON(t *testing.T) {
	body, err := json.Marshal(&MaintenanceWindow{Checks: []string{}, Heartbeats: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"checks", "heartbeats"} {
		value, ok := fields[key].([]any)
		if !ok || len(value) != 0 {
			t.Errorf("%s must be [], got %s", key, body)
		}
	}
}
