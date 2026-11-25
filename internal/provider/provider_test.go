package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeKayra is Kayra as its contracts describe it, in memory: a POST
// stores the body at the collection's path plus its name, adding what
// Kayra sets; a request carrying a read-only property fails the test.
type fakeKayra struct {
	t        *testing.T
	mu       sync.Mutex
	objects  map[string]map[string]any
	readOnly []string
}

func newFakeKayra(t *testing.T) (*fakeKayra, *httptest.Server) {
	k := &fakeKayra{t: t, objects: map[string]map[string]any{},
		readOnly: []string{"uid", "created_at", "updated_at", "platform", "region", "active", "state", "id", "user", "token", "expires_at"}}
	srv := httptest.NewServer(k)
	t.Cleanup(srv.Close)
	return k, srv
}

func (k *fakeKayra) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if r.URL.Path == "/.well-known/kayra.json" {
		_ = json.NewEncoder(w).Encode(map[string]any{"apis": map[string]string{
			"system": "/apis/system.infrapi.io/v1", "catalog": "/apis/catalog.infrapi.io/v1",
			"authorization": "/apis/authorization.infrapi.io/v1", "events": "/apis/events.infrapi.io/v1"}})
		return
	}
	if r.Header.Get("Authorization") != "Bearer secret" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "no token")
		return
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeProblem(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		for _, p := range k.readOnly {
			if _, ok := in[p]; ok && (p != "active" || !strings.Contains(r.URL.Path, "/apiregistrations")) {
				k.t.Errorf("%s %s sends read-only %q", r.Method, r.URL.Path, p)
			}
		}
		key := r.URL.Path
		if r.Method == http.MethodPost {
			name, _ := in["name"].(string)
			if api, ok := in["api"].(string); ok {
				name = api
			}
			if strings.HasSuffix(key, "/tokens") {
				name = "tok1"
				in["id"], in["token"], in["user"] = name, "pat_secret", "jdoe@example.com"
			}
			key += "/" + name
		} else if _, ok := k.objects[key]; !ok {
			writeProblem(w, http.StatusNotFound, "Not Found", "no "+key)
			return
		}
		in["uid"], in["created_at"], in["updated_at"] = "u-1", "2026-11-25T10:00:00Z", "2026-11-25T10:00:00Z"
		if i := strings.Index(key, "/namespaces/"); i >= 0 && strings.HasPrefix(key, "/apis/authorization.") {
			ns := strings.Split(key[i+len("/namespaces/"):], "/")[0]
			meta, _ := in["metadata"].(map[string]any)
			if meta == nil {
				meta = map[string]any{}
			}
			meta["namespace"] = ns
			in["metadata"] = meta
		}
		if strings.Contains(key, "/apiregistrations/") {
			if in["version"] == nil {
				in["version"] = "v1"
			}
			if in["sync_id"] == nil {
				in["sync_id"] = "s-1"
			}
		}
		k.objects[key] = in
		out := map[string]any{}
		for p, v := range in {
			out[p] = v
		}
		delete(out, "expires_in")
		status := http.StatusOK
		if r.Method == http.MethodPost {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(out)
	case http.MethodGet:
		o, ok := k.objects[r.URL.Path]
		if !ok {
			writeProblem(w, http.StatusNotFound, "Not Found", "no "+r.URL.Path)
			return
		}
		out := map[string]any{}
		for p, v := range o {
			if p != "token" && p != "expires_in" {
				out[p] = v
			}
		}
		_ = json.NewEncoder(w).Encode(out)
	case http.MethodDelete:
		if _, ok := k.objects[r.URL.Path]; !ok {
			writeProblem(w, http.StatusNotFound, "Not Found", "no "+r.URL.Path)
			return
		}
		delete(k.objects, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "title": title, "detail": detail, "solution": "do it right"})
}

