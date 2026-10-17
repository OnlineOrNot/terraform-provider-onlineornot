package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProjectLegacyStateUpgrade(t *testing.T) {
	for _, marker := range []any{nil, false, true, "omitted"} {
		t.Run(fmt.Sprint(marker), func(t *testing.T) {
			server, err := testAccProtoV6ProviderFactories["onlineornot"]()
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			providerSchema, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{
				"id":         projectSourceID,
				"name":       "Default",
				"created_at": "2026-10-16T00:00:00Z",
				"updated_at": "2026-10-16T00:00:00Z",
			}
			prior := make(map[string]any, len(expected)+1)
			for key, value := range expected {
				prior[key] = value
			}
			if marker != "omitted" {
				prior["is_default"] = marker
			}
			raw, err := json.Marshal(prior)
			if err != nil {
				t.Fatal(err)
			}
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "onlineornot_project", Version: 0, RawState: &tfprotov6.RawState{JSON: raw},
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range upgraded.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("state upgrade failed: %s: %s", diagnostic.Summary, diagnostic.Detail)
				}
			}
			if upgraded.UpgradedState == nil {
				t.Fatal("missing upgraded state")
			}
			value, err := upgraded.UpgradedState.Unmarshal(providerSchema.ResourceSchemas["onlineornot_project"].ValueType())
			if err != nil {
				t.Fatal(err)
			}
			var attributes map[string]tftypes.Value
			if err := value.As(&attributes); err != nil {
				t.Fatal(err)
			}
			if len(attributes) != len(expected) {
				t.Fatalf("unexpected upgraded project attributes: %v", attributes)
			}
			for key, want := range expected {
				var got string
				if err := attributes[key].As(&got); err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Errorf("upgraded %s = %q, want %q", key, got, want)
				}
			}
		})
	}
}
