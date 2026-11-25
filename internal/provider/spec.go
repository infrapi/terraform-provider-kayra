package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// specDataSource is an API's specification, what it serves: an API
// registered in Kayra's catalog, the one its reconciler last read, or one of
// Kayra's own groups, the one Kayra serves. A gateway routes from it.
// Written by hand: a specification is any OpenAPI document, no schema of
// Kayra's contracts.
type specDataSource struct {
	client *client
}

var _ datasource.DataSourceWithConfigure = (*specDataSource)(nil)

type specModel struct {
	API        types.String `tfsdk:"api"`
	Version    types.String `tfsdk:"version"`
	Root       types.String `tfsdk:"root"`
	Registered types.Bool   `tfsdk:"registered"`
	Available  types.Bool   `tfsdk:"available"`
	OpenAPI    types.String `tfsdk:"openapi"`
	Paths      []specPath   `tfsdk:"paths"`
}

type specPath struct {
	Path    types.String `tfsdk:"path"`
	Methods []string     `tfsdk:"methods"`
}

func (s *specDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_spec"
}

func (s *specDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An API's specification, the OpenAPI document of what it serves, its contract and its operational endpoints: " +
			"for an API registered in Kayra's catalog, the one Kayra's reconciler last read from it; for one of Kayra's own groups, " +
			"the one Kayra serves. A gateway's routes, one per path, come from it.",
		Attributes: map[string]schema.Attribute{
			"api":     schema.StringAttribute{Required: true, Description: "The API's group, compute.infrapi.io, or one of Kayra's own (authorization.<domain>)."},
			"version": schema.StringAttribute{Optional: true, Computed: true, Description: "Its version, v1 when unset: one of Kayra's own groups only; a registration's is its own."},
			"root":    schema.StringAttribute{Computed: true, Description: "Its paths' root, /apis/<api>/<version>."},
			"registered": schema.BoolAttribute{Computed: true,
				Description: "Whether the API is registered in Kayra's catalog: false for one of Kayra's own groups, and for an API not registered yet."},
			"available": schema.BoolAttribute{Computed: true,
				Description: "Whether there is a specification: false for an API not registered yet, or Kayra's reconciler has not read yet, its paths empty. " +
					"A configuration registering the API and routing from its specification gets its routes at the apply after Kayra read it."},
			"openapi": schema.StringAttribute{Computed: true, Description: "The specification, an OpenAPI document as JSON; empty when not available."},
			"paths": schema.ListNestedAttribute{Computed: true, Description: "Its paths, sorted, each with the methods it serves.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"path":    schema.StringAttribute{Computed: true, Description: "The path, as OpenAPI writes it: /apis/compute.infrapi.io/v1/namespaces/{namespace}/vms/{name}."},
					"methods": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Its methods, upper case, sorted: GET, POST."},
				}},
			},
		},
	}
}

func (s *specDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("%T, not the Kayra client", req.ProviderData))
		return
	}
	s.client = c
}

func (s *specDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m specModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := s.client.spec(ctx, &m)
	if err != nil {
		addError(&resp.Diagnostics, "Reading data kayra_api_spec", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// spec fills m from Kayra: its own group's specification, or a
// registration's.
func (c *client) spec(ctx context.Context, m *specModel) error {
	api := m.API.ValueString()
	version := m.Version.ValueString()
	if version == "" {
		version = "v1"
	}
	var path string
	m.Registered = types.BoolValue(false)
	if root, ok := c.ownRoot(api); ok {
		if version != "v1" {
			return fmt.Errorf("%s is Kayra's own group: version v1 only", api)
		}
		path = root + "/-/openapi.json"
	} else {
		var reg struct {
			Version string `json:"version"`
		}
		regPath := c.roots["catalog"] + "/apiregistrations/" + url.PathEscape(api)
		switch err := c.do(ctx, http.MethodGet, regPath, nil, &reg); {
		case errors.Is(err, errNotFound):
			// not registered yet: no routes until it is, and Kayra read it
			m.Version, m.Root = types.StringValue(version), types.StringValue("/apis/"+api+"/"+version)
			m.Available, m.OpenAPI, m.Paths = types.BoolValue(false), types.StringValue(""), []specPath{}
			return nil
		case err != nil:
			return err
		}
		m.Registered = types.BoolValue(true)
		if reg.Version != "" {
			if !m.Version.IsNull() && !m.Version.IsUnknown() && reg.Version != version {
				return fmt.Errorf("%s is registered at version %s, not %s", api, reg.Version, version)
			}
			version = reg.Version
		}
		path = regPath + "/openapi.json"
	}
	m.Version = types.StringValue(version)
	m.Root = types.StringValue("/apis/" + api + "/" + version)

	var doc map[string]any
	switch err := c.do(ctx, http.MethodGet, path, nil, &doc); {
	case errors.Is(err, errNotFound):
		// registered, not read yet: no routes until Kayra's reconciler reads it
		m.Available, m.OpenAPI, m.Paths = types.BoolValue(false), types.StringValue(""), []specPath{}
		return nil
	case err != nil:
		return err
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	m.Available, m.OpenAPI, m.Paths = types.BoolValue(true), types.StringValue(string(data)), specPaths(doc)
	return nil
}

// ownRoot is the root of api when it is one of Kayra's own groups, from its
// /.well-known/kayra.json.
func (c *client) ownRoot(api string) (string, bool) {
	for _, root := range c.roots {
		if group, _, ok := strings.Cut(strings.TrimPrefix(root, "/apis/"), "/"); ok && group == api {
			return root, true
		}
	}
	return "", false
}

// specPaths are doc's paths, sorted, with their methods.
func specPaths(doc map[string]any) []specPath {
	paths, _ := doc["paths"].(map[string]any)
	out := make([]specPath, 0, len(paths))
	for p, item := range paths {
		ops, _ := item.(map[string]any)
		var methods []string
		for m := range ops {
			switch m {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
				methods = append(methods, strings.ToUpper(m))
			}
		}
		if len(methods) == 0 {
			continue
		}
		slices.Sort(methods)
		out = append(out, specPath{Path: types.StringValue(p), Methods: methods})
	}
	slices.SortFunc(out, func(a, b specPath) int { return strings.Compare(a.Path.ValueString(), b.Path.ValueString()) })
	return out
}
