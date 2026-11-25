// Package provider is the Kayra provider for OpenTofu and Terraform: a
// resource per object of Kayra's (namespaces, API registrations,
// subscriptions, the RBAC, service accounts, personal access tokens), its
// attributes generated from Kayra's contracts, internal/contracts, one per
// property (zz_generated.go).
package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// kayraProvider is the provider, kayra.
type kayraProvider struct {
	version string
}

var _ provider.Provider = (*kayraProvider)(nil)

// New is the provider at version.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &kayraProvider{version: version} }
}

type config struct {
	URL       types.String `tfsdk:"url"`
	Token     types.String `tfsdk:"token"`
	TokenFile types.String `tfsdk:"token_file"`
}

func (p *kayraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName, resp.Version = "kayra", p.version
}

func (p *kayraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Kayra: namespaces, API registrations and subscriptions, the RBAC, service accounts and personal access tokens.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{Optional: true,
				MarkdownDescription: "Kayra's URL, `https://kayra.example.com`; `KAYRA_URL` when unset. Its groups are discovered at `/.well-known/kayra.json`."},
			"token": schema.StringAttribute{Optional: true, Sensitive: true,
				MarkdownDescription: "A token of Kayra's: a personal access token, or a service account's; `KAYRA_TOKEN` when unset."},
			"token_file": schema.StringAttribute{Optional: true,
				MarkdownDescription: "A file holding the token, read at each run; `KAYRA_TOKEN_FILE` when unset. `token` wins."},
		},
	}
}

func (p *kayraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var c config
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	or := func(v types.String, env string) string {
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			return v.ValueString()
		}
		return os.Getenv(env)
	}
	url, token := or(c.URL, "KAYRA_URL"), or(c.Token, "KAYRA_TOKEN")
	if url == "" {
		resp.Diagnostics.AddError("No Kayra URL", "Set url in the provider's configuration, or KAYRA_URL.")
		return
	}
	if file := or(c.TokenFile, "KAYRA_TOKEN_FILE"); token == "" && file != "" {
		data, err := os.ReadFile(file) //#nosec G304 G703 -- the file the configuration names
		if err != nil {
			resp.Diagnostics.AddError("Reading the token file", err.Error())
			return
		}
		token = strings.TrimSpace(string(data))
	}
	cl, err := newClient(ctx, url, token)
	if err != nil {
		resp.Diagnostics.AddError("Connecting to Kayra", err.Error())
		return
	}
	resp.ResourceData, resp.DataSourceData = cl, cl
}

func (p *kayraProvider) Resources(context.Context) []func() resource.Resource {
	var out []func() resource.Resource
	for _, d := range resourceDefs {
		out = append(out, func() resource.Resource { return &kayraResource{d: d} })
	}
	return out
}

func (p *kayraProvider) DataSources(context.Context) []func() datasource.DataSource {
	var out []func() datasource.DataSource
	for _, d := range dataSourceDefs {
		out = append(out, func() datasource.DataSource { return &kayraDataSource{d: d} })
	}
	return append(out, func() datasource.DataSource { return &specDataSource{} })
}
