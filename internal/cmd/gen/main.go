// Command gen writes internal/provider/zz_generated.go from Kayra's
// contracts, internal/contracts/<name>-v1.yaml: each resource's and data
// source's fields, one per property of its contract's schema, typed as the
// contract types it. The provider builds their schemas, and the requests
// and states, from these fields only.
//
//	go run ./internal/cmd/gen -contracts internal/contracts -o internal/provider/zz_generated.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	dir := flag.String("contracts", "internal/contracts", "the contracts' directory")
	out := flag.String("o", "internal/provider/zz_generated.go", "the file to write")
	flag.Parse()
	src, err := generate(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil { //#nosec G306 -- a source file
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

// resource is a Terraform resource over one of a contract's schemas.
type resource struct {
	Type       string // kayra_<Type>
	Contract   string // its contract's file name: rbac
	API        string // its group's key in /.well-known/kayra.json: authorization
	Schema     string // components.schemas.<Schema>
	Collection string // POST: relative to the group's root, {attribute} for a path parameter
	Item       string // GET, PUT and DELETE
	Keys       []string
	// Defaults are properties Kayra sets when the request leaves them
	// out: optional and computed. A nested one is parent.child.
	Defaults []string
	// PathOnly are attributes of the path only, not the schema's.
	PathOnly map[string]string // attribute: description
}

// dataSource is a Terraform data source: a GET of one of a contract's
// schemas.
type dataSource struct {
	Type, Contract, API, Schema, Path string
	Keys                              []string
	PathOnly                          map[string]string
}

const namespaceDesc = "The namespace it lives in."

var nsDefaults = []string{"metadata", "metadata.namespace"}

var resources = []resource{
	{Type: "namespace", Contract: "system", API: "system", Schema: "Namespace",
		Collection: "/namespaces", Item: "/namespaces/{name}", Keys: []string{"name"}},
	{Type: "api_registration", Contract: "catalog", API: "catalog", Schema: "ApiRegistration",
		Collection: "/apiregistrations", Item: "/apiregistrations/{name}", Keys: []string{"name"},
		Defaults: []string{"version", "sync_id"}},
	{Type: "subscription", Contract: "catalog", API: "catalog", Schema: "Subscription",
		Collection: "/namespaces/{namespace}/subscriptions", Item: "/namespaces/{namespace}/subscriptions/{api}",
		Keys: []string{"namespace", "api"}, PathOnly: map[string]string{"namespace": "The namespace subscribing."}},
	{Type: "cluster_group", Contract: "rbac", API: "authorization", Schema: "Group",
		Collection: "/clustergroups", Item: "/clustergroups/{name}", Keys: []string{"name"}},
	{Type: "cluster_role", Contract: "rbac", API: "authorization", Schema: "Role",
		Collection: "/clusterroles", Item: "/clusterroles/{name}", Keys: []string{"name"}},
	{Type: "cluster_role_binding", Contract: "rbac", API: "authorization", Schema: "RoleBinding",
		Collection: "/clusterrolebindings", Item: "/clusterrolebindings/{name}", Keys: []string{"name"}},
	{Type: "group", Contract: "rbac", API: "authorization", Schema: "Group",
		Collection: "/namespaces/{namespace}/groups", Item: "/namespaces/{namespace}/groups/{name}",
		Keys: []string{"namespace", "name"}, Defaults: nsDefaults, PathOnly: map[string]string{"namespace": namespaceDesc}},
	{Type: "role", Contract: "rbac", API: "authorization", Schema: "Role",
		Collection: "/namespaces/{namespace}/roles", Item: "/namespaces/{namespace}/roles/{name}",
		Keys: []string{"namespace", "name"}, Defaults: nsDefaults, PathOnly: map[string]string{"namespace": namespaceDesc}},
	{Type: "role_binding", Contract: "rbac", API: "authorization", Schema: "RoleBinding",
		Collection: "/namespaces/{namespace}/rolebindings", Item: "/namespaces/{namespace}/rolebindings/{name}",
		Keys: []string{"namespace", "name"}, Defaults: nsDefaults, PathOnly: map[string]string{"namespace": namespaceDesc}},
	{Type: "service_account", Contract: "rbac", API: "authorization", Schema: "ServiceAccount",
		Collection: "/namespaces/{namespace}/serviceaccounts", Item: "/namespaces/{namespace}/serviceaccounts/{name}",
		Keys: []string{"namespace", "name"}, Defaults: nsDefaults, PathOnly: map[string]string{"namespace": namespaceDesc}},
	{Type: "token", Contract: "rbac", API: "authorization", Schema: "Token",
		Collection: "/tokens", Item: "/tokens/{id}", Keys: []string{"id"}},
}

var dataSources = []dataSource{
	{Type: "namespace", Contract: "system", API: "system", Schema: "Namespace", Path: "/namespaces/{name}", Keys: []string{"name"}},
	{Type: "api_registration", Contract: "catalog", API: "catalog", Schema: "ApiRegistration", Path: "/apiregistrations/{name}", Keys: []string{"name"}},
	{Type: "api_registration_status", Contract: "catalog", API: "catalog", Schema: "ApiRegistrationStatus",
		Path: "/apiregistrations/{name}/status", Keys: []string{"name"}},
}

// stable are read-only properties that never change once set: the plan
// keeps them (UseStateForUnknown).
var stable = []string{"uid", "created_at", "platform", "region", "user", "expires_at", "token", "id"}

// sensitive are properties a state must not show.
var sensitive = []string{"token"}

// --- the contracts ---

type schemaDoc struct {
	Ref         string     `yaml:"$ref"`
	Type        string     `yaml:"type"`
	Description string     `yaml:"description"`
	Enum        []string   `yaml:"enum"`
	Required    []string   `yaml:"required"`
	ReadOnly    bool       `yaml:"readOnly"`
	WriteOnly   bool       `yaml:"writeOnly"`
	Items       *schemaDoc `yaml:"items"`
	Properties  yaml.Node  `yaml:"properties"`
	// AdditionalProperties is a schema (a map's values), or true or false
	// (a closed object's): additional reads it.
	AdditionalProperties yaml.Node    `yaml:"additionalProperties"`
	OneOf                []*schemaDoc `yaml:"oneOf"`
	props                map[string]*schemaDoc
	order                []string
}

type contract struct {
	Info struct {
		Group string `yaml:"x-infrapi-group"`
	} `yaml:"info"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components struct {
		Schemas map[string]*schemaDoc `yaml:"schemas"`
	} `yaml:"components"`
}

func readContract(path string) (*contract, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- the contracts given
	if err != nil {
		return nil, err
	}
	var c contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

// additional is the schema of s's additionalProperties, a map's values: nil
// when it has none, or says true or false.
func (s *schemaDoc) additional() *schemaDoc {
	if s.AdditionalProperties.Kind != yaml.MappingNode {
		return nil
	}
	var a schemaDoc
	if err := s.AdditionalProperties.Decode(&a); err != nil {
		return nil
	}
	return &a
}

// properties decodes s's properties, in the order they are written.
func (s *schemaDoc) properties() error {
	if s.props != nil || s.Properties.Kind == 0 {
		return nil
	}
	s.props = map[string]*schemaDoc{}
	for i := 0; i+1 < len(s.Properties.Content); i += 2 {
		var p schemaDoc
		if err := s.Properties.Content[i+1].Decode(&p); err != nil {
			return err
		}
		name := s.Properties.Content[i].Value
		s.order = append(s.order, name)
		s.props[name] = &p
	}
	return nil
}

func (c *contract) resolve(s *schemaDoc) (*schemaDoc, error) {
	if s.Ref == "" {
		return s, nil
	}
	name, ok := strings.CutPrefix(s.Ref, "#/components/schemas/")
	if !ok {
		return nil, fmt.Errorf("$ref %s: #/components/schemas/<name> only", s.Ref)
	}
	r, ok := c.Components.Schemas[name]
	if !ok {
		return nil, fmt.Errorf("$ref %s: no such schema", s.Ref)
	}
	return r, nil
}

// hasOperation tells whether the contract has method on path, path's
// parameters named as the resource names them: {name} for {group}.
func (c *contract) hasOperation(path, method string) bool {
	param := regexp.MustCompile(`\{[^}]+\}`)
	want := param.ReplaceAllString("/apis/"+c.Info.Group+"/v1"+path, "{}")
	for p, item := range c.Paths {
		if param.ReplaceAllString(p, "{}") == want {
			_, ok := item[method]
			return ok
		}
	}
	return false
}

// --- the fields ---

// reserved are Terraform's meta-arguments: a property of that name is an
// attribute of another.
var reserved = map[string]string{"depends_on": "dependencies", "count": "item_count", "for_each": "for_each_item",
	"provider": "provider_name", "lifecycle": "lifecycle_name", "connection": "connection_name", "provisioner": "provisioner_name"}

type field struct {
	Attr, JSON, Desc, Kind                                              string
	Required, ReadOnly, WriteOnly, Default, PathOnly, Stable, Sensitive bool
	Fields                                                              []field
}

func (c *contract) fields(s *schemaDoc, prefix string, defaults []string) ([]field, error) {
	s, err := c.resolve(s)
	if err != nil {
		return nil, err
	}
	if err := s.properties(); err != nil {
		return nil, err
	}
	if len(s.order) == 0 {
		return nil, fmt.Errorf("%s: an object without properties: no generic block", strings.TrimSuffix(prefix, "."))
	}
	var out []field
	for _, name := range s.order {
		p := s.props[name]
		attr := name
		if r, ok := reserved[name]; ok && prefix == "" {
			attr = r
		}
		f := field{
			Attr: attr, JSON: name, Desc: description(p), Required: slices.Contains(s.Required, name) && !p.ReadOnly,
			ReadOnly: p.ReadOnly, WriteOnly: p.WriteOnly, Default: slices.Contains(defaults, prefix+name),
			Stable: prefix == "" && p.ReadOnly && slices.Contains(stable, name), Sensitive: prefix == "" && slices.Contains(sensitive, name),
		}
		r, err := c.resolve(p)
		if err != nil {
			return nil, err
		}
		if f.Desc == "" {
			f.Desc = description(r)
		}
		switch {
		case r.Type == "string":
			f.Kind = "kString"
		case r.Type == "boolean":
			f.Kind = "kBool"
		case r.Type == "integer":
			f.Kind = "kInt"
		case r.Type == "array":
			item, err := c.resolve(r.Items)
			if err != nil {
				return nil, err
			}
			if item.Type == "string" {
				f.Kind = "kStrings"
				break
			}
			f.Kind = "kObjects"
			if f.Fields, err = c.fields(r.Items, prefix+name+".", defaults); err != nil {
				return nil, err
			}
		case r.Type == "object" && r.additional() != nil:
			if r.additional().Type != "string" {
				return nil, fmt.Errorf("%s%s: a map of strings only", prefix, name)
			}
			f.Kind = "kStringMap"
		case r.Type == "object":
			f.Kind = "kObject"
			if f.Fields, err = c.fields(p, prefix+name+".", defaults); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s%s: type %q", prefix, name, r.Type)
		}
		out = append(out, f)
	}
	return out, nil
}

func description(s *schemaDoc) string {
	d := strings.Join(strings.Fields(s.Description), " ")
	if len(s.Enum) > 0 {
		d += " One of: " + strings.Join(s.Enum, ", ") + "."
	}
	return strings.TrimSpace(d)
}

// --- the file ---

func generate(dir string) ([]byte, error) {
	contracts := map[string]*contract{}
	load := func(name string) (*contract, error) {
		if c, ok := contracts[name]; ok {
			return c, nil
		}
		c, err := readContract(filepath.Join(dir, name+"-v1.yaml"))
		contracts[name] = c
		return c, err
	}
	var b bytes.Buffer
	b.WriteString("// Code generated by go run ./internal/cmd/gen from internal/contracts. DO NOT EDIT.\n\npackage provider\n\n")
	b.WriteString("var resourceDefs = []resourceDef{\n")
	for _, r := range resources {
		c, err := load(r.Contract)
		if err != nil {
			return nil, err
		}
		s, ok := c.Components.Schemas[r.Schema]
		if !ok {
			return nil, fmt.Errorf("%s: no schema %s", r.Contract, r.Schema)
		}
		if !c.hasOperation(r.Collection, "post") || !c.hasOperation(r.Item, "get") || !c.hasOperation(r.Item, "delete") {
			return nil, fmt.Errorf("kayra_%s: %s has no POST %s, or no GET and DELETE %s", r.Type, r.Contract, r.Collection, r.Item)
		}
		fields, err := c.fields(s, "", r.Defaults)
		if err != nil {
			return nil, fmt.Errorf("kayra_%s: %w", r.Type, err)
		}
		fields = withPath(fields, r.PathOnly)
		fmt.Fprintf(&b, "\t{Type: %q, API: %q, Collection: %q, Item: %q, Keys: %#v, Updatable: %t, Desc: %q, Fields: ",
			r.Type, r.API, r.Collection, r.Item, r.Keys, c.hasOperation(r.Item, "put"), description(s))
		writeFields(&b, fields)
		b.WriteString("},\n")
	}
	b.WriteString("}\n\nvar dataSourceDefs = []dataSourceDef{\n")
	for _, d := range dataSources {
		c, err := load(d.Contract)
		if err != nil {
			return nil, err
		}
		s, ok := c.Components.Schemas[d.Schema]
		if !ok {
			return nil, fmt.Errorf("%s: no schema %s", d.Contract, d.Schema)
		}
		if !c.hasOperation(d.Path, "get") {
			return nil, fmt.Errorf("data kayra_%s: %s has no GET %s", d.Type, d.Contract, d.Path)
		}
		fields, err := c.fields(s, "", nil)
		if err != nil {
			return nil, fmt.Errorf("data kayra_%s: %w", d.Type, err)
		}
		fields = withPath(fields, d.PathOnly)
		fmt.Fprintf(&b, "\t{Type: %q, API: %q, Path: %q, Keys: %#v, Desc: %q, Fields: ", d.Type, d.API, d.Path, d.Keys, description(s))
		writeFields(&b, fields)
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	return format.Source(b.Bytes())
}

func withPath(fields []field, pathOnly map[string]string) []field {
	var extra []field
	for _, attr := range slices.Sorted(func(yield func(string) bool) {
		for k := range pathOnly {
			if !yield(k) {
				return
			}
		}
	}) {
		extra = append(extra, field{Attr: attr, Desc: pathOnly[attr], Kind: "kString", Required: true, PathOnly: true})
	}
	return append(extra, fields...)
}

func writeFields(b *bytes.Buffer, fields []field) {
	b.WriteString("[]field{\n")
	for _, f := range fields {
		fmt.Fprintf(b, "{Attr: %q, Kind: %s, Desc: %q", f.Attr, f.Kind, f.Desc)
		if f.JSON != f.Attr && f.JSON != "" {
			fmt.Fprintf(b, ", JSON: %q", f.JSON)
		}
		for _, flag := range []struct {
			name string
			on   bool
		}{{"Required", f.Required}, {"ReadOnly", f.ReadOnly}, {"WriteOnly", f.WriteOnly}, {"Default", f.Default},
			{"PathOnly", f.PathOnly}, {"Stable", f.Stable}, {"Sensitive", f.Sensitive}} {
			if flag.on {
				fmt.Fprintf(b, ", %s: true", flag.name)
			}
		}
		if len(f.Fields) > 0 {
			b.WriteString(", Fields: ")
			writeFields(b, f.Fields)
		}
		b.WriteString("},\n")
	}
	b.WriteString("}")
}
