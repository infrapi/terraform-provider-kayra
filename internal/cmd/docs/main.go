// Command docs writes the registry's documentation, docs/, from the
// provider's schemas and examples/, in tfplugindocs' layout: docs/index.md,
// docs/resources/<name>.md and docs/data-sources/<name>.md.
//
//	go run ./internal/cmd/docs
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/infrapi/terraform-provider-kayra/internal/provider"
)

func main() {
	if err := run("docs", "examples"); err != nil {
		fmt.Fprintln(os.Stderr, "docs:", err)
		os.Exit(1)
	}
}

func run(out, examples string) error {
	srv, err := providerserver.NewProtocol6WithError(provider.New("docs")())()
	if err != nil {
		return err
	}
	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		return err
	}
	if len(resp.Diagnostics) > 0 {
		return fmt.Errorf("%s: %s", resp.Diagnostics[0].Summary, resp.Diagnostics[0].Detail)
	}
	pages := map[string]string{
		"index.md": index(resp.Provider, examples),
	}
	for name, s := range resp.ResourceSchemas {
		pages[filepath.Join("resources", strings.TrimPrefix(name, "kayra_")+".md")] =
			page(name, "Resource", s, filepath.Join(examples, "resources", name, "resource.tf"), filepath.Join(examples, "resources", name, "import.sh"))
	}
	for name, s := range resp.DataSourceSchemas {
		pages[filepath.Join("data-sources", strings.TrimPrefix(name, "kayra_")+".md")] =
			page(name, "Data Source", s, filepath.Join(examples, "data-sources", name, "data-source.tf"), "")
	}
	for rel, content := range pages {
		path := filepath.Join(out, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //#nosec G301 -- docs
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //#nosec G306 -- docs
			return err
		}
	}
	return nil
}

func read(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path) //#nosec G304 -- the examples
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func index(s *tfprotov6.Schema, examples string) string {
	var b strings.Builder
	desc := s.Block.Description
	fmt.Fprintf(&b, "---\npage_title: \"Provider: Kayra\"\ndescription: |-\n  %s\n---\n\n# Kayra Provider\n\n%s\n\n", desc, desc)
	b.WriteString("Kayra is InfraPI's identity, namespaces, RBAC and API catalog: " +
		"<https://infrapi.github.io/products/kayra/>. Every attribute of a resource is a property of Kayra's object, " +
		"as its contract (OpenAPI) describes it, so a plan shows a change per property.\n\n")
	b.WriteString("The provider calls Kayra with a token of Kayra's: a personal access token " +
		"(`infrapi-cli create tokens`), or a service account's, a CI job's say. Its rights are its user's, " +
		"decided by Kayra's RBAC.\n\n")
	if ex := read(filepath.Join(examples, "provider", "provider.tf")); ex != "" {
		fmt.Fprintf(&b, "## Example Usage\n\n```terraform\n%s\n```\n\n", ex)
	}
	b.WriteString("## Schema\n\n")
	attributes(&b, s.Block.Attributes, "")
	return b.String()
}

func page(name, kind string, s *tfprotov6.Schema, example, imp string) string {
	var b strings.Builder
	desc := strings.TrimSpace(s.Block.Description)
	fmt.Fprintf(&b, "---\npage_title: \"%s %s - kayra\"\nsubcategory: \"\"\ndescription: |-\n  %s\n---\n\n# %s (%s)\n\n%s\n\n",
		name, kind, desc, name, kind, desc)
	if ex := read(example); ex != "" {
		fmt.Fprintf(&b, "## Example Usage\n\n```terraform\n%s\n```\n\n", ex)
	}
	b.WriteString("## Schema\n\n")
	attributes(&b, s.Block.Attributes, "")
	if ex := read(imp); ex != "" {
		fmt.Fprintf(&b, "## Import\n\nImport is supported using the following syntax:\n\n```shell\n%s\n```\n", ex)
	}
	return b.String()
}

// attributes writes attrs by Required, Optional and Read-Only, then their
// nested schemas.
func attributes(b *strings.Builder, attrs []*tfprotov6.SchemaAttribute, parent string) {
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Name < attrs[j].Name })
	var nested []*tfprotov6.SchemaAttribute
	for _, group := range []struct {
		title string
		is    func(*tfprotov6.SchemaAttribute) bool
	}{
		{"Required", func(a *tfprotov6.SchemaAttribute) bool { return a.Required }},
		{"Optional", func(a *tfprotov6.SchemaAttribute) bool { return a.Optional }},
		{"Read-Only", func(a *tfprotov6.SchemaAttribute) bool { return a.Computed && !a.Optional && !a.Required }},
	} {
		var lines []string
		for _, a := range attrs {
			if !group.is(a) {
				continue
			}
			typ := typeName(a)
			line := fmt.Sprintf("- `%s` (%s", a.Name, typ)
			if a.Sensitive {
				line += ", Sensitive"
			}
			line += ")"
			if a.NestedType != nil {
				anchor := "nestedatt--" + strings.TrimPrefix(parent+"--"+a.Name, "--")
				line += fmt.Sprintf(" %s (see [below for nested schema](#%s))", a.Description, anchor)
				nested = append(nested, a)
			} else if a.Description != "" {
				line += " " + a.Description
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			fmt.Fprintf(b, "### %s\n\n%s\n\n", group.title, strings.Join(lines, "\n"))
		}
	}
	for _, a := range nested {
		id := strings.TrimPrefix(parent+"--"+a.Name, "--")
		label := strings.ReplaceAll(id, "--", ".")
		fmt.Fprintf(b, "<a id=\"nestedatt--%s\"></a>\n### Nested Schema for `%s`\n\n", id, label)
		attributes(b, a.NestedType.Attributes, parent+"--"+a.Name)
	}
}

func typeName(a *tfprotov6.SchemaAttribute) string {
	if a.NestedType != nil {
		switch a.NestedType.Nesting {
		case tfprotov6.SchemaObjectNestingModeList:
			return "Attributes List"
		case tfprotov6.SchemaObjectNestingModeSingle:
			return "Attributes"
		}
		return "Attributes"
	}
	switch t := a.Type; {
	case t.Is(tftypes.String):
		return "String"
	case t.Is(tftypes.Bool):
		return "Boolean"
	case t.Is(tftypes.Number):
		return "Number"
	case t.Is(tftypes.List{ElementType: tftypes.String}):
		return "List of String"
	case t.Is(tftypes.Map{ElementType: tftypes.String}):
		return "Map of String"
	}
	return a.Type.String()
}
