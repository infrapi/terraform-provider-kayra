package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// liveResource is typ's resource against the Kayra of KAYRA_LIVE_URL, as
// KAYRA_LIVE_TOKEN: the provider's operations, without OpenTofu.
func liveResource(t *testing.T, typ string) *kayraResource {
	t.Helper()
	c, err := newClient(context.Background(), os.Getenv("KAYRA_LIVE_URL"), os.Getenv("KAYRA_LIVE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resourceDefs {
		if d.Type == typ {
			return &kayraResource{d: d, client: c}
		}
	}
	t.Fatalf("no kayra_%s", typ)
	return nil
}

// planOf is config as a plan: an attribute it leaves out is unknown when
// Kayra sets it, null otherwise.
func planOf(t *testing.T, fields []field, config map[string]any) tftypes.Value {
	t.Helper()
	v := map[string]any{}
	for _, f := range fields {
		switch c, ok := config[f.Attr]; {
		case ok:
			v[f.Attr] = c
		case f.ReadOnly || f.Default || f.Attr == "id":
			v[f.Attr] = unknown
		}
	}
	return value(t, fields, v)
}

// liveCase is an object TestLive makes.
type liveCase struct {
	typ, importID string
	config        map[string]any
	update        map[string]any
}

// liveCases are the objects TestLive makes, in order: what one needs
// before it. domain is the Kayra's, its groups'.
func liveCases(domain string) []liveCase {
	api := "tflive." + domain
	return []liveCase{
		{"namespace", "tf-live", map[string]any{"name": "tf-live"}, nil},
		{"cluster_group", "tf-live-admins", map[string]any{"name": "tf-live-admins", "groups": []string{"sandbox-tf-live"},
			"metadata": map[string]any{"namespace": "tf-live"}}, map[string]any{"description": "the tf-live admins"}},
		{"cluster_role", "tf-live-admin", map[string]any{"name": "tf-live-admin", "metadata": map[string]any{"namespace": "tf-live"},
			"rules": []map[string]any{{"apis": []string{"authorization." + domain}, "resources": []string{"*"}, "verbs": []string{"*"}}}},
			map[string]any{"description": "manages tf-live's RBAC"}},
		{"cluster_role_binding", "tf-live-admin", map[string]any{"name": "tf-live-admin", "role": "tf-live-admin",
			"groups": []string{"tf-live-admins"}, "metadata": map[string]any{"namespace": "tf-live"}}, map[string]any{"description": "binds them"}},
		{"group", "acme/tf-live-devs", map[string]any{"namespace": "acme", "name": "tf-live-devs", "users": []string{"jdoe@example.com"}},
			map[string]any{"description": "the developers"}},
		{"role", "acme/tf-live-reader", map[string]any{"namespace": "acme", "name": "tf-live-reader",
			"rules": []map[string]any{{"apis": []string{"events." + domain}, "resources": []string{"events"}, "verbs": []string{"get", "list"}}}},
			map[string]any{"description": "reads events"}},
		{"role_binding", "acme/tf-live-reader", map[string]any{"namespace": "acme", "name": "tf-live-reader", "role": "tf-live-reader",
			"groups": []string{"tf-live-devs"}}, map[string]any{"description": "devs read events"}},
		// a federated credential needs its issuer in Kayra's federation.issuers:
		// KAYRA_LIVE_ISSUER, when the Kayra has one
		{"service_account", "acme/tf-live-ci", map[string]any{"namespace": "acme", "name": "tf-live-ci"}, map[string]any{"description": "the CI"}},
		{"api_registration", api, map[string]any{"name": api, "owner": "team-tf",
			"allowed_namespaces": []string{"acme"}}, map[string]any{"description": "the live test's API"}},
		{"subscription", "acme/" + api, map[string]any{"namespace": "acme", "api": api}, nil},
		{"token", "", map[string]any{"name": "tf-live", "expires_in": "1h"}, nil},
	}
}

// Kayra's own group's specification, and a registration's, read live.
func TestLiveSpec(t *testing.T) {
	if os.Getenv("KAYRA_LIVE_URL") == "" {
		t.Skip("KAYRA_LIVE_URL and KAYRA_LIVE_TOKEN: a Kayra and a sysadmin's token")
	}
	c := liveResource(t, "namespace").client
	root := c.roots["authorization"]
	group := strings.TrimSuffix(strings.TrimPrefix(root, "/apis/"), "/v1")
	m := specModel{API: types.StringValue(group), Version: types.StringNull()}
	if err := c.spec(context.Background(), &m); err != nil || !m.Available.ValueBool() || len(m.Paths) < 20 {
		t.Fatalf("%s: %d paths, %v", group, len(m.Paths), err)
	}
	t.Logf("%s: %d paths, the first %s %v", group, len(m.Paths), m.Paths[0].Path.ValueString(), m.Paths[0].Methods)
	if api := os.Getenv("KAYRA_LIVE_API"); api != "" {
		m := specModel{API: types.StringValue(api), Version: types.StringNull()}
		if err := c.spec(context.Background(), &m); err != nil {
			t.Fatalf("%s: %v", api, err)
		}
		t.Logf("%s: available %v, %d paths", api, m.Available.ValueBool(), len(m.Paths))
	}
}

// Every resource, created, read, updated when it may be, imported, then
// deleted, the latest first, against a real Kayra: KAYRA_LIVE_URL and
// KAYRA_LIVE_TOKEN, a sysadmin's (go test -run TestLive ./internal/provider),
// its namespace acme existing.
func TestLive(t *testing.T) {
	if os.Getenv("KAYRA_LIVE_URL") == "" {
		t.Skip("KAYRA_LIVE_URL and KAYRA_LIVE_TOKEN: a Kayra and a sysadmin's token")
	}
	ctx := context.Background()
	type created struct {
		r     *kayraResource
		state tfsdk.State
	}
	var made []created
	defer func() {
		for i := len(made) - 1; i >= 0; i-- {
			r, state := made[i].r, made[i].state
			dresp := &resource.DeleteResponse{State: schemaOf(r)}
			r.Delete(ctx, resource.DeleteRequest{State: state}, dresp)
			if dresp.Diagnostics.HasError() {
				t.Errorf("delete kayra_%s: %v", r.d.Type, dresp.Diagnostics)
				continue
			}
			gone := &resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, gone)
			if gone.Diagnostics.HasError() || !gone.State.Raw.IsNull() {
				t.Errorf("kayra_%s read after delete: %v %v", r.d.Type, gone.State.Raw, gone.Diagnostics)
			}
		}
	}()
	root := liveResource(t, "namespace").client.roots["authorization"]
	domain := strings.TrimSuffix(strings.TrimPrefix(root, "/apis/authorization."), "/v1")
	cases := liveCases(domain)
	if iss := os.Getenv("KAYRA_LIVE_ISSUER"); iss != "" {
		for i, c := range cases {
			if c.typ == "service_account" {
				cases[i].config["federated_credentials"] = []map[string]any{{"name": "ci", "issuer": iss, "subject": "repo:acme/compute:ref:refs/heads/main"}}
			}
		}
	}
	for _, c := range cases {
		r := liveResource(t, c.typ)
		fields := r.d.allFields()
		st := schemaOf(r)
		cresp := &resource.CreateResponse{State: st}
		r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: planOf(t, fields, c.config)}}, cresp)
		if cresp.Diagnostics.HasError() {
			t.Fatalf("create kayra_%s: %v", c.typ, cresp.Diagnostics)
		}
		made = append(made, created{r, cresp.State})
		rresp := &resource.ReadResponse{State: cresp.State}
		r.Read(ctx, resource.ReadRequest{State: cresp.State}, rresp)
		if rresp.Diagnostics.HasError() || rresp.State.Raw.IsNull() {
			t.Fatalf("read kayra_%s: %v %v", c.typ, rresp.State.Raw, rresp.Diagnostics)
		}
		if c.update != nil {
			cfg := map[string]any{}
			for k, v := range c.config {
				cfg[k] = v
			}
			for k, v := range c.update {
				cfg[k] = v
			}
			uresp := &resource.UpdateResponse{State: st}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: planOf(t, fields, cfg)}, State: rresp.State}, uresp)
			if uresp.Diagnostics.HasError() {
				t.Fatalf("update kayra_%s: %v", c.typ, uresp.Diagnostics)
			}
			for k := range c.update {
				if got, want := attr(t, uresp.State.Raw, k), attr(t, planOf(t, fields, cfg), k); got != want {
					t.Errorf("kayra_%s updated %s: %v, want %v", c.typ, k, got, want)
				}
			}
			made[len(made)-1].state = uresp.State
		}
		if c.importID != "" {
			iresp := &resource.ImportStateResponse{State: st}
			r.ImportState(ctx, resource.ImportStateRequest{ID: c.importID}, iresp)
			if iresp.Diagnostics.HasError() {
				t.Fatalf("import kayra_%s: %v", c.typ, iresp.Diagnostics)
			}
			read := &resource.ReadResponse{State: iresp.State}
			r.Read(ctx, resource.ReadRequest{State: iresp.State}, read)
			if read.Diagnostics.HasError() || read.State.Raw.IsNull() {
				t.Fatalf("imported kayra_%s: %v %v", c.typ, read.State.Raw, read.Diagnostics)
			}
		}
		t.Logf("kayra_%s: created, read, updated: %v, imported: %v", c.typ, c.update != nil, c.importID != "")
	}
}
