package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// specKayra answers the discovery, a registered compute.infrapi.io at v2
// whose spec is stored, a registered storage.infrapi.io not read yet, and
// its own authorization group's spec.
func specKayra(t *testing.T) *httptest.Server {
	spec := map[string]any{"openapi": "3.0.3", "paths": map[string]any{
		"/apis/compute.infrapi.io/v2/namespaces/{namespace}/vms/{name}": map[string]any{"get": map[string]any{}, "delete": map[string]any{}, "parameters": []any{}},
		"/apis/compute.infrapi.io/v2/-/healthz":                         map[string]any{"get": map[string]any{}},
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := func(v any) { _ = json.NewEncoder(w).Encode(v) }
		switch r.URL.Path {
		case "/.well-known/kayra.json":
			answer(map[string]any{"apis": map[string]string{"system": "/apis/system.infrapi.io/v1",
				"catalog": "/apis/catalog.infrapi.io/v1", "authorization": "/apis/authorization.infrapi.io/v1"}})
		case "/apis/catalog.infrapi.io/v1/apiregistrations/compute.infrapi.io":
			answer(map[string]any{"name": "compute.infrapi.io", "version": "v2"})
		case "/apis/catalog.infrapi.io/v1/apiregistrations/compute.infrapi.io/openapi.json":
			answer(spec)
		case "/apis/catalog.infrapi.io/v1/apiregistrations/storage.infrapi.io":
			answer(map[string]any{"name": "storage.infrapi.io"})
		case "/apis/authorization.infrapi.io/v1/-/openapi.json":
			answer(map[string]any{"paths": map[string]any{"/apis/authorization.infrapi.io/v1/check": map[string]any{"post": map[string]any{}}}})
		default:
			writeProblem(w, http.StatusNotFound, "Not Found", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSpec(t *testing.T) {
	ctx := context.Background()
	c, err := newClient(ctx, specKayra(t).URL, "secret", "terraform-provider-kayra/test")
	if err != nil {
		t.Fatal(err)
	}
	get := func(api, version string) (specModel, error) {
		m := specModel{API: types.StringValue(api), Version: types.StringNull()}
		if version != "" {
			m.Version = types.StringValue(version)
		}
		return m, c.spec(ctx, &m)
	}

	// a registration: its version, its stored spec's paths, sorted
	m, err := get("compute.infrapi.io", "")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Available.ValueBool() || m.Root.ValueString() != "/apis/compute.infrapi.io/v2" || len(m.Paths) != 2 {
		t.Fatalf("compute: %+v", m)
	}
	if p := m.Paths[1]; p.Path.ValueString() != "/apis/compute.infrapi.io/v2/namespaces/{namespace}/vms/{name}" ||
		len(p.Methods) != 2 || p.Methods[0] != "DELETE" || p.Methods[1] != "GET" {
		t.Errorf("compute vms: %+v", p)
	}
	if _, err := get("compute.infrapi.io", "v1"); err == nil {
		t.Error("compute at v1: no error, it is registered at v2")
	}

	// registered, not read yet: no paths, no error
	if m, err := get("storage.infrapi.io", ""); err != nil || m.Available.ValueBool() || len(m.Paths) != 0 || m.Root.ValueString() != "/apis/storage.infrapi.io/v1" {
		t.Errorf("storage: %+v %v", m, err)
	}

	// Kayra's own group
	if m, err := get("authorization.infrapi.io", ""); err != nil || !m.Available.ValueBool() || len(m.Paths) != 1 || m.Paths[0].Methods[0] != "POST" {
		t.Errorf("authorization: %+v %v", m, err)
	}

	// not registered yet: no paths, no error
	if m, err := get("unknown.infrapi.io", ""); err != nil || m.Registered.ValueBool() || m.Available.ValueBool() || len(m.Paths) != 0 {
		t.Errorf("unknown: %+v %v", m, err)
	}
	if m, _ := get("compute.infrapi.io", ""); !m.Registered.ValueBool() {
		t.Error("compute: not registered")
	}
}