func testResource(t *testing.T, typ, url string) *kayraResource {
	t.Helper()
	c, err := newClient(context.Background(), url, "secret")
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

// unknown marks an attribute the plan does not know yet.
type unknownValue struct{}

var unknown = unknownValue{}

// value is v, Go values, as fields' object: an attribute it leaves out is
// null, unknown unknown.
func value(t *testing.T, fields []field, v map[string]any) tftypes.Value {
	t.Helper()
	attrs := map[string]tftypes.Value{}
	for _, f := range fields {
		attrs[f.Attr] = fieldValue(t, f, v[f.Attr])
	}
	return tftypes.NewValue(objectType(fields), attrs)
}

func fieldValue(t *testing.T, f field, v any) tftypes.Value {
	typ := tfType(f)
	switch x := v.(type) {
	case nil:
		return tftypes.NewValue(typ, nil)
	case unknownValue:
		return tftypes.NewValue(typ, tftypes.UnknownValue)
	case []string:
		elems := []tftypes.Value{}
		for _, s := range x {
			elems = append(elems, tftypes.NewValue(tftypes.String, s))
		}
		return tftypes.NewValue(typ, elems)
	case map[string]any:
		return value(t, f.Fields, x)
	case []map[string]any:
		elems := []tftypes.Value{}
		for _, o := range x {
			elems = append(elems, value(t, f.Fields, o))
		}
		return tftypes.NewValue(typ, elems)
	}
	return tftypes.NewValue(typ, v)
}

// attr is the Go value of path, a.b, in v.
func attr(t *testing.T, v tftypes.Value, path string) any {
	t.Helper()
	for _, step := range strings.Split(path, ".") {
		attrs := map[string]tftypes.Value{}
		if err := v.As(&attrs); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		v = attrs[step]
	}
	if v.IsNull() {
		return nil
	}
	switch {
	case v.Type().Is(tftypes.String):
		var s string
		_ = v.As(&s)
		return s
	case v.Type().Is(tftypes.Bool):
		var b bool
		_ = v.As(&b)
		return b
	case v.Type().Is(tftypes.List{ElementType: tftypes.String}):
		var elems []tftypes.Value
		_ = v.As(&elems)
		var out []string
		for _, e := range elems {
			var s string
			_ = e.As(&s)
			out = append(out, s)
		}
		return strings.Join(out, ",")
	}
	return v
}

func schemaOf(r *kayraResource) tfsdk.State {
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	return tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(objectType(r.d.allFields()), nil)}
}

// Every schema is a valid one, its type the one its values are built
// with.
func TestSchemas(t *testing.T) {
	ctx := context.Background()
	srv, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s", d.Summary, d.Detail)
	}
	if len(resp.ResourceSchemas) != len(resourceDefs) || len(resp.DataSourceSchemas) != len(dataSourceDefs)+1 { // and kayra_api_spec
		t.Errorf("%d resources, %d data sources", len(resp.ResourceSchemas), len(resp.DataSourceSchemas))
	}
	for _, d := range resourceDefs {
		r := &kayraResource{d: d}
		s := schemaOf(r)
		if got := s.Schema.Type().TerraformType(ctx); !got.Equal(objectType(d.allFields())) {
			t.Errorf("kayra_%s: schema type %s, values %s", d.Type, got, objectType(d.allFields()))
		}
	}
	for _, d := range dataSourceDefs {
		resp := &datasource.SchemaResponse{}
		(&kayraDataSource{d: d}).Schema(ctx, datasource.SchemaRequest{}, resp)
		if got := resp.Schema.Type().TerraformType(ctx); !got.Equal(objectType(d.Fields)) {
			t.Errorf("data kayra_%s: schema type %s", d.Type, got)
		}
	}
}

