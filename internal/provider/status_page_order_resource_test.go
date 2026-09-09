package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var orderFactories = []func() resource.Resource{NewStatusPageComponentOrderResource, NewStatusPageGroupOrderResource, NewStatusPageGroupComponentOrderResource}

func orderTestState(t *testing.T, r *StatusPageOrderResource, ids attr.Value) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var sch resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sch)
	values := map[string]attr.Value{"id": types.StringValue("page1234"), "status_page_id": types.StringValue("page1234"), r.idsKey: ids}
	if r.kind == "group_components" {
		values["group_id"] = types.StringValue("group123")
		values["id"] = types.StringValue("page1234/group123")
	}
	obj, ds := types.ObjectValue(sch.Schema.Type().(types.ObjectType).AttributeTypes(), values)
	if ds.HasError() {
		t.Fatal(ds)
	}
	raw, err := obj.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: sch.Schema, Raw: raw}
}
func orderTestList(t *testing.T, ids ...string) types.List {
	t.Helper()
	if ids == nil {
		ids = []string{}
	}
	v, ds := types.ListValueFrom(context.Background(), types.StringType, ids)
	if ds.HasError() {
		t.Fatal(ds)
	}
	return v
}
func orderStateIDs(t *testing.T, r *StatusPageOrderResource, s tfsdk.State) []string {
	t.Helper()
	var ids []string
	ds := s.GetAttribute(context.Background(), path.Root(r.idsKey), &ids)
	if ds.HasError() {
		t.Fatal(ds)
	}
	return ids
}

type orderMock struct {
	ids                      []string
	puts, calls              int
	parentStatus, listStatus int
	groupStatus              int
	wrongOrder               bool
	kind                     string
}

func (m *orderMock) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	m.calls++
	list := "/v1/status_pages/page1234/components"
	if m.kind == "groups" {
		list = "/v1/status_pages/page1234/groups"
	}
	if r.Method == "PUT" {
		m.puts++
		want := list + "/sort-order"
		if m.kind == "group_components" {
			want = "/v1/status_pages/page1234/groups/group123/sort-order"
		}
		if r.URL.Path != want {
			t.Errorf("wrong PUT path %s", r.URL.Path)
		}
		var b map[string][]string
		json.NewDecoder(r.Body).Decode(&b)
		key := "component_ids"
		if m.kind == "groups" {
			key = "group_ids"
		}
		if !m.wrongOrder {
			m.ids = b[key]
		}
		fmt.Fprint(w, `{"success":true,"result":{"message":"OK"}}`)
		return
	}
	if r.Method != "GET" {
		t.Errorf("unexpected mutation %s", r.Method)
		w.WriteHeader(500)
		return
	}
	if r.URL.Path == list {
		if m.listStatus != 0 {
			w.WriteHeader(m.listStatus)
			fmt.Fprint(w, `{"success":false}`)
			return
		}
		items := []map[string]any{}
		for _, id := range m.ids {
			group := any(nil)
			if m.kind == "group_components" {
				group = "group123"
			}
			items = append(items, map[string]any{"id": id, "group_id": group})
		}
		// Ensure grouped and ungrouped order resources exclude the other scope.
		if m.kind != "groups" {
			group := any("othergroup")
			items = append(items, map[string]any{"id": "outside1", "group_id": group})
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "result": items, "result_info": map[string]any{"page": r.URL.Query().Get("page"), "per_page": r.URL.Query().Get("per_page"), "count": len(items), "total_count": len(items)}})
		return
	}
	if r.URL.Path != "/v1/status_pages/page1234" && r.URL.Path != "/v1/status_pages/page1234/groups/group123" {
		t.Errorf("unexpected GET %s", r.URL.Path)
	}
	if m.groupStatus != 0 && r.URL.Path == "/v1/status_pages/page1234/groups/group123" {
		w.WriteHeader(m.groupStatus)
		fmt.Fprint(w, `{"success":false}`)
		return
	}
	if m.parentStatus != 0 {
		w.WriteHeader(m.parentStatus)
		fmt.Fprint(w, `{"success":false,"errors":[{"code":404,"message":"not found"}]}`)
		return
	}
	fmt.Fprint(w, `{"success":true,"result":{"id":"page1234","name":"parent"}}`)
}
func setupOrderMock(t *testing.T, r *StatusPageOrderResource) *orderMock {
	t.Helper()
	m := &orderMock{ids: []string{"first", "second"}, kind: r.kind}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { m.serve(t, w, req) }))
	t.Cleanup(s.Close)
	r.client = client.NewClient(&client.Config{BaseURL: s.URL, APIKey: "mock-only"})
	return m
}

func TestOrderResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, factory := range orderFactories {
		r := factory().(*StatusPageOrderResource)
		t.Run(r.kind, func(t *testing.T) {
			m := setupOrderMock(t, r)
			state := orderTestState(t, r, orderTestList(t, "second", "first"))
			create := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &create)
			if create.Diagnostics.HasError() {
				t.Fatal(create.Diagnostics)
			}
			if m.puts != 1 || !slices.Equal(m.ids, []string{"second", "first"}) {
				t.Fatalf("create %v", m)
			}
			state = create.State
			plan := orderTestState(t, r, orderTestList(t, "first", "second"))
			update := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &update)
			if update.Diagnostics.HasError() {
				t.Fatal(update.Diagnostics)
			}
			state = update.State
			if m.puts != 2 || !slices.Equal(orderStateIDs(t, r, state), []string{"first", "second"}) {
				t.Fatal("update failed")
			}
			for _, drift := range [][]string{{"second", "first"}, {"second", "newmember", "first"}, {"first"}, {}} {
				m.ids = drift
				read := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &read)
				if read.Diagnostics.HasError() {
					t.Fatal(read.Diagnostics)
				}
				if !slices.Equal(orderStateIDs(t, r, read.State), drift) {
					t.Fatal("missed drift")
				}
				state = read.State
			}
			// Empty scope must remain managed and serialize an actual [] on apply.
			empty := orderTestState(t, r, orderTestList(t))
			update = resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: empty.Schema, Raw: empty.Raw}}, &update)
			if update.Diagnostics.HasError() {
				t.Fatal(update.Diagnostics)
			}
			state = update.State
			var emptyList types.List
			if ds := state.GetAttribute(ctx, path.Root(r.idsKey), &emptyList); ds.HasError() || emptyList.IsNull() || emptyList.IsUnknown() || len(emptyList.Elements()) != 0 {
				t.Fatal("empty scope must save known []")
			}
			id := "page1234"
			if r.kind == "group_components" {
				id += "/group123"
			}
			imported := resource.ImportStateResponse{State: tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &imported)
			if imported.Diagnostics.HasError() {
				t.Fatal(imported.Diagnostics)
			}
			read := resource.ReadResponse{State: imported.State}
			r.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			before := m.calls
			del := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &del)
			if del.Diagnostics.HasError() || m.calls != before {
				t.Fatal("destroy must not call API")
			}
		})
	}
}

func TestOrderResourceRejectsInvalidMembership(t *testing.T) {
	ctx := context.Background()
	for _, factory := range orderFactories {
		r := factory().(*StatusPageOrderResource)
		t.Run(r.kind, func(t *testing.T) {
			m := setupOrderMock(t, r)
			cases := []types.List{orderTestList(t, "first"), orderTestList(t, "first", "second", "extra"), orderTestList(t, "first", "outside1"), orderTestList(t, "first", "first"), orderTestList(t, "first", "../second"), types.ListUnknown(types.StringType), types.ListNull(types.StringType)}
			unknown, _ := types.ListValue(types.StringType, []attr.Value{types.StringUnknown()})
			cases = append(cases, unknown)
			null, _ := types.ListValue(types.StringType, []attr.Value{types.StringNull()})
			cases = append(cases, null)
			for _, ids := range cases {
				state := orderTestState(t, r, ids)
				resp := resource.CreateResponse{State: state}
				r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatalf("accepted %s", ids)
				}
			}
			if m.puts != 0 {
				t.Fatal("invalid membership mutated ranks")
			}
		})
	}
}

func TestOrderResourceParentAndPermissionErrors(t *testing.T) {
	ctx := context.Background()
	for _, factory := range orderFactories {
		r := factory().(*StatusPageOrderResource)
		t.Run(r.kind, func(t *testing.T) {
			m := setupOrderMock(t, r)
			state := orderTestState(t, r, orderTestList(t, "first", "second"))
			for _, status := range []int{401, 403, 404, 500} {
				m.parentStatus = status
				resp := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &resp)
				if status == 404 {
					if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
						t.Fatal("404 should remove ownership")
					}
				} else {
					if !resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
						t.Fatal("permission/error must preserve state")
					}
				}
			}
			m.parentStatus = 0
			if r.kind == "group_components" {
				for _, status := range []int{403, 404} {
					m.groupStatus = status
					read := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &read)
					if status == 404 {
						if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
							t.Fatal("missing group must remove ownership")
						}
					} else if !read.Diagnostics.HasError() || read.State.Raw.IsNull() {
						t.Fatal("group permission failure must preserve ownership")
					}
				}
				m.groupStatus = 0
			}
			m.listStatus = 404
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if !resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
				t.Fatal("list 404 is not confirmed parent absence")
			}
			m.listStatus = 0
			m.wrongOrder = true
			plan := orderTestState(t, r, orderTestList(t, "second", "first"))
			created := resource.CreateResponse{State: state}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &created)
			if !created.Diagnostics.HasError() || created.State.Raw.IsNull() {
				t.Fatal("verification failure must retain ownership")
			}
		})
	}
}

func TestOrderResourceSchemaAndImportValidation(t *testing.T) {
	ctx := context.Background()
	for _, factory := range orderFactories {
		r := factory().(*StatusPageOrderResource)
		t.Run(r.kind, func(t *testing.T) {
			var sch resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &sch)
			for _, key := range []string{"status_page_id", "group_id"} {
				if a, ok := sch.Schema.Attributes[key]; ok {
					mods := a.(schema.StringAttribute).PlanModifiers
					if len(mods) != 1 {
						t.Fatalf("identity must replace: %v", mods)
					}
					prior := orderTestState(t, r, orderTestList(t, "first", "second"))
					modified := planmodifier.StringResponse{}
					mods[0].PlanModifyString(ctx, planmodifier.StringRequest{State: prior, Plan: tfsdk.Plan{Schema: prior.Schema, Raw: prior.Raw}, StateValue: types.StringValue("old"), PlanValue: types.StringValue("new"), ConfigValue: types.StringValue("new")}, &modified)
					if !modified.RequiresReplace || modified.Diagnostics.HasError() {
						t.Fatal("identity change did not require replacement")
					}

				}
			}
			if !sch.Schema.Attributes[r.idsKey].IsRequired() {
				t.Fatal("IDs must be required")
			}
			for _, id := range []string{"", "/", "page/", "/group", "page/group/extra", "../group", "page%2fgroup", "page?x", "page#x"} {
				resp := resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)}}
				r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Errorf("accepted bad import %q", id)
				}
			}
		})
	}
}
