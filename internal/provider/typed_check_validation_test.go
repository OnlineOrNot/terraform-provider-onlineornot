package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

func typedSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}

func checkStringEnum(t *testing.T, a schema.StringAttribute, valid, invalid []string) {
	t.Helper()
	for _, group := range []struct {
		values    []string
		wantError bool
	}{{valid, false}, {invalid, true}} {
		for _, value := range group.values {
			var resp validator.StringResponse
			for _, v := range a.Validators {
				v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("value"), ConfigValue: types.StringValue(value)}, &resp)
			}
			if resp.Diagnostics.HasError() != group.wantError {
				t.Errorf("%q: diagnostics %v, want error %v", value, resp.Diagnostics, group.wantError)
			}
		}
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		var resp validator.StringResponse
		for _, v := range a.Validators {
			v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("value"), ConfigValue: value}, &resp)
		}
		if resp.Diagnostics.HasError() {
			t.Errorf("null/unknown: %v", resp.Diagnostics)
		}
	}
}

func TestTypedCheckEnums(t *testing.T) {
	dns := typedSchema(t, NewDNSCheckResource())
	tcp := typedSchema(t, NewTCPCheckResource())
	checkStringEnum(t, dns.Attributes["dns_record_type"].(schema.StringAttribute), []string{"A", "AAAA", "CNAME", "MX", "NS", "SOA", "TXT"}, []string{"PTR", "SRV", "CAA", "a", ""})
	checkStringEnum(t, dns.Attributes["dns_protocol"].(schema.StringAttribute), []string{"UDP", "TCP"}, []string{"HTTPS", "udp", ""})
	checkStringEnum(t, tcp.Attributes["tcp_ip_family"].(schema.StringAttribute), []string{"IPv4", "IPv6"}, []string{"Any", "ipv4", ""})
	for _, tc := range []struct {
		name         string
		s            schema.Schema
		valid, wrong []string
	}{
		{"dns", dns, []string{"DNS_RESPONSE_CODE", "DNS_TEXT_ANSWER", "DNS_JSON_ANSWER"}, []string{"TCP_RESPONSE_TIME", "TCP_RESPONSE_DATA"}},
		{"tcp", tcp, []string{"TCP_RESPONSE_TIME", "TCP_RESPONSE_DATA"}, []string{"DNS_RESPONSE_CODE", "DNS_TEXT_ANSWER", "DNS_JSON_ANSWER"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.s.Attributes["assertions"].(schema.ListNestedAttribute)
			if !list.Optional || !list.Computed {
				t.Fatal("assertions must remain optional/computed")
			}
			fields := list.NestedObject.Attributes
			if len(fields) != 4 {
				t.Fatal(fields)
			}
			for _, name := range []string{"type", "property", "comparison", "expected"} {
				a := fields[name].(schema.StringAttribute)
				if !a.Required || a.Optional || a.Computed {
					t.Errorf("%s must be required", name)
				}
			}
			checkStringEnum(t, fields["type"].(schema.StringAttribute), tc.valid, append(tc.wrong, "JSON_BODY", "TEXT_BODY", "RESPONSE_HEADERS", "HTML_BODY", "", "unknown"))
			checkStringEnum(t, fields["comparison"].(schema.StringAttribute), []string{"EQUALS", "NOT_EQUALS", "GREATER_THAN", "LESS_THAN", "NULL", "NOT_NULL", "EMPTY", "NOT_EMPTY", "CONTAINS", "NOT_CONTAINS", "FALSE", "TRUE"}, []string{"equals", "MATCHES", ""})
			for _, name := range []string{"property", "expected"} {
				checkStringEnum(t, fields[name].(schema.StringAttribute), []string{"", "status", "$.answers[0].data", "responseTime"}, nil)
			}
		})
	}
}

// Exercise Terraform custom assertion values through model conversion and real
// client POST/PATCH encoding. Empty required strings must not be omitted.
func TestTypedCheckAssertionTransport(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"dns", "tcp"} {
		t.Run(kind, func(t *testing.T) {
			assertions := []client.MonitorAssertion{{Type: "DNS_RESPONSE_CODE", Property: "status", Comparison: "EQUALS", Expected: "NOERROR"}, {Type: "DNS_TEXT_ANSWER", Property: "", Comparison: "NOT_EMPTY", Expected: ""}, {Type: "DNS_JSON_ANSWER", Property: "$.answers[0].data", Comparison: "CONTAINS", Expected: "example"}}
			if kind == "tcp" {
				assertions = []client.MonitorAssertion{{Type: "TCP_RESPONSE_TIME", Property: "responseTime", Comparison: "LESS_THAN", Expected: "1000"}, {Type: "TCP_RESPONSE_DATA", Property: "", Comparison: "NOT_EMPTY", Expected: ""}}
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				wantMethod, wantPath := "POST", "/v1/checks/"+kind
				if calls == 2 {
					wantMethod, wantPath = "PATCH", wantPath+"/check-id"
				}
				if r.Method != wantMethod || r.URL.Path != wantPath {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				var got []map[string]string
				if err := json.Unmarshal(body["assertions"], &got); err != nil {
					t.Error(err)
				}
				want := make([]map[string]string, len(assertions))
				for i, a := range assertions {
					want[i] = map[string]string{"type": a.Type, "property": a.Property, "comparison": a.Comparison, "expected": a.Expected}
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("assertions = %#v, want %#v", got, want)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": body})
			}))
			defer server.Close()
			c := client.NewClient(&client.Config{APIKey: "test", BaseURL: server.URL})
			var diags diag.Diagnostics
			common := typedCheckModel{Name: types.StringValue("test"), Assertions: assertionListValue(ctx, assertions, &diags)}
			if kind == "dns" {
				model := DNSCheckModel{typedCheckModel: common, DNSDomain: types.StringValue("example.com"), DNSRecordType: types.StringValue("A"), DNSProtocol: types.StringValue("UDP")}
				input := dnsModelToClient(ctx, &model, &diags)
				created, err := c.CreateDNSCheck(input)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(created.Assertions, assertions) {
					t.Fatal(created.Assertions)
				}
				if _, err := c.UpdateDNSCheck("check-id", &client.DNSCheckPatch{DNSCheck: input}); err != nil {
					t.Fatal(err)
				}
			} else {
				model := TCPCheckModel{typedCheckModel: common, TCPHostname: types.StringValue("example.com"), TCPPort: types.Int64Value(443), TCPIPFamily: types.StringValue("IPv6")}
				input := tcpModelToClient(ctx, &model, &diags)
				created, err := c.CreateTCPCheck(input)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(created.Assertions, assertions) {
					t.Fatal(created.Assertions)
				}
				if _, err := c.UpdateTCPCheck("check-id", &client.TCPCheckPatch{TCPCheck: input}); err != nil {
					t.Fatal(err)
				}
			}
			if diags.HasError() {
				t.Fatal(diags)
			}
			if calls != 2 {
				t.Fatalf("got %d requests", calls)
			}
		})
	}
}
