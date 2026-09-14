package client

import (
	"encoding/json"
	"testing"
)

func TestCheckAlertPriorityEncoding(t *testing.T) {
	for _, priority := range []string{"", "LOW", "HIGH"} {
		for kind, payload := range map[string]any{
			"check create": &Check{AlertPriority: priority},
			"check patch":  &CheckPatch{Check: &Check{AlertPriority: priority}},
			"dns create":   &DNSCheck{AlertPriority: priority},
			"dns patch":    &DNSCheckPatch{DNSCheck: &DNSCheck{AlertPriority: priority}},
			"tcp create":   &TCPCheck{AlertPriority: priority},
			"tcp patch":    &TCPCheckPatch{TCPCheck: &TCPCheck{AlertPriority: priority}},
		} {
			t.Run(kind+"/"+priority, func(t *testing.T) {
				body, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]any
				if err := json.Unmarshal(body, &fields); err != nil {
					t.Fatal(err)
				}
				got, present := fields["alert_priority"]
				if priority == "" {
					if present {
						t.Fatalf("omission must allow server create default or preserve PATCH priority: %s", body)
					}
				} else if got != priority {
					t.Fatalf("priority = %v, want %s", got, priority)
				}
			})
		}
	}
}