// A namespace's group: created with what the configuration says only, its
// metadata.namespace Kayra's, the attributes left out null; updated per
// attribute; read; imported; deleted, and gone.
func TestGroupLifecycle(t *testing.T) {
	ctx := context.Background()
	k, srv := newFakeKayra(t)
	r := testResource(t, "group", srv.URL)
	fields := r.d.allFields()

	plan := value(t, fields, map[string]any{"id": unknown, "namespace": "acme", "name": "admins", "metadata": unknown,
		"users": []string{"jdoe@example.com"}, "created_at": unknown, "updated_at": unknown})
	st := schemaOf(r)
	cresp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: plan}}, cresp)
	if cresp.Diagnostics.HasError() {
		t.Fatal(cresp.Diagnostics)
	}
	got := cresp.State.Raw
	for path, want := range map[string]any{"id": "acme/admins", "namespace": "acme", "metadata.namespace": "acme",
		"metadata.region": nil, "users": "jdoe@example.com", "description": nil, "groups": nil, "created_at": "2026-11-25T10:00:00Z"} {
		if v := attr(t, got, path); v != want {
			t.Errorf("created %s: %v, want %v", path, v, want)
		}
	}
	if o := k.objects["/apis/authorization.infrapi.io/v1/namespaces/acme/groups/admins"]; o == nil || o["users"] == nil {
		t.Fatalf("stored: %v", k.objects)
	}

	// an update sends the new users, and the description set to ""
	plan = value(t, fields, map[string]any{"id": "acme/admins", "namespace": "acme", "name": "admins",
		"metadata": map[string]any{"namespace": "acme"}, "users": []string{"jdoe@example.com", "bob@example.com"},
		"description": "", "created_at": "2026-11-25T10:00:00Z", "updated_at": unknown})
	uresp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: plan}, State: tfsdk.State{Schema: st.Schema, Raw: got}}, uresp)
	if uresp.Diagnostics.HasError() {
		t.Fatal(uresp.Diagnostics)
	}
	if v := attr(t, uresp.State.Raw, "users"); v != "jdoe@example.com,bob@example.com" {
		t.Errorf("updated users: %v", v)
	}
	if v := attr(t, uresp.State.Raw, "description"); v != "" {
		t.Errorf("updated description: %v, want the configuration's \"\"", v)
	}

	// imported: its keys, then read
	iresp := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "acme/admins"}, iresp)
	if iresp.Diagnostics.HasError() {
		t.Fatal(iresp.Diagnostics)
	}
	rresp := &resource.ReadResponse{State: iresp.State}
	r.Read(ctx, resource.ReadRequest{State: iresp.State}, rresp)
	if rresp.Diagnostics.HasError() {
		t.Fatal(rresp.Diagnostics)
	}
	if v := attr(t, rresp.State.Raw, "users"); v != "jdoe@example.com,bob@example.com" {
		t.Errorf("imported users: %v", v)
	}
	if v := attr(t, rresp.State.Raw, "id"); v != "acme/admins" {
		t.Errorf("imported id: %v", v)
	}

	dresp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: rresp.State}, dresp)
	if dresp.Diagnostics.HasError() {
		t.Fatal(dresp.Diagnostics)
	}
	gone := &resource.ReadResponse{State: rresp.State}
	r.Read(ctx, resource.ReadRequest{State: rresp.State}, gone)
	if gone.Diagnostics.HasError() || !gone.State.Raw.IsNull() {
		t.Errorf("read after delete: %v, %v", gone.State.Raw, gone.Diagnostics)
	}
}

