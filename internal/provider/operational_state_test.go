package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOperationalStateChanges(t *testing.T) {
	t.Run("rejects simultaneously enabled states", func(t *testing.T) {
		_, err := operationalStateChanges(types.BoolValue(true), types.BoolValue(true), types.BoolValue(false), types.BoolValue(false))
		if err == nil {
			t.Fatal("expected conflicting operational states to fail")
		}
	})

	t.Run("moves from paused to muted in API-safe order", func(t *testing.T) {
		changes, err := operationalStateChanges(types.BoolValue(false), types.BoolValue(true), types.BoolValue(true), types.BoolValue(false))
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) != 2 || changes[0] != (operationalStateChange{field: "paused", value: false}) || changes[1] != (operationalStateChange{field: "muted", value: true}) {
			t.Fatalf("unexpected changes: %#v", changes)
		}
	})

	t.Run("moves from muted to paused in API-safe order", func(t *testing.T) {
		changes, err := operationalStateChanges(types.BoolValue(true), types.BoolValue(false), types.BoolValue(false), types.BoolValue(true))
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) != 2 || changes[0] != (operationalStateChange{field: "muted", value: false}) || changes[1] != (operationalStateChange{field: "paused", value: true}) {
			t.Fatalf("unexpected changes: %#v", changes)
		}
	})

	t.Run("leaves computed values unchanged", func(t *testing.T) {
		changes, err := operationalStateChanges(types.BoolUnknown(), types.BoolUnknown(), types.BoolValue(true), types.BoolValue(false))
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) != 0 {
			t.Fatalf("expected no changes, got %#v", changes)
		}
	})
}
