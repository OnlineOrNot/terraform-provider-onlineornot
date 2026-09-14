package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func TestHeartbeatReadHTTPErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" || req.URL.Path != "/v1/heartbeats/heartbeat1" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"success":false,"result":null,"errors":[{"code":10006,"message":"Heartbeat not found or unavailable"}],"messages":[]}`)
			}))
			defer server.Close()
			r := &HeartbeatResource{client: client.NewClient(&client.Config{BaseURL: server.URL, APIKey: "fixture"})}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			state := tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}
			if d := state.SetAttribute(ctx, path.Root("id"), types.StringValue("heartbeat1")); d.HasError() {
				t.Fatal(d)
			}
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if status == 404 {
				if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
					t.Fatalf("404 should remove resource: %v", response.Diagnostics)
				}
			} else if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("HTTP %d must report an error and preserve state: %v", status, response.Diagnostics)
			}
		})
	}
}