// A personal access token: its secret is known at its creation only, and
// kept; its lifetime, never returned, too.
func TestTokenSecretKept(t *testing.T) {
	ctx := context.Background()
	_, srv := newFakeKayra(t)
	r := testResource(t, "token", srv.URL)
	fields := r.d.allFields()
	plan := value(t, fields, map[string]any{"id": unknown, "name": "ci", "expires_in": "720h", "user": unknown,
		"token": unknown, "created_at": unknown, "expires_at": unknown, "last_used_at": unknown,
		"scope": map[string]any{"apis": []string{"compute.infrapi.io"}, "verbs": []string{"get", "list"}}})
	st := schemaOf(r)
	cresp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: plan}}, cresp)
	if cresp.Diagnostics.HasError() {
		t.Fatal(cresp.Diagnostics)
	}
	rresp := &resource.ReadResponse{State: cresp.State}
	r.Read(ctx, resource.ReadRequest{State: cresp.State}, rresp)
	if rresp.Diagnostics.HasError() {
		t.Fatal(rresp.Diagnostics)
	}
	for path, want := range map[string]any{"id": "tok1", "token": "pat_secret", "expires_in": "720h", "scope.verbs": "get,list", "last_used_at": nil} {
		if v := attr(t, rresp.State.Raw, path); v != want {
			t.Errorf("%s: %v, want %v", path, v, want)
		}
	}
}

// A registration: its version and sync_id Kayra's when left out; its
// dependencies sent as depends_on.
func TestApiRegistrationDefaults(t *testing.T) {
	ctx := context.Background()
	k, srv := newFakeKayra(t)
	r := testResource(t, "api_registration", srv.URL)
	plan := value(t, r.d.allFields(), map[string]any{"id": unknown, "name": "compute.infrapi.io", "version": unknown, "sync_id": unknown,
		"owner": "team-compute", "platform": unknown, "region": unknown, "uid": unknown, "canary_namespaces": unknown,
		"created_at": unknown, "updated_at": unknown,
		"dependencies": []map[string]any{{"platform": "sandbox", "region": "eu-west-1", "name": "storage.infrapi.io"}}})
	st := schemaOf(r)
	cresp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: st.Schema, Raw: plan}}, cresp)
	if cresp.Diagnostics.HasError() {
		t.Fatal(cresp.Diagnostics)
	}
	for path, want := range map[string]any{"version": "v1", "sync_id": "s-1", "owner": "team-compute", "active": nil, "id": "compute.infrapi.io"} {
		if v := attr(t, cresp.State.Raw, path); v != want {
			t.Errorf("%s: %v, want %v", path, v, want)
		}
	}
	stored := k.objects["/apis/catalog.infrapi.io/v1/apiregistrations/compute.infrapi.io"]
	if deps, _ := stored["depends_on"].([]any); len(deps) != 1 {
		t.Errorf("depends_on stored: %v", stored)
	}
}

// Kayra's problem, its solution, is the diagnostic.
func TestProblemDiagnostic(t *testing.T) {
	ctx := context.Background()
	_, srv := newFakeKayra(t)
	r := testResource(t, "namespace", srv.URL)
	st := schemaOf(r)
	dresp := &resource.DeleteResponse{State: st}
	state := value(t, r.d.allFields(), map[string]any{"id": "acme", "name": "acme"})
	rresp := &resource.ReadResponse{State: tfsdk.State{Schema: st.Schema, Raw: state}}
	r.client.token = "wrong"
	r.Read(ctx, resource.ReadRequest{State: rresp.State}, rresp)
	if !rresp.Diagnostics.HasError() || !strings.Contains(rresp.Diagnostics.Errors()[0].Detail(), "Solution: do it right") {
		t.Errorf("diagnostics: %v", rresp.Diagnostics)
	}
	r.client.token = "secret"
	r.Delete(ctx, resource.DeleteRequest{State: rresp.State}, dresp)
	if dresp.Diagnostics.HasError() {
		t.Errorf("deleting what is gone: %v", dresp.Diagnostics)
	}
}

func TestNewClient_refuses(t *testing.T) {
	for _, url := range []string{"", "kayra.example.com", "ftp://kayra.example.com"} {
		if _, err := newClient(context.Background(), url, ""); err == nil {
			t.Errorf("%q: no error", url)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"apis": {}}`)) }))
	defer srv.Close()
	if _, err := newClient(context.Background(), srv.URL, ""); err == nil || !strings.Contains(err.Error(), "is it Kayra") {
		t.Errorf("not Kayra: %v", err)
	}
}
